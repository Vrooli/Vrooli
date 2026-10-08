package research_test

import (
	"testing"

	research "web-search/internal/research"
)

func TestContractRejectsOversizedQuestionsBeforeExecution(t *testing.T) {
	questions := make([]research.ResearchQuestion, 21)
	if err := research.ValidateContractBounds("q", questions, research.EvidencePolicy{}); err == nil {
		t.Fatal("expected oversized question set to be rejected")
	}
}

func TestEvidencePolicyUsesConservativeDefaults(t *testing.T) {
	p := research.EvidencePolicy{Query: "current fact"}
	if err := p.Validate(); err != nil {
		t.Fatalf("default policy should validate: %v", err)
	}
	if p.Effort != "l2" || p.MinimumSources != 1 || p.TopN != research.DefaultTopN {
		t.Fatalf("defaults = effort %q, sources %d, top_n %d; want l2, 1, %d", p.Effort, p.MinimumSources, p.TopN, research.DefaultTopN)
	}
}

func TestUnknownAssessmentCannotBecomeSupported(t *testing.T) {
	if research.AssessmentDisposition("future_value").Supported() {
		t.Fatal("unknown disposition must not be supported")
	}
	if research.AssessmentUnknown.Supported() {
		t.Fatal("unknown assessment must remain unknown")
	}
}

func TestContractRejectsUnboundedEvidencePolicy(t *testing.T) {
	if err := research.ValidateContractBounds("q", nil, research.EvidencePolicy{MaxEvidenceBytes: 4<<20 + 1}); err == nil {
		t.Fatal("expected evidence byte limit to be enforced")
	}
}
