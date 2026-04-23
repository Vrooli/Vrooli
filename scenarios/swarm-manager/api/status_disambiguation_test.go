package main

import (
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/backlogstatus"
	"swarm-manager/internal/execution"
)

// TestBacklogStatus_NotConflatedWithExecutionStatus pins the semantic
// distinction between execution.StatusNeedsFixup (run-level, set by the
// finalization validator) and backlog.StatusNeedsFollowup (item-level
// terminal, set only by the user through review-decide).
//
// The two have similar-sounding names and are easy to conflate, but they
// live on different enums and mean different things. If a future refactor
// aliases the Go types or lets one flow into the other without going
// through a dedicated handler, this test will fail.
func TestBacklogStatus_NotConflatedWithExecutionStatus(t *testing.T) {
	// String values intentionally differ — matching strings would be the
	// easiest way to "helpfully" translate one into the other.
	if string(execution.StatusNeedsFixup) == string(backlog.StatusNeedsFollowup) {
		t.Fatalf("execution.StatusNeedsFixup (%q) must not share a string value with backlog.StatusNeedsFollowup (%q); they are different concepts",
			execution.StatusNeedsFixup, backlog.StatusNeedsFollowup)
	}

	// execution.Status.needs_fixup must NOT round-trip to a valid backlog
	// status. If someone adds "needs_fixup" to the backlog enum, the
	// validator on update_patch.go would start accepting a value that the
	// review-decide audit trail never produced — breaking the "terminal
	// transitions are user-only via review-decide" invariant.
	if backlogstatus.IsValid(string(execution.StatusNeedsFixup)) {
		t.Fatalf("execution.StatusNeedsFixup (%q) must not be a valid backlog status value",
			execution.StatusNeedsFixup)
	}

	// Similarly, backlog.StatusNeedsFollowup ("needs_followup") must not be
	// a valid execution run status. Execution runs finish in completed /
	// failed / canceled / needs_fixup / needs_review — not needs_followup.
	if isValidExecutionStatus(string(backlog.StatusNeedsFollowup)) {
		t.Fatalf("backlog.StatusNeedsFollowup (%q) must not be a valid execution run status",
			backlog.StatusNeedsFollowup)
	}
}

// isValidExecutionStatus is a local helper that enumerates the valid
// execution.Status values. Kept here (not in execution) because it only
// exists to pin the no-conflation invariant above.
func isValidExecutionStatus(s string) bool {
	switch execution.Status(s) {
	case execution.StatusPending,
		execution.StatusStarting,
		execution.StatusRunning,
		execution.StatusNeedsReview,
		execution.StatusValidating,
		execution.StatusNeedsFixup,
		execution.StatusCompleted,
		execution.StatusFailed,
		execution.StatusCanceled:
		return true
	}
	return false
}
