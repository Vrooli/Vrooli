package operatingmode

import (
	"fmt"
	"strings"
)

type ReadinessDimension struct {
	Key       string  `json:"key"`
	Label     string  `json:"label,omitempty"`
	Score     float64 `json:"score"`
	Rationale string  `json:"rationale,omitempty"`
}

type ReadinessReport struct {
	Dimensions   []ReadinessDimension `json:"dimensions"`
	OverallScore float64              `json:"overall_score"`
	Ready        bool                 `json:"ready"`
}

var holisticReadinessLabels = map[string]string{
	"problem_clarity":           "Problem clarity",
	"scope_defined":             "Scope defined",
	"approach_solid":            "Approach solid",
	"testable":                  "Testable",
	"risk_awareness":            "Risk awareness",
	"coupling_understood":       "Coupling understood",
	"system_acceptance_defined": "System acceptance defined",
}

func ScoreHolisticReadiness(dimensions []ReadinessDimension) (ReadinessReport, error) {
	if len(dimensions) == 0 {
		return ReadinessReport{}, fmt.Errorf("readiness dimensions are required")
	}
	out := make([]ReadinessDimension, 0, len(dimensions))
	var sum float64
	for _, dim := range dimensions {
		key := strings.TrimSpace(dim.Key)
		if key == "" {
			return ReadinessReport{}, fmt.Errorf("readiness dimension key is required")
		}
		if _, ok := holisticReadinessLabels[key]; !ok {
			return ReadinessReport{}, fmt.Errorf("unknown holistic readiness dimension %q", key)
		}
		if dim.Score < 0 || dim.Score > 1 {
			return ReadinessReport{}, fmt.Errorf("readiness dimension %q score must be between 0 and 1", key)
		}
		if strings.TrimSpace(dim.Label) == "" {
			dim.Label = holisticReadinessLabels[key]
		}
		dim.Key = key
		sum += dim.Score
		out = append(out, dim)
	}
	overall := sum / float64(len(out))
	return ReadinessReport{
		Dimensions:   out,
		OverallScore: overall,
		Ready:        overall >= 0.8,
	}, nil
}
