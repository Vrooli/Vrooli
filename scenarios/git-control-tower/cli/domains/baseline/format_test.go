package baseline

import (
	"strings"
	"testing"

	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
)

func TestAgentWaitBlockTimeouts(t *testing.T) {
	if got := recommendedWaitSeconds(600, true); got != 1050 {
		t.Fatalf("known ETA timeout = %d, want 1050", got)
	}
	if got := recommendedWaitSeconds(10, true); got != minRecommendedWaitSeconds {
		t.Fatalf("small ETA timeout = %d, want floor %d", got, minRecommendedWaitSeconds)
	}
	if got := recommendedWaitSeconds(0, false); got != unknownETAWaitSeconds {
		t.Fatalf("unknown ETA timeout = %d, want %d", got, unknownETAWaitSeconds)
	}
	out := agentWaitBlock("web", "run-1", 0, false)
	if !strings.Contains(out, "test-genie runs wait --json --timeout=900 web run-1") {
		t.Fatalf("unknown ETA block missing bounded wait command:\n%s", out)
	}
}

// The snapshot banner must surface the run id + ETA + the quiet `runs wait
// --json` block command up front (the agent re-attach verb), with `runs follow`
// only as the human watch-live alternative — the anti-polling contract. It must
// never imply a silent block.
func TestSnapshotBannerSurfacesHandleAndWait(t *testing.T) {
	out := snapshotBanner(&baselinesv1.SnapshotForBaselineResponse{
		RunId: "20260615-000000-abcd", Scenario: "web", Name: "pre-launch",
		EstimatedTotalSeconds: 600, EtaKnown: true,
	})
	for _, want := range []string{
		"20260615-000000-abcd", // run id
		"10m0s",                // ETA
		"Agent wait protocol",
		"test-genie runs wait --json --timeout=1050 web 20260615-000000-abcd", // quiet agent re-attach (primary)
		"recommended wait timeout: 17m30s",
		"tail --pid=<pid> -f /dev/null",
		"baseline snapshot status --scenario web --name pre-launch --run 20260615-000000-abcd",
		"watch live:    test-genie runs follow web 20260615-000000-abcd", // human alt
		"pins automatically when it completes",                           // durable, no silent block
	} {
		if !strings.Contains(out, want) {
			t.Errorf("banner missing %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{"Codex", "Claude"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("banner must stay provider-agnostic; found %q in:\n%s", forbidden, out)
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
		"Agent wait protocol",
		"test-genie runs wait --json --timeout=441 web 20260616-000000-abcd",
		"recommended wait timeout: 7m21s",
		"watch live:    test-genie runs follow web 20260616-000000-abcd",
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
