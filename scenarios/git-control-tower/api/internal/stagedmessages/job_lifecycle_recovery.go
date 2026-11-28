package stagedmessages

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/internal/reviewjobs"
)

// resolveCancelledDispatch never retries Start after cancellation. It first
// asks the owner for an execution already admitted under the durable key. An
// authoritative miss proves Start was not accepted; a found execution is
// attached durably and observed from the owner's post-cancellation receipt.
// Lookup outages preserve Cancelling and uncertainty without creating work.
func (s *Service) resolveCancelledDispatch(ctx context.Context, repo string, job reviewjobs.Job, input DraftInput, timeout time.Duration) (reviewjobs.Job, error) {
	fencer, ok := s.provider.(DispatchCancellationProvider)
	if !ok {
		return s.recordDispatchUnconfirmed(ctx, job)
	}
	result, err := fencer.CancelByKey(ctx, job.IdempotencyKey, input)
	if err != nil || !result.Fenced {
		// A failed or non-fenced owner response cannot prove that a Start RPC
		// still in flight will not be accepted later.
		return s.recordDispatchUnconfirmed(ctx, job)
	}
	if result.Execution == nil {
		// The owner committed its cancellation fence and found no admitted
		// execution in the same serialized boundary. A late Start is now refused.
		next := job
		if err := next.Transition(reviewjobs.Cancelled); err != nil {
			return reviewjobs.Job{}, err
		}
		next.ErrorCode = "cancelled_before_start"
		next.Error = "Cancellation fenced the request before Agent Manager accepted it. No agent work started."
		next.Retryable = false
		next.ExecutionStarted, next.ExecutionStartedKnown = false, true
		next.TokensUsed, next.TokensKnown = 0, true
		next.ChargeMicroUSD, next.ChargeKnown = 0, true
		if _, updateErr := s.store.UpdateAdvisory(ctx, job, next); updateErr != nil {
			return reviewjobs.Job{}, updateErr
		}
		return s.store.Get(ctx, job.ID)
	}
	execution := *result.Execution
	if execution.ID == "" {
		return s.recordDispatchUnconfirmed(ctx, job)
	}
	next := job
	next.ExecutionID = execution.ID
	next.ErrorCode = ""
	next.Error = ""
	next.Retryable = false
	if _, err := s.store.UpdateAdvisory(ctx, job, next); err != nil {
		return reviewjobs.Job{}, err
	}
	attached, err := s.store.Get(ctx, job.ID)
	if err != nil {
		return reviewjobs.Job{}, err
	}
	// The owner fence endpoint already requested cancellation when the execution
	// was nonterminal. Observe its exact receipt instead of issuing another
	// cancellation, which can be invalid if the execution completed in between.
	return s.reconcile(ctx, repo, attached, input, ownerReceiptAction, timeout)
}

func (s *Service) recordDispatchUnconfirmed(ctx context.Context, job reviewjobs.Job) (reviewjobs.Job, error) {
	if job.ErrorCode == "dispatch_unconfirmed" {
		return job, nil
	}
	next := job
	next.ErrorCode = "dispatch_unconfirmed"
	next.Error = "Cancellation is waiting for Agent Manager to confirm whether the request was accepted; no new Start will be sent after cancellation."
	next.Retryable = false
	if _, err := s.store.UpdateAdvisory(ctx, job, next); err != nil {
		return reviewjobs.Job{}, err
	}
	return s.store.Get(ctx, job.ID)
}

// classifyDispatchError persists a bounded, operator-safe receipt. Transport
// details can contain implementation or credential context, so they stay out
// of the durable job/UI. Ambiguous owner acknowledgements remain nonterminal.
func classifyDispatchError(err error, firstDispatch bool) (code, message string, retryable bool) {
	var evidenceErr EvidenceReadError
	if errors.As(err, &evidenceErr) {
		return "evidence_unavailable", "Agent Manager finished the execution, but its saved result evidence could not be read. The job remains attached; reconnect to retrieve the same result.", true
	}
	// Authentication and policy errors remain actionable after dispatch too:
	// the durable execution identity is retained, and reconnecting after the
	// access issue is fixed must observe that same run rather than start another.
	switch connect.CodeOf(err) {
	case connect.CodeUnauthenticated:
		return "authentication_required", "Agent Manager authentication is required to reconnect to this saved execution. Restore access, then reconnect to the same job; no replacement was started.", true
	case connect.CodePermissionDenied:
		if firstDispatch {
			return "policy_denied", "Agent Manager policy denied the request. Review the authorization policy before starting a new job.", false
		}
		return "policy_denied", "Agent Manager policy denied access to this saved execution. Resolve the policy, then reconnect to the same job; no replacement was started.", true
	}
	if !firstDispatch {
		return "execution_status_unavailable", "Agent Manager could not confirm the current execution status; the saved execution identity is retained.", true
	}
	switch connect.CodeOf(err) {
	case connect.CodeInvalidArgument:
		return "invalid_request", "Agent Manager rejected the request as invalid.", false
	case connect.CodeUnauthenticated:
		return "authentication_required", "Agent Manager authentication is required before this request can start.", true
	case connect.CodePermissionDenied:
		return "policy_denied", "Agent Manager policy denied the request.", false
	case connect.CodeFailedPrecondition:
		return "workflow_precondition_failed", "Agent Manager refused this workflow request before confirming an execution.", false
	case connect.CodeUnimplemented:
		return "unsupported_operation", "Agent Manager does not support this request.", false
	default:
		return "dispatch_unconfirmed", "Could not confirm whether Agent Manager accepted the request; the existing request will be reconciled with the same owner key.", false
	}
}

func waitAttachment(ctx context.Context, started time.Time, timeout time.Duration) error {
	remaining := timeout - time.Since(started)
	if remaining <= 0 {
		return nil
	}
	timer := time.NewTimer(remaining)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
