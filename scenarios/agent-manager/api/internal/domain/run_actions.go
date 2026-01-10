package domain

import "strings"

// RunActionContext supplies policy inputs for action evaluation.
type RunActionContext struct {
	InvestigationTagAllowlist []InvestigationTagRule
}

// RunActions captures which actions are allowed for a run.
type RunActions struct {
	CanInvestigate               bool `json:"canInvestigate"`
	CanApplyInvestigation        bool `json:"canApplyInvestigation"`
	CanDelete                    bool `json:"canDelete"`
	CanStop                      bool `json:"canStop"`
	CanRetry                     bool `json:"canRetry"`
	CanContinue                  bool `json:"canContinue"`
	CanApprove                   bool `json:"canApprove"`
	CanReject                    bool `json:"canReject"`
	CanReview                    bool `json:"canReview"`
	CanExtractRecommendations    bool `json:"canExtractRecommendations"`
	CanRegenerateRecommendations bool `json:"canRegenerateRecommendations"`
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
	canContinue, _ := CanContinueRun(run)
	canApprove, _ := CanApproveRun(run)
	canReject, _ := CanRejectRun(run)
	canReview, _ := CanReviewRun(run)
	canExtract, _ := CanExtractRecommendations(run, allowlist)
	canRegenerate, _ := CanRegenerateRecommendations(run, allowlist)

	return RunActions{
		CanInvestigate:               canInvestigate,
		CanApplyInvestigation:        canApplyInvestigation,
		CanDelete:                    canDelete,
		CanStop:                      canStop,
		CanRetry:                     canRetry,
		CanContinue:                  canContinue,
		CanApprove:                   canApprove,
		CanReject:                    canReject,
		CanReview:                    canReview,
		CanExtractRecommendations:    canExtract,
		CanRegenerateRecommendations: canRegenerate,
	}
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
func CanStopRun(run *Run) (bool, string) {
	if run == nil {
		return false, "run not found"
	}
	switch run.Status {
	case RunStatusRunning, RunStatusStarting:
		return true, ""
	default:
		return false, "can only stop running or starting runs"
	}
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
	default:
		return true, ""
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
