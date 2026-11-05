package execution

import (
	"context"
	"fmt"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/workflowcontract"
)

// cancelPlanExecutionLocked withdraws local dispatch/apply authority before
// any owner call. Transport failure and missing final usage remain a visible,
// durable cancelling state, never a freed reservation or a completed execution.
func (s *Service) cancelPlanExecutionLocked(ctx context.Context, records []Record, idx int) (Record, error) {
	record := records[idx]
	if record.Cancellation != nil && record.Cancellation.SettledAt != "" {
		return record, nil
	}
	previous := record.Status
	if record.Cancellation == nil {
		record.Cancellation = &CancellationStanding{RequestID: "cancel-" + record.ExecutionID, RequestedAt: nowRFC3339()}
	}
	record.Status = StatusCancelling
	record.FinishedAt = ""
	record.FailureReason = "cancellation requested; local authority withdrawn; owner acknowledgement and terminal accounting pending"
	record.UpdatedAt = nowRFC3339()
	records[idx] = record
	if err := s.store.Save(records); err != nil {
		return Record{}, err
	}
	if previous != record.Status {
		s.dispatchStatusAndLog(record, previous)
	}

	finishPending := func(reason string) (Record, error) {
		record.Cancellation.LastError = reason
		record.FailureReason = "cancellation pending: " + reason
		record.UpdatedAt = nowRFC3339()
		records[idx] = record
		if err := s.store.Save(records); err != nil {
			return Record{}, err
		}
		s.dispatchStatusUpdate(record)
		return record, nil
	}
	settle := func(usage *workflowcontract.Usage) (Record, error) {
		record.SettledUsage = usage
		record.Cancellation.SettledAt = nowRFC3339()
		record.Cancellation.LastError = ""
		record.Status = StatusCanceled
		record.FinishedAt = record.Cancellation.SettledAt
		record.UpdatedAt = record.FinishedAt
		record.FailureReason = "cancellation settled with original owner terminal accounting"
		records[idx] = record
		if err := s.store.Save(records); err != nil {
			return Record{}, err
		}
		s.dispatchStatusAndLog(record, StatusCancelling)
		return record, nil
	}
	// An execution admitted without reviewed limits holds no reservation, so its
	// cancellation needs no accounting to release: it needs only proof that the
	// owner's agent work has stopped. Nothing is recorded as settled usage.
	finishUnreserved := func(evidence string) (Record, error) {
		record.Cancellation.SettledAt = nowRFC3339()
		record.Cancellation.LastError = ""
		record.Status = StatusCanceled
		record.FinishedAt = record.Cancellation.SettledAt
		record.UpdatedAt = record.FinishedAt
		record.FailureReason = "cancellation settled without accounting: no reviewed allowance was reserved and " + evidence
		records[idx] = record
		if err := s.store.Save(records); err != nil {
			return Record{}, err
		}
		s.dispatchStatusAndLog(record, StatusCancelling)
		return record, nil
	}
	unreserved := record.WorkflowGrant == nil
	// A goal execution is one Agent Manager run with no workflow owner: stop the
	// run, then settle only from that run's own terminal accounting.
	if isGoalRecord(record) {
		if record.Cancellation.AcknowledgedAt == "" && s.stopper != nil && strings.TrimSpace(record.RunID) != "" {
			// Agent Manager refuses to stop a run that already ended; that run has
			// stopped, which is what the acknowledgement records.
			if err := s.stopper.StopRun(ctx, record.RunID); err != nil && !s.ownerRunTerminal(ctx, record) {
				return finishPending("goal run stop unavailable; reservation retained: " + err.Error())
			}
			record.Cancellation.AcknowledgedAt = nowRFC3339()
			records[idx] = record
			if err := s.store.Save(records); err != nil {
				return Record{}, err
			}
		}
		usage, err := s.settledGoalUsage(ctx, record)
		if err != nil {
			return finishPending("goal run accounting unresolved; reservation retained: " + err.Error())
		}
		if usage == nil {
			if unreserved && s.ownerRunTerminal(ctx, record) {
				return finishUnreserved("the goal run is terminal")
			}
			return finishPending("goal run termination or final usage remains unknown; reservation retained")
		}
		return settle(usage)
	}
	if s.transitionRunner == nil {
		return finishPending("transition owner is unavailable; reservation retained")
	}
	correlation, err := s.transitionCorrelation(record)
	if err != nil {
		// Recover only the original submission. This owner lookup must never
		// replay Start while local dispatch authority has been withdrawn.
		correlation, err = s.transitionRunner.ResolveDispatch(ctx, "plan.execute", record.ExecutionID)
		if err != nil {
			if unreserved && s.ownerRunTerminal(ctx, record) {
				return finishUnreserved("the original workflow identity is unavailable while the recorded owner run is terminal")
			}
			return finishPending("original owner identity remains unresolved; reservation retained: " + err.Error())
		}
	}
	var cancelErr error
	if record.Cancellation.AcknowledgedAt == "" {
		cancelErr = s.transitionRunner.Cancel(ctx, correlation.ExecutionID, record.Cancellation.RequestID, "consumer execution canceled")
		if cancelErr == nil {
			record.Cancellation.AcknowledgedAt = nowRFC3339()
			records[idx] = record
			if err := s.store.Save(records); err != nil {
				return Record{}, err
			}
		}
	}
	// A lost acknowledgement or an already-terminal owner can still supply
	// its original terminal receipt. Never require a second owner execution.
	ownerApproval := ""
	if record.WorkflowGrant != nil {
		ownerApproval = record.ApprovalDigest
	}
	// The ordinary ungrant invocation predates reviewed execution limits and
	// sends neither binding. Local plan acceptance is not an owner grant.
	usage, usageErr := s.transitionRunner.CollectUsage(ctx, correlation.ExecutionID, ownerApproval, workflowcontract.GrantDigest(record.WorkflowGrant))
	if usageErr != nil {
		if cancelErr != nil {
			return finishPending(fmt.Sprintf("owner cancellation unavailable (%v); terminal accounting unresolved (%v)", cancelErr, usageErr))
		}
		return finishPending("owner acknowledged; terminal accounting unresolved: " + usageErr.Error())
	}
	if usage == nil || !usage.TokensKnown || !usage.ChargeMeasured || usage.WallSeconds <= 0 {
		// CollectUsage returns a receipt only for a terminal owner workflow, so
		// the owner has stopped even though its accounting is incomplete.
		if unreserved {
			return finishUnreserved("the owner workflow is terminal")
		}
		return finishPending("owner termination or final usage remains unknown; reservation retained")
	}
	return settle(usage)
}

// WriteOffCancellation is the operator's explicit end for a reserved
// cancellation whose owner run has stopped but whose usage can never be known
// (an unpriced model, a launch failure with no receipt, a kill mid-turn). It
// charges the entire reservation as used, so unknown usage is never counted as
// less than it could have been, and records who decided and why. The automatic
// cancellation path never does this.
func (s *Service) WriteOffCancellation(ctx context.Context, executionID, actor, reason string) (Record, error) {
	actor, reason = strings.TrimSpace(actor), strings.TrimSpace(reason)
	if actor == "" || reason == "" {
		return Record{}, apierr.BadRequest("a cancellation write-off requires a non-blank actor and reason")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, err
	}
	record := records[idx]
	// An interrupted owner can leave a reservation without ever entering the
	// normal cancelling state (for example, a session-loss classification). It
	// is safe to expose the same explicit operator write-off once Agent Manager
	// confirms that owner is terminal; otherwise the reservation is stranded
	// forever and no retry can be admitted.
	if record.Cancellation != nil && record.Cancellation.SettledAt != "" {
		return Record{}, apierr.Conflict("execution %s is not an unsettled cancellation", executionID)
	}
	if record.Status != StatusCancelling && record.Status != StatusInterrupted {
		return Record{}, apierr.Conflict("execution %s is not an unsettled cancellation or interrupted reservation", executionID)
	}
	if record.Cancellation == nil {
		record.Cancellation = &CancellationStanding{RequestID: "writeoff-" + record.ExecutionID, RequestedAt: nowRFC3339()}
	}
	grant := record.WorkflowGrant
	if grant == nil {
		return Record{}, apierr.Conflict("execution %s holds no reservation; its cancellation finishes on its own once the owner run stops", executionID)
	}
	if !s.ownerRunTerminal(ctx, record) {
		return Record{}, apierr.Conflict("execution %s: Agent Manager does not confirm that owner run %q has stopped; a live or unreadable run cannot be written off", executionID, record.RunID)
	}
	now := nowRFC3339()
	record.SettledUsage = &workflowcontract.Usage{
		Tokens: grant.MaxTokens, Turns: int64(grant.MaxTurns), WallSeconds: grant.MaxWallTimeSeconds,
		ChargeMicroUSD: grant.MaxChargeMicroUSD, Children: int64(grant.MaxChildren), NodeAttempts: int64(grant.MaxNodeAttempts),
		Retries: int64(grant.MaxRetries), Slices: int64(record.MaxSlices), TokensKnown: true, ChargeMeasured: true,
	}
	record.Cancellation.WriteOffActor = actor
	record.Cancellation.WriteOffReason = reason
	record.Cancellation.WrittenOffAt = now
	record.Cancellation.SettledAt = now
	record.Cancellation.LastError = ""
	record.Status = StatusCanceled
	record.FinishedAt = now
	record.UpdatedAt = now
	record.FailureReason = fmt.Sprintf("cancellation written off by %s: owner run terminal, usage unknown; the full reservation is charged as used: %s", actor, reason)
	records[idx] = record
	if err := s.store.Save(records); err != nil {
		return Record{}, err
	}
	s.dispatchStatusAndLog(record, StatusCancelling)
	return record, nil
}

// reconcilePendingCancellations reuses the existing service maintenance cycle.
// Correlation cleanup runs outside the execution mutex: Apply owns the reverse
// lock order, so calling CloseUnapplied while holding it could deadlock.
func (s *Service) reconcilePendingCancellations(ctx context.Context) error {
	s.mu.Lock()
	records, err := s.store.Load()
	s.mu.Unlock()
	if err != nil {
		return err
	}
	for _, candidate := range records {
		if candidate.Cancellation == nil || candidate.Cancellation.ReconciledAt != "" {
			continue
		}
		record, err := s.Cancel(ctx, candidate.ExecutionID)
		if err != nil {
			return err
		}
		if record.Status != StatusCanceled {
			continue
		}
		// A goal run, or an execution whose workflow identity was lost, has no
		// correlation to close; that must not abort the rest of the cycle.
		if correlation, corrErr := s.transitionCorrelation(record); corrErr == nil {
			if err := s.transitionRunner.CloseUnapplied(correlation.ExecutionID, "cancelled"); err != nil {
				return err
			}
		}
		s.mu.Lock()
		current, idx, loadErr := s.loadRecordLocked(record.ExecutionID)
		if loadErr == nil && current[idx].Cancellation != nil {
			newerWork := false
			for _, other := range current {
				if other.ExecutionID != record.ExecutionID && other.BacklogKind == record.BacklogKind && other.BacklogName == record.BacklogName && (isInFlightRecord(other) || other.ParentExecutionID == record.ExecutionID) {
					newerWork = true
					break
				}
			}
			if !newerWork {
				loadErr = s.restoreBacklogStatusForRecord(record)
			}
			if loadErr == nil {
				current[idx].Cancellation.ReconciledAt = nowRFC3339()
				loadErr = s.store.Save(current)
			}
		}
		s.mu.Unlock()
		if loadErr != nil {
			return loadErr
		}
	}
	return nil
}
