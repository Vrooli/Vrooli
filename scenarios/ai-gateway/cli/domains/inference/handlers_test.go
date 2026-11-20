package inference

import (
	"strings"
	"testing"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

func TestDecisionLinesIncludeAnswerDistributionAndCandidateReason(t *testing.T) {
	response := &inferencev1.DecideResponse{
		Answers: []*inferencev1.DecisionAnswer{{
			QuestionId: "intent", Type: inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE,
			Choice: "ship", Confidence: float64Ptr(0.875),
			Probabilities: []*inferencev1.DecisionProbability{{Option: "ship", Probability: 0.875}},
		}},
		Candidates: []*inferencev1.DecisionCandidate{{Provider: "local", ResourceRole: "decision.fast", Reason: "available", LatencyMs: 12}},
	}

	lines := strings.Join(decisionLines(response), "\n")
	for _, expected := range []string{"intent choice=ship confidence=0.8750 probabilities={ship:0.8750}", "candidate local/decision.fast", "reason=available", "latency_ms=12"} {
		if !strings.Contains(lines, expected) {
			t.Errorf("decision lines missing %q in:\n%s", expected, lines)
		}
	}
}

func float64Ptr(value float64) *float64 { return &value }
