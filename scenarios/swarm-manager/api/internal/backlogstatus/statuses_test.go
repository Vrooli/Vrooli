package backlogstatus

import "testing"

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
