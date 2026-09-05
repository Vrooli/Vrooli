package workflowruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"agent-manager/internal/structuredresult"

	"github.com/google/uuid"
)

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
	visit := parallelVisitOrdinal(journal, fork.ID)
	visitKeyPrefix := fmt.Sprintf("workflow/%s/parallel/%s/visit/%d/node/", x.ID, fork.ID, visit)
	byNode := map[string]*domain.WorkflowNodeAttempt{}
	for _, attempt := range attempts {
		_, target := targets[attempt.NodeID]
		legacyVisitOne := visit == 1 && attempt.IdempotencyKey == fmt.Sprintf("workflow/%s/parallel/%s/attempt/1", x.ID, attempt.NodeID)
		if target && (strings.HasPrefix(attempt.IdempotencyKey, visitKeyPrefix) || legacyVisitOne) {
			byNode[attempt.NodeID] = attempt
		}
	}
	required := len(targets)
	if join.Join.Strategy == "any" {
		required = 1
	} else if join.Join.Strategy == "quorum" {
		required = join.Join.Quorum
	}
	if join.Join.Strategy != "all" {
		if joined, joinErr := e.completeSatisfiedParallelJoin(ctx, x, fork, join, visit, required, journal, byNode); joined || joinErr != nil {
			return x, joinErr
		}
	}
	if len(byNode) == 0 {
		if x.BudgetUsage.NodeAttempts+len(targets) > r.Definition.Budgets.MaxNodeAttempts || x.BudgetUsage.Children+len(targets) > r.Definition.Budgets.MaxChildren {
			return e.exhaust(ctx, x, "parallelism")
		}
		now := e.now()
		created := make([]*domain.WorkflowNodeAttempt, 0, len(targets))
		entries := make([]*domain.WorkflowJournalEntry, 0, len(targets)+1)
		forkPayload, _ := json.Marshal(map[string]any{"phase": "started", "visit": visit, "joinNodeId": join.ID, "members": sortedTargetIDs(targets), "strategy": join.Join.Strategy, "quorum": join.Join.Quorum})
		entries = append(entries, nextJournal(x.ID, journal, domain.WorkflowJournalJoin, fork.ID, nil, forkPayload, now))
		for _, targetID := range sortedTargetIDs(targets) {
			target := targets[targetID]
			attempt, prepErr := e.prepareParallelAttempt(x, fork.ID, visit, target, attempts, journal, now)
			if prepErr != nil {
				return e.fail(ctx, x, "parallel_member_invalid", prepErr.Error())
			}
			created = append(created, attempt)
			payload, _ := json.Marshal(map[string]any{"nodeId": targetID, "ordinal": attempt.Ordinal, "strategy": attempt.Strategy, "parallelParent": fork.ID, "parallelVisit": visit})
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
		var childStatus domain.WorkflowExecutionStatus
		if attempt.Strategy == domain.WorkflowAttemptChild {
			if attempt.ChildExecutionID == nil {
				return e.fail(ctx, x, "child_identity_missing", "parallel child workflow has no execution id")
			}
			state, inspectErr := e.Subworkflows.Inspect(ctx, *attempt.ChildExecutionID)
			if inspectErr != nil {
				return x, inspectErr
			}
			terminal, failed, usage, childOutput = state.Terminal, state.Failed, state.BudgetUsage, state.Output
			childStatus = state.Status
			if childStatus == "" && terminal {
				childStatus = domain.WorkflowExecutionSucceeded
				if failed {
					childStatus = domain.WorkflowExecutionFailed
				}
			}
		} else {
			if attempt.RunID == nil {
				return e.fail(ctx, x, "child_identity_missing", "parallel Run has no id")
			}
			state, inspectErr := e.Children.Inspect(ctx, *attempt.RunID)
			if inspectErr != nil {
				return x, inspectErr
			}
			terminal, failed, result = state.Terminal, state.Failed, state.Result
			usage = domain.WorkflowBudgetUsage{Turns: state.Turns, Tokens: state.Tokens, ChargeMicroUSD: state.ChargeMicroUSD, ChargeMeasured: state.ChargeMeasured}
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
		x.BudgetUsage.ChargeMicroUSD += usage.ChargeMicroUSD
		x.BudgetUsage.ChargeMeasured = x.BudgetUsage.ChargeMeasured || usage.ChargeMeasured
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
			data, _ := json.Marshal(map[string]any{"childExecutionId": attempt.ChildExecutionID, "status": childStatus, "output": childOutput})
			entries = append(entries, nextJournal(x.ID, journal, domain.WorkflowJournalChild, targetID, &attempt.ID, data, now))
		}
		if childStatus != "" && childStatus != domain.WorkflowExecutionSucceeded {
			attempt.Status, attempt.ErrorCode = domain.WorkflowAttemptFailed, "parallel_child_failed"
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
	if succeeded+(len(targets)-completed) < required {
		return e.fail(ctx, x, "join_unsatisfied", fmt.Sprintf("join required %d successful members, got %d with %d still possible", required, succeeded, len(targets)-completed))
	}
	if completed < len(targets) {
		x.Status = domain.WorkflowExecutionWaiting
		return x, nil
	}
	if succeeded < required {
		return e.fail(ctx, x, "join_unsatisfied", fmt.Sprintf("join required %d successful members, got %d", required, succeeded))
	}
	now := e.now()
	x.CurrentNodeID, x.Status, x.UpdatedAt = join.ID, domain.WorkflowExecutionRunning, now
	x.Version++
	payload, _ := json.Marshal(map[string]any{"phase": "completed", "visit": visit, "joinNodeId": join.ID})
	entry := nextJournal(x.ID, journal, domain.WorkflowJournalJoin, fork.ID, nil, payload, now)
	if ok, commitErr := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Journal: []*domain.WorkflowJournalEntry{entry}}); commitErr != nil {
		return nil, commitErr
	} else if !ok {
		return nil, ErrConcurrentAdvance
	}
	return x, nil
}

func (e *Engine) completeSatisfiedParallelJoin(ctx context.Context, x *domain.WorkflowExecution, fork, join *domain.WorkflowNode, visit, required int, journal []*domain.WorkflowJournalEntry, byNode map[string]*domain.WorkflowNodeAttempt) (bool, error) {
	succeeded := 0
	for _, attempt := range byNode {
		if attempt.Status == domain.WorkflowAttemptCompleted {
			succeeded++
		}
	}
	if succeeded < required {
		return false, nil
	}
	now := e.now()
	updated := make([]*domain.WorkflowNodeAttempt, 0)
	for _, attempt := range byNode {
		if attempt.Status == domain.WorkflowAttemptCompleted || attempt.Status == domain.WorkflowAttemptFailed {
			continue
		}
		if attempt.ChildExecutionID != nil {
			if err := e.Subworkflows.Cancel(ctx, *attempt.ChildExecutionID, "parallel join already satisfied"); err != nil {
				return false, err
			}
		} else if attempt.RunID != nil {
			if err := e.Children.Stop(ctx, *attempt.RunID); err != nil {
				return false, err
			}
		}
		attempt.Status, attempt.ErrorCode = domain.WorkflowAttemptFailed, "parallel_join_short_circuit"
		attempt.UpdatedAt, attempt.CompletedAt = now, &now
		attempt.Version++
		updated = append(updated, attempt)
	}
	x.CurrentNodeID, x.Status, x.UpdatedAt = join.ID, domain.WorkflowExecutionRunning, now
	x.Version++
	payload, _ := json.Marshal(map[string]any{"phase": "completed", "visit": visit, "joinNodeId": join.ID, "shortCircuited": true})
	entry := nextJournal(x.ID, journal, domain.WorkflowJournalJoin, fork.ID, nil, payload, now)
	ok, err := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempts: updated, Journal: []*domain.WorkflowJournalEntry{entry}})
	if err != nil {
		return false, err
	}
	if !ok {
		return false, ErrConcurrentAdvance
	}
	return true, nil
}

func parallelVisitOrdinal(journal []*domain.WorkflowJournalEntry, forkID string) int {
	latestStarted := 0
	completed := map[int]bool{}
	legacyVisit := 0
	for _, entry := range journal {
		if entry.Kind != domain.WorkflowJournalJoin || entry.NodeID != forkID {
			continue
		}
		var payload struct {
			Phase string `json:"phase"`
			Visit int    `json:"visit"`
		}
		if json.Unmarshal(entry.Payload, &payload) != nil {
			continue
		}
		if payload.Visit <= 0 {
			legacyVisit++
			payload.Visit = legacyVisit
			payload.Phase = "started"
		}
		switch payload.Phase {
		case "completed":
			completed[payload.Visit] = true
		default:
			if payload.Visit > latestStarted {
				latestStarted = payload.Visit
			}
		}
	}
	if latestStarted == 0 {
		return 1
	}
	if completed[latestStarted] {
		return latestStarted + 1
	}
	return latestStarted
}

func exceededBudgetName(usage domain.WorkflowBudgetUsage, budgets domain.WorkflowBudgets) string {
	switch {
	case usage.Turns > budgets.MaxTurns:
		return "turns"
	case usage.Tokens > budgets.MaxTokens:
		return "tokens"
	case usage.ChargeMicroUSD > budgets.MaxChargeMicroUSD:
		return "charge"
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

func (e *Engine) prepareParallelAttempt(x *domain.WorkflowExecution, forkID string, visit int, node *domain.WorkflowNode, attempts []*domain.WorkflowNodeAttempt, journal []*domain.WorkflowJournalEntry, now time.Time) (*domain.WorkflowNodeAttempt, error) {
	idempotencyKey := fmt.Sprintf("workflow/%s/parallel/%s/visit/%d/node/%s", x.ID, forkID, visit, node.ID)
	ordinal := 1
	for _, attempt := range attempts {
		if attempt.NodeID == node.ID && attempt.Ordinal >= ordinal {
			ordinal = attempt.Ordinal + 1
		}
	}
	if node.Kind == domain.WorkflowNodeChild {
		values, err := EvaluateBindings(node.Child.Bindings, BindingContext{Input: x.Input, Journal: journal})
		if err != nil {
			return nil, err
		}
		input, err := json.Marshal(values)
		if err != nil {
			return nil, err
		}
		return &domain.WorkflowNodeAttempt{ID: uuid.New(), ExecutionID: x.ID, NodeID: node.ID, Ordinal: ordinal, Strategy: domain.WorkflowAttemptChild, Status: domain.WorkflowAttemptDispatchPending, IdempotencyKey: idempotencyKey, InputSnapshot: input, Version: 1, CreatedAt: now, UpdatedAt: now}, nil
	}
	if node.Kind != domain.WorkflowNodeRun && node.Kind != domain.WorkflowNodeContinue {
		return nil, fmt.Errorf("parallel member %s must be run, continue, or child_workflow", node.ID)
	}
	// Parallel planning currently prepares its durable attempt inside the same
	// engine transaction-free phase. The resolver still runs before the attempt
	// is committed; Advance supplies cancellation at the outer boundary.
	assignment := PromptAssignmentIdentity{ExecutionID: x.ID, NodeID: node.ID, AttemptKey: fmt.Sprintf("%d", ordinal), IdempotencyKey: fmt.Sprintf("workflow-assignment/%s/node/%s", x.ID, node.ID)}
	input, prompt, resolution, _, strategy, source, _, err := e.resolveAgentInput(context.Background(), node, attempts, journal, x.Input, assignment)
	if err != nil {
		return nil, err
	}
	return &domain.WorkflowNodeAttempt{ID: uuid.New(), ExecutionID: x.ID, NodeID: node.ID, Ordinal: ordinal, Strategy: strategy, Status: domain.WorkflowAttemptDispatchPending, IdempotencyKey: idempotencyKey, InputSnapshot: input, PromptSnapshot: prompt, ExperimentID: resolution.ExperimentID, VariantID: resolution.VariantID, PromptHash: resolution.ContentHash, SourceAttemptID: source, Version: 1, CreatedAt: now, UpdatedAt: now}, nil
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
	journalValues := ProjectJournal(journal)
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
		matches, evalErr := e.Expressions.Evaluate(edge.Condition, ExpressionContext{Input: input, Journal: journalValues, Status: string(x.Status), Iteration: int64(x.BudgetUsage.NodeAttempts), EdgeTraversals: x.EdgeTraversals, Budget: map[string]any{"turns": x.BudgetUsage.Turns, "tokens": x.BudgetUsage.Tokens, "chargeMicroUsd": x.BudgetUsage.ChargeMicroUSD}})
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
	switch node.End.Status {
	case "succeeded":
		x.Status = domain.WorkflowExecutionSucceeded
	case "blocked":
		x.Status = domain.WorkflowExecutionBlocked
	case "abstained":
		x.Status = domain.WorkflowExecutionAbstained
	case "budget_exhausted":
		x.Status = domain.WorkflowExecutionBudgetExhausted
	default:
		x.Status = domain.WorkflowExecutionFailed
	}
	if x.TerminalReason == nil || x.TerminalReason.Code != "wait_timeout" {
		switch node.End.Status {
		case "succeeded":
			x.TerminalReason = &domain.WorkflowTerminalReason{Code: "completed"}
		case "blocked":
			x.TerminalReason = &domain.WorkflowTerminalReason{Code: "blocked"}
		case "abstained":
			x.TerminalReason = &domain.WorkflowTerminalReason{Code: "abstained"}
		case "budget_exhausted":
			x.TerminalReason = &domain.WorkflowTerminalReason{Code: "budget_exhausted", BudgetName: "authored_limit"}
		default:
			x.TerminalReason = &domain.WorkflowTerminalReason{Code: "definition_failed"}
		}
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

func (e *Engine) commitChildTerminal(ctx context.Context, x *domain.WorkflowExecution, a *domain.WorkflowNodeAttempt, entries []*domain.WorkflowJournalEntry, status domain.WorkflowExecutionStatus, output json.RawMessage, reason *domain.WorkflowTerminalReason) (*domain.WorkflowExecution, error) {
	now := e.now()
	x.Status = status
	x.Output = append(json.RawMessage(nil), output...)
	if reason != nil {
		copyReason := *reason
		x.TerminalReason = &copyReason
	} else {
		x.TerminalReason = &domain.WorkflowTerminalReason{Code: string(status)}
	}
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

// waitedDuration derives paused time from the durable journal. A wait starts at
// its intent and ends at either its matching signal or its routed timeout. An
// open wait is counted through now. Keeping this derived rather than persisted
// preserves replayability and excludes every human approval pause from the
// wall-time budget.
func waitedDuration(journal []*domain.WorkflowJournalEntry, now time.Time) time.Duration {
	type openWait struct {
		started time.Time
	}
	open := map[string]openWait{}
	var total time.Duration
	for _, entry := range journal {
		switch entry.Kind {
		case domain.WorkflowJournalWait:
			var intent waitIntent
			if json.Unmarshal(entry.Payload, &intent) == nil && intent.CorrelationKey != "" {
				open[intent.CorrelationKey] = openWait{started: entry.CreatedAt}
			}
		case domain.WorkflowJournalSignal:
			var signal signalRecord
			if json.Unmarshal(entry.Payload, &signal) == nil && signal.CorrelationKey != "" {
				if wait, ok := open[signal.CorrelationKey]; ok && entry.CreatedAt.After(wait.started) {
					total += entry.CreatedAt.Sub(wait.started)
					delete(open, signal.CorrelationKey)
				}
			}
		case domain.WorkflowJournalWaitTimeout:
			var timeout struct {
				CorrelationKey string `json:"correlationKey"`
			}
			if json.Unmarshal(entry.Payload, &timeout) == nil {
				if wait, ok := open[timeout.CorrelationKey]; ok && entry.CreatedAt.After(wait.started) {
					total += entry.CreatedAt.Sub(wait.started)
					delete(open, timeout.CorrelationKey)
				}
			}
		}
	}
	for _, wait := range open {
		if now.After(wait.started) {
			total += now.Sub(wait.started)
		}
	}
	return total
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

// ProjectJournal builds the CEL-visible journal: one map per entry carrying the
// entry's payload fields plus the authoritative nodeId, kind, and sequence drawn
// from the entry itself. Correlating a structured result to the node that
// produced it is impossible from the payload alone (the structured_result
// payload holds only status/value), so the projection overlays the entry-level
// nodeId/kind — which is what the latest()/count() helpers filter on. Overlaying
// preserves every existing payload key, so conditions that sniff has(j.value),
// has(j.status), has(j.output), or has(j.ordinal) evaluate exactly as before.
func ProjectJournal(journal []*domain.WorkflowJournalEntry) []any {
	values := make([]any, 0, len(journal))
	for _, entry := range journal {
		var payload any
		_ = json.Unmarshal(entry.Payload, &payload)
		if m, ok := payload.(map[string]any); ok {
			m["nodeId"] = entry.NodeID
			m["kind"] = string(entry.Kind)
			m["sequence"] = entry.Sequence
			values = append(values, m)
			continue
		}
		values = append(values, payload)
	}
	return values
}

func nextJournal(executionID uuid.UUID, existing []*domain.WorkflowJournalEntry, kind domain.WorkflowJournalKind, node string, attempt *uuid.UUID, payload []byte, now time.Time) *domain.WorkflowJournalEntry {
	seq := int64(1)
	if len(existing) > 0 {
		seq = existing[len(existing)-1].Sequence + 1
	}
	return &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: executionID, Sequence: seq, Kind: kind, NodeID: node, AttemptID: attempt, Payload: append(json.RawMessage(nil), payload...), CreatedAt: now}
}
