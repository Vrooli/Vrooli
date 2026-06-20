package execution

import (
	"encoding/json"
	"testing"

	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
)

func TestExitCodeForBaselineVerdict(t *testing.T) {
	cases := map[string]int{
		baselineVerdictClean:         0,
		baselineVerdictNewFailure:    0,
		baselineVerdictPreExisting:   0,
		baselineVerdictRegression:    1,
		baselineVerdictNotComparable: 2,
		"unknown":                    0,
	}
	for verdict, want := range cases {
		if got := exitCodeForBaselineVerdict(verdict); got != want {
			t.Errorf("exitCodeForBaselineVerdict(%q) = %d, want %d", verdict, got, want)
		}
	}
}

func TestBaselineDiffResultFromProto_NilMessage(t *testing.T) {
	got := baselineDiffResultFromProto("swarm-manager", nil)
	if got.ScenarioName != "swarm-manager" {
		t.Fatalf("scenario = %q, want swarm-manager", got.ScenarioName)
	}
	if got.Verdict != baselineVerdictClean || !got.Comparable {
		t.Fatalf("nil message should default to clean+comparable, got verdict=%q comparable=%v", got.Verdict, got.Comparable)
	}
}

func TestBaselineDiffResultFromProto_RegressionSplit(t *testing.T) {
	msg := &baselinesv1.DiffResult{
		Verdict: baselineVerdictRegression,
		Staleness: &baselinesv1.Staleness{
			LikelyStale: true,
		},
		Surfaces: []*baselinesv1.SurfaceDiff{
			{
				SurfaceId:   "tests",
				Verdict:     baselineVerdictRegression,
				Regressions: []string{"TestFoo", "TestBar"},
				Preexisting: []string{"TestOld"},
			},
			{
				SurfaceId:   "standards",
				Verdict:     baselineVerdictNewFailure,
				NewFailures: []string{"new-lint-rule"},
				Cleared:     []string{"fixed-violation"},
			},
		},
	}

	got := baselineDiffResultFromProto("swarm-manager", msg)

	if got.Verdict != baselineVerdictRegression {
		t.Errorf("verdict = %q, want regression", got.Verdict)
	}
	if got.ExitCode != 1 {
		t.Errorf("exit code = %d, want 1", got.ExitCode)
	}
	if !got.Comparable {
		t.Errorf("regression should be comparable")
	}
	if !got.Stale {
		t.Errorf("expected stale=true from staleness")
	}
	if !got.HasNewRegressions() {
		t.Errorf("expected HasNewRegressions=true")
	}

	if len(got.RegressedSurfaces) != 1 || got.RegressedSurfaces[0] != "tests" {
		t.Errorf("regressed surfaces = %v, want [tests]", got.RegressedSurfaces)
	}
	if len(got.Regressions) != 2 {
		t.Fatalf("regressions = %v, want 2", got.Regressions)
	}
	for _, f := range got.Regressions {
		if f.Surface != "tests" {
			t.Errorf("regression finding surface = %q, want tests", f.Surface)
		}
	}
	if len(got.PreExisting) != 1 || got.PreExisting[0].Detail != "TestOld" {
		t.Errorf("preexisting = %v, want [{tests TestOld}]", got.PreExisting)
	}
	if len(got.NewFailures) != 1 || got.NewFailures[0].Surface != "standards" {
		t.Errorf("new failures = %v, want one on standards", got.NewFailures)
	}
	if len(got.Cleared) != 1 || got.Cleared[0].Detail != "fixed-violation" {
		t.Errorf("cleared = %v, want [{standards fixed-violation}]", got.Cleared)
	}
}

func TestBaselineDiffResultFromProto_NotComparable(t *testing.T) {
	msg := &baselinesv1.DiffResult{Verdict: baselineVerdictNotComparable}
	got := baselineDiffResultFromProto("git-control-tower", msg)
	if got.Comparable {
		t.Errorf("not-comparable verdict must set Comparable=false")
	}
	if got.ExitCode != 2 {
		t.Errorf("exit code = %d, want 2", got.ExitCode)
	}
	if got.HasNewRegressions() {
		t.Errorf("no surfaces means no regressions")
	}
}

func TestMarshalBaselineDiffResults(t *testing.T) {
	if got := MarshalBaselineDiffResults(nil); got != "" {
		t.Errorf("empty map should marshal to empty string, got %q", got)
	}
	results := map[string]BaselineDiffResult{
		"swarm-manager": {
			ScenarioName: "swarm-manager",
			Verdict:      baselineVerdictRegression,
			ExitCode:     1,
			Comparable:   true,
			Regressions:  []SurfaceFinding{{Surface: "tests", Detail: "TestFoo"}},
		},
	}
	out := MarshalBaselineDiffResults(results)
	if out == "" {
		t.Fatal("expected non-empty JSON")
	}
	var round map[string]BaselineDiffResult
	if err := json.Unmarshal([]byte(out), &round); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if round["swarm-manager"].Regressions[0].Detail != "TestFoo" {
		t.Errorf("round-trip lost regression detail: %+v", round)
	}
}
