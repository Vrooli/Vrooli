package artifacts

import (
	"context"

	"github.com/vrooli/api-core/provenance"
)

// ApprovalInput is the already-evaluated state from each owning domain. The
// gate deliberately does not reimplement claim, post-type, or review policy.
type ApprovalInput struct {
	DraftStatus        DraftStatus
	UnverifiedClaimIDs []string
	PostTypeActive     bool
	ReviewPassed       bool
}

type ApprovalVerdict struct {
	Allowed  bool
	Blockers []string
}

// EvaluateApproval applies every mechanical prerequisite and the human-only
// rule. A verified Agent Manager provenance is never eligible; ordinary API
// requests resolve to API Core's operator provenance and can be approved.
func EvaluateApproval(ctx context.Context, input ApprovalInput) ApprovalVerdict {
	blockers := make([]string, 0, len(input.UnverifiedClaimIDs)+4)
	if input.DraftStatus != DraftReviewed {
		blockers = append(blockers, "draft is not ready for approval")
	}
	for _, id := range input.UnverifiedClaimIDs {
		blockers = append(blockers, "claim "+id+" is not verified")
	}
	if !input.PostTypeActive {
		blockers = append(blockers, "post type is inactive")
	}
	if !input.ReviewPassed {
		blockers = append(blockers, "review has blocking verdicts")
	}
	if provenance.FromContext(ctx).IsVerifiedAgent() {
		blockers = append(blockers, "agents cannot approve drafts")
	}
	return ApprovalVerdict{Allowed: len(blockers) == 0, Blockers: blockers}
}
