package skills

import (
	"encoding/json"
	"math"
	"sort"

	"prompt-manager/internal/store"
)

// Outcome status vocabulary. Agent-manager terminal status is guardrail data
// only. It is intentionally not translated into a primary success metric.
const (
	outcomeStatusUnknown = "unknown"
)

// outcomePayload is the best-effort parse of an outcome's data blob.
// Fields follow the shape agent-manager posts: {runId, status, tokensUsed}.
type outcomePayload struct {
	Status     string   `json:"status"`
	TokensUsed *float64 `json:"tokensUsed"`
}

// buildControlledReport is intentionally independent of buildExperimentReport:
// only assignment-bound evaluator outcomes may enter this projection. Serves,
// terminal states, and unattributed reads are observational data.
func buildControlledReport(exp *store.Experiment, assignments []store.ExperimentAssignment, exposures []store.ExperimentExposure, outcomes []store.ExperimentOutcome) *ControlledReport {
	report := &ControlledReport{Assignments: len(assignments), ExclusionReasons: map[string]int{}}
	byID := make(map[string]store.ExperimentAssignment, len(assignments))
	for _, a := range assignments {
		byID[a.IdempotencyKey] = a
	}
	contaminated := make(map[string]bool)
	for _, exposure := range exposures {
		for _, a := range assignments {
			if exposure.ExecutionID != a.ExecutionID || exposure.ReadSkillID != exp.SkillID {
				continue
			}
			// Reads of the treatment skill by another workflow node, or under a
			// different experimental arm, are not a clean single-treatment run.
			if exposure.NodeID != a.NodeID || exposure.VariantID != a.VariantID {
				contaminated[a.IdempotencyKey] = true
			}
		}
	}
	byAssignment := make(map[string]store.ExperimentOutcome, len(outcomes))
	for _, o := range outcomes {
		if o.Controlled == nil {
			continue
		}
		key := o.Controlled.AssignmentID
		// Agent-manager persists its own node-attempt UUID in the evaluator
		// provenance. Prompt-manager owns the durable dispatch assignment key.
		// Resolve that cross-service representation through the unique
		// execution/variant receipt instead of treating verified evidence as
		// incomplete solely because identifiers have different authorities.
		if _, exists := byID[key]; !exists {
			matches := make([]string, 0, 1)
			for assignmentKey, assignment := range byID {
				if assignment.ExecutionID == o.Controlled.ExecutionID && assignment.VariantID == o.VariantID {
					matches = append(matches, assignmentKey)
				}
			}
			if len(matches) == 1 {
				key = matches[0]
			}
		}
		byAssignment[key] = o
	}
	type counts struct{ assignments, eligible, complete, successes int }
	armCounts := map[string]*counts{}
	for _, arm := range exp.Arms {
		armCounts[arm.VariantID] = &counts{}
	}
	for key, a := range byID {
		c := armCounts[a.VariantID]
		if c == nil {
			c = &counts{}
			armCounts[a.VariantID] = c
		}
		c.assignments++
		if contaminated[key] {
			report.ExcludedAssignments++
			report.ExclusionReasons["contaminated"]++
			continue
		}
		o, ok := byAssignment[key]
		if !ok || o.Controlled.OutcomeStatus != "complete" || o.Controlled.Success == nil {
			report.IncompleteAssignments++
			report.ExclusionReasons["incomplete-outcome"]++
			continue
		}
		report.EligibleAssignments++
		c.eligible++
		c.complete++
		if *o.Controlled.Success {
			c.successes++
		}
	}
	if report.Assignments > 0 {
		report.OutcomeCompleteness = float64(report.EligibleAssignments) / float64(report.Assignments)
	}
	if len(report.ExclusionReasons) == 0 {
		report.ExclusionReasons = nil
	}
	ordered := make([]string, 0, len(armCounts))
	for _, arm := range exp.Arms {
		ordered = append(ordered, arm.VariantID)
	}
	for id := range armCounts {
		found := false
		for _, declared := range ordered {
			if id == declared {
				found = true
				break
			}
		}
		if !found {
			ordered = append(ordered, id)
		}
	}
	controlMean := math.NaN()
	var control *counts
	for _, id := range ordered {
		if id == store.ControlVariantID {
			c := armCounts[id]
			if c.eligible > 0 {
				controlMean = betaPosteriorMean(c.successes, c.eligible)
				control = c
			}
		}
	}
	for _, id := range ordered {
		c := armCounts[id]
		row := ControlledArmReport{VariantID: id, Assignments: c.assignments, Eligible: c.eligible, Complete: c.complete, Successes: c.successes}
		if c.eligible > 0 {
			mean, low, high := betaSummary(c.successes, c.eligible)
			row.PosteriorMean, row.CredibleLow, row.CredibleHigh = &mean, &low, &high
			if !math.IsNaN(controlMean) && id != store.ControlVariantID {
				effect := mean - controlMean
				row.EffectVsControl = &effect
				prob := betaProbBeats(c.successes, c.eligible, control.successes, control.eligible)
				row.ProbBeatsControl = &prob
			}
		}
		report.Arms = append(report.Arms, row)
	}
	return report
}

// betaProbBeats computes P(V > C) exactly for V ~ Beta(sv+1, nv-sv+1) and
// C ~ Beta(sc+1, nc-sc+1) — the Beta(1,1)-posterior probability that the
// variant's true success rate exceeds control's. Unlike the credible
// intervals above, this feeds a pre-registered gate, so the exact
// integer-parameter closed form is used rather than a normal approximation.
func betaProbBeats(sv, nv, sc, nc int) float64 {
	a1, b1 := float64(sv+1), float64(nv-sv+1)
	a2, b2 := float64(sc+1), float64(nc-sc+1)
	lnBeta := func(x, y float64) float64 {
		lx, _ := math.Lgamma(x)
		ly, _ := math.Lgamma(y)
		lxy, _ := math.Lgamma(x + y)
		return lx + ly - lxy
	}
	total := 0.0
	for i := 0.0; i < a1; i++ {
		total += math.Exp(lnBeta(a2+i, b1+b2) - math.Log(b1+i) - lnBeta(1+i, b1) - lnBeta(a2, b2))
	}
	return total
}

// betaSummary reports the Beta(1,1)-posterior mean and a normal approximation
// to its central 95% credible interval. It is descriptive uncertainty only;
// the pre-registered gates remain authoritative.
func betaSummary(successes, n int) (mean, low, high float64) {
	a, b := float64(successes+1), float64(n-successes+1)
	mean = a / (a + b)
	variance := a * b / ((a + b) * (a + b) * (a + b + 1))
	delta := 1.96 * math.Sqrt(variance)
	return mean, math.Max(0, mean-delta), math.Min(1, mean+delta)
}

func betaPosteriorMean(successes, n int) float64 {
	mean, _, _ := betaSummary(successes, n)
	return mean
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
			// Do not infer quality from a run's terminal status. A primary metric
			// must come from the separately declared evaluator contract.
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
