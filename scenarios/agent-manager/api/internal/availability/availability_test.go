package availability

import "testing"

func TestClosedVocabularyRetainsEveryEvidenceState(t *testing.T) {
	want := []State{
		Available, Unavailable, Degraded, Unobserved, Unknown, Resolved,
		PolicyAbsent, Oversized, NotCaptured, External, Empty, Complete,
	}
	seen := map[State]bool{}
	for _, state := range want {
		if state == "" {
			t.Fatal("availability state must not be empty")
		}
		if seen[state] {
			t.Fatalf("availability state %q is duplicated", state)
		}
		seen[state] = true
	}
	if len(seen) != 12 {
		t.Fatalf("availability vocabulary size = %d, want 12", len(seen))
	}
}

func TestAvailabilityPreservesStateAndReason(t *testing.T) {
	got := New(PolicyAbsent, "no capture policy applies")
	if got.State != PolicyAbsent || got.Reason != "no capture policy applies" {
		t.Fatalf("availability = %#v", got)
	}
}
