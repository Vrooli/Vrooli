package readiness

import (
	"testing"

	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/readiness"
)

// [REQ:PH-TIER-001] The tier matrix: no-UI / non-React / React drive the
// reachable capture tier; React is never assumed from the filesystem.
func TestDecideTierMatrix(t *testing.T) {
	cases := []struct {
		name  string
		facts Facts
		want  Tier
	}{
		{"no ui", Facts{Surfaces: []string{"api", "cli"}}, TierNone},
		{"non-react ui", Facts{Surfaces: []string{"ui"}, UIFramework: "vue"}, Tier0},
		{"react ui", Facts{Surfaces: []string{"ui"}, UIFramework: "react"}, Tier1},
		{"unknown framework ui", Facts{Surfaces: []string{"ui"}}, Tier0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := decideTier(tc.facts); got != tc.want {
				t.Fatalf("decideTier = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTierToProto(t *testing.T) {
	if TierToProto(Tier1) != readinessv1.CaptureTier_CAPTURE_TIER_1 {
		t.Fatal("Tier1 should map to CAPTURE_TIER_1")
	}
	if TierToProto(TierNone) != readinessv1.CaptureTier_CAPTURE_TIER_NONE {
		t.Fatal("TierNone should map to CAPTURE_TIER_NONE")
	}
}
