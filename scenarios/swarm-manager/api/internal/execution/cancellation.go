package execution

import (
	"context"
	"fmt"

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
	if s.transitionRunner == nil {
		return finishPending("transition owner is unavailable; reservation retained")
	}
	correlation, err := s.transitionCorrelation(record)
	if err != nil {
		// Recover only the original submission. This owner lookup must never
		// replay Start while local dispatch authority has been withdrawn.
		correlation, err = s.transitionRunner.ResolveDispatch(ctx, "plan.execute", record.ExecutionID)
		if err != nil {
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
		return finishPending("owner termination or final usage remains unknown; reservation retained")
	}
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
		correlation, err := s.transitionCorrelation(record)
		if err != nil {
			return err
		}
		if err := s.transitionRunner.CloseUnapplied(correlation.ExecutionID, "cancelled"); err != nil {
			return err
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
