// Failure-handling helpers used by the executor coordinator.
//
// These are setup-phase failure paths: they fire when SetupWorkspace,
// AcquireRunner, or status-update fail before the runner has a chance
// to run. The runner-result failure path (HandleFailure) lives in
// result.go because it operates on a runner's verdict, not on a setup
// error.

package phases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/config"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// FailWithErrorInput is the explicit input to FailWithError.
type FailWithErrorInput struct {
	Deps Deps
	Run  *domain.Run
	Err  error
}

// FailWithErrorOutput carries the classified outcome the caller should
// record on the executor.
type FailWithErrorOutput struct {
	Outcome domain.RunOutcome
}

const terminalPersistAttempts = 3

// persistTerminalRun retries a terminal transition with a short detached
// context. Terminal status is more important than the request that discovered
// the error: a cancelled request must not turn a real failure into a stranded
// active run. The bounded retry keeps the failure path finite and leaves a
// visible system event when durable persistence still cannot succeed.
func persistTerminalRun(parent context.Context, deps Deps, run *domain.Run) error {
	if deps.Runs == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), time.Second)
	defer cancel()
	var lastErr error
	for attempt := 0; attempt < terminalPersistAttempts; attempt++ {
		if err := deps.Runs.Update(ctx, run); err == nil {
			return nil
		} else {
			lastErr = err
			var stateErr *domain.StateError
			if errors.As(err, &stateErr) {
				return err
			} // A lost lifecycle fence is not transient.
		}
		if attempt+1 < terminalPersistAttempts {
			select {
			case <-ctx.Done():
				return lastErr
			case <-time.After(time.Duration(attempt+1) * 25 * time.Millisecond):
			}
		}
	}
	return lastErr
}

// FailWithError marks the run as failed with proper error classification.
// Errors are captured with full context, the failure event is emitted,
// the failure state is persisted, and the broadcaster is notified.
func FailWithError(ctx context.Context, in FailWithErrorInput) FailWithErrorOutput {
	previousStatus := in.Run.Status
	now := in.Deps.Now()
	in.Run.Status = domain.RunStatusFailed
	in.Run.EndedAt = &now
	in.Run.UpdatedAt = now
	providerMaintenanceRefusal := ApplyProviderMaintenanceFailureReceipt(in.Run, in.Err)

	if providerMaintenanceRefusal {
		var runnerErr *domain.RunnerError
		_ = errors.As(in.Err, &runnerErr)
		safeErr := *runnerErr
		safeErr.Cause = errors.New(in.Run.ErrorMsg)
		EmitFailureEvent(ctx, in.Deps, in.Run.ID, &safeErr)
	} else if domainErr, ok := in.Err.(domain.DomainError); ok {
		in.Run.ErrorMsg = domainErr.UserMessage()
		EmitFailureEvent(ctx, in.Deps, in.Run.ID, domainErr)
	} else {
		in.Run.ErrorMsg = in.Err.Error()
		EmitGenericFailureEvent(ctx, in.Deps, in.Run.ID, in.Err)
	}

	outcome := domain.ClassifyErrorOutcome(in.Err)

	if in.Deps.Runs != nil {
		if updateErr := persistTerminalRun(ctx, in.Deps, in.Run); updateErr != nil {
			EmitSystemEvent(ctx, in.Deps, in.Run.ID, "error",
				"failed to persist failure state: "+updateErr.Error())
			return FailWithErrorOutput{Outcome: outcome}
		}
		EmitStatusTransition(ctx, in.Deps, in.Run.ID, previousStatus, in.Run.Status, "run failed")
	}

	if in.Deps.Broadcaster != nil {
		in.Deps.Broadcaster.BroadcastRunStatus(ctx, in.Run)
	}

	return FailWithErrorOutput{Outcome: outcome}
}

// ApplyProviderMaintenanceFailureReceipt accepts only the runner owner's
// definitive pre-launch receipt plus Workspace Sandbox's specific refusal.
// HTTP status alone is ambiguous because a dispatch can be accepted before an
// acknowledgement is lost.
func ApplyProviderMaintenanceFailureReceipt(run *domain.Run, err error) bool {
	var runnerErr *domain.RunnerError
	if run == nil || !errors.As(err, &runnerErr) || !runnerErr.ExecutionStartedKnown || runnerErr.ExecutionStarted || runnerErr.Cause == nil || !domain.IsPreEffectRefusal(runnerErr.Cause) || !errors.Is(runnerErr.Cause, domain.ErrProviderAdmissionMaintenance) {
		return false
	}
	run.TerminalClass = domain.RunTerminalClassInterruption
	run.StopReason = domain.RunStopReasonProviderMaintenance
	run.ErrorMsg = "Agent execution is paused for Workspace Sandbox maintenance; no agent work started."
	return true
}

// HandleContextErrorInput is the explicit input to HandleContextError.
type HandleContextErrorInput struct {
	Deps      Deps
	Run       *domain.Run
	Profile   *domain.AgentProfile
	SandboxID *uuid.UUID
	Sandbox   sandbox.Provider
	// SessionID, if non-empty, is preserved on Run.SessionID across timeout
	// so continuation can pick up where the timed-out run left off.
	SessionID string
	Err       error
	Levers    config.Levers
}

// HandleContextErrorOutput carries the resulting outcome.
type HandleContextErrorOutput struct {
	Outcome domain.RunOutcome
}

// HandleContextError handles context cancellation or timeout. On
// DeadlineExceeded it records a typed RunnerError with the "timeout"
// operation; on Canceled it just marks the run as cancelled.
func HandleContextError(ctx context.Context, in HandleContextErrorInput) HandleContextErrorOutput {
	out := HandleContextErrorOutput{}

	switch in.Err {
	case context.DeadlineExceeded:
		// Preserve session ID for continuation even on timeout.
		if in.SessionID != "" {
			in.Run.SessionID = in.SessionID
		}
		fail := FailWithError(ctx, FailWithErrorInput{
			Deps: in.Deps,
			Run:  in.Run,
			Err: &domain.RunnerError{
				RunnerType:  GetRunnerType(in.Run, in.Profile),
				Operation:   "timeout",
				Cause:       fmt.Errorf("execution exceeded timeout of %v", in.Levers.Execution.DefaultTimeout),
				IsTransient: true,
			},
		})
		out.Outcome = fail.Outcome
		// Override the classifier's verdict: timeout is its own outcome.
		out.Outcome = domain.RunOutcomeTimeout
	case context.Canceled:
		EmitSystemEvent(ctx, in.Deps, in.Run.ID, "info", "execution cancelled")
		out.Outcome = domain.RunOutcomeCancelled
		previousStatus := in.Run.Status
		now := in.Deps.Now()
		in.Run.Status = domain.RunStatusCancelled
		in.Run.EndedAt = &now
		in.Run.UpdatedAt = now
		if in.Deps.Runs != nil {
			if updateErr := persistTerminalRun(ctx, in.Deps, in.Run); updateErr != nil {
				EmitSystemEvent(ctx, in.Deps, in.Run.ID, "warn",
					"failed to persist cancellation: "+updateErr.Error())
				return out
			}
		}
		if in.Deps.Runs != nil {
			EmitStatusTransition(ctx, in.Deps, in.Run.ID, previousStatus, in.Run.Status, "execution cancelled")
		}
	}

	if in.Deps.Broadcaster != nil {
		in.Deps.Broadcaster.BroadcastRunStatus(ctx, in.Run)
	}

	CleanupOnFailure(ctx, in.Deps, in.Run)
	return out
}

// CleanupOnFailure runs after a setup-phase failure. It emits an
// informational event noting whether the sandbox is being preserved per
// the lifecycle config; the actual Delete/Stop call is centralized in
// Finalize — see Execute()'s defer.
func CleanupOnFailure(ctx context.Context, deps Deps, run *domain.Run) {
	if ShouldPreserveSandbox(run, domain.SandboxLifecycleRunFailed) {
		EmitSystemEvent(ctx, deps, run.ID, "info",
			"run failed - sandbox preserved for inspection")
	}
}
