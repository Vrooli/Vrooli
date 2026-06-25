package flow

import (
	"fmt"

	"plan-manager/internal/authoring/flow/generated"
)

type (
	AuthoringStatus = generated.AuthoringStatus
	AuthoringEvent  = generated.AuthoringEvent
)

const (
	AuthoringDrafting  = generated.AuthoringDrafting
	AuthoringValidated = generated.AuthoringValidated
	AuthoringFinalized = generated.AuthoringFinalized
)

const (
	AuthoringSubmitSection = generated.AuthoringSubmitSection
	AuthoringValidate      = generated.AuthoringValidate
	AuthoringFinalize      = generated.AuthoringFinalize
)

// AuthoringState is the hand-authored projection of the guided-composer workflow
// status. It mirrors the structured authoring.Session lifecycle: a draft session
// accrues section submissions (Drafting), passes the structure gate (Validated),
// and writes its plan through the plans domain (Finalized). Re-editing a section
// after validation returns the session to Drafting.
type AuthoringState struct {
	Status AuthoringStatus
}

// InitialAuthoringState is the entry state — a freshly started session.
func InitialAuthoringState() AuthoringState {
	return AuthoringState{Status: AuthoringDrafting}
}

// TransitionAuthoring applies an event to the workflow state, validating the
// state first and delegating the legal-transition decision to the generated
// formal runtime so the hand-authored code and the verified model never drift.
func TransitionAuthoring(state AuthoringState, event AuthoringEvent) (AuthoringState, error) {
	if err := CheckAuthoringInvariants(state); err != nil {
		return state, err
	}
	next, err := generated.TransitionAuthoringStatus(state.Status, event)
	return AuthoringState{Status: next}, err
}

// CheckAuthoringInvariants guards against an unknown status reaching the
// transition function.
func CheckAuthoringInvariants(state AuthoringState) error {
	switch state.Status {
	case AuthoringDrafting,
		AuthoringValidated,
		AuthoringFinalized:
		return nil
	default:
		return fmt.Errorf("unknown authoring status %q", state.Status)
	}
}
