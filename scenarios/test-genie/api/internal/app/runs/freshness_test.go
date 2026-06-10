package runs

import (
	"testing"
	"time"

	"test-genie/internal/orchestrator/phases"
	sharedruns "test-genie/internal/shared/runs"

	freshness "github.com/vrooli/freshness-go"
)

// The fresh/stale/unknown verdict semantics are owned (and tested) by the
// shared freshness-go package. The tests here cover what stays test-genie's:
// the wire conversion and the required-set SSOT.

func TestToFreshnessResponseConvertsAllFields(t *testing.T) {
	rec := sharedruns.RunRecord{
		RunID:       "r1",
		Status:      sharedruns.StatusPassed,
		TreeDigest:  "td:x",
		CompletedAt: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		Phases:      []sharedruns.PhaseRecord{{Name: "unit", Status: "passed"}},
	}
	report := freshness.Check([]sharedruns.RunRecord{rec}, "td:x", []string{"unit", "business"})
	resp := toFreshnessResponse(report)

	if resp.GetTreeDigest() != "td:x" {
		t.Fatalf("TreeDigest = %q", resp.GetTreeDigest())
	}
	if len(resp.GetPhases()) != 2 {
		t.Fatalf("got %d phases, want 2", len(resp.GetPhases()))
	}
	unit := resp.GetPhases()[0]
	if unit.GetPhase() != "unit" || unit.GetStatus() != freshness.StatusFresh ||
		unit.GetLastRunId() != "r1" || unit.GetLastRunCompletedAt() != "2026-06-10T00:00:00Z" {
		t.Fatalf("unit verdict not converted faithfully: %+v", unit)
	}
	if business := resp.GetPhases()[1]; business.GetStatus() != freshness.StatusStale {
		t.Fatalf("business verdict = %q, want stale", business.GetStatus())
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

// TestRequiredSetMatchesFreshnessGo pins test-genie's quick-preset-derived
// required set to the shared freshness-go mirror that no-service consumers
// (scenario-completeness-scoring) use. If the quick preset changes, this fails
// until freshness.RequiredPhases() is updated in lockstep — drift between the
// two would make "stale" mean different things in different surfaces.
func TestRequiredSetMatchesFreshnessGo(t *testing.T) {
	want := phases.FreshnessRequired()
	got := freshness.RequiredPhases()
	if len(got) != len(want) {
		t.Fatalf("freshness.RequiredPhases() = %v, want %v (test-genie quick preset)", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("freshness.RequiredPhases() = %v, want %v (test-genie quick preset)", got, want)
		}
	}
}
