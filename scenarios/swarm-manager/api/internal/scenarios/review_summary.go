package scenarios

import (
	"time"

	"swarm-manager/internal/execution"
)

// ScenarioReviewSummary holds the last review classification and timestamp for a scenario.
type ScenarioReviewSummary struct {
	LastReviewClassification string
	LastReviewAt             time.Time
}

// ComputeReviewSummaries walks execution records and returns per-scenario
// last review classification and timestamp. This is a pure function.
func ComputeReviewSummaries(records []execution.Record) map[string]ScenarioReviewSummary {
	result := make(map[string]ScenarioReviewSummary)
	for _, rec := range records {
		if rec.Finalization == nil {
			continue
		}
		for _, sf := range rec.Finalization.Scenarios {
			if sf.Review.Result == nil || sf.Review.Result.ReviewedAt == "" {
				continue
			}
			reviewedAt, err := time.Parse(time.RFC3339, sf.Review.Result.ReviewedAt)
			if err != nil {
				continue
			}
			existing := result[sf.ScenarioName]
			if reviewedAt.After(existing.LastReviewAt) {
				result[sf.ScenarioName] = ScenarioReviewSummary{
					LastReviewClassification: sf.Review.Result.Classification,
					LastReviewAt:             reviewedAt,
				}
			}
		}
	}
	return result
}
