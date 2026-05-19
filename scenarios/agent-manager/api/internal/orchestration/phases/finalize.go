// The single terminal seam for a run: apply-at-run-end, sandbox teardown,
// final phase advancement, terminal-status broadcast. This is the package's
// most-load-bearing file because it pins the contract that prevented the
// 2026-04-28 mount-leak incident:
//
//   - Sandbox teardown MUST run even when the caller's ctx is cancelled.
//     The teardown HTTP call uses a fresh context.Background()-derived
//     deadline (Heartbeat.TeardownTimeout), so timing out the run does not
//     orphan the fuse-overlayfs mount.
//
//   - Every run reaches RunPhaseCompleted. Finalize advances the phase
//     ladder unconditionally; failure during teardown emits a warn event
//     but does not block phase advancement.
//
//   - Finalize is idempotent. Re-entry is a no-op so callers can invoke
//     directly without races against the deferred call. Idempotency lives
//     on a flag carried by the caller (RunExecutor.finalized), not here —
//     this package is pure logic.

package phases

import (
	"context"
	"fmt"
	"time"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/metrics"
	"agent-manager/internal/orchestration/obs"

	"github.com/google/uuid"
)

// FinalizeInput is the explicit input to Finalize.
type FinalizeInput struct {
	Deps      Deps
	Run       *domain.Run
	SandboxID *uuid.UUID
	Sandbox   sandbox.Provider
}

// Finalize is the terminal seam: it advances the phase ladder to
// RunPhaseCleaningUp → RunPhaseCompleted, runs the sandbox lifecycle
// dispatcher, persists the final run state, and broadcasts terminal status.
//
// Context discipline: Finalize uses a fresh context.Background()-derived
// timeout (TeardownTimeout). It does NOT take the run's execCtx — the
// whole point is to be independent of execCtx's deadline and cancellation.
func Finalize(in FinalizeInput) {
	ctx, cancel := context.WithTimeout(context.Background(), in.Deps.Levers.Heartbeat.TeardownTimeout)
	defer cancel()

	finalizeStart := time.Now()
	sink := finalizeSink(in.Deps)
	obs.EmitFinalizeStarted(sink, in.Run.ID, obs.FinalizeFields{
		SandboxID: in.SandboxID,
	})

	AdvancePhase(ctx, AdvancePhaseInput{
		Deps:  in.Deps,
		Run:   in.Run,
		Phase: domain.RunPhaseCleaningUp,
	})

	event := LifecycleEventForStatus(in.Run.Status)
	action := ApplySandboxLifecycle(ctx, ApplySandboxLifecycleInput{
		Deps:      in.Deps,
		Run:       in.Run,
		SandboxID: in.SandboxID,
		Sandbox:   in.Sandbox,
		Event:     event,
		Reason:    "finalize",
	})

	AdvancePhase(ctx, AdvancePhaseInput{
		Deps:  in.Deps,
		Run:   in.Run,
		Phase: domain.RunPhaseCompleted,
	})

	if in.Deps.Runs != nil {
		if err := in.Deps.Runs.Update(ctx, in.Run); err != nil {
			EmitSystemEvent(ctx, in.Deps, in.Run.ID, "warn",
				"failed to persist final run state: "+err.Error())
		}
	}
	if in.Deps.Broadcaster != nil {
		in.Deps.Broadcaster.BroadcastRunStatus(in.Run)
	}

	obs.EmitFinalizeCompleted(sink, in.Run.ID, obs.FinalizeFields{
		SandboxID: in.SandboxID,
		Action:    action,
	}, time.Since(finalizeStart))
}

// finalizeSink converts the phase Deps into the obs.Sink shape used by
// the lifecycle helpers. Falls back to nil — obs helpers tolerate it.
func finalizeSink(deps Deps) obs.Sink {
	if deps.Gate != nil {
		return deps.Gate
	}
	return nil
}

// LifecycleEventForStatus maps the run's terminal status to the
// SandboxLifecycleEvent that finalize should fire. Centralized so the
// switch is not duplicated across handlers.
//
// NeedsReview maps to RunCompleted: the apply step succeeded enough to
// require operator review, so any StopOn=run_completed rule should fire.
// Defensive default (status not yet terminal — e.g., panic before
// handleResult) maps to RunFailed: treat as a failed run for teardown
// purposes; the sandbox config decides whether to preserve.
func LifecycleEventForStatus(status domain.RunStatus) domain.SandboxLifecycleEvent {
	switch status {
	case domain.RunStatusComplete:
		return domain.SandboxLifecycleRunCompleted
	case domain.RunStatusCancelled:
		return domain.SandboxLifecycleRunCancelled
	case domain.RunStatusNeedsReview:
		return domain.SandboxLifecycleRunCompleted
	default:
		return domain.SandboxLifecycleRunFailed
	}
}

// ApplySandboxLifecycleInput is the explicit input to ApplySandboxLifecycle.
type ApplySandboxLifecycleInput struct {
	Deps      Deps
	Run       *domain.Run
	SandboxID *uuid.UUID
	Sandbox   sandbox.Provider
	Event     domain.SandboxLifecycleEvent
	Reason    string
}

// ApplySandboxLifecycle issues Delete or Stop on the sandbox per the run's
// lifecycle config. Called only from Finalize; not safe to call from other
// paths (no internal idempotency — the gate is the caller's finalized flag).
//
// Returns a short action label ("delete" | "stop" | "preserve") describing
// which path executed, so callers can include it in lifecycle events for
// auditability without re-deriving the decision.
//
// The HTTP call uses a DETACHED context, not the supplied ctx. The supplied
// ctx is used only for event emission. Detaching here makes the function's
// contract explicit: teardown is independent of any caller deadline.
func ApplySandboxLifecycle(ctx context.Context, in ApplySandboxLifecycleInput) string {
	if in.Run.RunMode != domain.RunModeSandboxed || in.SandboxID == nil || in.Sandbox == nil {
		return ""
	}
	cfg := EffectiveSandboxConfig(in.Run)
	if cfg == nil {
		return ""
	}

	events := []domain.SandboxLifecycleEvent{in.Event}
	if in.Event == domain.SandboxLifecycleRunCompleted ||
		in.Event == domain.SandboxLifecycleRunFailed ||
		in.Event == domain.SandboxLifecycleRunCancelled {
		events = append(events, domain.SandboxLifecycleTerminal)
	}

	teardownCtx, cancel := context.WithTimeout(context.Background(), in.Deps.Levers.Heartbeat.TeardownTimeout)
	defer cancel()

	if HasLifecycleEvent(cfg.Lifecycle.DeleteOn, events) {
		started := time.Now()
		if err := in.Sandbox.Delete(teardownCtx, *in.SandboxID); err != nil {
			EmitSandboxOperation(ctx, in.Deps, in.Run.ID, eventlog.SandboxOperationPayload{
				Operation:  eventlog.SandboxOpDelete,
				Success:    false,
				DurationMS: time.Since(started).Milliseconds(),
				Reason:     in.Reason,
				Message:    err.Error(),
			})
		} else {
			EmitSandboxOperation(ctx, in.Deps, in.Run.ID, eventlog.SandboxOperationPayload{
				Operation:  eventlog.SandboxOpDelete,
				Success:    true,
				DurationMS: time.Since(started).Milliseconds(),
				Reason:     in.Reason,
			})
		}
		return "delete"
	}

	if HasLifecycleEvent(cfg.Lifecycle.StopOn, events) {
		started := time.Now()
		if err := in.Sandbox.Stop(teardownCtx, *in.SandboxID); err != nil {
			EmitSandboxOperation(ctx, in.Deps, in.Run.ID, eventlog.SandboxOperationPayload{
				Operation:  eventlog.SandboxOpStop,
				Success:    false,
				DurationMS: time.Since(started).Milliseconds(),
				Reason:     in.Reason,
				Message:    err.Error(),
			})
		} else {
			EmitSandboxOperation(ctx, in.Deps, in.Run.ID, eventlog.SandboxOperationPayload{
				Operation:  eventlog.SandboxOpStop,
				Success:    true,
				DurationMS: time.Since(started).Milliseconds(),
				Reason:     in.Reason,
			})
		}
		return "stop"
	}
	return "preserve"
}

// EffectiveSandboxConfig returns the run's resolved sandbox config or nil.
func EffectiveSandboxConfig(run *domain.Run) *domain.SandboxConfig {
	if run != nil && run.SandboxConfig != nil {
		return run.SandboxConfig
	}
	if run != nil && run.ResolvedConfig != nil && run.ResolvedConfig.SandboxConfig != nil {
		return run.ResolvedConfig.SandboxConfig
	}
	return nil
}

// HasLifecycleEvent returns true if any candidate appears in events.
func HasLifecycleEvent(events []domain.SandboxLifecycleEvent, candidates []domain.SandboxLifecycleEvent) bool {
	for _, candidate := range candidates {
		for _, event := range events {
			if event == candidate {
				return true
			}
		}
	}
	return false
}

// ShouldPreserveSandbox returns true when neither StopOn nor DeleteOn would
// fire for the supplied event (or the synthetic Terminal event). Used by
// the failure path to log "sandbox preserved for inspection" before
// finalize runs the actual teardown.
func ShouldPreserveSandbox(run *domain.Run, event domain.SandboxLifecycleEvent) bool {
	cfg := EffectiveSandboxConfig(run)
	if cfg == nil {
		return true
	}
	events := []domain.SandboxLifecycleEvent{event, domain.SandboxLifecycleTerminal}
	if HasLifecycleEvent(cfg.Lifecycle.StopOn, events) || HasLifecycleEvent(cfg.Lifecycle.DeleteOn, events) {
		return false
	}
	return true
}

// ApplyAtRunEndInput is the explicit input to ApplyAtRunEnd.
type ApplyAtRunEndInput struct {
	Deps      Deps
	Run       *domain.Run
	SandboxID *uuid.UUID
	Sandbox   sandbox.Provider
	Outcome   domain.ContractRunOutcome
	Cost      float64
}

// ApplyAtRunEnd is the single shared post-turn apply seam called from every
// terminal handler (success, failure, cancel, timeout). It encodes the
// auditability contract: in-acceptance changes apply at turn end, regardless
// of run outcome, and out-of-acceptance changes are retained as
// state=pending-review on the resulting provenance record. When the sandbox
// lifecycle declares a continuable turn checkpoint, this seam calls
// TurnCheckpoint explicitly; otherwise it calls the final apply-at-run-end
// provider method.
//
// Returns true iff the run's terminal status should be RunStatusComplete
// with ApprovalState=Approved (i.e., apply succeeded). Returns false in
// three cases: ManualReview=true (sandbox persists for operator approval),
// AutoApply=false (operator opted out), or apply failed.
//
// Mutates in.Run: Status, ApprovalState, ApprovedBy, ApprovedAt are set
// based on the apply outcome.
func ApplyAtRunEnd(ctx context.Context, in ApplyAtRunEndInput) bool {
	cfg := EffectiveSandboxConfig(in.Run)
	if cfg == nil {
		// Defensive: resolveSandboxConfig guarantees a non-nil config for
		// sandboxed runs since 2026-04-24. If we land here, the orchestrator
		// constructed a run without going through resolveSandboxConfig — a bug.
		EmitSystemEvent(ctx, in.Deps, in.Run.ID, "warn",
			"apply-at-run-end skipped: run has no sandbox config (resolve bug — please report)")
		domain.MarkFinalizationFailed(in.Run, fmt.Errorf("run has no sandbox config"), time.Now())
		return false
	}
	if in.Sandbox == nil || in.SandboxID == nil {
		EmitSystemEvent(ctx, in.Deps, in.Run.ID, "warn",
			"apply-at-run-end skipped: no sandbox available")
		domain.MarkFinalizationFailed(in.Run, fmt.Errorf("no sandbox available"), time.Now())
		return false
	}

	// ManualReview=true defers apply until operator approval. The sandbox
	// persists past run end; the run terminates as Complete with
	// ApprovalState=Pending so the AI Changes review queue surfaces it.
	if cfg.ManualReview {
		now := time.Now()
		in.Run.ApprovalState = domain.ApprovalStatePending
		in.Run.ApprovedAt = &now
		in.Run.Status = domain.RunStatusNeedsReview
		domain.MarkFinalizationSkipped(in.Run, "manualReview=true", now)
		EmitSystemEvent(ctx, in.Deps, in.Run.ID, "info",
			"apply deferred: manualReview=true (operator approval required)")
		return false
	}

	if !cfg.GetAutoApply() {
		EmitSystemEvent(ctx, in.Deps, in.Run.ID, "info", "apply skipped: autoApply=false")
		domain.MarkFinalizationSkipped(in.Run, "autoApply=false", time.Now())
		return false
	}

	if in.Outcome != domain.ContractRunOutcomeSuccess && !cfg.GetApplyOnFailure() {
		EmitSystemEvent(ctx, in.Deps, in.Run.ID, "info",
			fmt.Sprintf("apply skipped: applyOnFailure=false (outcome=%s)", in.Outcome))
		domain.MarkFinalizationSkipped(in.Run, fmt.Sprintf("applyOnFailure=false (outcome=%s)", in.Outcome), time.Now())
		return false
	}

	domain.MarkFinalizationRunning(in.Run, time.Now())
	result, err := applyOrCheckpointTurn(ctx, cfg, in)
	if err != nil {
		EmitSystemEvent(ctx, in.Deps, in.Run.ID, "warn", "apply-at-run-end failed: "+err.Error())
		metrics.Get().RecordProvenanceSkipped()
		domain.MarkFinalizationFailed(in.Run, err, time.Now())
		return false
	}

	metrics.Get().RecordProvenanceWrite()

	now := time.Now()
	in.Run.ApprovedBy = "applyAtRunEnd"
	in.Run.ApprovedAt = &now

	if result != nil && result.IsPartial {
		// Out-of-acceptance files retained as state=pending-review.
		in.Run.ApprovalState = domain.ApprovalStateApproved
		if in.Outcome == domain.ContractRunOutcomeSuccess {
			in.Run.Status = domain.RunStatusComplete
		}
		domain.MarkFinalizationSucceeded(in.Run, now)
		EmitSystemEvent(ctx, in.Deps, in.Run.ID, "info",
			fmt.Sprintf("partial apply: %d applied, %d pending review", result.Applied, result.Remaining))
		return true
	}

	if result != nil && result.Applied == 0 {
		EmitSystemEvent(ctx, in.Deps, in.Run.ID, "info",
			"apply-at-run-end recorded empty provenance (no changes)")
		if in.Outcome != domain.ContractRunOutcomeSuccess {
			// Empty provenance + non-success outcome: nothing was applied,
			// so the "change is the audit unit" contract does NOT promote
			// the failed run to Complete.
			// Regression of the 2026-04-28 silent-COMPLETE-despite-error bug.
			domain.MarkFinalizationSucceeded(in.Run, now)
			return false
		}
	}
	in.Run.ApprovalState = domain.ApprovalStateApproved
	if in.Outcome == domain.ContractRunOutcomeSuccess {
		in.Run.Status = domain.RunStatusComplete
	}
	domain.MarkFinalizationSucceeded(in.Run, now)
	return true
}

type postTurnApplyResult struct {
	Applied    int
	Remaining  int
	IsPartial  bool
	CommitHash string
	AppliedAt  time.Time
}

func applyOrCheckpointTurn(ctx context.Context, cfg *domain.SandboxConfig, in ApplyAtRunEndInput) (*postTurnApplyResult, error) {
	turnEvent := turnLifecycleEventForOutcome(in.Outcome)
	if HasLifecycleEvent(cfg.Lifecycle.CheckpointOn, []domain.SandboxLifecycleEvent{turnEvent}) {
		req := sandbox.TurnCheckpointRequest{
			SandboxID:      *in.SandboxID,
			RunID:          in.Run.ID.String(),
			ConversationID: in.Run.ConversationID,
			Cost:           in.Cost,
			RunOutcome:     string(in.Outcome),
			Actor:          "applyAtRunEnd",
		}
		result, err := postTurnSandboxOperation(ctx, in, "turn_checkpoint", func() (*sandbox.TurnCheckpointResult, error) {
			return in.Sandbox.TurnCheckpoint(ctx, req)
		})
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, nil
		}
		return &postTurnApplyResult{
			Applied:    result.Applied,
			Remaining:  result.Remaining,
			IsPartial:  result.IsPartial,
			CommitHash: result.CommitHash,
			AppliedAt:  result.AppliedAt,
		}, nil
	}

	req := sandbox.ApplyAtRunEndRequest{
		SandboxID:      *in.SandboxID,
		RunID:          in.Run.ID.String(),
		ConversationID: in.Run.ConversationID,
		Cost:           in.Cost,
		RunOutcome:     string(in.Outcome),
		Actor:          "applyAtRunEnd",
	}
	result, err := postTurnSandboxOperation(ctx, in, "apply_at_run_end", func() (*sandbox.ApplyAtRunEndResult, error) {
		return in.Sandbox.ApplyAtRunEnd(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return &postTurnApplyResult{
		Applied:    result.Applied,
		Remaining:  result.Remaining,
		IsPartial:  result.IsPartial,
		CommitHash: result.CommitHash,
		AppliedAt:  result.AppliedAt,
	}, nil
}

func postTurnSandboxOperation[T any](ctx context.Context, in ApplyAtRunEndInput, operation string, call func() (*T, error)) (*T, error) {
	maxAttempts := in.Deps.Levers.Sandbox.OperationMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	var lastErr error
	ensured := false
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, err := call()
		if err == nil {
			if attempt > 1 {
				EmitSystemEvent(ctx, in.Deps, in.Run.ID, "info",
					fmt.Sprintf("%s succeeded after %d attempts", operation, attempt))
			}
			return result, nil
		}
		lastErr = err
		if !retryableSandboxError(err) || attempt == maxAttempts {
			break
		}
		EmitSystemEvent(ctx, in.Deps, in.Run.ID, "warn",
			fmt.Sprintf("%s attempt %d/%d failed; retrying: %v", operation, attempt, maxAttempts, err))
		if !ensured && in.Deps.WorkspaceSandbox != nil {
			ensured = true
			if ensureErr := in.Deps.WorkspaceSandbox.EnsureAvailable(ctx); ensureErr != nil {
				return nil, workspaceSandboxUnavailableError(operation, err.Error(), ensureErr)
			}
		}
		if waitErr := waitForSandboxRetry(ctx, in.Deps.Levers.Sandbox, attempt); waitErr != nil {
			return nil, waitErr
		}
	}
	return nil, lastErr
}

func turnLifecycleEventForOutcome(outcome domain.ContractRunOutcome) domain.SandboxLifecycleEvent {
	switch outcome {
	case domain.ContractRunOutcomeFailure, domain.ContractRunOutcomeTimeout:
		return domain.SandboxLifecycleTurnFailed
	case domain.ContractRunOutcomeCancelled:
		return domain.SandboxLifecycleTurnCancelled
	default:
		return domain.SandboxLifecycleTurnCompleted
	}
}
