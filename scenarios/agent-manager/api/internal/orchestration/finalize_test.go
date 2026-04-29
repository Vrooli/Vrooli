// Regression gates for the deferred finalize() seam in run_executor.go.
//
// These tests pin the contract that prevented the 2026-04-28 mount-leak
// incident from recurring:
//
//   - Sandbox teardown MUST run even when the caller's ctx is already
//     cancelled. Before the fix, applySandboxLifecycle was called with
//     execCtx (which carries the run's 60-min execution deadline); a
//     slow applyAtRunEnd could exhaust the deadline, after which Delete
//     failed silently with "context deadline exceeded" and the
//     fuse-overlayfs mount was leaked. After the fix, applySandboxLifecycle
//     uses a detached context and is unaffected by caller cancellation.
//
//   - Every run reaches RunPhaseCompleted. Before the fix, Execute returned
//     after handleResult without advancing past RunPhaseCollectingResults;
//     456 of 590 runs were stuck at the COLLECTING_RESULTS phase. After
//     the fix, finalize() advances the phase ladder to its terminal value.
//
//   - finalize() is idempotent. Re-entry is a no-op so callers can invoke
//     it directly without races against the deferred call.

package orchestration

import (
	"context"
	"errors"
	"testing"

	"agent-manager/internal/domain"
)

// sandboxedRunCfg returns a SandboxConfig that mirrors what
// resolveSandboxConfig produces for an auto-apply protected run:
// DeleteOn=[terminal] so finalize will issue Delete on terminal events.
func sandboxedRunCfg() *domain.SandboxConfig {
	cfg := domain.DefaultSandboxConfig()
	cfg.Lifecycle.DeleteOn = []domain.SandboxLifecycleEvent{domain.SandboxLifecycleTerminal}
	return cfg
}

// TestApplySandboxLifecycle_DeletesEvenWithCancelledCallerCtx is the
// regression gate for the 2026-04-28 incident. The caller passes a
// context that is already cancelled (simulating execCtx after timeout);
// applySandboxLifecycle must STILL call Delete, because it derives its
// own detached context for the HTTP call.
func TestApplySandboxLifecycle_DeletesEvenWithCancelledCallerCtx(t *testing.T) {
	stub := &stubSandbox{}
	exec, _ := newTestExecutor(t, sandboxedRunCfg(), stub)
	exec.run.Status = domain.RunStatusComplete

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already-dead caller ctx

	exec.applySandboxLifecycle(ctx, domain.SandboxLifecycleRunCompleted, "regression")

	if stub.deleteHits != 1 {
		t.Fatalf("expected 1 Delete call despite cancelled caller ctx, got %d (regression: 2026-04-28 mount leak)", stub.deleteHits)
	}
	if stub.deleteCtxErr != nil {
		t.Errorf("Delete was called with a context that already had ctx.Err()=%v — teardown ctx is not detached", stub.deleteCtxErr)
	}
}

// TestFinalize_AdvancesPhaseToCompleted pins Bug B: every run must reach
// RunPhaseCompleted. Before the fix the run was left at
// RunPhaseCollectingResults (456 of 590 production runs).
func TestFinalize_AdvancesPhaseToCompleted(t *testing.T) {
	stub := &stubSandbox{}
	exec, _ := newTestExecutor(t, sandboxedRunCfg(), stub)
	exec.run.Status = domain.RunStatusComplete

	exec.finalize()

	if exec.run.Phase != domain.RunPhaseCompleted {
		t.Errorf("expected run.Phase=%s after finalize, got %s", domain.RunPhaseCompleted, exec.run.Phase)
	}
}

// TestFinalize_IsIdempotent — calling finalize twice must not double-fire
// teardown. The first call sets `finalized`; the second short-circuits.
func TestFinalize_IsIdempotent(t *testing.T) {
	stub := &stubSandbox{}
	exec, _ := newTestExecutor(t, sandboxedRunCfg(), stub)
	exec.run.Status = domain.RunStatusComplete

	exec.finalize()
	exec.finalize()

	if stub.deleteHits != 1 {
		t.Errorf("expected exactly 1 Delete call across 2 finalize() calls, got %d", stub.deleteHits)
	}
}

// TestFinalize_DeletesSandboxOnSuccess verifies the success → Delete wire
// shape: Delete is called once with the sandbox ID, no Stop.
func TestFinalize_DeletesSandboxOnSuccess(t *testing.T) {
	stub := &stubSandbox{}
	exec, _ := newTestExecutor(t, sandboxedRunCfg(), stub)
	exec.run.Status = domain.RunStatusComplete

	exec.finalize()

	if stub.deleteHits != 1 {
		t.Errorf("expected 1 Delete on successful run, got %d", stub.deleteHits)
	}
	if stub.stopHits != 0 {
		t.Errorf("expected 0 Stop calls when DeleteOn matches, got %d", stub.stopHits)
	}
}

// TestFinalize_DeletesSandboxOnFailure verifies that failed runs also
// release their sandbox under the default lifecycle (DeleteOn=[terminal]).
// Pre-fix this path also leaked because handleFailure used execCtx.
func TestFinalize_DeletesSandboxOnFailure(t *testing.T) {
	stub := &stubSandbox{}
	exec, _ := newTestExecutor(t, sandboxedRunCfg(), stub)
	exec.run.Status = domain.RunStatusFailed

	exec.finalize()

	if stub.deleteHits != 1 {
		t.Errorf("expected 1 Delete on failed run (DeleteOn=[terminal] matches RunFailed), got %d", stub.deleteHits)
	}
}

// TestFinalize_DeleteFailureDoesNotBlockPhaseAdvance — when Delete returns
// an error (e.g., workspace-sandbox unreachable), finalize must STILL
// advance the phase to Completed. The error is recorded as a warn event
// but is not load-bearing for the run's terminal status.
func TestFinalize_DeleteFailureDoesNotBlockPhaseAdvance(t *testing.T) {
	stub := &stubSandbox{deleteErr: errors.New("workspace-sandbox unreachable")}
	exec, ev := newTestExecutor(t, sandboxedRunCfg(), stub)
	exec.run.Status = domain.RunStatusComplete

	exec.finalize()

	if exec.run.Phase != domain.RunPhaseCompleted {
		t.Errorf("phase must advance to Completed even when Delete errors, got %s", exec.run.Phase)
	}
	if _, ok := ev.findMessage("failed to delete sandbox"); !ok {
		t.Error("expected warn event recording the Delete error")
	}
}

// TestFinalize_NoOpForInPlaceRun — in-place runs have no sandbox, so
// finalize must not call Delete or Stop. The phase ladder still advances.
func TestFinalize_NoOpForInPlaceRun(t *testing.T) {
	stub := &stubSandbox{}
	exec, _ := newTestExecutor(t, sandboxedRunCfg(), stub)
	exec.run.RunMode = domain.RunModeInPlace
	exec.run.Status = domain.RunStatusComplete

	exec.finalize()

	if stub.deleteHits != 0 || stub.stopHits != 0 {
		t.Errorf("in-place run should not touch sandbox: deleteHits=%d stopHits=%d", stub.deleteHits, stub.stopHits)
	}
	if exec.run.Phase != domain.RunPhaseCompleted {
		t.Errorf("phase ladder must still advance for in-place runs, got %s", exec.run.Phase)
	}
}

// TestFinalize_StopsSandboxWhenLifecycleSaysStop — non-default lifecycle
// configurations (StopOn=[terminal] without DeleteOn) must invoke Stop
// instead of Delete. Pins the dispatch logic in applySandboxLifecycle.
func TestFinalize_StopsSandboxWhenLifecycleSaysStop(t *testing.T) {
	cfg := domain.DefaultSandboxConfig()
	cfg.Lifecycle.StopOn = []domain.SandboxLifecycleEvent{domain.SandboxLifecycleTerminal}
	cfg.Lifecycle.DeleteOn = nil

	stub := &stubSandbox{}
	exec, _ := newTestExecutor(t, cfg, stub)
	exec.run.Status = domain.RunStatusComplete

	exec.finalize()

	if stub.stopHits != 1 {
		t.Errorf("expected 1 Stop call when StopOn=[terminal] and DeleteOn empty, got %d", stub.stopHits)
	}
	if stub.deleteHits != 0 {
		t.Errorf("expected 0 Delete calls under StopOn lifecycle, got %d", stub.deleteHits)
	}
}

// TestLifecycleEventForStatus_MapsAllTerminalStatuses pins the static
// mapping of run status → lifecycle event used by finalize. Treat this
// as a contract test; changing the map requires updating both halves.
func TestLifecycleEventForStatus_MapsAllTerminalStatuses(t *testing.T) {
	cases := []struct {
		status domain.RunStatus
		want   domain.SandboxLifecycleEvent
	}{
		{domain.RunStatusComplete, domain.SandboxLifecycleRunCompleted},
		{domain.RunStatusFailed, domain.SandboxLifecycleRunFailed},
		{domain.RunStatusCancelled, domain.SandboxLifecycleRunCancelled},
		{domain.RunStatusNeedsReview, domain.SandboxLifecycleRunCompleted},
		// Defensive default: non-terminal status (panic before handleResult)
		// maps to RunFailed so the lifecycle config decides whether to
		// preserve. Pin this so the default case can't silently change.
		{domain.RunStatusRunning, domain.SandboxLifecycleRunFailed},
	}
	for _, tc := range cases {
		got := lifecycleEventForStatus(tc.status)
		if got != tc.want {
			t.Errorf("lifecycleEventForStatus(%s) = %s, want %s", tc.status, got, tc.want)
		}
	}
}
