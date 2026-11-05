package agentcatalog

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// TokenWeightPolicyVersion identifies an operator heuristic in non-cache token
// units, not billing or measured subscription quota. v2 preserves Sol10/Luna1
// thresholds and declares Terra5 (half Sol); historical models remain readable.
const TokenWeightPolicyVersion = "operator-non-cache-v2-2026-10-27"

func DefaultTokenWeights() map[string]float64 {
	return map[string]float64{"sol": 10, "terra": 5, "luna": 1}
}

// ParseTokenWeights overlays explicit weights on the supplied defaults. Passing
// nil preserves the cap environment's existing replacement-map semantics.
func ParseTokenWeights(raw string, defaults map[string]float64) (map[string]float64, error) {
	weights := make(map[string]float64, len(defaults))
	for tier, weight := range defaults {
		weights[tier] = weight
	}
	if strings.TrimSpace(raw) == "" {
		return weights, nil
	}
	for _, pair := range strings.Split(raw, ",") {
		tier, value, ok := strings.Cut(strings.TrimSpace(pair), "=")
		weight, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if !ok || strings.TrimSpace(tier) == "" || err != nil || weight < 0 || math.IsNaN(weight) || math.IsInf(weight, 0) {
			return nil, fmt.Errorf("invalid weight %q (want tier=finite non-negative number)", pair)
		}
		weights[strings.ToLower(strings.TrimSpace(tier))] = weight
	}
	return weights, nil
}

// TokenWeight returns a known heuristic weight. An unmatched model is unknown,
// not implicitly an economical model. Sorted matching preserves existing ties.
func TokenWeight(model string, weights map[string]float64) (float64, bool) {
	model = strings.ToLower(model)
	tiers := make([]string, 0, len(weights))
	for tier := range weights {
		tiers = append(tiers, tier)
	}
	sort.Strings(tiers)
	for _, tier := range tiers {
		if strings.Contains(model, tier) {
			return weights[tier], true
		}
	}
	return 0, false
}
