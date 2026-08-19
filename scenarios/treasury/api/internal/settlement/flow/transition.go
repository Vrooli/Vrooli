package flow

import (
	"fmt"

	"treasury/internal/settlement/flow/generated"
)

func TransitionSettlement(state SettlementState, event SettlementEvent) (SettlementState, error) {
	if err := CheckInvariants(state); err != nil {
		return state, err
	}
	next, err := generated.TransitionSettlementStatus(state.Status, event)
	return SettlementState{Status: next}, err
}

func CheckInvariants(state SettlementState) error {
	switch state.Status {
	case SettlementReady, SettlementCalling, SettlementSettled, SettlementFailed, SettlementUnknown:
		return nil
	default:
		return fmt.Errorf("unknown settlement status %q", state.Status)
	}
}
