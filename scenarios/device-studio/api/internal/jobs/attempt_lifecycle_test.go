package jobs_test

import (
	"errors"
	"testing"

	"device-studio/internal/jobs"
)

func TestAttemptLifecycleTransitionMatrix(t *testing.T) {
	states := []jobs.AttemptState{
		jobs.AttemptQueued, jobs.AttemptRunning, jobs.AttemptCheckpointed,
		jobs.AttemptSucceeded, jobs.AttemptFailed, jobs.AttemptCancelled,
	}
	events := []jobs.AttemptEvent{
		jobs.AttemptStarted, jobs.AttemptCheckpointedEvent,
		jobs.AttemptSucceededEvent, jobs.AttemptFailedEvent,
		jobs.AttemptCancelledEvent,
	}
	want := map[jobs.AttemptState]map[jobs.AttemptEvent]jobs.AttemptState{
		jobs.AttemptQueued: {
			jobs.AttemptStarted:        jobs.AttemptRunning,
			jobs.AttemptFailedEvent:    jobs.AttemptFailed,
			jobs.AttemptCancelledEvent: jobs.AttemptCancelled,
		},
		jobs.AttemptRunning: {
			jobs.AttemptCheckpointedEvent: jobs.AttemptCheckpointed,
			jobs.AttemptSucceededEvent:    jobs.AttemptSucceeded,
			jobs.AttemptFailedEvent:       jobs.AttemptFailed,
			jobs.AttemptCancelledEvent:    jobs.AttemptCancelled,
		},
		jobs.AttemptCheckpointed: {
			jobs.AttemptCancelledEvent: jobs.AttemptCancelled,
		},
	}

	for _, from := range states {
		for _, event := range events {
			t.Run(string(from)+"/"+string(event), func(t *testing.T) {
				attempt := jobs.Attempt{State: from}
				next, legal := want[from][event]
				err := attempt.Transition(event)
				if legal {
					if err != nil || attempt.State != next {
						t.Fatalf("transition state=%q want=%q err=%v", attempt.State, next, err)
					}
					return
				}
				if !errors.Is(err, jobs.ErrInvalidTransition) {
					t.Fatalf("transition error=%v, want invalid transition", err)
				}
				if attempt.State != from {
					t.Fatalf("rejected transition mutated state to %q", attempt.State)
				}
			})
		}
	}
}

func TestAttemptLifecycleRejectsUnknownStateAndEventWithoutMutation(t *testing.T) {
	for _, test := range []struct {
		name  string
		state jobs.AttemptState
		event jobs.AttemptEvent
	}{
		{name: "unknown source", state: jobs.AttemptState("legacy-unknown"), event: jobs.AttemptStarted},
		{name: "unknown event", state: jobs.AttemptQueued, event: jobs.AttemptEvent("force-succeeded")},
	} {
		t.Run(test.name, func(t *testing.T) {
			attempt := jobs.Attempt{State: test.state}
			if err := attempt.Transition(test.event); !errors.Is(err, jobs.ErrInvalidTransition) {
				t.Fatalf("transition error=%v, want invalid transition", err)
			}
			if attempt.State != test.state {
				t.Fatalf("rejected transition mutated state from %q to %q", test.state, attempt.State)
			}
		})
	}
	var attempt *jobs.Attempt
	if err := attempt.Transition(jobs.AttemptStarted); !errors.Is(err, jobs.ErrInvalidTransition) {
		t.Fatalf("nil attempt transition error=%v, want invalid transition", err)
	}
}
