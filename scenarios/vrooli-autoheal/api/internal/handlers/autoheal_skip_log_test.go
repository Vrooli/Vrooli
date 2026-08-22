package handlers

import "testing"

// Feature: the action history records state changes, not ticks
//
//	As an operator reading the action history
//	I want one row when a check starts skipping and one when the reason changes
//	So that the first failure is findable, instead of buried under thousands of
//	copies of itself.

// Scenario: a repeated skip reason is recorded once.
func TestSkipLogGateRecordsTheFirstSkipAndDropsRepeats(t *testing.T) {
	// Given a gate and a check that skips for the same reason every tick
	gate := newSkipLogGate()
	const check = "resource-reranker"
	const reason = "auto-heal not enabled for this check"

	// When the first skip is offered
	if !gate.ShouldLog(check, reason) {
		t.Fatal("the first skip was not recorded; the first failure must always be findable")
	}

	// Then the next thousand identical skips are not recorded
	for i := range 1000 {
		if gate.ShouldLog(check, reason) {
			t.Fatalf("repeat skip %d was recorded; a steady state is not an event", i)
		}
	}
}

// Scenario: a changed reason is a new state and is recorded.
func TestSkipLogGateRecordsAChangedReason(t *testing.T) {
	// Given a check already skipping for one reason
	gate := newSkipLogGate()
	const check = "resource-searxng"
	gate.ShouldLog(check, "in cooldown (240s remaining)")

	// When the reason changes
	// Then it is recorded, because the operator's picture just changed
	if !gate.ShouldLog(check, "in cooldown (120s remaining)") {
		t.Fatal("a changed skip reason was not recorded")
	}
	if !gate.ShouldLog(check, "no auto-heal recovery action available") {
		t.Fatal("a different skip class was not recorded")
	}
}

// Scenario: each check is tracked separately.
func TestSkipLogGateTracksChecksIndependently(t *testing.T) {
	// Given two checks skipping for the same reason
	gate := newSkipLogGate()
	const reason = "in cooldown (60s remaining)"

	// When each offers its first skip
	// Then both are recorded: one check's noise must not silence another's
	if !gate.ShouldLog("resource-whisper", reason) {
		t.Fatal("whisper's first skip was not recorded")
	}
	if !gate.ShouldLog("resource-reranker", reason) {
		t.Fatal("reranker's first skip was suppressed by whisper's identical reason")
	}
}

// Scenario: an attempted heal resets the state, so the next skip is news again.
func TestSkipLogGateClearsAfterAnAttempt(t *testing.T) {
	// Given a check that has been skipping
	gate := newSkipLogGate()
	const check = "resource-ollama"
	const reason = "in cooldown (300s remaining)"
	gate.ShouldLog(check, reason)

	// When a heal is attempted
	gate.Clear(check)

	// Then the next skip for the same reason is recorded again, because it
	// follows a different event than the last one did
	if !gate.ShouldLog(check, reason) {
		t.Fatal("the skip after an attempted heal was suppressed; it is a new state change")
	}
}

// Scenario: an empty reason is never recorded.
func TestSkipLogGateIgnoresAnEmptyReason(t *testing.T) {
	// Given a gate
	gate := newSkipLogGate()

	// When a skip with no reason is offered
	// Then nothing is recorded, because a row with no reason helps nobody
	for _, reason := range []string{"", "   ", "\t"} {
		if gate.ShouldLog("resource-qdrant", reason) {
			t.Fatalf("a skip with reason %q was recorded", reason)
		}
	}
}
