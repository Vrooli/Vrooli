package stagedmessages

import (
	"context"
	"fmt"

	"git-control-tower/internal/reviewjobs"
)

// recordExecutionSuccess handles the cancellation race after an ambiguous
// dispatch, then applies and persists the owner receipt against its claim.
func (s *Service) recordExecutionSuccess(ctx context.Context, job reviewjobs.Job, input DraftInput, execution Execution) (reviewjobs.Job, bool, error) {
	next := job
	if job.ExecutionID == "" && job.DispatchAttempted {
		latest, err := s.store.Get(ctx, job.ID)
		if err != nil {
			return reviewjobs.Job{}, false, err
		}
		if latest.State.Terminal() {
			return latest, true, nil
		}
		if latest.CancelRequested {
			job, next = latest, latest
			resolved, finished, err := s.reconcileCancelledReceipt(ctx, job, next, input, execution)
			if err != nil {
				return reviewjobs.Job{}, false, err
			}
			if finished != nil {
				return *finished, true, nil
			}
			execution = resolved
		}
	}
	next, err := s.applyExecutionReceipt(ctx, job, &input, execution)
	if err != nil {
		return reviewjobs.Job{}, false, err
	}
	if _, err := s.store.UpdateAdvisory(ctx, job, next); err != nil {
		return reviewjobs.Job{}, false, err
	}
	return reviewjobs.Job{}, false, nil
}

// reconcileCancelledReceipt either persists an unresolved cancellation receipt,
// observes or requests cancellation for the exact execution, or returns the
// original response for normal owner-receipt processing.
func (s *Service) reconcileCancelledReceipt(ctx context.Context, job, next reviewjobs.Job, input DraftInput, execution Execution) (Execution, *reviewjobs.Job, error) {
	if execution.ID == "" && job.ExecutionID == "" {
		return s.recordUnconfirmedCancellation(ctx, job, next)
	}
	if execution.ID == "" || terminalExecution(execution.Status) {
		return execution, nil, nil
	}
	return s.cancelExecutionReceipt(ctx, job, next, input, execution)
}

func (s *Service) recordUnconfirmedCancellation(ctx context.Context, job, next reviewjobs.Job) (Execution, *reviewjobs.Job, error) {
	next.ErrorCode = "dispatch_unconfirmed"
	next.Error = "Cancellation is waiting for Agent Manager to confirm whether the request was accepted; the same owner key will be reconciled."
	next.Retryable = false
	if _, err := s.store.UpdateAdvisory(ctx, job, next); err != nil {
		return Execution{}, nil, err
	}
	latest, err := s.store.Get(ctx, job.ID)
	return Execution{}, &latest, err
}

func (s *Service) cancelExecutionReceipt(ctx context.Context, job, next reviewjobs.Job, input DraftInput, execution Execution) (Execution, *reviewjobs.Job, error) {
	cancelled, err := s.cancelOrObserveExecution(ctx, job, input, execution)
	if err != nil {
		return s.recordCancellationUnconfirmed(ctx, job, next, execution.ID)
	}
	if cancelled.ID != "" && cancelled.ID != execution.ID {
		return Execution{}, nil, fmt.Errorf("execution owner returned a different handle during cancellation")
	}
	if cancelled.ID == "" {
		cancelled.ID = execution.ID
	}
	return cancelled, nil, nil
}

func (s *Service) recordCancellationUnconfirmed(ctx context.Context, job, next reviewjobs.Job, executionID string) (Execution, *reviewjobs.Job, error) {
	next.ExecutionID = executionID
	next.ErrorCode = "execution_status_unavailable"
	next.Error = "Agent Manager could not confirm cancellation; the saved execution identity is retained for recovery."
	next.Retryable = true
	if _, err := s.store.UpdateAdvisory(ctx, job, next); err != nil {
		return Execution{}, nil, err
	}
	latest, err := s.store.Get(ctx, job.ID)
	return Execution{}, &latest, err
}

func (s *Service) cancelOrObserveExecution(ctx context.Context, job reviewjobs.Job, input DraftInput, execution Execution) (Execution, error) {
	if job.ExecutionID == execution.ID {
		// The owner-side key fence already adopted and cancelled this execution.
		// Observe it instead of issuing a second cancellation that can race terminal completion.
		return s.provider.Get(ctx, execution.ID, input)
	}
	return s.provider.Cancel(ctx, execution.ID, input)
}

func terminalExecution(status string) bool {
	return status == "succeeded" || status == "failed" || status == "cancelled"
}
