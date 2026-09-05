package flow

import (
	"fmt"

	"treasury/internal/authorization/flow/generated"
)

func TransitionAuthorization(state AuthorizationState, event AuthorizationEvent) (AuthorizationState, error) {
	if err := CheckInvariants(state); err != nil {
		return state, err
	}
	next, err := generated.TransitionAuthorizationStatus(state.Status, event)
	return AuthorizationState{Status: next}, err
}

func CheckInvariants(state AuthorizationState) error {
	switch state.Status {
	case AuthorizationEvaluating, AuthorizationRefused, AuthorizationPending, AuthorizationApproved, AuthorizationReleased, AuthorizationSettled:
		return nil
	default:
		return fmt.Errorf("unknown authorization status %q", state.Status)
	}
}
