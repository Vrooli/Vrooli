package drills

import "testing"

type drillTransitionPair struct {
	state Status
	event Event
}

func TestDrillTransitionMatrixAndInvalidStateImmutability(t *testing.T) {
	states := []Status{StatusRequested, StatusRunning, StatusVerified, StatusFailed, "future"}
	events := []Event{EventStart, EventVerify, EventFail, "future"}
	want := map[drillTransitionPair]Status{
		{StatusRequested, EventStart}: StatusRunning,
		{StatusRequested, EventFail}:  StatusFailed,
		{StatusRunning, EventVerify}:  StatusVerified,
		{StatusRunning, EventFail}:    StatusFailed,
	}
	for _, state := range states {
		for _, event := range events {
			t.Run(string(state)+"/"+string(event), func(t *testing.T) {
				got, ok := Transition(state, event)
				expected, valid := want[drillTransitionPair{state, event}]
				if ok != valid || (valid && got != expected) {
					t.Fatalf("Transition(%q,%q) = %q,%v; want %q,%v", state, event, got, ok, expected, valid)
				}
				drill := Drill{Status: state}
				err := drill.TransitionStatus(event)
				if valid && err != nil {
					t.Fatalf("valid transition: %v", err)
				}
				if !valid && (err == nil || drill.Status != state) {
					t.Fatalf("invalid transition err=%v status=%q; want error and unchanged state", err, drill.Status)
				}
			})
		}
	}
}
