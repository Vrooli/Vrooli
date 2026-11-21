package research_test

import (
	"testing"

	research "web-search/internal/research"
)

func TestQuestionCoverageDoesNotBorrowSupportAcrossParts(t *testing.T) {
	questions := []research.ResearchQuestion{{ID: "a", Prompt: "first", Required: true}, {ID: "b", Prompt: "second", Required: true}}
	coverage := research.EvaluateQuestionCoverage(questions, []research.ClaimAssessment{{ClaimID: "claim-a", Disposition: research.AssessmentSupported}}, map[string][]string{"a": {"claim-a"}, "b": {"claim-b"}})
	if coverage[0].Status != "supported" {
		t.Fatalf("first question status = %q", coverage[0].Status)
	}
	if coverage[1].Status == "supported" {
		t.Fatal("unsupported question borrowed another question's support")
	}
}

func TestUnknownCoverageRemainsUnknown(t *testing.T) {
	coverage := research.EvaluateQuestionCoverage([]research.ResearchQuestion{{ID: "q", Prompt: "question", Required: true}}, []research.ClaimAssessment{{ClaimID: "c", Disposition: research.AssessmentUnknown}}, map[string][]string{"q": {"c"}})
	if coverage[0].Status != "unknown" {
		t.Fatalf("status = %q, want unknown", coverage[0].Status)
	}
}

func TestContradictionCannotProduceCompleteCoverage(t *testing.T) {
	coverage := research.EvaluateQuestionCoverage(
		[]research.ResearchQuestion{{ID: "q", Prompt: "question", Required: true}},
		[]research.ClaimAssessment{{ClaimID: "c", Disposition: research.AssessmentContradicted}},
		map[string][]string{"q": {"c"}},
	)
	if coverage[0].Status != "unresolved" || coverage[0].UnresolvedReason != "claim_not_supported" {
		t.Fatalf("contradiction coverage = %+v, want unresolved claim_not_supported", coverage[0])
	}
}

func TestMissingClaimPreservesQuestionDenominator(t *testing.T) {
	questions := []research.ResearchQuestion{
		{ID: "known", Prompt: "known", Required: true},
		{ID: "missing", Prompt: "missing", Required: true},
	}
	coverage := research.EvaluateQuestionCoverage(
		questions,
		[]research.ClaimAssessment{{ClaimID: "known-claim", Disposition: research.AssessmentSupported}},
		map[string][]string{"known": {"known-claim"}},
	)
	if len(coverage) != len(questions) {
		t.Fatalf("coverage length = %d, want %d", len(coverage), len(questions))
	}
	if coverage[1].Status != "unresolved" || coverage[1].UnresolvedReason != "no_claim_assigned" {
		t.Fatalf("missing question coverage = %+v", coverage[1])
	}
}
