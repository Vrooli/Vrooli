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

// A coalesced snapshot must say it rode an in-flight run (no new suite).
func TestSnapshotBannerCoalesced(t *testing.T) {
	out := snapshotBanner(&baselinesv1.SnapshotForBaselineResponse{
		RunId: "run-1", Scenario: "web", Name: "n", Coalesced: true,
	})
	if !strings.Contains(out, "re-using in-flight comprehensive run run-1") || !strings.Contains(out, "no new suite") {
		t.Errorf("coalesced snapshot banner must say it rode the in-flight run:\n%s", out)
	}
}

// The diff START banner must surface a run id + a streaming follow command + the
// resolve command for every admission outcome — the anti-polling contract.
func TestDiffStartBannerFresh(t *testing.T) {
	out := diffStartBanner(&baselinesv1.StartDiffResponse{
		RunId: "20260616-000000-abcd", Scenario: "web", Name: "pre-launch",
		EstimatedTotalSeconds: 252, EtaKnown: true,
	})
	for _, want := range []string{
		"20260616-000000-abcd",
		"comprehensive run 20260616-000000-abcd started",
		"4m12s",
		"test-genie runs follow web 20260616-000000-abcd",
		"baseline diff status --scenario web --name pre-launch --run 20260616-000000-abcd",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("fresh diff banner missing %q:\n%s", want, out)
		}
	}
}

func TestDiffStartBannerCoalesced(t *testing.T) {
	out := diffStartBanner(&baselinesv1.StartDiffResponse{
		RunId: "run-1", Scenario: "web", Name: "n", Coalesced: true,
	})
	if !strings.Contains(out, "re-using in-flight comprehensive run run-1") || !strings.Contains(out, "no new suite") {
		t.Errorf("coalesced diff banner must say it rode the in-flight run:\n%s", out)
	}
}

func TestDiffStartBannerReused(t *testing.T) {
	out := diffStartBanner(&baselinesv1.StartDiffResponse{
		RunId: "run-9", Scenario: "web", Name: "n", ReusedRun: true, ReusedSha: "abc12345",
	})
	if !strings.Contains(out, "re-using completed run run-9 at abc12345") || !strings.Contains(out, "no suite re-run") {
		t.Errorf("reused diff banner must say it reused the completed run:\n%s", out)
	}
}
