package workflowruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
)

// TerminalAccountingInspector reads original children without advancing an
// unfinished workflow. Normal workflow inspection may drive it; recovery must
// never create more work while collecting a terminal parent's receipts.
type TerminalAccountingInspector interface {
	InspectTerminalAccounting(context.Context, uuid.UUID) (SubworkflowState, error)
}

// A new workflow starts with an empty, complete aggregate. Once cleanup finds
// an unresolved child that initial flag no longer describes a final receipt.
// Keep observed totals and the original outcome; only withdraw completeness.
func (e *Engine) recordIncompleteCleanup(ctx context.Context, x *domain.WorkflowExecution, cause error) (*domain.WorkflowExecution, error) {
	if !x.BudgetUsage.AccountingComplete {
		return x, cause
	}
	x.BudgetUsage.AccountingComplete = false
	x.UpdatedAt, x.Version = e.now(), x.Version+1
	if ok, err := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x}); err != nil {
		return x, fmt.Errorf("persist incomplete cleanup accounting: %w; original failure: %v", err, cause)
	} else if !ok {
		return nil, ErrConcurrentAdvance
	}
	return x, cause
}

// ReconcileTerminalAccounting refreshes late receipts without changing the
// original result, end time or execution identity, including successful work.
func (e *Engine) ReconcileTerminalAccounting(ctx context.Context, id uuid.UUID) (*domain.WorkflowExecution, error) {
	x, err := e.Store.Get(ctx, id)
	if err != nil || x == nil || !x.Status.Terminal() || x.BudgetUsage.AccountingComplete {
		return x, err
	}
	journal, err := e.Store.ListJournal(ctx, id, 0, 0)
	if err != nil {
		return x, err
	}
	attempts, _, err := e.reconcileOrdinaryCleanup(ctx, x, journal)
	if err != nil {
		return x, err
	}
	now := e.now()
	payload, _ := json.Marshal(map[string]any{"code": "terminal_accounting_reconciled", "usage": x.BudgetUsage})
	entry := nextJournal(x.ID, journal, domain.WorkflowJournalDiagnostic, x.CurrentNodeID, nil, payload, now)
	x.UpdatedAt, x.Version = now, x.Version+1
	if ok, err := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempts: attempts, Journal: []*domain.WorkflowJournalEntry{entry}}); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrConcurrentAdvance
	}
	return x, nil
}

// reconcileOrdinaryCleanup rebuilds accounting from original terminal child
// receipts. Rebuilding includes already-completed attempts: an older cleanup
// may have recorded a terminal status before the accounting tail arrived.
// Nothing commits until every dispatched child is accounted for, and replay
// replaces the aggregate rather than adding the same receipt a second time.
func (e *Engine) reconcileOrdinaryCleanup(ctx context.Context, x *domain.WorkflowExecution, journal []*domain.WorkflowJournalEntry) ([]*domain.WorkflowNodeAttempt, []meteredSettlement, error) {
	attempts, err := e.Store.ListAttempts(ctx, x.ID)
	if err != nil {
		return nil, nil, err
	}
	return e.rebuildOrdinaryUsage(ctx, x, journal, attempts, true)
}

// Continue may reuse an existing Run. Its owner receipt is cumulative, so
// aggregate each original identity once while still counting every dispatch
// attempt. Normal advancement and recovery share this same reconstruction.
func (e *Engine) rebuildOrdinaryUsage(ctx context.Context, x *domain.WorkflowExecution, journal []*domain.WorkflowJournalEntry, attempts []*domain.WorkflowNodeAttempt, requireComplete bool) ([]*domain.WorkflowNodeAttempt, []meteredSettlement, error) {
	usage := domain.WorkflowBudgetUsage{AccountingComplete: true, ChargeMeasured: true, NodeAttempts: len(attempts), CostUSD: x.BudgetUsage.CostUSD}
	seenRuns := map[uuid.UUID]bool{}
	seenWorkflows := map[uuid.UUID]bool{}
	for _, entry := range journal {
		if entry.Kind == domain.WorkflowJournalRetry {
			usage.Retries++
		}
	}
	var settled []*domain.WorkflowNodeAttempt
	for _, attempt := range attempts {
		if x.Status == domain.WorkflowExecutionSucceeded && attempt.Status != domain.WorkflowAttemptCompleted && attempt.Status != domain.WorkflowAttemptFailed {
			return nil, nil, fmt.Errorf("successful execution has unsettled attempt %s; owner reconciliation required", attempt.ID)
		}
		var child domain.WorkflowBudgetUsage
		switch {
		case attempt.ChildExecutionID != nil:
			if seenWorkflows[*attempt.ChildExecutionID] {
				break
			}
			seenWorkflows[*attempt.ChildExecutionID] = true
			if e.Subworkflows == nil {
				return nil, nil, fmt.Errorf("child workflow accounting owner is unavailable")
			}
			inspect := e.Subworkflows.Inspect
			if reader, ok := e.Subworkflows.(TerminalAccountingInspector); ok {
				inspect = reader.InspectTerminalAccounting
			}
			state, inspectErr := inspect(ctx, *attempt.ChildExecutionID)
			if inspectErr != nil {
				return nil, nil, inspectErr
			}
			known := state.Terminal && state.BudgetUsage.AccountingComplete && state.BudgetUsage.ChargeMeasured
			if state.ExecutionID != *attempt.ChildExecutionID || (requireComplete && !known) {
				return nil, nil, fmt.Errorf("child workflow %s requires original terminal accounting before cleanup", *attempt.ChildExecutionID)
			}
			child = state.BudgetUsage
			usage.AccountingComplete = usage.AccountingComplete && known
			usage.ChargeMeasured = usage.ChargeMeasured && state.BudgetUsage.ChargeMeasured
		case attempt.RunID != nil:
			if seenRuns[*attempt.RunID] {
				break
			}
			seenRuns[*attempt.RunID] = true
			if e.Children == nil {
				return nil, nil, fmt.Errorf("child run accounting owner is unavailable")
			}
			state, inspectErr := e.Children.Inspect(ctx, *attempt.RunID)
			if inspectErr != nil {
				return nil, nil, inspectErr
			}
			known := state.Terminal && state.TokensKnown && state.ChargeMeasured
			if state.RunID != *attempt.RunID || (requireComplete && !known) {
				return nil, nil, fmt.Errorf("child run %s requires original terminal accounting before cleanup", *attempt.RunID)
			}
			child = domain.WorkflowBudgetUsage{Tokens: state.Tokens, Turns: state.Turns, ChargeMicroUSD: state.ChargeMicroUSD}
			usage.AccountingComplete = usage.AccountingComplete && known
			usage.ChargeMeasured = usage.ChargeMeasured && state.ChargeMeasured
		default:
			return nil, nil, fmt.Errorf("unbound attempt %s requires dispatch reconciliation before cleanup", attempt.ID)
		}
		if err := addCleanupUsage(&usage, child); err != nil {
			return nil, nil, err
		}
		if requireComplete && attempt.Status != domain.WorkflowAttemptCompleted && attempt.Status != domain.WorkflowAttemptFailed {
			now := e.now()
			attempt.Status, attempt.ErrorCode = domain.WorkflowAttemptFailed, "cancelled"
			attempt.UpdatedAt, attempt.CompletedAt = now, &now
			attempt.Version++
			settled = append(settled, attempt)
		}
	}
	x.BudgetUsage = usage
	return settled, nil, nil
}

func addCleanupUsage(total *domain.WorkflowBudgetUsage, child domain.WorkflowBudgetUsage) error {
	for _, pair := range []struct {
		total *int
		value int
	}{
		{&total.Tokens, child.Tokens}, {&total.Turns, child.Turns},
		{&total.NodeAttempts, child.NodeAttempts}, {&total.Retries, child.Retries},
		{&total.Children, child.Children},
	} {
		if pair.value < 0 || pair.value > math.MaxInt-*pair.total {
			return fmt.Errorf("invalid or overflowing terminal child accounting")
		}
		*pair.total += pair.value
	}
	if total.Children == math.MaxInt || child.ChargeMicroUSD < 0 || child.ChargeMicroUSD > math.MaxInt64-total.ChargeMicroUSD {
		return fmt.Errorf("invalid or overflowing terminal child accounting")
	}
	total.Children++ // The immediate run/workflow, in addition to its descendants.
	total.ChargeMicroUSD += child.ChargeMicroUSD
	return nil
}
