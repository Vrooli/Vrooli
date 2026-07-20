package baseline

import (
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	"testing"
)

func TestGateDecisionForComparisonKeepsAxesIndependent(t *testing.T) {
	for _, tc := range []struct {
		name, behavior, coverage, compatibility, provenance string
		want                                                Verdict
	}{
		{"regression", "regression", "measured", "compatible", "verified", VerdictRegression},
		{"outage", "clean", "unmeasured", "compatible", "verified", VerdictNotComparable},
		{"oracle", "unknown", "measured", "changed-unreviewed", "verified", VerdictNotComparable},
		{"volatile", "clean", "measured", "compatible", "volatile", VerdictNotComparable},
		{"shared-scoped", "clean", "measured", "compatible", "shared-scoped", VerdictClean},
		{"preexisting", "preexisting", "measured", "compatible", "verified", VerdictPreexisting},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := GateDecisionForComparison(&runspb.CompareRunsResponse{SchemaVersion: 2, Behavior: tc.behavior, Coverage: tc.coverage, Compatibility: tc.compatibility, Provenance: tc.provenance})
			if got.LegacyVerdict != tc.want {
				t.Fatalf("decision = %+v, want %s", got, tc.want)
			}
		})
	}
}
