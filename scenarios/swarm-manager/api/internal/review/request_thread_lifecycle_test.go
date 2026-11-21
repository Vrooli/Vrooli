package review

import (
	"errors"
	"testing"
)

func TestRequestThreadTransitionStatus(t *testing.T) {
	for from, edges := range requestThreadTransitions {
		for to := range edges {
			thread := RequestThread{Status: from}
			if err := thread.TransitionStatus(to); err != nil {
				t.Errorf("TransitionStatus(%s -> %s) error = %v", from, to, err)
			}
			if thread.Status != to {
				t.Errorf("TransitionStatus(%s -> %s) left status %s", from, to, thread.Status)
			}
		}
	}
}

func TestRequestThreadTransitionRejectsInvalidAndTerminalReversal(t *testing.T) {
	tests := []struct {
		name string
		from RequestThreadStatus
		to   RequestThreadStatus
	}{
		{name: "unknown stored status", from: "unknown", to: RequestThreadStatusFulfilled},
		{name: "unknown next status", from: RequestThreadStatusPending, to: "running"},
		{name: "dismissed cannot fulfill", from: RequestThreadStatusDismissed, to: RequestThreadStatusFulfilled},
		{name: "fulfilled cannot dismiss", from: RequestThreadStatusFulfilled, to: RequestThreadStatusDismissed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			thread := RequestThread{Status: test.from}
			err := thread.TransitionStatus(test.to)
			var transitionErr *InvalidRequestThreadTransitionError
			if !errors.As(err, &transitionErr) {
				t.Fatalf("TransitionStatus(%s -> %s) error = %v, want InvalidRequestThreadTransitionError", test.from, test.to, err)
			}
			if thread.Status != test.from {
				t.Fatalf("rejected transition mutated status to %s", thread.Status)
			}
		})
	}
}

func TestRequestThreadValidateStatusSupportsRecoveryReplay(t *testing.T) {
	thread := RequestThread{Status: RequestThreadStatusFulfilled}
	if err := thread.ValidateStatus(); err != nil {
		t.Fatalf("validate restored fulfilled thread: %v", err)
	}
	if err := thread.TransitionStatus(RequestThreadStatusFulfilled); err != nil {
		t.Fatalf("idempotent fulfilled replay: %v", err)
	}
	if err := thread.TransitionStatus(RequestThreadStatusDismissed); err == nil {
		t.Fatal("terminal replay changed fulfilled thread to dismissed")
	}
}
