package jobs

import (
	"errors"
	"testing"
)

func TestJobStateTransitionMatrix(t *testing.T) {
	allowed := map[State]map[State]bool{
		StateQueued:  {StateRunning: true, StateFailed: true, StateCanceled: true},
		StateRunning: {StateSucceeded: true, StateFailed: true, StateCanceled: true},
	}
	states := []State{StateQueued, StateRunning, StateSucceeded, StateFailed, StateCanceled}
	for _, from := range states {
		for _, to := range states {
			job := Job{State: from}
			err := job.Transition(to)
			if allowed[from][to] || from == to {
				if err != nil || job.State != to {
					t.Errorf("%s -> %s: state=%s err=%v", from, to, job.State, err)
				}
				continue
			}
			if !errors.Is(err, ErrInvalidStateTransition) {
				t.Errorf("%s -> %s: error=%v, want ErrInvalidStateTransition", from, to, err)
			}
			if job.State != from {
				t.Errorf("rejected %s -> %s mutated state to %s", from, to, job.State)
			}
		}
	}
}

func TestJobStateRejectsUnknownAndNil(t *testing.T) {
	if (State("corrupt")).Valid() {
		t.Fatal("unknown state reported valid")
	}
	job := Job{State: StateQueued}
	if err := job.Transition(State("corrupt")); !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("unknown target error=%v", err)
	}
	if job.State != StateQueued {
		t.Fatalf("unknown transition mutated state: %s", job.State)
	}
	job.State = State("corrupt")
	if err := job.Transition(StateRunning); !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("unknown source error=%v", err)
	}
	if err := (*Job)(nil).Transition(StateQueued); !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("nil job error=%v", err)
	}
}
