package domain

import (
	"strings"
	"time"
)

// MarkFinalizationRunning records that post-run sandbox effects are in flight.
func MarkFinalizationRunning(run *Run, now time.Time) {
	if run == nil {
		return
	}
	run.FinalizationStatus = RunFinalizationStatusRunning
	run.FinalizationError = ""
	run.FinalizedAt = nil
	run.UpdatedAt = now
}

// MarkFinalizationSucceeded records successful post-run sandbox effects.
func MarkFinalizationSucceeded(run *Run, now time.Time) {
	if run == nil {
		return
	}
	run.FinalizationStatus = RunFinalizationStatusSucceeded
	run.FinalizationError = ""
	run.FinalizedAt = &now
	run.UpdatedAt = now
}

// MarkFinalizationSkipped records that no post-run sandbox effects were needed.
func MarkFinalizationSkipped(run *Run, reason string, now time.Time) {
	if run == nil {
		return
	}
	run.FinalizationStatus = RunFinalizationStatusSkipped
	run.FinalizationError = strings.TrimSpace(reason)
	run.FinalizedAt = &now
	run.UpdatedAt = now
}

// MarkFinalizationFailed records failed post-run sandbox effects without
// changing the runner turn status.
func MarkFinalizationFailed(run *Run, err error, now time.Time) {
	if run == nil {
		return
	}
	run.FinalizationStatus = RunFinalizationStatusFailed
	run.FinalizationError = ""
	if err != nil {
		run.FinalizationError = strings.TrimSpace(err.Error())
	}
	run.FinalizedAt = &now
	run.UpdatedAt = now
}

// RequiredFinalizationPending separates runner termination from delivery of
// automatic sandbox effects. Explicit manual-review/no-apply policies remain
// valid terminal results. Missing config or an unrecorded required outcome is
// unresolved, not evidence that accepted changes reached the workspace.
func RequiredFinalizationPending(run *Run) bool {
	if run == nil || run.RunMode != RunModeSandboxed || !run.Status.IsTerminal() {
		return false
	}
	cfg := run.SandboxConfig
	if cfg == nil && run.ResolvedConfig != nil {
		cfg = run.ResolvedConfig.SandboxConfig
	}
	if cfg != nil {
		if cfg.ManualReview || !cfg.GetAutoApply() {
			return false
		}
		if (run.Status == RunStatusFailed || run.Status == RunStatusCancelled) && !cfg.GetApplyOnFailure() {
			return false
		}
	}
	return run.FinalizationStatus != RunFinalizationStatusSucceeded
}
