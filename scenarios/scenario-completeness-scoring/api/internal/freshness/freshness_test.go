package freshness

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/freshness-go/runindex"
)

func fixedService(digest string, digestErr error, records []runindex.RunRecord) *Service {
	return &Service{
		ComputeDigest: func(string) (string, error) { return digest, digestErr },
		LoadRecords:   func(string) ([]runindex.RunRecord, error) { return records, nil },
	}
}

func TestCheckFreshAndStaleVerdicts(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	records := []runindex.RunRecord{
		{
			RunID:       "run-new",
			CompletedAt: now,
			TreeDigest:  "td:current",
			Phases: []runindex.PhaseRecord{
				{Name: "unit", Status: "passed"},
				{Name: "proto", Status: "failed"},
			},
		},
		{
			RunID:       "run-old",
			CompletedAt: now.Add(-time.Hour),
			TreeDigest:  "td:old",
			Phases: []runindex.PhaseRecord{
				{Name: "structure", Status: "passed"},
				{Name: "docs", Status: "passed"},
				{Name: "business", Status: "passed"},
				{Name: "proto", Status: "passed"},
			},
		},
	}

	res := fixedService("td:current", nil, records).Check("fixture", "/unused")

	if res.Digest != "td:current" || res.DigestErr != "" {
		t.Fatalf("digest = %q err=%q", res.Digest, res.DigestErr)
	}
	byPhase := map[string]PhaseStatus{}
	for _, p := range res.Phases {
		byPhase[p.Phase] = p
	}

	unit := byPhase["unit"]
	if unit.Verdict != "fresh" || unit.LastRunID != "run-new" || unit.LastDigest != "td:current" || unit.LastStatus != "passed" {
		t.Fatalf("unit verdict = %+v", unit)
	}
	// proto passed only at the OLD digest -> stale, with last-passed context.
	proto := byPhase["proto"]
	if proto.Verdict != "stale" || proto.LastRunID != "run-old" || proto.LastDigest != "td:old" {
		t.Fatalf("proto verdict = %+v", proto)
	}
	if res.SuggestedCommand == "" || !strings.Contains(res.SuggestedCommand, "fixture") {
		t.Fatalf("suggested command = %q", res.SuggestedCommand)
	}
}

func TestCheckNeverTestedScenarioIsUnknown(t *testing.T) {
	res := fixedService("td:current", nil, nil).Check("fixture", "/unused")
	for _, p := range res.Phases {
		if p.Verdict != "unknown" {
			t.Fatalf("phase %s verdict = %q, want unknown", p.Phase, p.Verdict)
		}
	}
	if res.SuggestedCommand == "" {
		t.Fatal("expected a suggested command for unknown verdicts")
	}
}

func TestCheckDigestFailureDegradesToUnknown(t *testing.T) {
	records := []runindex.RunRecord{{
		RunID:      "run-new",
		TreeDigest: "td:something",
		Phases:     []runindex.PhaseRecord{{Name: "unit", Status: "passed"}},
	}}
	res := fixedService("", errors.New("not a directory"), records).Check("fixture", "/unused")

	if res.Digest != "" || res.DigestErr == "" {
		t.Fatalf("expected digest error, got %+v", res)
	}
	for _, p := range res.Phases {
		if p.Verdict != "unknown" {
			t.Fatalf("phase %s verdict = %q, want unknown (digest unavailable)", p.Phase, p.Verdict)
		}
	}
}

func TestCheckUsesRequiredPhaseSet(t *testing.T) {
	res := fixedService("td:x", nil, nil).Check("fixture", "/unused")
	want := []string{"structure", "docs", "unit", "business", "proto"}
	if len(res.Phases) != len(want) {
		t.Fatalf("phase count = %d, want %d", len(res.Phases), len(want))
	}
	for i, p := range res.Phases {
		if p.Phase != want[i] {
			t.Fatalf("phase[%d] = %q, want %q", i, p.Phase, want[i])
		}
	}
}

// TestCheckScenarioLevelLastRun verifies the scenario-level recency fields:
// the newest run overall (regardless of phase/digest) drives LastRun*, with the
// start time used as a fallback when the run has not completed.
func TestCheckScenarioLevelLastRun(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	records := []runindex.RunRecord{
		{RunID: "run-new", StartedAt: now.Add(-time.Minute), CompletedAt: now, Status: "passed", TreeDigest: "td:current"},
		{RunID: "run-old", StartedAt: now.Add(-2 * time.Hour), CompletedAt: now.Add(-time.Hour), Status: "failed", TreeDigest: "td:old"},
	}
	res := fixedService("td:current", nil, records).Check("fixture", "/unused")
	if res.LastRunID != "run-new" || res.LastStatus != "passed" || !res.LastRunAt.Equal(now) {
		t.Fatalf("scenario recency = %s/%s/%v, want run-new/passed/%v", res.LastRunID, res.LastStatus, res.LastRunAt, now)
	}

	// In-progress newest run (no completed_at) falls back to started_at.
	inprog := []runindex.RunRecord{
		{RunID: "run-live", StartedAt: now, Status: "in_progress", TreeDigest: "td:current"},
	}
	res = fixedService("td:current", nil, inprog).Check("fixture", "/unused")
	if res.LastRunID != "run-live" || res.LastStatus != "in_progress" || !res.LastRunAt.Equal(now) {
		t.Fatalf("in-progress recency = %s/%s/%v, want run-live/in_progress/%v", res.LastRunID, res.LastStatus, res.LastRunAt, now)
	}

	// No runs -> empty recency (not a zero that reads as a real run).
	res = fixedService("td:current", nil, nil).Check("fixture", "/unused")
	if res.LastRunID != "" || res.LastStatus != "" || !res.LastRunAt.IsZero() {
		t.Fatalf("empty recency = %s/%s/%v, want all empty", res.LastRunID, res.LastStatus, res.LastRunAt)
	}
}
