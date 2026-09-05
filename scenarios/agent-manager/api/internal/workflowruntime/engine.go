package workflowruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"agent-manager/internal/structuredresult"

	"github.com/google/uuid"
)

var (
	ErrConcurrentAdvance   = errors.New("workflow execution advanced concurrently")
	errEmptyPromptTemplate = errors.New("empty_prompt_template")
)

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
		ScopePath      string
		Tag            string
		Force          bool
		Prompt         string
		Until          string
		ResultSpec     *domain.ResultSpec
		SourceRunID    *uuid.UUID
		MaxTurns       int
		Timeout        time.Duration
		ExperimentID   string
		VariantID      string
		PromptHash     string
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
		ChargeMicroUSD int64
		ChargeMeasured bool
	}
	ChildLauncher interface {
		StartFresh(context.Context, ChildRequest) (ChildState, error)
		Continue(context.Context, ChildRequest) (ChildState, error)
		Inspect(context.Context, uuid.UUID) (ChildState, error)
		Stop(context.Context, uuid.UUID) error
	}
	// PromptResolution is the immutable treatment selected for one workflow
	// attempt. It is committed with dispatch_pending before a child run starts.
	PromptResolution struct {
		Content      string
		ExperimentID string
		VariantID    string
		ContentHash  string
	}
	PromptResolver interface {
		Resolve(context.Context, *domain.WorkflowPromptRef, PromptAssignmentIdentity) (PromptResolution, error)
	}
	PromptAssignmentIdentity struct {
		ExecutionID    uuid.UUID
		NodeID         string
		AttemptKey     string
		IdempotencyKey string
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
		ExecutionID    uuid.UUID
		Terminal       bool
		Failed         bool
		Status         domain.WorkflowExecutionStatus
		Output         json.RawMessage
		TerminalReason *domain.WorkflowTerminalReason
		BudgetUsage    domain.WorkflowBudgetUsage
	}
	SubworkflowLauncher interface {
		Start(context.Context, SubworkflowRequest) (SubworkflowState, error)
		Inspect(context.Context, uuid.UUID) (SubworkflowState, error)
		Cancel(context.Context, uuid.UUID, string) error
	}
	Clock func() time.Time
)

type Engine struct {
	Store          repository.WorkflowExecutionRepository
	Catalog        Catalog
	Children       ChildLauncher
	Subworkflows   SubworkflowLauncher
	Expressions    *ExpressionEvaluator
	PromptResolver PromptResolver
	Now            Clock
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
	if err != nil || execution == nil || execution.Status.Terminal() || execution.Status == domain.WorkflowExecutionCancelling {
		return execution, err
	}
	revision, err := e.Catalog.GetByDigest(ctx, execution.DefinitionDigest)
	if err != nil || revision == nil {
		return e.fail(ctx, execution, "definition_missing", "pinned workflow revision is unavailable")
	}
	journal, err := e.Store.ListJournal(ctx, execution.ID, 0, 0)
	if err != nil {
		return nil, err
	}
	now := e.now()
	if now.Sub(execution.CreatedAt)-waitedDuration(journal, now) > time.Duration(revision.Definition.Budgets.WallTimeSeconds)*time.Second {
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

// Fail transitions a non-terminal execution to failed through the same
// optimistic commit path used by interpreter failures. It is deliberately
// narrow: external supervisors use it only when their own worker panics and
// must not leave the durable execution stuck behind a dead goroutine.
func (e *Engine) Fail(ctx context.Context, id uuid.UUID, code, message string) (*domain.WorkflowExecution, error) {
	execution, err := e.Store.Get(ctx, id)
	if err != nil || execution == nil || execution.Status.Terminal() {
		return execution, err
	}
	return e.commitFailure(ctx, execution, nil, nil, code, message)
}

// RecordDiagnostic appends durable operator evidence without changing workflow
// semantics. The execution version still advances, preserving the repository's
// single serial commit boundary and making concurrent diagnostics harmless.
func (e *Engine) RecordDiagnostic(ctx context.Context, id uuid.UUID, code, message string) (*domain.WorkflowExecution, error) {
	execution, err := e.Store.Get(ctx, id)
	if err != nil || execution == nil || execution.Status.Terminal() {
		return execution, err
	}
	journal, err := e.Store.ListJournal(ctx, id, 0, 0)
	if err != nil {
		return nil, err
	}
	now := e.now()
	payload, _ := json.Marshal(map[string]string{"code": code, "message": message})
	entry := nextJournal(id, journal, domain.WorkflowJournalDiagnostic, execution.CurrentNodeID, nil, payload, now)
	// A diagnostic is evidence, not lifecycle progress. In particular, the
	// unarmed-wait reaper measures the execution's last real progress using
	// UpdatedAt; refreshing it here would let repeated warnings postpone the
	// terminal reap forever.
	execution.Version++
	ok, err := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: execution.Version - 1, Execution: execution, Journal: []*domain.WorkflowJournalEntry{entry}})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrConcurrentAdvance
	}
	return execution, nil
}

type waitIntent struct {
	Signal         string          `json:"signal"`
	CorrelationKey string          `json:"correlationKey"`
	ResumeToken    string          `json:"resumeToken"`
	Deadline       time.Time       `json:"deadline"`
	OnTimeout      string          `json:"onTimeout,omitempty"`
	PayloadSchema  json.RawMessage `json:"payloadSchema,omitempty"`
}

type signalRecord struct {
	Signal         string          `json:"signal"`
	WaitNodeID     string          `json:"waitNodeId"`
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
	waitVisits := 0
	for _, entry := range journal {
		if entry.NodeID != node.ID || entry.Kind != domain.WorkflowJournalWait {
			continue
		}
		waitVisits++
		var candidate waitIntent
		if json.Unmarshal(entry.Payload, &candidate) == nil {
			intent = &candidate
		}
	}
	completedVisits := 0
	for edge, traversals := range x.EdgeTraversals {
		if strings.HasPrefix(edge, node.ID+"->") {
			completedVisits += traversals
		}
	}
	if waitVisits <= completedVisits {
		intent = nil
	}
	if intent == nil {
		limit := node.Wait.TimeoutSeconds
		if limit > r.Definition.Budgets.MaxWaitSeconds {
			limit = r.Definition.Budgets.MaxWaitSeconds
		}
		now := e.now()
		visit := completedVisits + 1
		intent = &waitIntent{Signal: node.Wait.Signal, CorrelationKey: fmt.Sprintf("workflow/%s/wait/%s/visit/%d", x.ID, node.ID, visit), ResumeToken: uuid.NewString(), OnTimeout: node.Wait.OnTimeout, PayloadSchema: append(json.RawMessage(nil), node.Wait.PayloadSchema...)}
		if limit > 0 {
			intent.Deadline = now.Add(time.Duration(limit) * time.Second)
		}
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
		// A signal may have arrived after the preceding node completed but before
		// this wait armed. Re-enter after persisting the intent so the buffered
		// signal is consumed as part of arming, without asking callers to race an
		// explicit Advance between completion and Signal.
		return e.advanceWait(ctx, x, r, node)
	}
	for i := len(journal) - 1; i >= 0; i-- {
		entry := journal[i]
		if entry.NodeID != node.ID || entry.Kind != domain.WorkflowJournalSignal {
			continue
		}
		var signal signalRecord
		if json.Unmarshal(entry.Payload, &signal) == nil && signal.Signal == intent.Signal && signal.WaitNodeID == node.ID && (signal.CorrelationKey == "" || signal.CorrelationKey == intent.CorrelationKey) {
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
	if !intent.Deadline.IsZero() && !e.now().Before(intent.Deadline) {
		return e.routeWaitTimeout(ctx, x, r, node, intent, journal)
	}
	x.Status = domain.WorkflowExecutionWaiting
	return x, nil
}

func (e *Engine) routeWaitTimeout(ctx context.Context, x *domain.WorkflowExecution, r *domain.WorkflowRevision, node *domain.WorkflowNode, intent *waitIntent, journal []*domain.WorkflowJournalEntry) (*domain.WorkflowExecution, error) {
	if intent.OnTimeout == "" {
		return e.fail(ctx, x, "wait_timeout", "external signal deadline elapsed")
	}
	if findNode(r.Definition.Nodes, intent.OnTimeout) == nil {
		return e.fail(ctx, x, "wait_timeout_target_missing", "declared timeout target is unavailable")
	}
	now := e.now()
	payload, _ := json.Marshal(map[string]any{"signal": intent.Signal, "correlationKey": intent.CorrelationKey, "deadline": intent.Deadline, "target": intent.OnTimeout})
	entry := nextJournal(x.ID, journal, domain.WorkflowJournalWaitTimeout, node.ID, nil, payload, now)
	x.CurrentNodeID = intent.OnTimeout
	x.Status = domain.WorkflowExecutionRunning
	x.TerminalReason = &domain.WorkflowTerminalReason{Code: "wait_timeout", Message: "external signal deadline elapsed"}
	x.UpdatedAt = now
	x.Version++
	ok, err := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Journal: []*domain.WorkflowJournalEntry{entry}})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrConcurrentAdvance
	}
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
	if x.Status.Terminal() {
		return x, false, fmt.Errorf("workflow execution is already terminal")
	}
	if expectedVersion > 0 && x.Version != expectedVersion {
		return x, false, ErrConcurrentAdvance
	}
	revision, err := e.Catalog.GetByDigest(ctx, x.DefinitionDigest)
	if err != nil || revision == nil {
		return x, false, fmt.Errorf("pinned workflow revision is unavailable")
	}
	node, err := signalWaitNode(revision.Definition, x.CurrentNodeID, signal)
	if err != nil {
		return x, false, err
	}
	if len(node.Wait.PayloadSchema) > 0 {
		if err := structuredresult.ValidateValue(node.Wait.PayloadSchema, payload); err != nil {
			return x, false, fmt.Errorf("signal payload: %w", err)
		}
	}
	var intent waitIntent
	for i := len(journal) - 1; i >= 0; i-- {
		if journal[i].Kind == domain.WorkflowJournalWait && journal[i].NodeID == node.ID && json.Unmarshal(journal[i].Payload, &intent) == nil {
			break
		}
	}
	record := signalRecord{Signal: signal, WaitNodeID: node.ID, CorrelationKey: intent.CorrelationKey, IdempotencyKey: idempotencyKey, Payload: append(json.RawMessage(nil), payload...)}
	data, _ := json.Marshal(record)
	now := e.now()
	entry := nextJournal(x.ID, journal, domain.WorkflowJournalSignal, node.ID, nil, data, now)
	if x.Status == domain.WorkflowExecutionWaiting {
		x.Status = domain.WorkflowExecutionRunning
	}
	x.UpdatedAt = now
	x.Version++
	if ok, commitErr := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Journal: []*domain.WorkflowJournalEntry{entry}}); commitErr != nil {
		return nil, false, commitErr
	} else if !ok {
		return nil, false, ErrConcurrentAdvance
	}
	return x, false, nil
}

// signalWaitNode resolves a signal against the pinned declaration rather than
// only an already-armed wait. A signal name must therefore identify one wait
// contract unless that wait is the current node; this prevents a buffered
// signal from being ambiguously delivered to a future wait.
func signalWaitNode(definition domain.WorkflowDefinition, currentNodeID, signal string) (*domain.WorkflowNode, error) {
	if current := findNode(definition.Nodes, currentNodeID); current != nil && current.Wait != nil && current.Wait.Signal == signal {
		return current, nil
	}
	var match *domain.WorkflowNode
	for i := range definition.Nodes {
		node := &definition.Nodes[i]
		if node.Wait == nil || node.Wait.Signal != signal {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("signal matches multiple declared wait contracts")
		}
		match = node
	}
	if match == nil {
		return nil, fmt.Errorf("signal does not match a declared wait contract")
	}
	return match, nil
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
	x.Status = domain.WorkflowExecutionCancelling
	x.TerminalReason = &domain.WorkflowTerminalReason{Code: "cancelled", Message: reason}
	x.UpdatedAt = now
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
	return e.RecordCleanupDisposition(ctx, id, stoppedRuns, stoppedWorkflows, failures)
}

// RecordCleanupDisposition is the durable barrier between stopping children
// and publishing an abnormal parent terminal state. Failed cleanup writes no
// disposition, so recovery retries it instead of orphaning child agents.
func (e *Engine) RecordCleanupDisposition(ctx context.Context, id uuid.UUID, stoppedRuns, stoppedWorkflows int, failures []string) (*domain.WorkflowExecution, error) {
	x, err := e.Store.Get(ctx, id)
	if err != nil || x == nil {
		return x, err
	}
	if x.Status != domain.WorkflowExecutionCancelling && x.Status != domain.WorkflowExecutionFailed && x.Status != domain.WorkflowExecutionBudgetExhausted && x.Status != domain.WorkflowExecutionCancelled {
		return x, fmt.Errorf("cleanup disposition requires cancelling or abnormal terminal execution")
	}
	journal, err := e.Store.ListJournal(ctx, id, 0, 0)
	if err != nil {
		return nil, err
	}
	for _, entry := range journal {
		if entry.Kind == domain.WorkflowJournalCleanup {
			var prior struct {
				Retry int `json:"retry"`
			}
			if json.Unmarshal(entry.Payload, &prior) == nil && prior.Retry == x.BudgetUsage.Retries {
				return x, nil
			}
		}
	}
	if len(failures) > 0 {
		return x, fmt.Errorf("child cleanup incomplete: %s", strings.Join(failures, "; "))
	}
	now := e.now()
	payload, _ := json.Marshal(map[string]any{"retry": x.BudgetUsage.Retries, "stoppedRuns": stoppedRuns, "stoppedWorkflows": stoppedWorkflows, "failures": failures})
	entry := nextJournal(x.ID, journal, domain.WorkflowJournalCleanup, x.CurrentNodeID, nil, payload, now)
	if x.Status == domain.WorkflowExecutionCancelling {
		x.Status = domain.WorkflowExecutionCancelled
		x.EndedAt = &now
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
		// Treatment is randomized once per execution/node. Retries and schema
		// repairs deliberately re-request this same receipt while their child-run
		// idempotency remains attempt-specific below.
		assignment := PromptAssignmentIdentity{ExecutionID: x.ID, NodeID: node.ID, AttemptKey: fmt.Sprintf("%d", ordinal), IdempotencyKey: fmt.Sprintf("workflow-assignment/%s/node/%s", x.ID, node.ID)}
		bindings, prompt, resolution, spec, strategy, source, diagnostics, err := e.resolveAgentInput(ctx, node, attempts, journal, x.Input, assignment)
		if err != nil {
			if errors.Is(err, errEmptyPromptTemplate) {
				return e.fail(ctx, x, "empty_prompt_template", "workflow node has no rendered prompt template")
			}
			return e.fail(ctx, x, "binding_invalid", err.Error())
		}
		now := e.now()
		active = &domain.WorkflowNodeAttempt{ID: uuid.New(), ExecutionID: x.ID, NodeID: node.ID, Ordinal: ordinal, Strategy: strategy, Status: domain.WorkflowAttemptDispatchPending, IdempotencyKey: fmt.Sprintf("workflow/%s/node/%s/attempt/%d", x.ID, node.ID, ordinal), InputSnapshot: bindings, PromptSnapshot: prompt, ExperimentID: resolution.ExperimentID, VariantID: resolution.VariantID, PromptHash: resolution.ContentHash, SourceAttemptID: source, Version: 1, CreatedAt: now, UpdatedAt: now}
		x.BudgetUsage.NodeAttempts++
		x.Version++
		x.UpdatedAt = now
		payload, _ := json.Marshal(map[string]any{"nodeId": node.ID, "ordinal": ordinal, "strategy": strategy})
		entries := []*domain.WorkflowJournalEntry{nextJournal(x.ID, journal, domain.WorkflowJournalAttempt, node.ID, &active.ID, payload, now)}
		for _, diagnostic := range diagnostics {
			payload, _ := json.Marshal(diagnostic)
			entries = append(entries, nextJournal(x.ID, append(journal, entries...), domain.WorkflowJournalDiagnostic, node.ID, &active.ID, payload, now))
		}
		if ok, err := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempt: active, Journal: entries}); err != nil {
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
	x.BudgetUsage.ChargeMicroUSD += state.ChargeMicroUSD
	x.BudgetUsage.ChargeMeasured = x.BudgetUsage.ChargeMeasured || state.ChargeMeasured
	entries := []*domain.WorkflowJournalEntry{}
	if state.Result != nil {
		active.RawOutput = state.Result.FinalOutput
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
	if validationError := structuredValidationError(node, state.Result); validationError != "" {
		active.Status = domain.WorkflowAttemptFailed
		active.ErrorCode = "structured_result_invalid"
		active.ValidationError = validationError
		// A repair is a continuation of the existing child session, not a new
		// child. It is bounded by the explicit node-attempt budget but must not
		// be denied merely because the original run consumed MaxChildren.
		if schemaRepairCount(attempts, node.ID) < schemaRepairLimit(node) && x.BudgetUsage.NodeAttempts < r.Definition.Budgets.MaxNodeAttempts {
			repair := &domain.WorkflowNodeAttempt{ID: uuid.New(), ExecutionID: x.ID, NodeID: node.ID, Ordinal: ordinal, Strategy: domain.WorkflowAttemptContinue, Status: domain.WorkflowAttemptDispatchPending, IdempotencyKey: fmt.Sprintf("workflow/%s/node/%s/attempt/%d/schema-repair", x.ID, node.ID, ordinal), InputSnapshot: append(json.RawMessage(nil), active.InputSnapshot...), PromptSnapshot: schemaRepairPrompt(nodeResultSpec(node), validationError), SourceAttemptID: &active.ID, Version: 1, CreatedAt: now, UpdatedAt: now}
			x.Status, x.UpdatedAt = domain.WorkflowExecutionRunning, now
			x.BudgetUsage.NodeAttempts++
			x.Version++
			payload, _ := json.Marshal(map[string]any{"nodeId": node.ID, "ordinal": repair.Ordinal, "strategy": repair.Strategy, "repair": true, "sourceAttemptId": active.ID})
			entries = append(entries, nextJournal(x.ID, append(journal, entries...), domain.WorkflowJournalAttempt, node.ID, &repair.ID, payload, now))
			ok, commitErr := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempts: []*domain.WorkflowNodeAttempt{active, repair}, Journal: entries})
			if commitErr != nil {
				return nil, commitErr
			}
			if !ok {
				return nil, ErrConcurrentAdvance
			}
			return x, nil
		}
		return e.commitFailure(ctx, x, active, entries, "structured_result_invalid", validationError)
	}
	if x.BudgetUsage.Turns > r.Definition.Budgets.MaxTurns {
		return e.commitExhaust(ctx, x, active, entries, "turns")
	}
	if x.BudgetUsage.Tokens > r.Definition.Budgets.MaxTokens {
		return e.commitExhaust(ctx, x, active, entries, "tokens")
	}
	if x.BudgetUsage.ChargeMicroUSD > r.Definition.Budgets.MaxChargeMicroUSD {
		return e.commitExhaust(ctx, x, active, entries, "charge")
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

func schemaRepairCount(attempts []*domain.WorkflowNodeAttempt, nodeID string) int {
	count := 0
	for _, attempt := range attempts {
		if attempt.NodeID == nodeID && strings.HasSuffix(attempt.IdempotencyKey, "/schema-repair") {
			count++
		}
	}
	return count
}

func nodeResultSpec(node *domain.WorkflowNode) *domain.ResultSpec {
	if node.Run != nil {
		return node.Run.ResultSpec
	}
	if node.Continue != nil {
		return node.Continue.ResultSpec
	}
	return nil
}

func schemaRepairLimit(node *domain.WorkflowNode) int {
	spec := nodeResultSpec(node)
	if spec == nil || spec.SchemaRepairAttempts == nil {
		return 1
	}
	return *spec.SchemaRepairAttempts
}

func schemaRepairPrompt(spec *domain.ResultSpec, validationError string) string {
	prompt := "Your previous response did not satisfy the required structured-result schema. Return only a corrected JSON value that fixes these validation errors; do not add explanation.\n\nValidation errors:\n" + validationError
	if spec != nil && len(spec.Schema) > 0 {
		prompt += "\n\nRequired schema:\n```json\n" + string(spec.Schema) + "\n```"
	}
	return prompt
}

// structuredResultInstruction renders the node's typed-result contract into the
// agent-visible prompt. The result schema is otherwise validated only after the
// run, so without this block the executing agent never sees the shape it must
// return and can only guess it from prose.
func structuredResultInstruction(spec *domain.ResultSpec) string {
	if spec == nil || spec.Kind == domain.ResultSpecKindNone || len(spec.Schema) == 0 {
		return ""
	}
	return "\n\n## Required structured result\n\nMake your final message exactly one JSON value — a bare JSON document or one ```json fence — that validates against the schema below. Do not put any other JSON value or fence in the final message.\n\n```json\n" + string(spec.Schema) + "\n```"
}

func structuredValidationError(node *domain.WorkflowNode, result *domain.RunResult) string {
	if result == nil || result.Structured == nil || (node.Run == nil || node.Run.ResultSpec == nil) && (node.Continue == nil || node.Continue.ResultSpec == nil) {
		return ""
	}
	if result.Structured.Status == domain.StructuredResultSuccess {
		return ""
	}
	parts := make([]string, 0, len(result.Structured.Diagnostics)+1)
	parts = append(parts, string(result.Structured.Status))
	for _, diagnostic := range result.Structured.Diagnostics {
		part := diagnostic.Code
		if diagnostic.Path != "" {
			part += " at " + diagnostic.Path
		}
		if diagnostic.Message != "" {
			part += ": " + diagnostic.Message
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "; ")
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
	childStatus := state.Status
	if childStatus == "" {
		childStatus = domain.WorkflowExecutionSucceeded
		if state.Failed {
			childStatus = domain.WorkflowExecutionFailed
		}
	}
	active.Status = domain.WorkflowAttemptCompleted
	if childStatus == domain.WorkflowExecutionFailed {
		active.Status, active.ErrorCode = domain.WorkflowAttemptFailed, "child_workflow_failed"
	}
	active.Version++
	active.UpdatedAt, active.CompletedAt = now, &now
	x.BudgetUsage.Turns += state.BudgetUsage.Turns
	x.BudgetUsage.Tokens += state.BudgetUsage.Tokens
	x.BudgetUsage.ChargeMicroUSD += state.BudgetUsage.ChargeMicroUSD
	x.BudgetUsage.ChargeMeasured = x.BudgetUsage.ChargeMeasured || state.BudgetUsage.ChargeMeasured
	x.BudgetUsage.NodeAttempts += state.BudgetUsage.NodeAttempts
	x.BudgetUsage.Children += state.BudgetUsage.Children
	x.BudgetUsage.Retries += state.BudgetUsage.Retries
	payload, _ := json.Marshal(map[string]any{"childExecutionId": state.ExecutionID, "status": childStatus, "output": json.RawMessage(state.Output)})
	entry := nextJournal(x.ID, journal, domain.WorkflowJournalChild, node.ID, &active.ID, payload, now)
	if childStatus == domain.WorkflowExecutionFailed {
		return e.commitFailure(ctx, x, active, []*domain.WorkflowJournalEntry{entry}, "child_workflow_failed", "child workflow failed")
	}
	if childStatus != domain.WorkflowExecutionSucceeded {
		return e.commitChildTerminal(ctx, x, active, []*domain.WorkflowJournalEntry{entry}, childStatus, state.Output, state.TerminalReason)
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

func (e *Engine) resolveAgentInput(ctx context.Context, node *domain.WorkflowNode, attempts []*domain.WorkflowNodeAttempt, journal []*domain.WorkflowJournalEntry, input json.RawMessage, assignment PromptAssignmentIdentity) (json.RawMessage, string, PromptResolution, *domain.ResultSpec, domain.WorkflowAttemptStrategy, *uuid.UUID, []BindingDiagnostic, error) {
	var bindings []domain.WorkflowInputBinding
	var tmpl string
	var spec *domain.ResultSpec
	strategy := domain.WorkflowAttemptFreshRun
	var source *uuid.UUID
	var resolution PromptResolution
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
			return nil, "", PromptResolution{}, nil, "", nil, nil, fmt.Errorf("explicit continuation source has no completed attempt")
		}
	}
	ref := armedPromptRef(node)
	if ref != nil {
		if e.PromptResolver == nil {
			return nil, "", PromptResolution{}, nil, "", nil, nil, fmt.Errorf("armed promptRef requires a prompt resolver")
		}
		var err error
		resolution, err = e.PromptResolver.Resolve(ctx, ref, assignment)
		if err != nil {
			return nil, "", PromptResolution{}, nil, "", nil, nil, err
		}
		if resolution.ExperimentID == "" || resolution.VariantID == "" || resolution.Content == "" || resolution.ContentHash == "" {
			return nil, "", PromptResolution{}, nil, "", nil, nil, fmt.Errorf("armed prompt resolver returned incomplete assignment")
		}
		tmpl = resolution.Content
	}
	if strings.TrimSpace(tmpl) == "" {
		return nil, "", PromptResolution{}, nil, "", nil, nil, errEmptyPromptTemplate
	}
	values, diagnostics, err := EvaluateBindingsWithDiagnostics(bindings, BindingContext{Input: input, Journal: journal})
	if err != nil {
		return nil, "", PromptResolution{}, nil, "", nil, nil, err
	}
	snapshot, err := json.Marshal(values)
	if err != nil {
		return nil, "", PromptResolution{}, nil, "", nil, nil, err
	}
	prompt, err := RenderPrompt(tmpl, values)
	if err != nil {
		return snapshot, prompt, resolution, spec, strategy, source, diagnostics, err
	}
	prompt += structuredResultInstruction(spec)
	if node.Run != nil && strings.TrimSpace(node.Run.Until) != "" {
		prompt += "\n\nUNTIL (engine-owned completion test; evaluate against authoritative plan state):\n" + strings.TrimSpace(node.Run.Until) + "\n"
	}
	return snapshot, prompt, resolution, spec, strategy, source, diagnostics, nil
}

func armedPromptRef(node *domain.WorkflowNode) *domain.WorkflowPromptRef {
	if node.Run != nil && node.Run.PromptRef != nil && node.Run.PromptRef.ExperimentID != "" {
		return node.Run.PromptRef
	}
	if node.Continue != nil && node.Continue.PromptRef != nil && node.Continue.PromptRef.ExperimentID != "" {
		return node.Continue.PromptRef
	}
	return nil
}

func (e *Engine) childRequest(node *domain.WorkflowNode, x *domain.WorkflowExecution, a *domain.WorkflowNodeAttempt, attempts []*domain.WorkflowNodeAttempt, journal []*domain.WorkflowJournalEntry) (ChildRequest, error) {
	prompt := a.PromptSnapshot
	var spec *domain.ResultSpec
	if node.Run != nil {
		spec = node.Run.ResultSpec
	} else {
		spec = node.Continue.ResultSpec
	}
	request := ChildRequest{ExecutionID: x.ID, AttemptID: a.ID, NodeID: node.ID, IdempotencyKey: a.IdempotencyKey, Prompt: prompt, ResultSpec: spec, ExperimentID: a.ExperimentID, VariantID: a.VariantID, PromptHash: a.PromptHash}
	if node.Run != nil {
		request.ProfileKey = node.Run.ProfileKey
		request.RoleRef = node.Run.RoleRef
		if node.Run.ScopePathTemplate != "" {
			values := map[string]any{}
			if err := json.Unmarshal(a.InputSnapshot, &values); err != nil {
				return ChildRequest{}, fmt.Errorf("decode workflow input snapshot for scope path: %w", err)
			}
			scopePath, err := RenderPrompt(node.Run.ScopePathTemplate, values)
			if err != nil {
				return ChildRequest{}, fmt.Errorf("render workflow scope path: %w", err)
			}
			request.ScopePath = strings.TrimSpace(scopePath)
		}
		request.Tag = node.Run.Tag
		request.Force = node.Run.Force
		request.MaxTurns = node.Run.MaxTurns
		request.Timeout = time.Duration(node.Run.TimeoutSeconds) * time.Second
		request.Until = node.Run.Until
	} else {
		request.MaxTurns = node.Continue.MaxTurns
		request.Timeout = time.Duration(node.Continue.TimeoutSeconds) * time.Second
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
	if a.Strategy == domain.WorkflowAttemptContinue && a.SourceAttemptID != nil {
		for _, prior := range attempts {
			if prior.ID == *a.SourceAttemptID && prior.RunID != nil {
				request.SourceRunID = prior.RunID
				break
			}
		}
		if request.SourceRunID == nil {
			return ChildRequest{}, fmt.Errorf("schema repair source attempt has no Run id")
		}
	}
	return request, nil
}
