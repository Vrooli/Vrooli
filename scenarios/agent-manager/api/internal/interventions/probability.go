package interventions

import (
	"fmt"
	"math"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

// decodeProbabilityDistribution validates model scores before either
// intervention classifier persists them or uses them for a rate gate. Choice
// answers without probabilities remain supported for the LLM enum fallback;
// a non-empty distribution must be complete, unique, finite and normalized.
func decodeProbabilityDistribution[T ~string](values []*inferencev1.DecisionProbability, labels []T) (map[T]float64, error) {
	if len(values) == 0 {
		return nil, nil
	}
	allowed := make(map[T]struct{}, len(labels))
	for _, label := range labels {
		allowed[label] = struct{}{}
	}
	if len(values) != len(allowed) {
		return nil, fmt.Errorf("got %d probabilities for %d labels", len(values), len(allowed))
	}
	probabilities := make(map[T]float64, len(values))
	total := 0.0
	for _, value := range values {
		label := T(value.GetOption())
		if _, ok := allowed[label]; !ok {
			return nil, fmt.Errorf("unknown option %q", label)
		}
		if _, duplicate := probabilities[label]; duplicate {
			return nil, fmt.Errorf("duplicate option %q", label)
		}
		probability := value.GetProbability()
		if math.IsNaN(probability) || math.IsInf(probability, 0) || probability < 0 || probability > 1 {
			return nil, fmt.Errorf("option %q has invalid probability %v", label, probability)
		}
		probabilities[label] = probability
		total += probability
	}
	if math.Abs(total-1) > 0.02 {
		return nil, fmt.Errorf("probabilities sum to %.6f, want 1", total)
	}
	return probabilities, nil
}
