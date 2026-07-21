package skills

import (
	"encoding/json"
	"sort"

	"prompt-manager/store"
)

// Outcome status vocabulary. Agent-manager posts the run's terminal status
// (complete|failed|cancelled) inside the opaque data blob; outcomes whose data
// carries no parseable status are bucketed as "unknown" and excluded from the
// success-rate denominator.
const (
	outcomeStatusSuccess = "complete"
	outcomeStatusUnknown = "unknown"
)

// outcomePayload is the best-effort parse of an outcome's data blob.
// Fields follow the shape agent-manager posts: {runId, status, tokensUsed}.
type outcomePayload struct {
	Status     string   `json:"status"`
	TokensUsed *float64 `json:"tokensUsed"`
}

// buildExperimentReport aggregates raw serves and outcomes into a per-arm
// report. Declared arms appear first (in experiment order); variant IDs that
// appear in serves or outcomes but not in the arms are appended sorted.
func buildExperimentReport(exp *store.Experiment, serves []store.ExperimentServe, outcomes []store.ExperimentOutcome) ExperimentReportResponse {
	serveCounts := make(map[string]int)
	for _, s := range serves {
		serveCounts[s.VariantID]++
	}

	type armAgg struct {
		outcomes    int
		statuses    map[string]int
		tokensSum   float64
		tokensCount int
	}
	aggs := make(map[string]*armAgg)
	for _, o := range outcomes {
		a, ok := aggs[o.VariantID]
		if !ok {
			a = &armAgg{statuses: make(map[string]int)}
			aggs[o.VariantID] = a
		}
		a.outcomes++

		status := outcomeStatusUnknown
		var payload outcomePayload
		if len(o.Data) > 0 && json.Unmarshal(o.Data, &payload) == nil {
			if payload.Status != "" {
				status = payload.Status
			}
			if payload.TokensUsed != nil {
				a.tokensSum += *payload.TokensUsed
				a.tokensCount++
			}
		}
		a.statuses[status]++
	}

	// Declared arms first, then any extra variant IDs seen in the data.
	ordered := make([]string, 0, len(exp.Arms))
	weights := make(map[string]float64, len(exp.Arms))
	seen := make(map[string]bool, len(exp.Arms))
	for _, arm := range exp.Arms {
		ordered = append(ordered, arm.VariantID)
		weights[arm.VariantID] = arm.Weight
		seen[arm.VariantID] = true
	}
	var extras []string
	for vid := range serveCounts {
		if !seen[vid] {
			seen[vid] = true
			extras = append(extras, vid)
		}
	}
	for vid := range aggs {
		if !seen[vid] {
			seen[vid] = true
			extras = append(extras, vid)
		}
	}
	sort.Strings(extras)
	ordered = append(ordered, extras...)

	report := ExperimentReportResponse{
		ExperimentID:  exp.ID,
		SkillID:       exp.SkillID,
		Name:          exp.Name,
		Status:        exp.Status,
		TotalServes:   len(serves),
		TotalOutcomes: len(outcomes),
		Arms:          make([]ExperimentArmReport, 0, len(ordered)),
	}

	for _, vid := range ordered {
		armReport := ExperimentArmReport{
			VariantID: vid,
			Weight:    weights[vid],
			Serves:    serveCounts[vid],
		}
		if a, ok := aggs[vid]; ok {
			armReport.Outcomes = a.outcomes
			armReport.StatusCounts = a.statuses
			known := a.outcomes - a.statuses[outcomeStatusUnknown]
			if known > 0 {
				rate := float64(a.statuses[outcomeStatusSuccess]) / float64(known)
				armReport.SuccessRate = &rate
			}
			if a.tokensCount > 0 {
				mean := a.tokensSum / float64(a.tokensCount)
				armReport.MeanTokensUsed = &mean
			}
		}
		if armReport.Serves == 0 && armReport.Outcomes == 0 {
			report.ZeroDataArms = append(report.ZeroDataArms, vid)
		}
		report.Arms = append(report.Arms, armReport)
	}

	return report
}
