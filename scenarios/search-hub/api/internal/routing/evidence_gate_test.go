package routing

import "testing"

func TestAutomaticEvidenceExclusionMakesAllStaleCorpusUnusable(t *testing.T) {
	got := automaticEvidenceExclusion(EvalQualityEvidence{
		EvidenceAvailable:    true,
		SuitePresent:         true,
		LiveReviewedPositive: true,
		RecentPassingRun:     true,
		CorpusAllStale:       true,
	})
	if got != "all reviewed positives are stale" {
		t.Fatalf("automaticEvidenceExclusion() = %q", got)
	}
}
