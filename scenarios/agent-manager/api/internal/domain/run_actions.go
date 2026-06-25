package domain

import "strings"

// RunActionContext supplies policy inputs for action evaluation.
type RunActionContext struct {
	InvestigationTagAllowlist []InvestigationTagRule
}

// RunActions captures which actions are allowed for a run.
type RunActions struct {
	CanInvestigate               bool   `json:"canInvestigate"`
	CanApplyInvestigation        bool   `json:"canApplyInvestigation"`
	CanDelete                    bool   `json:"canDelete"`
	CanStop                      bool   `json:"canStop"`
	CanRetry                     bool   `json:"canRetry"`
	CanContinue                  bool   `json:"canContinue"`
	CanContinueReason            string `json:"canContinueReason,omitempty"`
	CanApprove                   bool   `json:"canApprove"`
	CanReject                    bool   `json:"canReject"`
	CanReview                    bool   `json:"canReview"`
	CanExtractRecommendations    bool   `json:"canExtractRecommendations"`
	CanRegenerateRecommendations bool   `json:"canRegenerateRecommendations"`
	CanResumeFromFailure         bool   `json:"canResumeFromFailure"`
	CanResumeFromFailureReason   string `json:"canResumeFromFailureReason,omitempty"`
	FinalizationWarning          string `json:"finalizationWarning,omitempty"`
	CanRetryFinalization         bool   `json:"canRetryFinalization"`
}

// RunActionsFor computes the action flags for a run using the provided context.
func RunActionsFor(run *Run, ctx RunActionContext) RunActions {
	if run == nil {
		return RunActions{}
	}

	allowlist := NormalizeInvestigationTagAllowlist(ctx.InvestigationTagAllowlist)

	canInvestigate, _ := CanInvestigateRun(run)
	canApplyInvestigation, _ := CanApplyInvestigationRun(run, allowlist)
	canDelete, _ := CanDeleteRun(run)
	canStop, _ := CanStopRun(run)
	canRetry, _ := CanRetryRun(run)
	canContinue, canContinueReason := CanContinueRun(run)
	canApprove, _ := CanApproveRun(run)
	canReject, _ := CanRejectRun(run)
	canReview, _ := CanReviewRun(run)
	canExtract, _ := CanExtractRecommendations(run, allowlist)
	canRegenerate, _ := CanRegenerateRecommendations(run, allowlist)
	canResume, canResumeReason := CanResumeFromFailureRun(run)
	finalizationWarning := FinalizationWarning(run)

	return RunActions{
		CanInvestigate:               canInvestigate,
		CanApplyInvestigation:        canApplyInvestigation,
		CanDelete:                    canDelete,
		CanStop:                      canStop,
		CanRetry:                     canRetry,
		CanContinue:                  canContinue,
		CanContinueReason:            canContinueReason,
		CanApprove:                   canApprove,
		CanReject:                    canReject,
		CanReview:                    canReview,
		CanExtractRecommendations:    canExtract,
		CanRegenerateRecommendations: canRegenerate,
		CanResumeFromFailure:         canResume,
		CanResumeFromFailureReason:   canResumeReason,
		FinalizationWarning:          finalizationWarning,
		CanRetryFinalization:         finalizationWarning != "",
	}
}

// FinalizationWarning returns user-facing warning copy for post-run sandbox
// finalization failures. Runner turn activity is modeled by Run.Status; this
// warning is intentionally separate so follow-up eligibility can remain tied to
// the runner turn instead of checkpoint infrastructure.
func FinalizationWarning(run *Run) string {
	if run == nil || run.FinalizationStatus != RunFinalizationStatusFailed {
		return ""
	}
	if strings.TrimSpace(run.FinalizationError) != "" {
		return "Sandbox finalization failed: " + strings.TrimSpace(run.FinalizationError)
	}
	return "Sandbox finalization failed. Changes may require repair before provenance is complete."
}

// CanInvestigateRun returns whether a run can be investigated.
func CanInvestigateRun(_ *Run) (bool, string) {
	return true, ""
}

// CanApplyInvestigationRun returns whether an apply investigation run can be created.
func CanApplyInvestigationRun(run *Run, allowlist []InvestigationTagRule) (bool, string) {
	if run == nil {
		return false, "run not found"
	}
	if run.Status != RunStatusComplete {
		return false, "investigation must be complete before applying fixes"
	}
	if !MatchesInvestigationTag(run.Tag, allowlist) {
		return false, "run tag is not eligible for apply investigation"
	}
	return true, ""
}

// CanExtractRecommendations returns whether recommendation extraction can run.
func CanExtractRecommendations(run *Run, allowlist []InvestigationTagRule) (bool, string) {
	if run == nil {
		return false, "run not found"
	}
	if run.Status != RunStatusComplete {
		return false, "investigation must be complete before extracting recommendations"
	}
	if !MatchesInvestigationTag(run.Tag, allowlist) {
		return false, "run tag is not eligible for recommendation extraction"
	}
	return true, ""
}

// CanRegenerateRecommendations returns whether recommendation extraction can be re-triggered.
func CanRegenerateRecommendations(run *Run, allowlist []InvestigationTagRule) (bool, string) {
	if run == nil {
		return false, "run not found"
	}
	if run.Status != RunStatusComplete {
		return false, "investigation must be complete before regenerating recommendations"
	}
	if !MatchesInvestigationTag(run.Tag, allowlist) {
		return false, "run tag is not eligible for recommendation extraction"
	}
	return true, ""
}

// CanDeleteRun returns whether a run can be deleted.
func CanDeleteRun(run *Run) (bool, string) {
	if run == nil {
		return false, "run not found"
	}
	switch run.Status {
	case RunStatusPending, RunStatusStarting, RunStatusRunning:
		return false, "stop the run before deleting it"
	default:
		return true, ""
	}
}

// CanStopRun returns whether a run can be stopped.
//
// parked is stoppable: a parked run has no live process, but stopping it must
// still cancel the await-handle's waiter and move the run to a terminal state
// (cancelled) rather than leaving it suspended forever. The orchestrator's
// StopRun special-cases parked (no process to terminate).
func CanStopRun(run *Run) (bool, string) {
	if run == nil {
		return false, "run not found"
	}
	switch run.Status {
	case RunStatusRunning, RunStatusStarting, RunStatusParked:
		return true, ""
	default:
		return false, "can only stop running, starting, or parked runs"
	}
}

// CanParkRun returns whether a run can be parked (suspended waiting on
// externally-owned async work). Only a live, running run can be parked, and
// only one open await-handle is permitted per run — a second park while already
// parked is rejected. Park requires a SessionID so the run can later be woken
// via session resume.
func CanParkRun(run *Run) (bool, string) {
	if run == nil {
		return false, "run not found"
	}
	if run.Status == RunStatusParked {
		return false, "run is already parked"
	}
	if run.Status != RunStatusRunning {
		return false, "only a running run can be parked"
	}
	if strings.TrimSpace(run.SessionID) == "" {
		return false, "run has no session ID - cannot be woken after parking"
	}
	return true, ""
}

// CanWakeRun returns whether a parked run can be woken (resumed with the awaited
// result injected). Wake is idempotent at the orchestrator layer: a run that is
// not parked is treated as already-woken rather than an error.
func CanWakeRun(run *Run) (bool, string) {
	if run == nil {
		return false, "run not found"
	}
	if run.Status != RunStatusParked {
		return false, "run is not parked"
	}
	return true, ""
}

// CanRetryRun returns whether a run can be retried.
func CanRetryRun(run *Run) (bool, string) {
	if run == nil {
		return false, "run not found"
	}
	if run.Status == RunStatusFailed || run.Status == RunStatusCancelled || run.Status == RunStatusComplete {
		return true, ""
	}
	if run.ApprovalState == ApprovalStateApproved || run.ApprovalState == ApprovalStateRejected {
		return true, ""
	}
	return false, "run cannot be retried in its current state"
}

// CanContinueRun returns whether a run can be continued.
func CanContinueRun(run *Run) (bool, string) {
	if run == nil {
		return false, "run not found"
	}
	if strings.TrimSpace(run.SessionID) == "" {
		return false, "run has no session ID - continuation not available for this run"
	}
	switch run.Status {
	case RunStatusRunning, RunStatusStarting, RunStatusPending:
		return false, "cannot continue a run that is still in progress"
	case RunStatusParked:
		// A parked run is owned by its waiter and resumes via wake (with the
		// awaited result injected). An operator-driven continue would race the
		// waiter, so it is disallowed — stop the run first to take it over.
		return false, "run is parked waiting on async work - it will resume automatically (or stop it first)"
	default:
		return true, ""
	}
}

// CanResumeFromFailureRun returns whether a run can be resumed-from-failure:
// a brand-new run that inherits the original task + profile and is seeded with
// the failed attempt's transcript and diff so the agent can complete the
// remaining work instead of starting over (Retry) or replaying a Codex
// session (Continue). Allowed for terminal-but-incomplete states only.
func CanResumeFromFailureRun(run *Run) (bool, string) {
	if run == nil {
		return false, "run not found"
	}
	switch run.Status {
	case RunStatusFailed, RunStatusCancelled:
		return true, ""
	default:
		return false, "resume is only available for failed or cancelled runs"
	}
}

// CanApproveRun returns whether a run can be approved.
func CanApproveRun(run *Run) (bool, string) {
	if run == nil {
		return false, "run not found"
	}
	return run.IsApprovable()
}

// CanRejectRun returns whether a run can be rejected.
func CanRejectRun(run *Run) (bool, string) {
	if run == nil {
		return false, "run not found"
	}
	return run.IsRejectable()
}

// CanReviewRun returns whether a run can be reviewed.
func CanReviewRun(run *Run) (bool, string) {
	if run == nil {
		return false, "run not found"
	}
	if run.Status != RunStatusNeedsReview {
		return false, "run is not awaiting review"
	}
	if run.SandboxID == nil {
		return false, "run has no sandbox to review"
	}
	return true, ""
}
