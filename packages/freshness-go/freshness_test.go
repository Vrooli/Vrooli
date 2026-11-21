package freshness

import (
	"testing"
	"time"

	"github.com/vrooli/freshness-go/runindex"
)

func passedRun(id, digest string, phaseNames ...string) runindex.RunRecord {
	rec := runindex.RunRecord{
		RunID:       id,
		Status:      runindex.StatusPassed,
		TreeDigest:  digest,
		CompletedAt: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
	}
	for _, p := range phaseNames {
		rec.Phases = append(rec.Phases, runindex.PhaseRecord{Name: p, Status: "passed"})
	}
	return rec
}

func verdictsByPhase(t *testing.T, resp map[string]string, phase, want string) {
	t.Helper()
	if got := resp[phase]; got != want {
		t.Errorf("phase %s: status = %q, want %q", phase, got, want)
	}
}

func collect(records []runindex.RunRecord, digest string, names []string) map[string]string {
	report := Check(records, digest, names)
	out := make(map[string]string)
	for _, v := range report.Phases {
		out[v.Phase] = v.Status
	}
	return out
}

func TestCheckFreshStaleUnknown(t *testing.T) {
	current := "td:aaa"

	// fresh: run at current digest passed unit; stale: business only passed at
	// an older digest.
	records := []runindex.RunRecord{
		passedRun("r2", current, "unit"),
		passedRun("r1", "td:old", "unit", "business"),
	}
	got := collect(records, current, []string{"unit", "business"})
	verdictsByPhase(t, got, "unit", StatusFresh)
	verdictsByPhase(t, got, "business", StatusStale)

	// unknown: no digest-stamped runs at all (pre-digest history).
	legacy := []runindex.RunRecord{passedRun("r0", "", "unit")}
	got = collect(legacy, current, []string{"unit"})
	verdictsByPhase(t, got, "unit", StatusUnknown)

	// empty index is also unknown.
	got = collect(nil, current, []string{"unit"})
	verdictsByPhase(t, got, "unit", StatusUnknown)
}

func TestCheckFailedPhaseIsNotFresh(t *testing.T) {
	current := "td:aaa"
	rec := passedRun("r1", current)
	rec.Phases = []runindex.PhaseRecord{{Name: "unit", Status: "failed"}}
	got := collect([]runindex.RunRecord{rec}, current, []string{"unit"})
	verdictsByPhase(t, got, "unit", StatusStale)
}

func TestCheckReportsNewestEvidence(t *testing.T) {
	current := "td:aaa"
	records := []runindex.RunRecord{ // newest-first
		passedRun("r3", current, "unit"),
		passedRun("r2", current, "unit"),
	}
	report := Check(records, current, []string{"unit"})
	if report.Phases[0].LastRunID != "r3" {
		t.Fatalf("expected newest run r3, got %q", report.Phases[0].LastRunID)
	}
}

func TestCheckStaleCarriesLastPassedContext(t *testing.T) {
	current := "td:new"
	records := []runindex.RunRecord{passedRun("r1", "td:old", "unit")}
	report := Check(records, current, []string{"unit"})
	v := report.Phases[0]
	if v.Status != StatusStale || v.LastRunID != "r1" || v.LastRunCompletedAt == "" {
		t.Fatalf("stale verdict should carry newest any-digest evidence, got %+v", v)
	}
}

func TestSuggestedCommand(t *testing.T) {
	report := Check(nil, "td:x", []string{"unit", "business"})
	if cmd := SuggestedCommand("demo", report.Phases, true); cmd != "test-genie execute demo --preset quick" {
		t.Fatalf("defaulted suggestion = %q", cmd)
	}
	if cmd := SuggestedCommand("demo", report.Phases, false); cmd != "test-genie execute demo unit business" {
		t.Fatalf("explicit suggestion = %q", cmd)
	}

	fresh := Check([]runindex.RunRecord{passedRun("r1", "td:x", "unit")}, "td:x", []string{"unit"})
	if cmd := SuggestedCommand("demo", fresh.Phases, true); cmd != "" {
		t.Fatalf("all-fresh suggestion should be empty, got %q", cmd)
	}
}

func TestNormalizePhases(t *testing.T) {
	got := NormalizePhases([]string{" Unit ", "unit", "", "BUSINESS"})
	want := []string{"unit", "business"}
	if len(got) != len(want) {
		t.Fatalf("NormalizePhases = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("NormalizePhases = %v, want %v", got, want)
		}
	}
}

func TestRequiredPhasesIncludesBusiness(t *testing.T) {
	found := false
	for _, p := range RequiredPhases() {
		if p == "business" {
			found = true
		}
	}
	if !found {
		t.Fatal("required set must include the business phase")
	}
}
