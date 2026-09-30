package validation

import (
	"encoding/json"
	"testing"

	"unit-health/internal/evidence"
	"unit-health/internal/executor"
)

// [REQ:UH-ANALYZE-011]
func TestEvidenceStagesNeverInferExecutionOrReviewFromStaticStatus(t *testing.T) {
	response := Response{RunID: "source", Status: "passed", ProjectionChecks: []ProjectionCheck{{}}}
	stages := summarizeEvidenceStages(response, false)
	if stages.Configured != "observed" || stages.Executed != "not_requested" || stages.Analyzed != "unknown" || stages.Reviewed != "not_supplied" {
		t.Fatalf("static pass overclaimed: %+v", stages)
	}
	if got := summarizeEvidenceStages(response, true).Executed; got != "not_executed" {
		t.Fatalf("empty results=%s", got)
	}
	response.CommandResults = []CommandResult{{Status: executor.StatusError, FailureClass: executor.ClassUnsupported}}
	if got := summarizeEvidenceStages(response, true).Executed; got != "refused" {
		t.Fatalf("refused execution=%s", got)
	}
	response.CommandResults[0] = CommandResult{Status: "future"}
	if got := summarizeEvidenceStages(response, true).Executed; got != "unknown" {
		t.Fatalf("future execution=%s", got)
	}
}

func TestCachedEvidenceStagesRetainOriginalRunAndDoNotClaimFreshExecution(t *testing.T) {
	response := Response{RunID: "original", EvidenceStages: &EvidenceStages{Configured: "observed", Analyzed: "observed", Executed: "passed", Reviewed: "not_supplied", SourceRunID: "original"}}
	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	cached, ok := cachedResponse(evidence.Record{Payload: raw}, "new")
	if !ok || cached.RunID != "new" || cached.EvidenceStages.SourceRunID != "original" || cached.EvidenceStages.Executed != "cached" || cached.EvidenceStages.Reviewed != "not_supplied" {
		t.Fatalf("cache provenance lost: %+v", cached)
	}
}

func TestEvidenceStagesNormalizationClampsUnknownValues(t *testing.T) {
	normalized := (EvidenceStages{Configured: "future", Analyzed: "future", Executed: "future", Reviewed: "future"}).Normalized()
	if normalized.Configured != "unknown" || normalized.Analyzed != "unknown" || normalized.Executed != "unknown" || normalized.Reviewed != "unknown" {
		t.Fatalf("normalized = %+v", normalized)
	}
	if got := (EvidenceStages{Configured: "cached", Analyzed: "partial", Executed: "failed", Reviewed: "not_supplied"}).Normalized(); got.Configured != "cached" || got.Executed != "failed" {
		t.Fatalf("known stages changed: %+v", got)
	}
}

func TestReviewedEvidenceRequiresAttachedObservedCohort(t *testing.T) {
	base := summarizeEvidenceStages(Response{RunID: "run"}, false)
	attachReviewedEvidence(base, Request{ReviewedCohortID: "cohort", ReviewedSourceIdentity: "sha256:source", ReviewedObservationCount: 0})
	if base.Reviewed != "not_supplied" {
		t.Fatalf("empty cohort overclaimed review: %+v", base)
	}
	attachReviewedEvidence(base, Request{ReviewedCohortID: "cohort", ReviewedSourceIdentity: "sha256:source", ReviewedObservationCount: 1})
	if base.Reviewed != "supplied" {
		t.Fatalf("observed cohort was not supplied: %+v", base)
	}
}
