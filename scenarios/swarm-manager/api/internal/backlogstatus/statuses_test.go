package backlogstatus

import (
	"errors"
	"testing"
)

func TestSuggestedStatusPolicy(t *testing.T) {
	if !IsValid(Suggested) {
		t.Fatal("suggested should be a valid backlog status")
	}
	if IsUserSettable(Suggested) {
		t.Fatal("suggested should not be directly user-settable")
	}
	if IsTerminal(Suggested) {
		t.Fatal("suggested should not be terminal")
	}
}

func TestDroppedStatusPolicy(t *testing.T) {
	if !IsValid(Dropped) {
		t.Fatal("dropped should be a valid backlog status")
	}
	if !IsTerminal(Dropped) {
		t.Fatal("dropped should be terminal — the item will not be worked again")
	}
	// Unlike the other terminals, dropped records an operator decision rather
	// than a run outcome, so it must be reachable without a review round.
	if !IsUserSettable(Dropped) {
		t.Fatal("dropped should be settable directly via PATCH, with no run required")
	}
	if !CanTransition(Backlog, Dropped, TransitionOperator) {
		t.Fatal("an untouched backlog item should be droppable")
	}
}

func TestIsResolved(t *testing.T) {
	cases := []struct {
		status string
		want   bool
		reason string
	}{
		{Completed, true, "work was done — dependents may proceed"},
		{Dropped, true, "work will never be done — dependents must not wait forever"},

		// Failure states are still live work. Their dependents are genuinely
		// blocked, so resolving them here would let downstream items start on
		// top of a prerequisite that never landed.
		{Failed, false, "failed work may still be retried"},
		{NeedsFollowup, false, "delivered but incomplete — dependents still wait"},

		{Backlog, false, "not started"},
		{Ready, false, "not started"},
		{InProgress, false, "in flight"},
		{InReview, false, "in flight"},
		{ReviewPending, false, "awaiting operator verdict"},
		{Suggested, false, "not yet accepted"},
	}
	for _, c := range cases {
		if got := IsResolved(c.status); got != c.want {
			t.Errorf("IsResolved(%s) = %v, want %v (%s)", c.status, got, c.want, c.reason)
		}
	}
}

func TestCanTransitionEnforcesOwnerEdges(t *testing.T) {
	cases := []struct {
		from, to string
		owner    TransitionOwner
		want     bool
	}{
		{Queued, InProgress, TransitionExecution, true},
		{InProgress, InReview, TransitionExecution, true},
		{ReviewPending, Completed, TransitionReviewDecision, true},
		{Failed, InProgress, TransitionRetry, true},
		{NeedsFollowup, Backlog, TransitionFollowUp, true},
		{InReview, Backlog, TransitionRecovery, true},
		{Ready, Backlog, TransitionReset, true},
		{Backlog, Dropped, TransitionArchive, true},
		{InProgress, Dropped, TransitionGoalArchive, true},
		{Completed, Dropped, TransitionGoalArchive, false},
		{ReviewPending, Completed, TransitionOperator, false},
		{Completed, Backlog, TransitionRetry, false},
		{InProgress, Completed, TransitionExecution, false},
		{"banana", Backlog, TransitionOperator, false},
		{Backlog, "banana", TransitionImport, false},
	}
	for _, tc := range cases {
		if got := CanTransition(tc.from, tc.to, tc.owner); got != tc.want {
			t.Errorf("CanTransition(%q, %q, %q) = %v, want %v", tc.from, tc.to, tc.owner, got, tc.want)
		}
	}
}

func TestTransitionReturnsTypedInvalidEdgeError(t *testing.T) {
	if _, err := Transition(Completed, InProgress, TransitionExecution); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("Transition returned %v, want ErrInvalidTransition", err)
	}
}
