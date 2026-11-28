package planworkshop

import (
	"errors"
	"testing"
)

func TestReviewRunTransitionMatrix(t *testing.T) {
	states := []ReviewState{ReviewPending, ReviewRunning, ReviewApplied, ReviewStale, ReviewFailed}
	allowed := map[ReviewState]map[ReviewState]bool{
		"":            {ReviewPending: true},
		ReviewPending: {ReviewPending: true, ReviewRunning: true, ReviewStale: true, ReviewFailed: true},
		ReviewRunning: {ReviewRunning: true, ReviewApplied: true, ReviewStale: true},
		ReviewApplied: {ReviewApplied: true, ReviewPending: true},
		ReviewStale:   {ReviewStale: true, ReviewPending: true},
		ReviewFailed:  {ReviewFailed: true, ReviewPending: true},
	}
	fromStates := append([]ReviewState{""}, states...)
	for _, from := range fromStates {
		for _, to := range states {
			run := ReviewRun{State: from}
			err := run.Transition(to)
			if allowed[from][to] {
				if err != nil || run.State != to {
					t.Errorf("transition %q -> %q produced state %q, err=%v", from, to, run.State, err)
				}
			} else if !errors.Is(err, ErrInvalidReviewRunTransition) || run.State != from {
				t.Errorf("invalid transition %q -> %q mutated to %q, err=%v", from, to, run.State, err)
			}
		}
	}
}

func TestReviewRunTransitionRejectsUnknownStatesAndNilReceiver(t *testing.T) {
	for name, run := range map[string]*ReviewRun{
		"unknown current": {State: "future_state"},
		"nil receiver":    nil,
	} {
		if err := run.Transition(ReviewPending); !errors.Is(err, ErrInvalidReviewRunTransition) {
			t.Errorf("%s error = %v, want ErrInvalidReviewRunTransition", name, err)
		}
	}
	unknownTarget := ReviewRun{State: ReviewPending}
	if err := unknownTarget.Transition("future_state"); !errors.Is(err, ErrInvalidReviewRunTransition) || unknownTarget.State != ReviewPending {
		t.Fatalf("unknown target transition state=%q err=%v", unknownTarget.State, err)
	}
}

func TestSessionValidationRejectsUnknownReviewState(t *testing.T) {
	session := Session{
		ID:             "workshop-1",
		Subject:        Subject{Kind: SubjectBacklog, Ref: "execute/example"},
		SubjectVersion: "version-1",
		Review:         &ReviewRun{State: "future_state"},
	}
	if err := session.Validate(); err == nil {
		t.Fatal("session accepted a persisted review with unknown state")
	}
}
