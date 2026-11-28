package stagedmessages

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/internal/reviewjobs"
)

// recordExecutionError turns an owner failure into the durable lifecycle
// receipt. It returns done when a concurrent terminal result already won.
func (s *Service) recordExecutionError(ctx context.Context, job reviewjobs.Job, action string, timeout time.Duration, attachedAt time.Time, firstDispatch bool, execution Execution, cause error) (reviewjobs.Job, bool, error) {
	next := job
	next.ErrorCode, next.Error, next.Retryable = classifyDispatchError(cause, firstDispatch)
	var done bool
	var err error
	job, next, done, err = s.refreshErrorJob(ctx, job, next, firstDispatch, cause)
	if err != nil || done {
		return job, done, err
	}
	next, err = attachUnreadableExecution(job, next, execution, cause)
	if err != nil {
		return reviewjobs.Job{}, false, err
	}
	next, err = classifyExecutionError(job, next, firstDispatch, cause)
	if err != nil {
		return reviewjobs.Job{}, false, err
	}
	if action == "wait" && !next.State.Terminal() {
		if err := waitAttachment(ctx, attachedAt, timeout); err != nil {
			return reviewjobs.Job{}, false, err
		}
	}
	if _, err := s.store.UpdateAdvisory(ctx, job, next); err != nil {
		return reviewjobs.Job{}, false, err
	}
	return reviewjobs.Job{}, false, nil
}

func (s *Service) refreshErrorJob(ctx context.Context, job, next reviewjobs.Job, firstDispatch bool, cause error) (reviewjobs.Job, reviewjobs.Job, bool, error) {
	if job.ExecutionID != "" || !job.DispatchAttempted {
		return job, next, false, nil
	}
	latest, err := s.store.Get(ctx, job.ID)
	if err != nil {
		return reviewjobs.Job{}, reviewjobs.Job{}, false, err
	}
	if latest.State.Terminal() {
		return latest, latest, true, nil
	}
	if !latest.CancelRequested {
		return job, next, false, nil
	}
	job, next = latest, latest
	next.ErrorCode, next.Error, next.Retryable = classifyDispatchError(cause, firstDispatch)
	if next.ErrorCode == "dispatch_unconfirmed" || !firstDispatch {
		next.ErrorCode = "dispatch_unconfirmed"
		next.Error = "Cancellation is waiting for Agent Manager to confirm whether the request was accepted; the same owner key will be reconciled."
		next.Retryable = false
	}
	return job, next, false, nil
}

func attachUnreadableExecution(job, next reviewjobs.Job, execution Execution, cause error) (reviewjobs.Job, error) {
	var evidenceErr EvidenceReadError
	if !errors.As(cause, &evidenceErr) || execution.ID == "" || job.CancelRequested || (job.ExecutionID != "" && job.ExecutionID != execution.ID) {
		return next, nil
	}
	next.ExecutionID = execution.ID
	if execution.Phase != "" {
		next.Phase = execution.Phase
	}
	if execution.ExecutionStartedKnown {
		next.ExecutionStarted, next.ExecutionStartedKnown = execution.ExecutionStarted, true
	}
	if execution.TokensKnown {
		next.TokensUsed, next.TokensKnown = execution.TokensUsed, true
	}
	if execution.ChargeKnown {
		next.ChargeMicroUSD, next.ChargeKnown = execution.ChargeMicroUSD, true
	}
	if job.ExecutionID == "" && execution.Status == "succeeded" && next.State == reviewjobs.Dispatching {
		if err := next.Transition(reviewjobs.Running); err != nil {
			return reviewjobs.Job{}, err
		}
	}
	return next, nil
}

func classifyExecutionError(job, next reviewjobs.Job, firstDispatch bool, cause error) (reviewjobs.Job, error) {
	if job.ExecutionID != "" && connect.CodeOf(cause) == connect.CodeNotFound {
		next.ErrorCode = "execution_missing"
		next.Error = "Agent Manager no longer has the saved execution. This job is stale; no replacement was started."
		next.Retryable = false
		if err := next.Transition(reviewjobs.Stale); err != nil {
			return reviewjobs.Job{}, err
		}
	}
	if !definitiveAdmissionRefusal(cause) || !firstDispatch || job.ExecutionID != "" {
		return next, nil
	}
	target := reviewjobs.Failed
	if job.CancelRequested {
		target = reviewjobs.Cancelled
	}
	if err := next.Transition(target); err != nil {
		return reviewjobs.Job{}, err
	}
	return next, nil
}

func definitiveAdmissionRefusal(err error) bool {
	switch connect.CodeOf(err) {
	case connect.CodeInvalidArgument, connect.CodePermissionDenied, connect.CodeUnauthenticated, connect.CodeFailedPrecondition, connect.CodeUnimplemented:
		return true
	default:
		return false
	}
}
