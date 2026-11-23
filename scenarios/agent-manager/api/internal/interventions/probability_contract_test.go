package interventions

import (
	"math"
	"strings"
	"testing"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

func probabilityValues(labels []string) []*inferencev1.DecisionProbability {
	values := make([]*inferencev1.DecisionProbability, len(labels))
	for i, label := range labels {
		values[i] = &inferencev1.DecisionProbability{Option: label, Probability: 1 / float64(len(labels))}
	}
	return values
}

func TestProbabilityDecodersRejectMalformedModelDistributions(t *testing.T) {
	cases := []struct {
		name   string
		mutate func([]*inferencev1.DecisionProbability) []*inferencev1.DecisionProbability
	}{
		{name: "duplicate option", mutate: func(values []*inferencev1.DecisionProbability) []*inferencev1.DecisionProbability {
			values[0].Option = values[1].Option
			return values
		}},
		{name: "unknown option", mutate: func(values []*inferencev1.DecisionProbability) []*inferencev1.DecisionProbability {
			values[0].Option = "not_a_contract_label"
			return values
		}},
		{name: "missing option", mutate: func(values []*inferencev1.DecisionProbability) []*inferencev1.DecisionProbability {
			return values[:len(values)-1]
		}},
		{name: "non-finite probability", mutate: func(values []*inferencev1.DecisionProbability) []*inferencev1.DecisionProbability {
			values[0].Probability = math.NaN()
			return values
		}},
		{name: "probability outside unit interval", mutate: func(values []*inferencev1.DecisionProbability) []*inferencev1.DecisionProbability {
			values[0].Probability = 1.01
			return values
		}},
		{name: "unnormalized mass", mutate: func(values []*inferencev1.DecisionProbability) []*inferencev1.DecisionProbability {
			values[0].Probability += 0.03
			return values
		}},
	}
	categoryLabels := []string{"early_stop", "false_inability", "wrong_claim", "quality_miss", "too_dense", "instruction_ignored", "scope_change", "approval_or_info"}
	signalLabels := []string{"reported_failure", "correction", "frustration", "satisfaction", "neutral"}
	decoders := []struct {
		name       string
		labels     []string
		questionID string
		choice     string
		decode     func(*inferencev1.DecideResponse) (Status, string)
	}{
		{name: "category", labels: categoryLabels, questionID: questionID, choice: string(QualityMiss), decode: func(response *inferencev1.DecideResponse) (Status, string) {
			answer := answerFrom(response)
			return answer.Status, answer.Reason
		}},
		{name: "signal", labels: signalLabels, questionID: signalQuestionID, choice: string(ReportedFailure), decode: func(response *inferencev1.DecideResponse) (Status, string) {
			answer := signalAnswerFrom(response)
			return answer.Status, answer.Reason
		}},
	}

	for _, tc := range cases {
		for _, decoder := range decoders {
			decoder := decoder
			t.Run(tc.name+"/"+decoder.name, func(t *testing.T) {
				values := tc.mutate(probabilityValues(decoder.labels))
				response := &inferencev1.DecideResponse{Answers: []*inferencev1.DecisionAnswer{{QuestionId: decoder.questionID, Choice: decoder.choice, Probabilities: values}}}
				status, reason := decoder.decode(response)
				if status != StatusUnclassified || !strings.Contains(reason, "malformed probability distribution") {
					t.Fatalf("answer = %s: %s, want malformed distribution rejected", status, reason)
				}
			})
		}
	}
}
