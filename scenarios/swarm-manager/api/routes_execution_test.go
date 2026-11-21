package main

import (
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
)

// TestEngagementCloseDecisionForStatus pins the mapping from a backlog
// terminal status to the Baseline Modes engagement-close decision that the
// review-decide hook (registerExecutionRoutes) feeds into
// execution.CloseOwnerEngagements. The contract:
//
//   - completed → promote the owner's shadow engagement set (accept).
//   - failed    → abandon it (reject; roll the shadow restore points back).
//   - anything else (needs_followup, in_progress, ready, ...) → leave open
//     so the next run under the same owner can retry.
//
// Getting this wrong silently promotes failed work or abandons in-flight
// work, so every status is exercised, not just the two terminal ones.
func TestEngagementCloseDecisionForStatus(t *testing.T) {
	cases := []struct {
		status backlog.BacklogStatus
		want   execution.EngagementCloseDecision
	}{
		{backlog.StatusCompleted, execution.EngagementPromote},
		{backlog.StatusFailed, execution.EngagementAbandon},
		{backlog.StatusNeedsFollowup, execution.EngagementLeaveOpen},
		{backlog.StatusInProgress, execution.EngagementLeaveOpen},
		{backlog.StatusReady, execution.EngagementLeaveOpen},
		{backlog.StatusReviewPending, execution.EngagementLeaveOpen},
		{backlog.StatusInReview, execution.EngagementLeaveOpen},
		{backlog.StatusBacklog, execution.EngagementLeaveOpen},
		{backlog.BacklogStatus("totally-unknown"), execution.EngagementLeaveOpen},
	}
	for _, c := range cases {
		t.Run(string(c.status), func(t *testing.T) {
			if got := engagementCloseDecisionForStatus(c.status); got != c.want {
				t.Fatalf("engagementCloseDecisionForStatus(%q) = %v, want %v", c.status, got, c.want)
			}
		})
	}
}

// TestBaselineEngagementEnabled pins the env-var parse for the opt-in
// Baseline Modes engagement flag. Default (unset/empty) must be OFF so the
// reflexive kernel runs unperturbed; only the documented truthy spellings
// flip it on, and parsing is case-insensitive with surrounding whitespace
// trimmed. A defaulted-off flag that silently turns on would change live
// execution behavior for every run, so the negative cases matter as much
// as the positive ones.
func TestBaselineEngagementEnabled(t *testing.T) {
	cases := []struct {
		name string
		set  bool
		val  string
		want bool
	}{
		{name: "unset defaults off", set: false, want: false},
		{name: "empty string off", set: true, val: "", want: false},
		{name: "whitespace off", set: true, val: "   ", want: false},
		{name: "1 on", set: true, val: "1", want: true},
		{name: "true on", set: true, val: "true", want: true},
		{name: "TRUE case-insensitive on", set: true, val: "TRUE", want: true},
		{name: "yes on", set: true, val: "yes", want: true},
		{name: "on on", set: true, val: "on", want: true},
		{name: "padded true on", set: true, val: "  true  ", want: true},
		{name: "0 off", set: true, val: "0", want: false},
		{name: "false off", set: true, val: "false", want: false},
		{name: "garbage off", set: true, val: "maybe", want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// t.Setenv handles save/restore (and unset is the zero state
			// since each subtest gets a fresh process env view).
			if c.set {
				t.Setenv("SWARM_MANAGER_BASELINE_ENGAGEMENT", c.val)
			} else {
				// Ensure no inherited value leaks in.
				t.Setenv("SWARM_MANAGER_BASELINE_ENGAGEMENT", "")
			}
			if got := baselineEngagementEnabled(); got != c.want {
				t.Fatalf("baselineEngagementEnabled() with %q = %v, want %v", c.val, got, c.want)
			}
		})
	}
}

// TestRepoRootFromScenarioRoot pins the path derivation that supplies the
// git-control-tower engagement runner's working directory. The scenario
// source path is `<repo>/scenarios/<name>`, so the repo root is two
// directories up — but only when the grandparent is literally `scenarios`.
// Any other shape returns "" (the runner then inherits the process cwd),
// which is the safe degraded mode rather than guessing a wrong root.
func TestRepoRootFromScenarioRoot(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "canonical scenario root yields repo root",
			in:   "/home/me/Vrooli/scenarios/swarm-manager",
			want: "/home/me/Vrooli",
		},
		{
			name: "trailing whitespace trimmed",
			in:   "  /home/me/Vrooli/scenarios/swarm-manager  ",
			want: "/home/me/Vrooli",
		},
		{
			name: "relative scenario root yields relative repo root",
			in:   "Vrooli/scenarios/swarm-manager",
			want: "Vrooli",
		},
		{
			name: "empty input yields empty",
			in:   "",
			want: "",
		},
		{
			name: "whitespace-only input yields empty",
			in:   "   ",
			want: "",
		},
		{
			name: "parent not named scenarios yields empty",
			in:   "/home/me/Vrooli/apps/swarm-manager",
			want: "",
		},
		{
			name: "scenarios-rooted but too shallow yields empty",
			in:   "scenarios/swarm-manager",
			want: ".",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := repoRootFromScenarioRoot(c.in); got != c.want {
				t.Fatalf("repoRootFromScenarioRoot(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
