package validation

import (
	"encoding/json"
	"testing"

	"unit-health/internal/evidence"
	"unit-health/internal/executor"
)

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
