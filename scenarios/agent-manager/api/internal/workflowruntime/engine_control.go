package workflowruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"agent-manager/internal/structuredresult"

	"github.com/google/uuid"
)

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
	intent, _ := activeWaitIntent(journal, node.ID, x.EdgeTraversals)
	if intent != nil && !intent.Deadline.IsZero() && !e.now().Before(intent.Deadline) {
		return x, false, fmt.Errorf("workflow wait deadline has elapsed")
	}
	if len(node.Wait.PayloadSchema) > 0 {
		if err := structuredresult.ValidateValue(node.Wait.PayloadSchema, payload); err != nil {
			return x, false, fmt.Errorf("signal payload: %w", err)
		}
	}
	correlationKey := ""
	if intent != nil {
		correlationKey = intent.CorrelationKey
	}
	record := signalRecord{Signal: signal, WaitNodeID: node.ID, CorrelationKey: correlationKey, IdempotencyKey: idempotencyKey, Payload: append(json.RawMessage(nil), payload...)}
	data, _ := json.Marshal(record)
	now := e.now()
	entry := nextJournal(x.ID, journal, domain.WorkflowJournalSignal, node.ID, nil, data, now)
	if x.Status == domain.WorkflowExecutionWaiting {
		if err := x.TransitionStatus(domain.WorkflowExecutionRunning); err != nil {
			return nil, false, err
		}
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
	if err := x.TransitionStatus(domain.WorkflowExecutionCancelling); err != nil {
		return nil, false, err
	}
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
	revision = applyEngagementGrant(revision, x.EngagementGrant)
	if x.EngagementGrant != nil && !x.BudgetUsage.AccountingComplete {
		return x, false, fmt.Errorf("workflow retry requires complete terminal usage; the prior reservation remains held")
	}
	if x.BudgetUsage.Retries >= revision.Definition.Budgets.MaxRetries {
		return x, false, fmt.Errorf("workflow retry budget exhausted")
	}
	now := e.now()
	payload, _ := json.Marshal(map[string]any{"idempotencyKey": idempotencyKey})
	entry := nextJournal(x.ID, journal, domain.WorkflowJournalRetry, x.CurrentNodeID, nil, payload, now)
	if err := x.TransitionStatus(domain.WorkflowExecutionRunning); err != nil {
		return nil, false, err
	}
	x.TerminalReason, x.EndedAt, x.UpdatedAt = nil, nil, now
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
	return e.recordCleanupDisposition(ctx, id, stoppedRuns, stoppedWorkflows, failures, false)
}

// RecordRecoveryCleanupDisposition is RecordCleanupDisposition for the recovery
// sweep. A child run that retains a terminal but has no terminal accounting
// settles as unknown so recovery proceeds instead of looping forever (the
// wedged workflow 93213d93 loop). Ordinary cancellation keeps the strict
// contract above and still waits for accounting.
func (e *Engine) RecordRecoveryCleanupDisposition(ctx context.Context, id uuid.UUID, stoppedRuns, stoppedWorkflows int, failures []string) (*domain.WorkflowExecution, error) {
	return e.recordCleanupDisposition(ctx, id, stoppedRuns, stoppedWorkflows, failures, true)
}

func (e *Engine) recordCleanupDisposition(ctx context.Context, id uuid.UUID, stoppedRuns, stoppedWorkflows int, failures []string, allowUnknown bool) (*domain.WorkflowExecution, error) {
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
	// Recovery may finalize only an old terminal execution. A recent cleanup
	// failure remains strict so a late receipt or transient owner outage can be
	// repaired. The accounting pass below still has to inspect every bound
	// child; an active/inspectable child therefore cannot be hidden by this
	// age-based escape hatch.
	finalizeUnknown := allowUnknown && x.Status != domain.WorkflowExecutionCancelling && e.terminalAccountingUnknownEligible(x)
	if len(failures) > 0 && !finalizeUnknown {
		return e.recordIncompleteCleanup(ctx, x, fmt.Errorf("child cleanup incomplete: %s", strings.Join(failures, "; ")))
	}
	priorCleanup := false
	for _, entry := range journal {
		if entry.Kind == domain.WorkflowJournalCleanup {
			var prior struct {
				Retry int `json:"retry"`
			}
			if json.Unmarshal(entry.Payload, &prior) == nil && prior.Retry == x.BudgetUsage.Retries {
				priorCleanup = true
				if x.BudgetUsage.AccountingComplete {
					return x, nil
				}
			}
		}
	}
	priorUsage := x.BudgetUsage
	// Operator cancellation always waits for accounting, even on the recovery
	// sweep: a missing receipt must never be fabricated as zero. Only abnormal
	// terminal recovery (failed/budget-exhausted) settles a retained terminal as
	// unknown and proceeds.
	allowUnknown = finalizeUnknown
	settledAttempts, settlements, err := e.reconcileMeteredCleanup(ctx, x, journal, allowUnknown)
	if err != nil {
		x.BudgetUsage = priorUsage
		return e.recordIncompleteCleanup(ctx, x, err)
	}
	if priorCleanup && len(settledAttempts) == 0 && x.BudgetUsage == priorUsage {
		return x, nil
	}
	if finalizeUnknown && !x.BudgetUsage.AccountingComplete {
		x.BudgetUsage.AccountingFinalizedUnknown = true
	}
	now := e.now()
	payload, _ := json.Marshal(map[string]any{"retry": x.BudgetUsage.Retries, "stoppedRuns": stoppedRuns, "stoppedWorkflows": stoppedWorkflows, "failures": failures, "settlements": settlements, "accountingFinalizedUnknown": x.BudgetUsage.AccountingFinalizedUnknown})
	entry := nextJournal(x.ID, journal, domain.WorkflowJournalCleanup, x.CurrentNodeID, nil, payload, now)
	if x.Status == domain.WorkflowExecutionCancelling {
		if err := x.TransitionStatus(domain.WorkflowExecutionCancelled); err != nil {
			return nil, err
		}
		x.EndedAt = &now
	}
	x.UpdatedAt = now
	x.Version++
	if ok, commitErr := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempts: settledAttempts, Journal: []*domain.WorkflowJournalEntry{entry}}); commitErr != nil {
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
