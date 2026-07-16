package workflowruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"agent-manager/internal/structuredresult"

	"github.com/google/uuid"
)

var ErrConcurrentAdvance = errors.New("workflow execution advanced concurrently")

type (
	Catalog interface {
		GetByDigest(context.Context, string) (*domain.WorkflowRevision, error)
	}
	ChildRequest struct {
		ExecutionID    uuid.UUID
		AttemptID      uuid.UUID
		NodeID         string
		IdempotencyKey string
		ProfileKey     string
		RoleRef        string
		Prompt         string
		ResultSpec     *domain.ResultSpec
		SourceRunID    *uuid.UUID
	}
	ChildState struct {
		RunID          uuid.UUID
		ConversationID string
		Terminal       bool
		Failed         bool
		Result         *domain.RunResult
		Turns          int
		Tokens         int
		CostUSD        float64
	}
	ChildLauncher interface {
		StartFresh(context.Context, ChildRequest) (ChildState, error)
		Continue(context.Context, ChildRequest) (ChildState, error)
		Inspect(context.Context, uuid.UUID) (ChildState, error)
	}
	SubworkflowRequest struct {
		ParentExecutionID uuid.UUID
		ParentAttemptID   uuid.UUID
		Owner             string
		WorkflowKey       string
		Version           string
		Input             json.RawMessage
		IdempotencyKey    string
		Depth             int
	}
	SubworkflowState struct {
		ExecutionID uuid.UUID
		Terminal    bool
		Failed      bool
		Output      json.RawMessage
		BudgetUsage domain.WorkflowBudgetUsage
	}
	SubworkflowLauncher interface {
		Start(context.Context, SubworkflowRequest) (SubworkflowState, error)
		Inspect(context.Context, uuid.UUID) (SubworkflowState, error)
		Cancel(context.Context, uuid.UUID, string) error
	}
	Clock func() time.Time
)

type Engine struct {
	Store        repository.WorkflowExecutionRepository
	Catalog      Catalog
	Children     ChildLauncher
	Subworkflows SubworkflowLauncher
	Expressions  *ExpressionEvaluator
	Now          Clock
}

func (e *Engine) Start(ctx context.Context, revision *domain.WorkflowRevision, input json.RawMessage, idempotencyKey string) (*domain.WorkflowExecution, error) {
	return e.start(ctx, revision, input, idempotencyKey, nil, nil, 0)
}

func (e *Engine) StartChild(ctx context.Context, revision *domain.WorkflowRevision, input json.RawMessage, idempotencyKey string, parentExecutionID, parentAttemptID uuid.UUID, depth int) (*domain.WorkflowExecution, error) {
	return e.start(ctx, revision, input, idempotencyKey, &parentExecutionID, &parentAttemptID, depth)
}

func (e *Engine) start(ctx context.Context, revision *domain.WorkflowRevision, input json.RawMessage, idempotencyKey string, parentExecutionID, parentAttemptID *uuid.UUID, depth int) (*domain.WorkflowExecution, error) {
	if e.Store == nil || revision == nil {
		return nil, errors.New("workflow store and revision are required")
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return nil, errors.New("idempotency key is required")
	}
	if existing, err := e.Store.GetByIdempotencyKey(ctx, idempotencyKey); err != nil || existing != nil {
		return existing, err
	}
	if err := structuredresult.ValidateValue(revision.Definition.InputSchema, input); err != nil {
		return nil, fmt.Errorf("workflow input: %w", err)
	}
	now := e.now()
	execution := &domain.WorkflowExecution{ID: uuid.New(), Owner: revision.Owner, WorkflowKey: revision.Key, DefinitionDigest: revision.Digest, Status: domain.WorkflowExecutionRunning, CurrentNodeID: revision.Definition.EntryNode, Input: append(json.RawMessage(nil), input...), EdgeTraversals: map[string]int{}, Version: 1, IdempotencyKey: idempotencyKey, ParentExecutionID: parentExecutionID, ParentAttemptID: parentAttemptID, Depth: depth, CreatedAt: now, UpdatedAt: now}
	entry := &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: execution.ID, Sequence: 1, Kind: domain.WorkflowJournalInput, Payload: append(json.RawMessage(nil), input...), CreatedAt: now}
	if err := e.Store.Create(ctx, execution, entry); err != nil {
		if existing, getErr := e.Store.GetByIdempotencyKey(ctx, idempotencyKey); getErr == nil && existing != nil {
			return existing, nil
		}
		return nil, err
	}
	return execution, nil
}

func (e *Engine) Advance(ctx context.Context, id uuid.UUID) (*domain.WorkflowExecution, error) {
	execution, err := e.Store.Get(ctx, id)
	if err != nil || execution == nil || execution.Status.Terminal() {
		return execution, err
	}
	revision, err := e.Catalog.GetByDigest(ctx, execution.DefinitionDigest)
	if err != nil || revision == nil {
		return e.fail(ctx, execution, "definition_missing", "pinned workflow revision is unavailable")
	}
	if e.now().Sub(execution.CreatedAt) > time.Duration(revision.Definition.Budgets.WallTimeSeconds)*time.Second {
		return e.exhaust(ctx, execution, "wall_time")
	}
	node := findNode(revision.Definition.Nodes, execution.CurrentNodeID)
	if node == nil {
		return e.fail(ctx, execution, "node_missing", "current node is absent from pinned definition")
	}
	switch node.Kind {
	case domain.WorkflowNodeRun, domain.WorkflowNodeContinue:
		return e.advanceAgent(ctx, execution, revision, node)
	case domain.WorkflowNodeChild:
		return e.advanceChild(ctx, execution, revision, node)
	case domain.WorkflowNodeBranch:
		if node.Branch.Parallel {
			return e.advanceParallelBranch(ctx, execution, revision, node)
		}
		return e.advanceBranch(ctx, execution, revision, node)
	case domain.WorkflowNodeWait:
		return e.advanceWait(ctx, execution, revision, node)
	case domain.WorkflowNodeJoin:
		return e.advanceJoin(ctx, execution, revision, node)
	case domain.WorkflowNodeEnd:
		return e.advanceEnd(ctx, execution, revision, node)
	default:
		return e.fail(ctx, execution, "unsupported_node", "node kind is not implemented by core interpreter")
	}
}

type waitIntent struct {
	Signal         string          `json:"signal"`
	CorrelationKey string          `json:"correlationKey"`
	ResumeToken    string          `json:"resumeToken"`
	Deadline       time.Time       `json:"deadline"`
	PayloadSchema  json.RawMessage `json:"payloadSchema,omitempty"`
}

type signalRecord struct {
	Signal         string          `json:"signal"`
	CorrelationKey string          `json:"correlationKey"`
	IdempotencyKey string          `json:"idempotencyKey"`
	Payload        json.RawMessage `json:"payload"`
}

func (e *Engine) advanceWait(ctx context.Context, x *domain.WorkflowExecution, r *domain.WorkflowRevision, node *domain.WorkflowNode) (*domain.WorkflowExecution, error) {
	journal, err := e.Store.ListJournal(ctx, x.ID, 0, 0)
	if err != nil {
		return nil, err
	}
	var intent *waitIntent
	var waitSequence int64
	for _, entry := range journal {
		if entry.NodeID != node.ID || entry.Kind != domain.WorkflowJournalWait {
			continue
		}
		var candidate waitIntent
		if json.Unmarshal(entry.Payload, &candidate) == nil {
			intent, waitSequence = &candidate, entry.Sequence
		}
	}
	if intent == nil {
		limit := node.Wait.TimeoutSeconds
		if limit > r.Definition.Budgets.MaxWaitSeconds {
			limit = r.Definition.Budgets.MaxWaitSeconds
		}
		now := e.now()
		intent = &waitIntent{Signal: node.Wait.Signal, CorrelationKey: fmt.Sprintf("workflow/%s/wait/%s", x.ID, node.ID), ResumeToken: uuid.NewString(), Deadline: now.Add(time.Duration(limit) * time.Second), PayloadSchema: append(json.RawMessage(nil), node.Wait.PayloadSchema...)}
		payload, _ := json.Marshal(intent)
		x.Status = domain.WorkflowExecutionWaiting
		x.Version++
		x.UpdatedAt = now
		entry := nextJournal(x.ID, journal, domain.WorkflowJournalWait, node.ID, nil, payload, now)
		if ok, commitErr := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Journal: []*domain.WorkflowJournalEntry{entry}}); commitErr != nil {
			return nil, commitErr
		} else if !ok {
			return nil, ErrConcurrentAdvance
		}
		return x, nil
	}
	for _, entry := range journal {
		if entry.Sequence <= waitSequence || entry.NodeID != node.ID || entry.Kind != domain.WorkflowJournalSignal {
			continue
		}
		var signal signalRecord
		if json.Unmarshal(entry.Payload, &signal) == nil && signal.Signal == intent.Signal && signal.CorrelationKey == intent.CorrelationKey {
			next, edgeErr := selectUnconditionalEdge(r.Definition.Edges, node.ID)
			if edgeErr != nil {
				return e.fail(ctx, x, "edge_invalid", edgeErr.Error())
			}
			if edgeErr = takeEdge(x, next, r.Definition.Budgets); edgeErr != nil {
				return e.exhaust(ctx, x, "edge_traversal")
			}
			now := e.now()
			x.CurrentNodeID, x.Status, x.UpdatedAt = next.To, domain.WorkflowExecutionRunning, now
			x.Version++
			return e.commitOnly(ctx, x)
		}
	}
	if !e.now().Before(intent.Deadline) {
		return e.fail(ctx, x, "wait_timeout", "external signal deadline elapsed")
	}
	x.Status = domain.WorkflowExecutionWaiting
	return x, nil
}

func (e *Engine) advanceJoin(ctx context.Context, x *domain.WorkflowExecution, r *domain.WorkflowRevision, node *domain.WorkflowNode) (*domain.WorkflowExecution, error) {
	next, err := selectUnconditionalEdge(r.Definition.Edges, node.ID)
	if err != nil {
		return e.fail(ctx, x, "join_invalid", err.Error())
	}
	if err := takeEdge(x, next, r.Definition.Budgets); err != nil {
		return e.exhaust(ctx, x, "edge_traversal")
	}
	journal, err := e.Store.ListJournal(ctx, x.ID, 0, 0)
	if err != nil {
		return nil, err
	}
	now := e.now()
	payload, _ := json.Marshal(map[string]any{"strategy": node.Join.Strategy, "next": next.To})
	entry := nextJournal(x.ID, journal, domain.WorkflowJournalJoin, node.ID, nil, payload, now)
	x.CurrentNodeID, x.Status, x.UpdatedAt = next.To, domain.WorkflowExecutionRunning, now
	x.Version++
	if ok, commitErr := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Journal: []*domain.WorkflowJournalEntry{entry}}); commitErr != nil {
		return nil, commitErr
	} else if !ok {
		return nil, ErrConcurrentAdvance
	}
	return x, nil
}

func (e *Engine) Signal(ctx context.Context, id uuid.UUID, signal string, payload json.RawMessage, idempotencyKey string, expectedVersion int64) (*domain.WorkflowExecution, bool, error) {
	x, err := e.Store.Get(ctx, id)
	if err != nil || x == nil {
		return x, false, err
	}
	journal, err := e.Store.ListJournal(ctx, id, 0, 0)
	if err != nil {
		return nil, false, err
	}
	for _, entry := range journal {
		if entry.Kind != domain.WorkflowJournalSignal {
			continue
		}
		var prior signalRecord
		if json.Unmarshal(entry.Payload, &prior) == nil && prior.IdempotencyKey == idempotencyKey {
			return x, true, nil
		}
	}
	if x.Status.Terminal() || x.Status != domain.WorkflowExecutionWaiting {
		return x, false, fmt.Errorf("workflow execution is not waiting")
	}
	if expectedVersion > 0 && x.Version != expectedVersion {
		return x, false, ErrConcurrentAdvance
	}
	revision, err := e.Catalog.GetByDigest(ctx, x.DefinitionDigest)
	if err != nil || revision == nil {
		return x, false, fmt.Errorf("pinned workflow revision is unavailable")
	}
	node := findNode(revision.Definition.Nodes, x.CurrentNodeID)
	if node == nil || node.Wait == nil || node.Wait.Signal != signal {
		return x, false, fmt.Errorf("signal does not match the active wait contract")
	}
	if len(node.Wait.PayloadSchema) > 0 {
		if err := structuredresult.ValidateValue(node.Wait.PayloadSchema, payload); err != nil {
			return x, false, fmt.Errorf("signal payload: %w", err)
		}
	}
	var intent waitIntent
	found := false
	for i := len(journal) - 1; i >= 0; i-- {
		if journal[i].Kind == domain.WorkflowJournalWait && journal[i].NodeID == node.ID && json.Unmarshal(journal[i].Payload, &intent) == nil {
			found = true
			break
		}
	}
	if !found || intent.Signal != signal {
		return x, false, fmt.Errorf("durable wait intent is unavailable")
	}
	record := signalRecord{Signal: signal, CorrelationKey: intent.CorrelationKey, IdempotencyKey: idempotencyKey, Payload: append(json.RawMessage(nil), payload...)}
	data, _ := json.Marshal(record)
	now := e.now()
	entry := nextJournal(x.ID, journal, domain.WorkflowJournalSignal, node.ID, nil, data, now)
	x.Status, x.UpdatedAt = domain.WorkflowExecutionRunning, now
	x.Version++
	if ok, commitErr := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Journal: []*domain.WorkflowJournalEntry{entry}}); commitErr != nil {
		return nil, false, commitErr
	} else if !ok {
		return nil, false, ErrConcurrentAdvance
	}
	return x, false, nil
}

func (e *Engine) Cancel(ctx context.Context, id uuid.UUID, idempotencyKey, reason string, expectedVersion int64) (*domain.WorkflowExecution, bool, error) {
	x, err := e.Store.Get(ctx, id)
	if err != nil || x == nil {
		return x, false, err
	}
	journal, err := e.Store.ListJournal(ctx, id, 0, 0)
	if err != nil {
		return nil, false, err
	}
	if journalHasOperation(journal, domain.WorkflowJournalCancel, idempotencyKey) {
		return x, true, nil
	}
	if x.Status.Terminal() {
		return x, false, fmt.Errorf("workflow execution is already terminal")
	}
	if expectedVersion > 0 && x.Version != expectedVersion {
		return x, false, ErrConcurrentAdvance
	}
	now := e.now()
	payload, _ := json.Marshal(map[string]any{"idempotencyKey": idempotencyKey, "reason": reason})
	entry := nextJournal(x.ID, journal, domain.WorkflowJournalCancel, x.CurrentNodeID, nil, payload, now)
	x.Status = domain.WorkflowExecutionCancelled
	x.TerminalReason = &domain.WorkflowTerminalReason{Code: "cancelled", Message: reason}
	x.EndedAt, x.UpdatedAt = &now, now
	x.Version++
	if ok, commitErr := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Journal: []*domain.WorkflowJournalEntry{entry}}); commitErr != nil {
		return nil, false, commitErr
	} else if !ok {
		return nil, false, ErrConcurrentAdvance
	}
	return x, false, nil
}

func (e *Engine) Retry(ctx context.Context, id uuid.UUID, idempotencyKey string, expectedVersion int64) (*domain.WorkflowExecution, bool, error) {
	x, err := e.Store.Get(ctx, id)
	if err != nil || x == nil {
		return x, false, err
	}
	journal, err := e.Store.ListJournal(ctx, id, 0, 0)
	if err != nil {
		return nil, false, err
	}
	if journalHasOperation(journal, domain.WorkflowJournalRetry, idempotencyKey) {
		return x, true, nil
	}
	if x.Status != domain.WorkflowExecutionFailed {
		return x, false, fmt.Errorf("only a failed workflow execution can be retried")
	}
	if expectedVersion > 0 && x.Version != expectedVersion {
		return x, false, ErrConcurrentAdvance
	}
	revision, err := e.Catalog.GetByDigest(ctx, x.DefinitionDigest)
	if err != nil || revision == nil {
		return x, false, fmt.Errorf("pinned workflow revision is unavailable")
	}
	if x.BudgetUsage.Retries >= revision.Definition.Budgets.MaxRetries {
		return x, false, fmt.Errorf("workflow retry budget exhausted")
	}
	now := e.now()
	payload, _ := json.Marshal(map[string]any{"idempotencyKey": idempotencyKey})
	entry := nextJournal(x.ID, journal, domain.WorkflowJournalRetry, x.CurrentNodeID, nil, payload, now)
	x.Status, x.TerminalReason, x.EndedAt, x.UpdatedAt = domain.WorkflowExecutionRunning, nil, nil, now
	x.BudgetUsage.Retries++
	x.Version++
	if ok, commitErr := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Journal: []*domain.WorkflowJournalEntry{entry}}); commitErr != nil {
		return nil, false, commitErr
	} else if !ok {
		return nil, false, ErrConcurrentAdvance
	}
	return x, false, nil
}

func (e *Engine) Resume(ctx context.Context, id uuid.UUID, idempotencyKey string, expectedVersion int64) (*domain.WorkflowExecution, bool, error) {
	x, err := e.Store.Get(ctx, id)
	if err != nil || x == nil {
		return x, false, err
	}
	journal, err := e.Store.ListJournal(ctx, id, 0, 0)
	if err != nil {
		return nil, false, err
	}
	if journalHasOperation(journal, domain.WorkflowJournalResume, idempotencyKey) {
		return x, true, nil
	}
	if x.Status.Terminal() {
		return x, false, fmt.Errorf("workflow execution is terminal")
	}
	if expectedVersion > 0 && x.Version != expectedVersion {
		return x, false, ErrConcurrentAdvance
	}
	now := e.now()
	payload, _ := json.Marshal(map[string]any{"idempotencyKey": idempotencyKey})
	entry := nextJournal(x.ID, journal, domain.WorkflowJournalResume, x.CurrentNodeID, nil, payload, now)
	x.UpdatedAt = now
	x.Version++
	if ok, commitErr := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Journal: []*domain.WorkflowJournalEntry{entry}}); commitErr != nil {
		return nil, false, commitErr
	} else if !ok {
		return nil, false, ErrConcurrentAdvance
	}
	return x, false, nil
}

func (e *Engine) RecordCancellationDisposition(ctx context.Context, id uuid.UUID, stoppedRuns, stoppedWorkflows int, failures []string) (*domain.WorkflowExecution, error) {
	x, err := e.Store.Get(ctx, id)
	if err != nil || x == nil {
		return x, err
	}
	if x.Status != domain.WorkflowExecutionCancelled {
		return x, fmt.Errorf("cancellation disposition requires cancelled execution")
	}
	journal, err := e.Store.ListJournal(ctx, id, 0, 0)
	if err != nil {
		return nil, err
	}
	now := e.now()
	payload, _ := json.Marshal(map[string]any{"stoppedRuns": stoppedRuns, "stoppedWorkflows": stoppedWorkflows, "failures": failures})
	entry := nextJournal(x.ID, journal, domain.WorkflowJournalCancel, x.CurrentNodeID, nil, payload, now)
	if len(failures) > 0 {
		x.TerminalReason.Message = strings.TrimSpace(x.TerminalReason.Message + "; cancellation incomplete: " + strings.Join(failures, "; "))
	}
	x.UpdatedAt = now
	x.Version++
	if ok, commitErr := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Journal: []*domain.WorkflowJournalEntry{entry}}); commitErr != nil {
		return nil, commitErr
	} else if !ok {
		return nil, ErrConcurrentAdvance
	}
	return x, nil
}

func journalHasOperation(journal []*domain.WorkflowJournalEntry, kind domain.WorkflowJournalKind, key string) bool {
	for _, entry := range journal {
		if entry.Kind != kind {
			continue
		}
		var value map[string]any
		if json.Unmarshal(entry.Payload, &value) == nil && value["idempotencyKey"] == key {
			return true
		}
	}
	return false
}

func (e *Engine) advanceAgent(ctx context.Context, x *domain.WorkflowExecution, r *domain.WorkflowRevision, node *domain.WorkflowNode) (*domain.WorkflowExecution, error) {
	attempts, err := e.Store.ListAttempts(ctx, x.ID)
	if err != nil {
		return nil, err
	}
	var active *domain.WorkflowNodeAttempt
	ordinal := 1
	for _, a := range attempts {
		if a.NodeID == node.ID {
			if a.Ordinal >= ordinal {
				ordinal = a.Ordinal + 1
			}
			if a.Status != domain.WorkflowAttemptCompleted && a.Status != domain.WorkflowAttemptFailed {
				active = a
			}
		}
	}
	journal, err := e.Store.ListJournal(ctx, x.ID, 0, 0)
	if err != nil {
		return nil, err
	}
	if active == nil {
		if x.BudgetUsage.NodeAttempts >= r.Definition.Budgets.MaxNodeAttempts {
			return e.exhaust(ctx, x, "node_attempts")
		}
		if x.BudgetUsage.Children >= r.Definition.Budgets.MaxChildren {
			return e.exhaust(ctx, x, "children")
		}
		bindings, prompt, spec, strategy, source, err := e.resolveAgentInput(node, attempts, journal, x.Input)
		if err != nil {
			return e.fail(ctx, x, "binding_invalid", err.Error())
		}
		now := e.now()
		active = &domain.WorkflowNodeAttempt{ID: uuid.New(), ExecutionID: x.ID, NodeID: node.ID, Ordinal: ordinal, Strategy: strategy, Status: domain.WorkflowAttemptDispatchPending, IdempotencyKey: fmt.Sprintf("workflow/%s/node/%s/attempt/%d", x.ID, node.ID, ordinal), InputSnapshot: bindings, PromptSnapshot: prompt, SourceAttemptID: source, Version: 1, CreatedAt: now, UpdatedAt: now}
		x.BudgetUsage.NodeAttempts++
		x.Version++
		x.UpdatedAt = now
		payload, _ := json.Marshal(map[string]any{"nodeId": node.ID, "ordinal": ordinal, "strategy": strategy})
		entry := nextJournal(x.ID, journal, domain.WorkflowJournalAttempt, node.ID, &active.ID, payload, now)
		if ok, err := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempt: active, Journal: []*domain.WorkflowJournalEntry{entry}}); err != nil {
			return nil, err
		} else if !ok {
			return nil, ErrConcurrentAdvance
		}
		_ = spec
		return x, nil
	}
	request, err := e.childRequest(node, x, active, attempts, journal)
	if err != nil {
		return e.fail(ctx, x, "dispatch_invalid", err.Error())
	}
	if active.Status == domain.WorkflowAttemptDispatchPending {
		var state ChildState
		if active.Strategy == domain.WorkflowAttemptFreshRun {
			state, err = e.Children.StartFresh(ctx, request)
		} else {
			state, err = e.Children.Continue(ctx, request)
		}
		if err != nil {
			return x, err
		}
		now := e.now()
		active.RunID = &state.RunID
		active.ConversationID = state.ConversationID
		active.Status = domain.WorkflowAttemptDispatched
		active.Version++
		active.UpdatedAt = now
		x.Status = domain.WorkflowExecutionWaiting
		x.BudgetUsage.Children++
		x.Version++
		x.UpdatedAt = now
		if ok, commitErr := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempt: active}); commitErr != nil {
			return nil, commitErr
		} else if !ok {
			return nil, ErrConcurrentAdvance
		}
		return x, nil
	}
	if active.RunID == nil {
		return e.fail(ctx, x, "child_identity_missing", "dispatched attempt has no Run id")
	}
	state, err := e.Children.Inspect(ctx, *active.RunID)
	if err != nil {
		return x, err
	}
	if !state.Terminal {
		return x, nil
	}
	now := e.now()
	active.Status = domain.WorkflowAttemptCompleted
	if state.Failed {
		active.Status = domain.WorkflowAttemptFailed
		active.ErrorCode = "child_failed"
	}
	active.Version++
	active.UpdatedAt = now
	active.CompletedAt = &now
	x.BudgetUsage.Turns += state.Turns
	x.BudgetUsage.Tokens += state.Tokens
	x.BudgetUsage.CostUSD += state.CostUSD
	entries := []*domain.WorkflowJournalEntry{}
	if state.Result != nil {
		payload, _ := json.Marshal(state.Result)
		entries = append(entries, nextJournal(x.ID, append(journal, entries...), domain.WorkflowJournalRunResult, node.ID, &active.ID, payload, now))
		if state.Result.Structured != nil {
			payload, _ = json.Marshal(state.Result.Structured)
			entries = append(entries, nextJournal(x.ID, append(journal, entries...), domain.WorkflowJournalStructured, node.ID, &active.ID, payload, now))
		}
		if state.Result.FinalOutput != "" {
			payload, _ = json.Marshal(map[string]any{"text": state.Result.FinalOutput})
			entries = append(entries, nextJournal(x.ID, append(journal, entries...), domain.WorkflowJournalHandoff, node.ID, &active.ID, payload, now))
		}
	}
	if state.Failed {
		return e.commitFailure(ctx, x, active, entries, "child_failed", "child Run failed")
	}
	if x.BudgetUsage.Turns > r.Definition.Budgets.MaxTurns {
		return e.commitExhaust(ctx, x, active, entries, "turns")
	}
	if x.BudgetUsage.Tokens > r.Definition.Budgets.MaxTokens {
		return e.commitExhaust(ctx, x, active, entries, "tokens")
	}
	if x.BudgetUsage.CostUSD > r.Definition.Budgets.MaxCostUSD {
		return e.commitExhaust(ctx, x, active, entries, "cost")
	}
	next, err := selectUnconditionalEdge(r.Definition.Edges, node.ID)
	if err != nil {
		return e.fail(ctx, x, "edge_invalid", err.Error())
	}
	if err := takeEdge(x, next, r.Definition.Budgets); err != nil {
		return e.exhaust(ctx, x, "edge_traversal")
	}
	x.CurrentNodeID = next.To
	x.Status = domain.WorkflowExecutionRunning
	x.Version++
	x.UpdatedAt = now
	if ok, err := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempt: active, Journal: entries}); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrConcurrentAdvance
	}
	return x, nil
}

func (e *Engine) advanceChild(ctx context.Context, x *domain.WorkflowExecution, r *domain.WorkflowRevision, node *domain.WorkflowNode) (*domain.WorkflowExecution, error) {
	if e.Subworkflows == nil {
		return e.fail(ctx, x, "subworkflow_unavailable", "subworkflow launcher is not configured")
	}
	if x.Depth+1 > r.Definition.Budgets.MaxRecursion || node.Child.MaxDepth < x.Depth+1 {
		return e.exhaust(ctx, x, "recursion")
	}
	attempts, err := e.Store.ListAttempts(ctx, x.ID)
	if err != nil {
		return nil, err
	}
	journal, err := e.Store.ListJournal(ctx, x.ID, 0, 0)
	if err != nil {
		return nil, err
	}
	var active *domain.WorkflowNodeAttempt
	ordinal := 1
	for _, attempt := range attempts {
		if attempt.NodeID != node.ID {
			continue
		}
		if attempt.Ordinal >= ordinal {
			ordinal = attempt.Ordinal + 1
		}
		if attempt.Status != domain.WorkflowAttemptCompleted && attempt.Status != domain.WorkflowAttemptFailed {
			active = attempt
		}
	}
	if active == nil {
		if x.BudgetUsage.NodeAttempts >= r.Definition.Budgets.MaxNodeAttempts {
			return e.exhaust(ctx, x, "node_attempts")
		}
		if x.BudgetUsage.Children >= r.Definition.Budgets.MaxChildren {
			return e.exhaust(ctx, x, "children")
		}
		values, evalErr := EvaluateBindings(node.Child.Bindings, BindingContext{Input: x.Input, Journal: journal})
		if evalErr != nil {
			return e.fail(ctx, x, "child_input_invalid", evalErr.Error())
		}
		input, marshalErr := json.Marshal(values)
		if marshalErr != nil {
			return nil, marshalErr
		}
		now := e.now()
		active = &domain.WorkflowNodeAttempt{ID: uuid.New(), ExecutionID: x.ID, NodeID: node.ID, Ordinal: ordinal, Strategy: domain.WorkflowAttemptChild, Status: domain.WorkflowAttemptDispatchPending, IdempotencyKey: fmt.Sprintf("workflow/%s/node/%s/attempt/%d", x.ID, node.ID, ordinal), InputSnapshot: input, Version: 1, CreatedAt: now, UpdatedAt: now}
		x.BudgetUsage.NodeAttempts++
		x.Version++
		x.UpdatedAt = now
		payload, _ := json.Marshal(map[string]any{"nodeId": node.ID, "ordinal": ordinal, "strategy": domain.WorkflowAttemptChild, "workflowKey": node.Child.WorkflowKey})
		entry := nextJournal(x.ID, journal, domain.WorkflowJournalAttempt, node.ID, &active.ID, payload, now)
		if ok, commitErr := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempt: active, Journal: []*domain.WorkflowJournalEntry{entry}}); commitErr != nil {
			return nil, commitErr
		} else if !ok {
			return nil, ErrConcurrentAdvance
		}
		return x, nil
	}
	if active.Status == domain.WorkflowAttemptDispatchPending {
		state, launchErr := e.Subworkflows.Start(ctx, SubworkflowRequest{ParentExecutionID: x.ID, ParentAttemptID: active.ID, Owner: x.Owner, WorkflowKey: node.Child.WorkflowKey, Version: node.Child.Version, Input: active.InputSnapshot, IdempotencyKey: active.IdempotencyKey, Depth: x.Depth + 1})
		if launchErr != nil {
			return x, launchErr
		}
		now := e.now()
		active.ChildExecutionID = &state.ExecutionID
		active.Status = domain.WorkflowAttemptDispatched
		active.Version++
		active.UpdatedAt = now
		x.Status = domain.WorkflowExecutionWaiting
		x.BudgetUsage.Children++
		x.Version++
		x.UpdatedAt = now
		payload, _ := json.Marshal(map[string]any{"childExecutionId": state.ExecutionID, "workflowKey": node.Child.WorkflowKey})
		entry := nextJournal(x.ID, journal, domain.WorkflowJournalChild, node.ID, &active.ID, payload, now)
		if ok, commitErr := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempt: active, Journal: []*domain.WorkflowJournalEntry{entry}}); commitErr != nil {
			return nil, commitErr
		} else if !ok {
			return nil, ErrConcurrentAdvance
		}
		return x, nil
	}
	if active.ChildExecutionID == nil {
		return e.fail(ctx, x, "child_identity_missing", "dispatched subworkflow has no execution id")
	}
	state, err := e.Subworkflows.Inspect(ctx, *active.ChildExecutionID)
	if err != nil {
		return x, err
	}
	if !state.Terminal {
		return x, nil
	}
	now := e.now()
	active.Status = domain.WorkflowAttemptCompleted
	if state.Failed {
		active.Status, active.ErrorCode = domain.WorkflowAttemptFailed, "child_workflow_failed"
	}
	active.Version++
	active.UpdatedAt, active.CompletedAt = now, &now
	x.BudgetUsage.Turns += state.BudgetUsage.Turns
	x.BudgetUsage.Tokens += state.BudgetUsage.Tokens
	x.BudgetUsage.CostUSD += state.BudgetUsage.CostUSD
	x.BudgetUsage.NodeAttempts += state.BudgetUsage.NodeAttempts
	x.BudgetUsage.Children += state.BudgetUsage.Children
	x.BudgetUsage.Retries += state.BudgetUsage.Retries
	payload, _ := json.Marshal(map[string]any{"childExecutionId": state.ExecutionID, "status": map[bool]string{true: "failed", false: "succeeded"}[state.Failed], "output": json.RawMessage(state.Output)})
	entry := nextJournal(x.ID, journal, domain.WorkflowJournalChild, node.ID, &active.ID, payload, now)
	if state.Failed {
		return e.commitFailure(ctx, x, active, []*domain.WorkflowJournalEntry{entry}, "child_workflow_failed", "child workflow failed")
	}
	if budget := exceededBudgetName(x.BudgetUsage, r.Definition.Budgets); budget != "" {
		return e.commitExhaust(ctx, x, active, []*domain.WorkflowJournalEntry{entry}, budget)
	}
	next, edgeErr := selectUnconditionalEdge(r.Definition.Edges, node.ID)
	if edgeErr != nil {
		return e.commitFailure(ctx, x, active, []*domain.WorkflowJournalEntry{entry}, "edge_invalid", edgeErr.Error())
	}
	if edgeErr = takeEdge(x, next, r.Definition.Budgets); edgeErr != nil {
		return e.commitExhaust(ctx, x, active, []*domain.WorkflowJournalEntry{entry}, "edge_traversal")
	}
	x.CurrentNodeID, x.Status, x.UpdatedAt = next.To, domain.WorkflowExecutionRunning, now
	x.Version++
	if ok, commitErr := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempt: active, Journal: []*domain.WorkflowJournalEntry{entry}}); commitErr != nil {
		return nil, commitErr
	} else if !ok {
		return nil, ErrConcurrentAdvance
	}
	return x, nil
}

func (e *Engine) resolveAgentInput(node *domain.WorkflowNode, attempts []*domain.WorkflowNodeAttempt, journal []*domain.WorkflowJournalEntry, input json.RawMessage) (json.RawMessage, string, *domain.ResultSpec, domain.WorkflowAttemptStrategy, *uuid.UUID, error) {
	var bindings []domain.WorkflowInputBinding
	var tmpl string
	var spec *domain.ResultSpec
	strategy := domain.WorkflowAttemptFreshRun
	var source *uuid.UUID
	if node.Run != nil {
		bindings = node.Run.Bindings
		tmpl = node.Run.PromptTemplate
		spec = node.Run.ResultSpec
	} else {
		strategy = domain.WorkflowAttemptContinue
		bindings = node.Continue.Bindings
		tmpl = node.Continue.PromptTemplate
		spec = node.Continue.ResultSpec
		for i := len(attempts) - 1; i >= 0; i-- {
			if attempts[i].NodeID == node.Continue.ConversationFromNode && attempts[i].Status == domain.WorkflowAttemptCompleted {
				v := attempts[i].ID
				source = &v
				break
			}
		}
		if source == nil {
			return nil, "", nil, "", nil, fmt.Errorf("explicit continuation source has no completed attempt")
		}
	}
	values, err := EvaluateBindings(bindings, BindingContext{Input: input, Journal: journal})
	if err != nil {
		return nil, "", nil, "", nil, err
	}
	snapshot, err := json.Marshal(values)
	if err != nil {
		return nil, "", nil, "", nil, err
	}
	prompt, err := RenderPrompt(tmpl, values)
	return snapshot, prompt, spec, strategy, source, err
}

func (e *Engine) childRequest(node *domain.WorkflowNode, x *domain.WorkflowExecution, a *domain.WorkflowNodeAttempt, attempts []*domain.WorkflowNodeAttempt, journal []*domain.WorkflowJournalEntry) (ChildRequest, error) {
	prompt := a.PromptSnapshot
	var spec *domain.ResultSpec
	if node.Run != nil {
		spec = node.Run.ResultSpec
	} else {
		spec = node.Continue.ResultSpec
	}
	request := ChildRequest{ExecutionID: x.ID, AttemptID: a.ID, NodeID: node.ID, IdempotencyKey: a.IdempotencyKey, Prompt: prompt, ResultSpec: spec}
	if node.Run != nil {
		request.ProfileKey = node.Run.ProfileKey
		request.RoleRef = node.Run.RoleRef
	} else {
		for _, candidate := range attempts {
			if candidate.ID == *a.SourceAttemptID {
				request.SourceRunID = candidate.RunID
				break
			}
		}
		if request.SourceRunID == nil {
			return ChildRequest{}, fmt.Errorf("continuation source Run is unavailable")
		}
	}
	return request, nil
}

func (e *Engine) advanceParallelBranch(ctx context.Context, x *domain.WorkflowExecution, r *domain.WorkflowRevision, fork *domain.WorkflowNode) (*domain.WorkflowExecution, error) {
	attempts, err := e.Store.ListAttempts(ctx, x.ID)
	if err != nil {
		return nil, err
	}
	journal, err := e.Store.ListJournal(ctx, x.ID, 0, 0)
	if err != nil {
		return nil, err
	}
	targets, join, err := parallelPlan(r.Definition, fork.ID)
	if err != nil {
		return e.fail(ctx, x, "parallel_invalid", err.Error())
	}
	byNode := map[string]*domain.WorkflowNodeAttempt{}
	for _, attempt := range attempts {
		if _, ok := targets[attempt.NodeID]; ok {
			byNode[attempt.NodeID] = attempt
		}
	}
	if len(byNode) == 0 {
		if len(targets) > r.Definition.Budgets.MaxConcurrency || x.BudgetUsage.NodeAttempts+len(targets) > r.Definition.Budgets.MaxNodeAttempts || x.BudgetUsage.Children+len(targets) > r.Definition.Budgets.MaxChildren {
			return e.exhaust(ctx, x, "parallelism")
		}
		now := e.now()
		created := make([]*domain.WorkflowNodeAttempt, 0, len(targets))
		entries := make([]*domain.WorkflowJournalEntry, 0, len(targets)+1)
		forkPayload, _ := json.Marshal(map[string]any{"joinNodeId": join.ID, "members": sortedTargetIDs(targets), "strategy": join.Join.Strategy, "quorum": join.Join.Quorum})
		entries = append(entries, nextJournal(x.ID, journal, domain.WorkflowJournalJoin, fork.ID, nil, forkPayload, now))
		for _, targetID := range sortedTargetIDs(targets) {
			target := targets[targetID]
			attempt, prepErr := e.prepareParallelAttempt(x, target, attempts, journal, now)
			if prepErr != nil {
				return e.fail(ctx, x, "parallel_member_invalid", prepErr.Error())
			}
			created = append(created, attempt)
			payload, _ := json.Marshal(map[string]any{"nodeId": targetID, "ordinal": 1, "strategy": attempt.Strategy, "parallelParent": fork.ID})
			entries = append(entries, nextJournal(x.ID, append(journal, entries...), domain.WorkflowJournalAttempt, targetID, &attempt.ID, payload, now))
			for _, edge := range r.Definition.Edges {
				if edge.From == fork.ID && edge.To == targetID {
					if takeErr := takeEdge(x, edge, r.Definition.Budgets); takeErr != nil {
						return e.exhaust(ctx, x, "edge_traversal")
					}
				}
			}
		}
		x.BudgetUsage.NodeAttempts += len(created)
		x.Version++
		x.UpdatedAt = now
		if ok, commitErr := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempts: created, Journal: entries}); commitErr != nil {
			return nil, commitErr
		} else if !ok {
			return nil, ErrConcurrentAdvance
		}
		return x, nil
	}
	inFlight := 0
	for _, attempt := range byNode {
		if attempt.Status == domain.WorkflowAttemptDispatched || attempt.Status == domain.WorkflowAttemptWaiting {
			inFlight++
		}
	}
	for _, targetID := range sortedTargetIDs(targets) {
		attempt := byNode[targetID]
		if attempt == nil || attempt.Status != domain.WorkflowAttemptDispatchPending || inFlight >= r.Definition.Budgets.MaxConcurrency {
			continue
		}
		target := targets[targetID]
		now := e.now()
		if attempt.Strategy == domain.WorkflowAttemptChild {
			state, launchErr := e.Subworkflows.Start(ctx, SubworkflowRequest{ParentExecutionID: x.ID, ParentAttemptID: attempt.ID, Owner: x.Owner, WorkflowKey: target.Child.WorkflowKey, Version: target.Child.Version, Input: attempt.InputSnapshot, IdempotencyKey: attempt.IdempotencyKey, Depth: x.Depth + 1})
			if launchErr != nil {
				return x, launchErr
			}
			attempt.ChildExecutionID = &state.ExecutionID
		} else {
			request, requestErr := e.childRequest(target, x, attempt, attempts, journal)
			if requestErr != nil {
				return e.fail(ctx, x, "parallel_dispatch_invalid", requestErr.Error())
			}
			var state ChildState
			if attempt.Strategy == domain.WorkflowAttemptFreshRun {
				state, err = e.Children.StartFresh(ctx, request)
			} else {
				state, err = e.Children.Continue(ctx, request)
			}
			if err != nil {
				return x, err
			}
			attempt.RunID, attempt.ConversationID = &state.RunID, state.ConversationID
		}
		attempt.Status, attempt.UpdatedAt = domain.WorkflowAttemptDispatched, now
		attempt.Version++
		x.BudgetUsage.Children++
		x.Status, x.UpdatedAt = domain.WorkflowExecutionRunning, now
		x.Version++
		if ok, commitErr := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempt: attempt}); commitErr != nil {
			return nil, commitErr
		} else if !ok {
			return nil, ErrConcurrentAdvance
		}
		return x, nil
	}
	for _, targetID := range sortedTargetIDs(targets) {
		attempt := byNode[targetID]
		if attempt == nil || attempt.Status != domain.WorkflowAttemptDispatched {
			continue
		}
		target := targets[targetID]
		terminal, failed := false, false
		usage := domain.WorkflowBudgetUsage{}
		var result *domain.RunResult
		var childOutput json.RawMessage
		if attempt.Strategy == domain.WorkflowAttemptChild {
			if attempt.ChildExecutionID == nil {
				return e.fail(ctx, x, "child_identity_missing", "parallel child workflow has no execution id")
			}
			state, inspectErr := e.Subworkflows.Inspect(ctx, *attempt.ChildExecutionID)
			if inspectErr != nil {
				return x, inspectErr
			}
			terminal, failed, usage, childOutput = state.Terminal, state.Failed, state.BudgetUsage, state.Output
		} else {
			if attempt.RunID == nil {
				return e.fail(ctx, x, "child_identity_missing", "parallel Run has no id")
			}
			state, inspectErr := e.Children.Inspect(ctx, *attempt.RunID)
			if inspectErr != nil {
				return x, inspectErr
			}
			terminal, failed, result = state.Terminal, state.Failed, state.Result
			usage = domain.WorkflowBudgetUsage{Turns: state.Turns, Tokens: state.Tokens, CostUSD: state.CostUSD}
		}
		if !terminal {
			continue
		}
		now := e.now()
		attempt.Status = domain.WorkflowAttemptCompleted
		if failed {
			attempt.Status, attempt.ErrorCode = domain.WorkflowAttemptFailed, "parallel_child_failed"
		}
		attempt.Version++
		attempt.UpdatedAt, attempt.CompletedAt = now, &now
		x.BudgetUsage.Turns += usage.Turns
		x.BudgetUsage.Tokens += usage.Tokens
		x.BudgetUsage.CostUSD += usage.CostUSD
		x.BudgetUsage.NodeAttempts += usage.NodeAttempts
		x.BudgetUsage.Children += usage.Children
		x.BudgetUsage.Retries += usage.Retries
		entries := []*domain.WorkflowJournalEntry{}
		if result != nil {
			data, _ := json.Marshal(result)
			entries = append(entries, nextJournal(x.ID, journal, domain.WorkflowJournalRunResult, targetID, &attempt.ID, data, now))
			if result.Structured != nil {
				data, _ = json.Marshal(result.Structured)
				entries = append(entries, nextJournal(x.ID, append(journal, entries...), domain.WorkflowJournalStructured, targetID, &attempt.ID, data, now))
			}
			if result.FinalOutput != "" {
				data, _ = json.Marshal(map[string]any{"text": result.FinalOutput})
				entries = append(entries, nextJournal(x.ID, append(journal, entries...), domain.WorkflowJournalHandoff, targetID, &attempt.ID, data, now))
			}
		}
		if len(childOutput) > 0 {
			data, _ := json.Marshal(map[string]any{"childExecutionId": attempt.ChildExecutionID, "output": childOutput})
			entries = append(entries, nextJournal(x.ID, journal, domain.WorkflowJournalChild, targetID, &attempt.ID, data, now))
		}
		memberEdge, edgeErr := selectUnconditionalEdge(r.Definition.Edges, target.ID)
		if edgeErr != nil || memberEdge.To != join.ID {
			return e.commitFailure(ctx, x, attempt, entries, "parallel_join_invalid", "parallel member does not terminate at declared join")
		}
		if edgeErr = takeEdge(x, memberEdge, r.Definition.Budgets); edgeErr != nil {
			return e.commitExhaust(ctx, x, attempt, entries, "edge_traversal")
		}
		if budget := exceededBudgetName(x.BudgetUsage, r.Definition.Budgets); budget != "" {
			return e.commitExhaust(ctx, x, attempt, entries, budget)
		}
		x.Version++
		x.UpdatedAt = now
		if ok, commitErr := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempt: attempt, Journal: entries}); commitErr != nil {
			return nil, commitErr
		} else if !ok {
			return nil, ErrConcurrentAdvance
		}
		return x, nil
	}
	completed, succeeded := 0, 0
	for _, attempt := range byNode {
		if attempt.Status == domain.WorkflowAttemptCompleted || attempt.Status == domain.WorkflowAttemptFailed {
			completed++
		}
		if attempt.Status == domain.WorkflowAttemptCompleted {
			succeeded++
		}
	}
	if completed < len(targets) {
		x.Status = domain.WorkflowExecutionWaiting
		return x, nil
	}
	required := len(targets)
	if join.Join.Strategy == "any" {
		required = 1
	} else if join.Join.Strategy == "quorum" {
		required = join.Join.Quorum
	}
	if succeeded < required {
		return e.fail(ctx, x, "join_unsatisfied", fmt.Sprintf("join required %d successful members, got %d", required, succeeded))
	}
	now := e.now()
	x.CurrentNodeID, x.Status, x.UpdatedAt = join.ID, domain.WorkflowExecutionRunning, now
	x.Version++
	return e.commitOnly(ctx, x)
}

func exceededBudgetName(usage domain.WorkflowBudgetUsage, budgets domain.WorkflowBudgets) string {
	switch {
	case usage.Turns > budgets.MaxTurns:
		return "turns"
	case usage.Tokens > budgets.MaxTokens:
		return "tokens"
	case usage.CostUSD > budgets.MaxCostUSD:
		return "cost"
	case usage.NodeAttempts > budgets.MaxNodeAttempts:
		return "node_attempts"
	case usage.Children > budgets.MaxChildren:
		return "children"
	case usage.Retries > budgets.MaxRetries:
		return "retries"
	default:
		return ""
	}
}

func (e *Engine) prepareParallelAttempt(x *domain.WorkflowExecution, node *domain.WorkflowNode, attempts []*domain.WorkflowNodeAttempt, journal []*domain.WorkflowJournalEntry, now time.Time) (*domain.WorkflowNodeAttempt, error) {
	idempotencyKey := fmt.Sprintf("workflow/%s/parallel/%s/attempt/1", x.ID, node.ID)
	if node.Kind == domain.WorkflowNodeChild {
		values, err := EvaluateBindings(node.Child.Bindings, BindingContext{Input: x.Input, Journal: journal})
		if err != nil {
			return nil, err
		}
		input, err := json.Marshal(values)
		if err != nil {
			return nil, err
		}
		return &domain.WorkflowNodeAttempt{ID: uuid.New(), ExecutionID: x.ID, NodeID: node.ID, Ordinal: 1, Strategy: domain.WorkflowAttemptChild, Status: domain.WorkflowAttemptDispatchPending, IdempotencyKey: idempotencyKey, InputSnapshot: input, Version: 1, CreatedAt: now, UpdatedAt: now}, nil
	}
	if node.Kind != domain.WorkflowNodeRun && node.Kind != domain.WorkflowNodeContinue {
		return nil, fmt.Errorf("parallel member %s must be run, continue, or child_workflow", node.ID)
	}
	input, prompt, _, strategy, source, err := e.resolveAgentInput(node, attempts, journal, x.Input)
	if err != nil {
		return nil, err
	}
	return &domain.WorkflowNodeAttempt{ID: uuid.New(), ExecutionID: x.ID, NodeID: node.ID, Ordinal: 1, Strategy: strategy, Status: domain.WorkflowAttemptDispatchPending, IdempotencyKey: idempotencyKey, InputSnapshot: input, PromptSnapshot: prompt, SourceAttemptID: source, Version: 1, CreatedAt: now, UpdatedAt: now}, nil
}

func parallelPlan(definition domain.WorkflowDefinition, forkID string) (map[string]*domain.WorkflowNode, *domain.WorkflowNode, error) {
	targets := map[string]*domain.WorkflowNode{}
	joinID := ""
	for _, edge := range definition.Edges {
		if edge.From != forkID {
			continue
		}
		target := findNode(definition.Nodes, edge.To)
		if target == nil {
			return nil, nil, fmt.Errorf("parallel target %s is missing", edge.To)
		}
		memberEdge, err := selectUnconditionalEdge(definition.Edges, target.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("parallel target %s: %w", target.ID, err)
		}
		if joinID == "" {
			joinID = memberEdge.To
		} else if joinID != memberEdge.To {
			return nil, nil, fmt.Errorf("parallel targets do not converge on one join")
		}
		targets[target.ID] = target
	}
	join := findNode(definition.Nodes, joinID)
	if len(targets) < 2 || join == nil || join.Kind != domain.WorkflowNodeJoin {
		return nil, nil, fmt.Errorf("parallel branch requires at least two direct members converging on a join")
	}
	if join.Join.Strategy == "quorum" && join.Join.Quorum > len(targets) {
		return nil, nil, fmt.Errorf("join quorum exceeds member count")
	}
	return targets, join, nil
}

func sortedTargetIDs(targets map[string]*domain.WorkflowNode) []string {
	ids := make([]string, 0, len(targets))
	for id := range targets {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func (e *Engine) advanceBranch(ctx context.Context, x *domain.WorkflowExecution, r *domain.WorkflowRevision, node *domain.WorkflowNode) (*domain.WorkflowExecution, error) {
	journal, err := e.Store.ListJournal(ctx, x.ID, 0, 0)
	if err != nil {
		return nil, err
	}
	var input any
	_ = json.Unmarshal(x.Input, &input)
	journalValues := make([]any, 0, len(journal))
	for _, entry := range journal {
		var value any
		_ = json.Unmarshal(entry.Payload, &value)
		journalValues = append(journalValues, value)
	}
	var selected *domain.WorkflowEdge
	var fallback *domain.WorkflowEdge
	for i := range r.Definition.Edges {
		edge := &r.Definition.Edges[i]
		if edge.From != node.ID {
			continue
		}
		if edge.Condition == "" {
			if fallback != nil {
				return e.fail(ctx, x, "branch_ambiguous", "multiple fallback branch edges")
			}
			fallback = edge
			continue
		}
		matches, evalErr := e.Expressions.Evaluate(edge.Condition, ExpressionContext{Input: input, Journal: journalValues, Status: string(x.Status), Iteration: int64(x.BudgetUsage.NodeAttempts), EdgeTraversals: x.EdgeTraversals, Budget: map[string]any{"turns": x.BudgetUsage.Turns, "tokens": x.BudgetUsage.Tokens, "costUsd": x.BudgetUsage.CostUSD}})
		err = evalErr
		if edge.Condition != "" {
			if err != nil {
				return e.fail(ctx, x, "expression_invalid", err.Error())
			}
		}
		if matches {
			if selected != nil {
				return e.fail(ctx, x, "branch_ambiguous", "multiple branch edges matched")
			}
			selected = edge
		}
	}
	if selected == nil {
		selected = fallback
	}
	if selected == nil {
		return e.fail(ctx, x, "branch_unmatched", "no branch edge matched")
	}
	if err := takeEdge(x, *selected, r.Definition.Budgets); err != nil {
		return e.exhaust(ctx, x, "edge_traversal")
	}
	x.CurrentNodeID = selected.To
	x.Status = domain.WorkflowExecutionRunning
	x.Version++
	x.UpdatedAt = e.now()
	return e.commitOnly(ctx, x)
}

func (e *Engine) advanceEnd(ctx context.Context, x *domain.WorkflowExecution, r *domain.WorkflowRevision, node *domain.WorkflowNode) (*domain.WorkflowExecution, error) {
	journal, err := e.Store.ListJournal(ctx, x.ID, 0, 0)
	if err != nil {
		return nil, err
	}
	values, err := EvaluateBindings(node.End.Bindings, BindingContext{Input: x.Input, Journal: journal})
	if err != nil {
		return e.fail(ctx, x, "output_binding_invalid", err.Error())
	}
	output, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	if err := structuredresult.ValidateValue(r.Definition.OutputSchema, output); err != nil {
		return e.fail(ctx, x, "output_invalid", err.Error())
	}
	now := e.now()
	x.Output = output
	if node.End.Status == "succeeded" {
		x.Status = domain.WorkflowExecutionSucceeded
		x.TerminalReason = &domain.WorkflowTerminalReason{Code: "completed"}
	} else {
		x.Status = domain.WorkflowExecutionFailed
		x.TerminalReason = &domain.WorkflowTerminalReason{Code: "definition_failed"}
	}
	x.EndedAt = &now
	x.UpdatedAt = now
	x.Version++
	return e.commitOnly(ctx, x)
}

func (e *Engine) fail(ctx context.Context, x *domain.WorkflowExecution, code, message string) (*domain.WorkflowExecution, error) {
	now := e.now()
	x.Status = domain.WorkflowExecutionFailed
	x.TerminalReason = &domain.WorkflowTerminalReason{Code: code, Message: message}
	x.EndedAt = &now
	x.UpdatedAt = now
	x.Version++
	return e.commitOnly(ctx, x)
}

func (e *Engine) exhaust(ctx context.Context, x *domain.WorkflowExecution, budget string) (*domain.WorkflowExecution, error) {
	now := e.now()
	x.Status = domain.WorkflowExecutionBudgetExhausted
	x.TerminalReason = &domain.WorkflowTerminalReason{Code: "budget_exhausted", BudgetName: budget}
	x.EndedAt = &now
	x.UpdatedAt = now
	x.Version++
	return e.commitOnly(ctx, x)
}

func (e *Engine) commitFailure(ctx context.Context, x *domain.WorkflowExecution, a *domain.WorkflowNodeAttempt, entries []*domain.WorkflowJournalEntry, code, message string) (*domain.WorkflowExecution, error) {
	now := e.now()
	x.Status = domain.WorkflowExecutionFailed
	x.TerminalReason = &domain.WorkflowTerminalReason{Code: code, Message: message}
	x.EndedAt = &now
	x.UpdatedAt = now
	x.Version++
	ok, err := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempt: a, Journal: entries})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrConcurrentAdvance
	}
	return x, nil
}

func (e *Engine) commitExhaust(ctx context.Context, x *domain.WorkflowExecution, a *domain.WorkflowNodeAttempt, entries []*domain.WorkflowJournalEntry, budget string) (*domain.WorkflowExecution, error) {
	now := e.now()
	x.Status = domain.WorkflowExecutionBudgetExhausted
	x.TerminalReason = &domain.WorkflowTerminalReason{Code: "budget_exhausted", BudgetName: budget}
	x.EndedAt = &now
	x.UpdatedAt = now
	x.Version++
	ok, err := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempt: a, Journal: entries})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrConcurrentAdvance
	}
	return x, nil
}

func (e *Engine) commitOnly(ctx context.Context, x *domain.WorkflowExecution) (*domain.WorkflowExecution, error) {
	ok, err := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrConcurrentAdvance
	}
	return x, nil
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}

func findNode(nodes []domain.WorkflowNode, id string) *domain.WorkflowNode {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
	}
	return nil
}

func selectUnconditionalEdge(edges []domain.WorkflowEdge, from string) (domain.WorkflowEdge, error) {
	var selected *domain.WorkflowEdge
	for i := range edges {
		if edges[i].From == from && edges[i].Condition == "" {
			if selected != nil {
				return domain.WorkflowEdge{}, fmt.Errorf("multiple unconditional edges")
			}
			selected = &edges[i]
		}
	}
	if selected == nil {
		return domain.WorkflowEdge{}, fmt.Errorf("no unconditional edge")
	}
	return *selected, nil
}

func takeEdge(x *domain.WorkflowExecution, edge domain.WorkflowEdge, _ domain.WorkflowBudgets) error {
	key := edge.From + "->" + edge.To
	next := x.EdgeTraversals[key] + 1
	if edge.MaxTraversals > 0 && next > edge.MaxTraversals {
		return fmt.Errorf("edge %s exhausted", key)
	}
	x.EdgeTraversals[key] = next
	return nil
}

func nextJournal(executionID uuid.UUID, existing []*domain.WorkflowJournalEntry, kind domain.WorkflowJournalKind, node string, attempt *uuid.UUID, payload []byte, now time.Time) *domain.WorkflowJournalEntry {
	seq := int64(1)
	if len(existing) > 0 {
		seq = existing[len(existing)-1].Sequence + 1
	}
	return &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: executionID, Sequence: seq, Kind: kind, NodeID: node, AttemptID: attempt, Payload: append(json.RawMessage(nil), payload...), CreatedAt: now}
}
