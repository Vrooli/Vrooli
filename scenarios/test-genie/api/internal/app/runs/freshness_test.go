package runs

import (
	"testing"
	"time"

	"test-genie/internal/orchestrator/phases"
	sharedruns "test-genie/internal/shared/runs"
)

func passedRun(id, digest string, phaseNames ...string) sharedruns.RunRecord {
	rec := sharedruns.RunRecord{
		RunID:       id,
		Status:      sharedruns.StatusPassed,
		TreeDigest:  digest,
		CompletedAt: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
	}
	for _, p := range phaseNames {
		rec.Phases = append(rec.Phases, sharedruns.PhaseRecord{Name: p, Status: "passed"})
	}
	return rec
}

func verdictsByPhase(t *testing.T, resp map[string]string, phase, want string) {
	t.Helper()
	if got := resp[phase]; got != want {
		t.Errorf("phase %s: status = %q, want %q", phase, got, want)
	}
}

func collect(records []sharedruns.RunRecord, digest string, names []string) map[string]string {
	resp := checkFreshness(records, digest, names)
	out := make(map[string]string)
	for _, v := range resp.GetPhases() {
		out[v.GetPhase()] = v.GetStatus()
	}
	return out
}

func TestCheckFreshnessFreshStaleUnknown(t *testing.T) {
	current := "td:aaa"

	// fresh: run at current digest passed unit; stale: business only passed at
	// an older digest.
	records := []sharedruns.RunRecord{
		passedRun("r2", current, "unit"),
		passedRun("r1", "td:old", "unit", "business"),
	}
	got := collect(records, current, []string{"unit", "business"})
	verdictsByPhase(t, got, "unit", "fresh")
	verdictsByPhase(t, got, "business", "stale")

	// unknown: no digest-stamped runs at all (pre-digest history).
	legacy := []sharedruns.RunRecord{passedRun("r0", "", "unit")}
	got = collect(legacy, current, []string{"unit"})
	verdictsByPhase(t, got, "unit", "unknown")

	// empty index is also unknown.
	got = collect(nil, current, []string{"unit"})
	verdictsByPhase(t, got, "unit", "unknown")
}

func TestCheckFreshnessFailedPhaseIsNotFresh(t *testing.T) {
	current := "td:aaa"
	rec := passedRun("r1", current)
	rec.Phases = []sharedruns.PhaseRecord{{Name: "unit", Status: "failed"}}
	got := collect([]sharedruns.RunRecord{rec}, current, []string{"unit"})
	verdictsByPhase(t, got, "unit", "stale")
}

func TestCheckFreshnessReportsNewestEvidence(t *testing.T) {
	current := "td:aaa"
	records := []sharedruns.RunRecord{ // newest-first
		passedRun("r3", current, "unit"),
		passedRun("r2", current, "unit"),
	}
	resp := checkFreshness(records, current, []string{"unit"})
	if resp.GetPhases()[0].GetLastRunId() != "r3" {
		t.Fatalf("expected newest run r3, got %q", resp.GetPhases()[0].GetLastRunId())
	}
}

func TestFreshnessDefaultSetIsQuickPreset(t *testing.T) {
	want := phases.DefaultPresets()["quick"]
	got := phases.FreshnessRequired()
	if len(got) != len(want) {
		t.Fatalf("FreshnessRequired = %v, want quick preset %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("FreshnessRequired = %v, want quick preset %v", got, want)
		}
	}
	// The promoted business phase must be part of the required set — that is
	// the whole point of WS-B feeding WS-D.
	found := false
	for _, p := range got {
		if p == "business" {
			found = true
		}
	}
	if !found {
		t.Fatal("required set must include the business phase")
	}
}

func TestSuggestedCommand(t *testing.T) {
	resp := checkFreshness(nil, "td:x", []string{"unit", "business"})
	if cmd := suggestedCommand("demo", resp.GetPhases(), true); cmd != "test-genie execute demo --preset quick" {
		t.Fatalf("defaulted suggestion = %q", cmd)
	}
	if cmd := suggestedCommand("demo", resp.GetPhases(), false); cmd != "test-genie execute demo unit business" {
		t.Fatalf("explicit suggestion = %q", cmd)
	}

	fresh := checkFreshness([]sharedruns.RunRecord{passedRun("r1", "td:x", "unit")}, "td:x", []string{"unit"})
	if cmd := suggestedCommand("demo", fresh.GetPhases(), true); cmd != "" {
		t.Fatalf("all-fresh suggestion should be empty, got %q", cmd)
	}
}
