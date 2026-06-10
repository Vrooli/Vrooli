package scores

import (
	"strings"
	"testing"

	scoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring"
)

func sampleResponse() *scoringv1.GetScoreResponse {
	return &scoringv1.GetScoreResponse{
		Scenario: "cli-health",
		Category: "utility",
		Maturity: &scoringv1.MaturityHeadline{
			WorkingRung:      "R1 Safe & standards-clean",
			SatisfiedThrough: "R0 Runnable & green",
			BuildPassing:     true,
			Dimensions: []*scoringv1.DimensionCount{
				{Dimension: "standards", ErrorPlus: 2, Total: 5, Approximate: true},
				{Dimension: "tests", ErrorPlus: 0, Total: 1},
			},
		},
		Composite: &scoringv1.CompositeScore{
			Score:               72,
			Classification:      "mostly_complete",
			ClassificationLabel: "Mostly complete, needs refinement and validation",
			Groups: []*scoringv1.ScoreGroup{
				{
					Id: "quality", Label: "Quality", Score: 40, Max: 50,
					Metrics: []*scoringv1.MetricLine{
						{Id: "requirement_pass_rate", Label: "Requirements", Observed: "34 total, 30 passing (88%)", Points: 17.6, MaxPoints: 20},
						{Id: "phase_pass_rate", Label: "Test Phases", Observed: "10 recorded, 8 passing (80%)", Points: 12, MaxPoints: 15},
					},
				},
			},
		},
		Freshness: &scoringv1.FreshnessBlock{
			CurrentDigest:    "td:1d6f1c94aaaa",
			SuggestedCommand: "test-genie execute cli-health --preset quick",
			Phases: []*scoringv1.PhaseFreshness{
				{Phase: "structure", Verdict: "fresh", LastRunId: "run-9"},
				{Phase: "standards", Verdict: "stale", LastRunId: "run-3", LastDigest: "td:old"},
				{Phase: "unit", Verdict: "unknown"},
			},
		},
		Recommendations: []*scoringv1.Recommendation{
			{Priority: "high", Description: "Fix failing test phases", ImpactPoints: 3},
		},
		ActionPlan: []*scoringv1.ActionPhase{
			{Title: "Restore quality gates", Actions: []string{"Fix failing test phases"}, EstimatedPoints: 3},
		},
		Degradations: []*scoringv1.CollectorDegradation{
			{Collector: "ui", State: "failed", Reason: "no ui sources"},
		},
	}
}

// TestFormatReportSections pins the output contract: every product section
// renders, in order, with the load-bearing details present.
func TestFormatReportSections(t *testing.T) {
	out := FormatReport(sampleResponse())

	ordered := []string{
		"🪜 MATURITY — cli-health (utility)",
		"Working rung: R1 Safe & standards-clean",
		"Satisfied:    R0 Runnable & green",
		"As of digest: td:1d6f1c94aaaa",
		"standards: 2 error+, 5 open (approximated from phase status)",
		"📊 COMPLETENESS SCORE: 72/100 (mostly_complete)",
		"Quality (40/50):",
		"Requirements: 34 total, 30 passing (88%) → 17.6/20 pts",
		"⏱  FRESHNESS",
		"structure",
		"standards",
		"last passed in run-3 at td:old",
		"Refresh: test-genie execute cli-health --preset quick",
		"🎯 RECOMMENDATIONS",
		"[high] Fix failing test phases (+3 pts)",
		"🗺  ACTION PLAN",
		"Estimated score after fixes: ~75/100",
		"⚠️  DEGRADED COLLECTION",
		"ui collector failed: no ui sources",
	}
	idx := 0
	for _, want := range ordered {
		next := strings.Index(out[idx:], want)
		if next < 0 {
			t.Fatalf("missing or out-of-order section %q in:\n%s", want, out)
		}
		idx += next
	}
}

// TestFormatReportCleanLadderAndNoRecs renders the happy path: no
// recommendation/degradation sections, clean-ladder line, empty refresh.
func TestFormatReportCleanLadderAndNoRecs(t *testing.T) {
	msg := sampleResponse()
	msg.Maturity.WorkingRung = ""
	msg.Maturity.LadderClean = true
	msg.Recommendations = nil
	msg.ActionPlan = nil
	msg.Degradations = nil
	msg.Freshness.SuggestedCommand = ""

	out := FormatReport(msg)
	if !strings.Contains(out, "✅ clean through R4") {
		t.Fatalf("missing clean-ladder line:\n%s", out)
	}
	for _, absent := range []string{"RECOMMENDATIONS", "ACTION PLAN", "DEGRADED", "Refresh:"} {
		if strings.Contains(out, absent) {
			t.Fatalf("unexpected section %q on clean payload:\n%s", absent, out)
		}
	}
}

// TestFormatReportDigestFailure keeps the honesty contract visible: a
// digest failure renders the reason, never a fake digest.
func TestFormatReportDigestFailure(t *testing.T) {
	msg := sampleResponse()
	msg.Freshness.CurrentDigest = ""
	msg.Freshness.DigestError = "not a git checkout"

	out := FormatReport(msg)
	if !strings.Contains(out, "As of digest: unavailable (not a git checkout)") {
		t.Fatalf("digest failure not surfaced:\n%s", out)
	}
}
