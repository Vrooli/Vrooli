package redemption

import (
	"errors"
	"testing"
)

func TestRedemptionTransitionsRejectIllegalEdgesWithoutMutation(t *testing.T) {
	states := []State{StatePendingApproval, StateSettled, StateDenied}
	allowed := map[State][]State{StatePendingApproval: {StateSettled, StateDenied}}
	for _, from := range states {
		for _, to := range states {
			t.Run(string(from)+"_to_"+string(to), func(t *testing.T) {
				value := &Redemption{State: from}
				err := value.TransitionState(to)
				valid := from == to
				for _, candidate := range allowed[from] {
					valid = valid || candidate == to
				}
				if valid {
					if err != nil || value.State != to {
						t.Fatalf("transition %s -> %s: state=%s err=%v", from, to, value.State, err)
					}
					return
				}
				if !errors.Is(err, ErrInvalidRedemptionTransition) || value.State != from {
					t.Fatalf("illegal transition mutated state or returned wrong error: state=%s err=%v", value.State, err)
				}
			})
		}
	}
}

func TestReservationTransitionsRejectIllegalEdges(t *testing.T) {
	for _, test := range []struct {
		from ReservationState
		to   ReservationState
		want bool
	}{
		{ReservationStateActive, ReservationStateActive, true},
		{ReservationStateActive, ReservationStateSettled, true},
		{ReservationStateActive, ReservationStateReleased, true},
		{ReservationStateSettled, ReservationStateReleased, false},
		{ReservationStateReleased, ReservationStateSettled, false},
		{"future", ReservationStateActive, false},
	} {
		value := &Reservation{State: test.from}
		err := value.TransitionState(test.to)
		if test.want && (err != nil || value.State != test.to) {
			t.Errorf("transition %s -> %s failed: state=%s err=%v", test.from, test.to, value.State, err)
		}
		if !test.want && (!errors.Is(err, ErrInvalidRedemptionTransition) || value.State != test.from) {
			t.Errorf("illegal transition %s -> %s changed state: state=%s err=%v", test.from, test.to, value.State, err)
		}
	}
}

func TestInitialStatesFollowApprovalPosture(t *testing.T) {
	for _, test := range []struct {
		posture ApprovalPosture
		state   State
		reserve ReservationState
	}{
		{ApprovalPostureRequiresApproval, StatePendingApproval, ReservationStateActive},
		{ApprovalPostureImmediate, StateSettled, ReservationStateSettled},
	} {
		state, err := InitialRedemptionState(test.posture)
		if err != nil || state != test.state {
			t.Errorf("initial redemption state for %s = %s, %v", test.posture, state, err)
		}
		reservation, err := InitialReservationState(test.posture)
		if err != nil || reservation != test.reserve {
			t.Errorf("initial reservation state for %s = %s, %v", test.posture, reservation, err)
		}
	}
	if _, err := InitialRedemptionState("future"); !errors.Is(err, ErrInvalidRedemptionTransition) {
		t.Errorf("unknown approval posture accepted: %v", err)
	}
}
