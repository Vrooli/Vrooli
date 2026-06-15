package baseline

import (
	"strings"
	"testing"

	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
)

// The snapshot banner must surface the run id + ETA + a streaming `runs follow`
// command up front — the anti-polling contract. It must never imply a silent
// block.
func TestSnapshotBannerSurfacesHandleAndFollow(t *testing.T) {
	out := snapshotBanner(&baselinesv1.SnapshotForBaselineResponse{
		RunId: "20260615-000000-abcd", Scenario: "web", Name: "pre-launch",
		EstimatedTotalSeconds: 600, EtaKnown: true,
	})
	for _, want := range []string{
		"20260615-000000-abcd", // run id
		"10m0s",                // ETA
		"test-genie runs follow web 20260615-000000-abcd", // streaming re-attach
		"pins automatically when it completes",            // durable, no silent block
	} {
		if !strings.Contains(out, want) {
			t.Errorf("banner missing %q:\n%s", want, out)
		}
	}
}

func TestSnapshotBannerUnknownETA(t *testing.T) {
	out := snapshotBanner(&baselinesv1.SnapshotForBaselineResponse{
		RunId: "r", Scenario: "s", Name: "n", EtaKnown: false,
	})
	if !strings.Contains(out, "estimated unknown") {
		t.Errorf("unknown ETA must render as 'unknown':\n%s", out)
	}
}
