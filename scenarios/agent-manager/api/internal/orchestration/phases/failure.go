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

// FailWithError marks the run as failed with proper error classification.
// Errors are captured with full context, the failure event is emitted,
// the failure state is persisted, and the broadcaster is notified.
func FailWithError(ctx context.Context, in FailWithErrorInput) FailWithErrorOutput {
	now := time.Now()
	in.Run.Status = domain.RunStatusFailed
	in.Run.EndedAt = &now
	in.Run.UpdatedAt = now

	if domainErr, ok := in.Err.(domain.DomainError); ok {
		in.Run.ErrorMsg = domainErr.UserMessage()
		EmitFailureEvent(ctx, in.Deps, in.Run.ID, domainErr)
	} else {
		in.Run.ErrorMsg = in.Err.Error()
		EmitGenericFailureEvent(ctx, in.Deps, in.Run.ID, in.Err)
	}

	outcome := ClassifyErrorOutcome(in.Err)

	if in.Deps.Runs != nil {
		if updateErr := in.Deps.Runs.Update(ctx, in.Run); updateErr != nil {
			EmitSystemEvent(ctx, in.Deps, in.Run.ID, "error",
				"failed to persist failure state: "+updateErr.Error())
		}
	}

	if in.Deps.Broadcaster != nil {
		in.Deps.Broadcaster.BroadcastRunStatus(in.Run)
	}

	return FailWithErrorOutput{Outcome: outcome}
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
		now := time.Now()
		in.Run.Status = domain.RunStatusCancelled
		in.Run.EndedAt = &now
		in.Run.UpdatedAt = now
		if in.Deps.Runs != nil {
			if updateErr := in.Deps.Runs.Update(ctx, in.Run); updateErr != nil {
				EmitSystemEvent(ctx, in.Deps, in.Run.ID, "warn",
					"failed to persist cancellation: "+updateErr.Error())
			}
		}
	}

	if in.Deps.Broadcaster != nil {
		in.Deps.Broadcaster.BroadcastRunStatus(in.Run)
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

// PhaseOrdinal returns the numeric order of a phase for comparison.
// Used by the executor to detect "should we skip this phase on resume".
func PhaseOrdinal(phase domain.RunPhase) int {
	switch phase {
	case domain.RunPhaseQueued:
		return 0
	case domain.RunPhaseInitializing:
		return 1
	case domain.RunPhaseSandboxCreating:
		return 2
	case domain.RunPhaseRunnerAcquiring:
		return 3
	case domain.RunPhaseExecuting:
		return 4
	case domain.RunPhaseCollectingResults:
		return 5
	case domain.RunPhaseAwaitingReview:
		return 6
	case domain.RunPhaseApplying:
		return 7
	case domain.RunPhaseCleaningUp:
		return 8
	case domain.RunPhaseCompleted:
		return 9
	default:
		return 0
	}
}
