// Package artifacts owns drafts and their lifecycle transition policy.
package artifacts

import "fmt"

type (
	DraftStatus string
	DraftEvent  string
)

const (
	DraftRequested DraftStatus = "requested"
	DraftDrafting  DraftStatus = "drafting"
	DraftDrafted   DraftStatus = "drafted"
	DraftChecking  DraftStatus = "checking"
	DraftBlocked   DraftStatus = "blocked"
	DraftReviewed  DraftStatus = "reviewed"
	DraftApproved  DraftStatus = "approved"
	DraftPublished DraftStatus = "published"
	DraftAbandoned DraftStatus = "abandoned"
)

const (
	DraftBegin      DraftEvent = "begin"
	DraftComplete   DraftEvent = "complete"
	DraftCheck      DraftEvent = "check"
	DraftBlock      DraftEvent = "block"
	DraftReviewPass DraftEvent = "review_pass"
	DraftApprove    DraftEvent = "approve"
	DraftPublish    DraftEvent = "publish"
	DraftAbandon    DraftEvent = "abandon"
)

type DraftState struct{ Status DraftStatus }

func InitialDraftState() DraftState { return DraftState{Status: DraftRequested} }

// TransitionDraft is pure lifecycle policy. Approval gates (claims, review,
// posttype, actor) supply the Approve event only after their domain checks
// succeed; this function still rejects every skipped lifecycle state.
func TransitionDraft(state DraftState, event DraftEvent) (DraftState, error) {
	if !knownStatus(state.Status) {
		return state, fmt.Errorf("unknown draft status %q", state.Status)
	}
	if event == DraftAbandon && state.Status != DraftApproved && state.Status != DraftPublished && state.Status != DraftAbandoned {
		return DraftState{Status: DraftAbandoned}, nil
	}
	var next DraftStatus
	switch state.Status {
	case DraftRequested:
		if event == DraftBegin {
			next = DraftDrafting
		}
	case DraftDrafting:
		if event == DraftComplete {
			next = DraftDrafted
		}
	case DraftDrafted:
		if event == DraftCheck {
			next = DraftChecking
		}
	case DraftChecking:
		if event == DraftBlock {
			next = DraftBlocked
		}
		if event == DraftReviewPass {
			next = DraftReviewed
		}
	case DraftBlocked:
		if event == DraftCheck {
			next = DraftChecking
		}
	case DraftReviewed:
		if event == DraftBlock {
			next = DraftBlocked
		}
		if event == DraftApprove {
			next = DraftApproved
		}
	case DraftApproved:
		if event == DraftPublish {
			next = DraftPublished
		}
	}
	if next == "" {
		return state, fmt.Errorf("cannot apply %s from %s", event, state.Status)
	}
	return DraftState{Status: next}, nil
}

func knownStatus(status DraftStatus) bool {
	switch status {
	case DraftRequested, DraftDrafting, DraftDrafted, DraftChecking, DraftBlocked, DraftReviewed, DraftApproved, DraftPublished, DraftAbandoned:
		return true
	}
	return false
}
