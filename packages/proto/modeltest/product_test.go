package modeltest

import "testing"

func TestValidateProductObservationRequiresCompleteStateAndEffects(t *testing.T) {
	want := ProductObservation{
		State:   map[string]string{"reservation": "finalized", "wallet": "nonnegative"},
		Effects: map[string]int64{"allocations": 1, "usage": 200},
	}
	if errs := ValidateProductObservation(want, want); len(errs) != 0 {
		t.Fatalf("exact observation mismatch: %v", errs)
	}
	got := ProductObservation{
		State:   map[string]string{"reservation": "finalized"},
		Effects: map[string]int64{"allocations": 1},
	}
	if errs := ValidateProductObservation(want, got); len(errs) != 2 {
		t.Fatalf("omitted state/effect keys errors=%v, want both", errs)
	}
}
