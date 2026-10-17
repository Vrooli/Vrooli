package capture

import (
	"testing"

	"performance-health/internal/readiness"
)

// [REQ:PH-CAPTURE-002] reachedTier downgrades a Tier-1-eligible capture to Tier 0
// when no ⚛ marks were present, and preserves the eligible tier otherwise.
func TestReachedTier(t *testing.T) {
	cases := []struct {
		eligible readiness.Tier
		marks    bool
		want     readiness.Tier
	}{
		{readiness.Tier1, true, readiness.Tier1},
		{readiness.Tier1, false, readiness.Tier0},
		{readiness.Tier0, false, readiness.Tier0},
		{readiness.Tier0, true, readiness.Tier0},
	}
	for _, tc := range cases {
		if got := reachedTier(tc.eligible, tc.marks); got != tc.want {
			t.Fatalf("reachedTier(%v, %v) = %v, want %v", tc.eligible, tc.marks, got, tc.want)
		}
	}
}
