package flow

import (
	"content-desk/internal/artifacts"
)

// TransitionDraftLifecycle is the hand-authored wrapper around the
// generated state machine for the artifacts.draft-lifecycle.api flow.
func TransitionDraftLifecycle(status artifacts.DraftStatus, event artifacts.DraftEvent) (artifacts.DraftStatus, error) {
	next, err := artifacts.TransitionDraft(artifacts.DraftState{Status: status}, event)
	return next.Status, err
}
