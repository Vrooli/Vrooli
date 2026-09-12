package flow

import (
	"fmt"

	"treasury/internal/approval/flow/generated"
)

func TransitionApproval(state ApprovalState, event ApprovalEvent) (ApprovalState, error) {
	if err := CheckInvariants(state); err != nil {
		return state, err
	}
	next, err := generated.TransitionApprovalStatus(state.Status, event)
	return ApprovalState{Status: next}, err
}

func CheckInvariants(state ApprovalState) error {
	switch state.Status {
	case ApprovalQueued, ApprovalApproved, ApprovalDeclined, ApprovalExpired:
		return nil
	default:
		return fmt.Errorf("unknown approval status %q", state.Status)
	}
}
