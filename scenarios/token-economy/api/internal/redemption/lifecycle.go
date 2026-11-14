package redemption

import (
	"errors"
	"fmt"
)

var ErrInvalidRedemptionTransition = errors.New("invalid redemption state transition")

var redemptionTransitions = map[State]map[State]struct{}{
	StatePendingApproval: {StateSettled: {}, StateDenied: {}},
}

var reservationTransitions = map[ReservationState]map[ReservationState]struct{}{
	ReservationStateActive: {ReservationStateSettled: {}, ReservationStateReleased: {}},
}

func (state State) Valid() bool {
	switch state {
	case StatePendingApproval, StateSettled, StateDenied:
		return true
	default:
		return false
	}
}

func (state ReservationState) Valid() bool {
	switch state {
	case ReservationStateActive, ReservationStateSettled, ReservationStateReleased:
		return true
	default:
		return false
	}
}

// InitialRedemptionState owns the durable state selected when a redemption is
// first created; all later changes pass through TransitionState.
func InitialRedemptionState(posture ApprovalPosture) (State, error) {
	switch posture {
	case ApprovalPostureRequiresApproval:
		return StatePendingApproval, nil
	case ApprovalPostureImmediate:
		return StateSettled, nil
	default:
		return "", fmt.Errorf("%w: unknown approval posture %q", ErrInvalidRedemptionTransition, posture)
	}
}

// InitialReservationState mirrors the initial redemption decision while
// keeping inventory reservation ownership in the same lifecycle vocabulary.
func InitialReservationState(posture ApprovalPosture) (ReservationState, error) {
	switch posture {
	case ApprovalPostureRequiresApproval:
		return ReservationStateActive, nil
	case ApprovalPostureImmediate:
		return ReservationStateSettled, nil
	default:
		return "", fmt.Errorf("%w: unknown approval posture %q", ErrInvalidRedemptionTransition, posture)
	}
}

func (value *Redemption) TransitionState(next State) error {
	if value == nil || !value.State.Valid() || !next.Valid() {
		return fmt.Errorf("%w: unknown redemption state %q -> %q", ErrInvalidRedemptionTransition, redemptionStateOf(value), next)
	}
	if value.State == next {
		return nil
	}
	if _, ok := redemptionTransitions[value.State][next]; !ok {
		return fmt.Errorf("%w: redemption %q -> %q", ErrInvalidRedemptionTransition, value.State, next)
	}
	value.State = next
	return nil
}

func (value *Reservation) TransitionState(next ReservationState) error {
	if value == nil || !value.State.Valid() || !next.Valid() {
		return fmt.Errorf("%w: unknown reservation state %q -> %q", ErrInvalidRedemptionTransition, reservationStateOf(value), next)
	}
	if value.State == next {
		return nil
	}
	if _, ok := reservationTransitions[value.State][next]; !ok {
		return fmt.Errorf("%w: reservation %q -> %q", ErrInvalidRedemptionTransition, value.State, next)
	}
	value.State = next
	return nil
}

func redemptionStateOf(value *Redemption) State {
	if value == nil {
		return "<nil>"
	}
	return value.State
}

func reservationStateOf(value *Reservation) ReservationState {
	if value == nil {
		return "<nil>"
	}
	return value.State
}
