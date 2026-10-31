package flow

import (
	"fmt"

	"treasury/internal/mandate/flow/generated"
)

func TransitionMandate(state MandateState, event MandateEvent) (MandateState, error) {
	if err := CheckInvariants(state); err != nil {
		return state, err
	}
	next, err := generated.TransitionMandateStatus(state.Status, event)
	return MandateState{Status: next}, err
}

func CheckInvariants(state MandateState) error {
	switch state.Status {
	case MandateDraft, MandateLive, MandateExhausted, MandateExpired, MandateRevoked:
		return nil
	default:
		return fmt.Errorf("unknown mandate status %q", state.Status)
	}
}
