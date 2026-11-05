package backlogstatus

import "testing"

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
	if !IsValidTransition(Backlog, Dropped) {
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

func TestIsValidTransition(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
		reason   string
	}{
		// Canonical path through the review flow.
		{InProgress, InReview, true, "run completed, review agent gathers evidence"},
		{InProgress, ReviewPending, true, "non-finalization items skip in_review"},
		{InReview, ReviewPending, true, "review agent finished"},
		{ReviewPending, Completed, true, "user accepted via review-decide"},
		{ReviewPending, Failed, true, "user rejected via review-decide"},
		{ReviewPending, NeedsFollowup, true, "user marked delivered-but-needs-more"},

		// Newly created items.
		{"", Backlog, true, "new item default"},
		{"", Ready, true, "new item pre-prioritized"},
		{"", Suggested, true, "auto-filer can create suggested items internally"},
		{"", "whatever", false, "unknown target rejected even for new items"},

		// Self-transitions: allowed (handlers may carry other field changes).
		{Backlog, Backlog, true, "no-op self"},
		{Completed, Completed, true, "no-op on terminal"},

		// Terminal → anything (including other terminals) is allowed by
		// this predicate. The manual-accept escape hatch (failed →
		// completed via PATCH) is a legitimate user override. Guardrails
		// on *who* may perform these transitions live in update_patch.go
		// (IsUserSettable + review-gate) and review_decide.go.
		{Completed, Backlog, true, "revive path — caller authorizes"},
		{Failed, Completed, true, "manual-accept escape hatch"},
		{NeedsFollowup, Queued, true, "scheduled follow-up run"},

		// Unknown statuses rejected on either end.
		{"banana", Backlog, false, "unknown source"},
		{Backlog, "banana", false, "unknown target"},

		// Within non-terminal states, transitions are permissive — finer
		// rules live in update_patch.go.
		{Suggested, Backlog, true, "operator accepts suggestion into backlog"},
		{Suggested, Ready, true, "operator accepts suggestion as ready"},
		{Backlog, Ready, true, ""},
		{Ready, Backlog, true, "user can deprioritize"},
		{Researching, Backlog, true, ""},
		{Queued, InProgress, true, "execution drain"},
	}
	for _, c := range cases {
		if got := IsValidTransition(c.from, c.to); got != c.want {
			label := c.from + "→" + c.to
			if c.reason != "" {
				label += " (" + c.reason + ")"
			}
			t.Errorf("IsValidTransition(%s) = %v, want %v", label, got, c.want)
		}
	}
}
