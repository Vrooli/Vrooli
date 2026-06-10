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
				{Name: "standards", Status: "failed"},
			},
		},
		{
			RunID:       "run-old",
			CompletedAt: now.Add(-time.Hour),
			TreeDigest:  "td:old",
			Phases: []runindex.PhaseRecord{
				{Name: "standards", Status: "passed"},
				{Name: "structure", Status: "passed"},
				{Name: "docs", Status: "passed"},
				{Name: "business", Status: "passed"},
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
	// standards passed only at the OLD digest -> stale, with last-passed context.
	standards := byPhase["standards"]
	if standards.Verdict != "stale" || standards.LastRunID != "run-old" || standards.LastDigest != "td:old" {
		t.Fatalf("standards verdict = %+v", standards)
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
	want := []string{"structure", "standards", "docs", "business", "unit"}
	if len(res.Phases) != len(want) {
		t.Fatalf("phase count = %d, want %d", len(res.Phases), len(want))
	}
	for i, p := range res.Phases {
		if p.Phase != want[i] {
			t.Fatalf("phase[%d] = %q, want %q", i, p.Phase, want[i])
		}
	}
}
