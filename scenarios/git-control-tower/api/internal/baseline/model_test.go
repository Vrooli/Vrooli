package baseline

import (
	"encoding/json"
	"testing"
	"time"

	"git-control-tower/internal/git"
)

func TestManifestJSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 5, 26, 15, 0, 0, 0, time.UTC)
	in := BaselineManifest{
		Name:      "plan-7c3",
		Scenario:  "foo",
		Branch:    "agi",
		CreatedAt: ts,
		CreatedBy: "agent",
		Git: git.State{
			Sha: "abc123", Branch: "agi", Dirty: true, DirtySummary: "1 modified",
			CommitMessage: "m", CommitAuthor: "a", CommitDate: ts,
		},
		Surfaces: map[string]SurfacePointer{
			SurfaceTests: {SurfaceID: SurfaceTests, Kind: KindTestGenieRun, Ref: "run-1", CapturedAt: ts, Summary: json.RawMessage(`{"passed":3}`)},
		},
		SchemaVersion: SchemaVersion,
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out BaselineManifest
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	redata, _ := json.Marshal(out)
	if string(data) != string(redata) {
		t.Fatalf("round-trip mismatch:\n%s\n%s", data, redata)
	}
}

func TestWorseVerdictOrdering(t *testing.T) {
	cases := []struct{ a, b, want Verdict }{
		{VerdictClean, VerdictRegression, VerdictRegression},
		{VerdictNewFailure, VerdictPreexisting, VerdictNewFailure},
		{VerdictNotComparable, VerdictNewFailure, VerdictNotComparable},
		{VerdictRegression, VerdictNotComparable, VerdictRegression},
		{VerdictClean, VerdictClean, VerdictClean},
	}
	for _, c := range cases {
		if got := WorseVerdict(c.a, c.b); got != c.want {
			t.Errorf("WorseVerdict(%s,%s)=%s want %s", c.a, c.b, got, c.want)
		}
	}
}

func TestValidateRequiresFields(t *testing.T) {
	if err := (BaselineManifest{Scenario: "s", Branch: "b"}).Validate(); err == nil {
		t.Error("expected error for missing name")
	}
	if err := (BaselineManifest{Name: "n", Branch: "b"}).Validate(); err == nil {
		t.Error("expected error for missing scenario")
	}
	if err := (BaselineManifest{Name: "n", Scenario: "s", Branch: "b"}).Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResolveStorageBranch(t *testing.T) {
	if got := ResolveStorageBranch(git.State{Branch: "agi"}); got != "agi" {
		t.Errorf("branch = %q", got)
	}
	if got := ResolveStorageBranch(git.State{Detached: true, Sha: "deadbeefcafe"}); got != "detached-deadbeef" {
		t.Errorf("detached = %q", got)
	}
	if got := ResolveStorageBranch(git.State{Detached: true}); got != "detached-unknown" {
		t.Errorf("detached-unknown = %q", got)
	}
}
