package calibration

import (
	"fmt"
	"sort"
	"unit-health/internal/testquality"
)

// HoldoutComparison compares independently authored labels with observations.
// Missing or invalid observations stay in the denominator as unknowns.
type HoldoutComparison struct {
	SchemaVersion    string                    `json:"schemaVersion"`
	RuleVersion      string                    `json:"ruleVersion"`
	Denominator      int                       `json:"denominator"`
	Compared         int                       `json:"compared"`
	Unknown          int                       `json:"unknown"`
	Excluded         int                       `json:"excluded"`
	FalsePositives   int                       `json:"falsePositives"`
	FalseNegatives   int                       `json:"falseNegatives"`
	Confusion        map[string]map[string]int `json:"confusion"`
	Disagreements    []string                  `json:"disagreements"`
	PromotionAllowed bool                      `json:"promotionAllowed"`
	PromotionReason  string                    `json:"promotionReason"`
}

// CompareHoldout requires one independent label per test identity and never
// infers a human label from model output.
func CompareHoldout(labels []testquality.HoldoutLabel, observations []testquality.SampledObservation, ruleVersion string) (HoldoutComparison, error) {
	if len(labels) == 0 {
		return HoldoutComparison{}, fmt.Errorf("independent holdout labels are required")
	}
	if ruleVersion == "" {
		return HoldoutComparison{}, fmt.Errorf("rule version is required")
	}
	out := HoldoutComparison{SchemaVersion: "test-quality-holdout/v1", RuleVersion: ruleVersion, Denominator: len(labels), Confusion: map[string]map[string]int{}, Disagreements: []string{}}
	obs := map[string]testquality.SampledObservation{}
	for _, row := range observations {
		if row.TestIdentity != "" {
			obs[row.TestIdentity] = row
		}
	}
	seen := map[string]bool{}
	for _, label := range labels {
		if label.TestIdentity == "" || label.Reviewer == "" || label.ReviewedAt == "" || !testquality.ValidReviewLabel(label.Label) {
			out.Unknown++
			continue
		}
		if seen[label.TestIdentity] {
			out.Excluded++
			continue
		}
		seen[label.TestIdentity] = true
		row, ok := obs[label.TestIdentity]
		if !ok || row.Status != "observed" || !testquality.ValidReviewLabel(row.Label) || row.SourceIdentity != label.SourceIdentity {
			out.Unknown++
			out.Disagreements = append(out.Disagreements, label.TestIdentity)
			continue
		}
		out.Compared++
		if out.Confusion[label.Label] == nil {
			out.Confusion[label.Label] = map[string]int{}
		}
		out.Confusion[label.Label][row.Label]++
		if label.Label != row.Label {
			out.Disagreements = append(out.Disagreements, label.TestIdentity)
		}
		// Labels which indicate a quality problem are the positive class.
		positive := func(s string) bool {
			return s == "weak_oracle" || s == "missing_negative_case" || s == "implementation_coupled"
		}
		if positive(row.Label) && !positive(label.Label) {
			out.FalsePositives++
		}
		if !positive(row.Label) && positive(label.Label) {
			out.FalseNegatives++
		}
	}
	sort.Strings(out.Disagreements)
	// Comparison can establish eligibility evidence, but an owner decision is
	// always required. Keep this false so callers cannot accidentally promote.
	out.PromotionAllowed = false
	out.PromotionReason = "owner decision required; comparison evidence never promotes a rule automatically"
	return out, nil
}
