package scoring

import (
	"fmt"
	"math"
	"sort"

	"scenario-completeness-scoring/internal/signals"
)

// recommendedPassRate is the legacy bar below which pass-rate work is
// recommended.
const recommendedPassRate = 0.90

// buildRecommendations derives prioritized improvements from the composite
// breakdown. Impact = composite points currently left on the table by that
// metric, so the list is sortable by payoff (ported semantics).
func buildRecommendations(snap signals.Snapshot, comp Composite, mat Maturity) []Recommendation {
	var recs []Recommendation

	add := func(priority, description string, impact float64) {
		if impact <= 0 {
			return
		}
		recs = append(recs, Recommendation{Priority: priority, Description: description, Impact: round1(impact)})
	}

	metricGap := func(groupID, metricID string) (Metric, float64) {
		for _, g := range comp.Groups {
			if g.ID != groupID {
				continue
			}
			for _, m := range g.Metrics {
				if m.ID == metricID {
					return m, m.MaxPoints - m.Points
				}
			}
		}
		return Metric{}, 0
	}

	// Build/rung blockers first: they gate everything else. Rung blockers
	// carry zero composite impact (the rung is not part of the 0-100) —
	// renderers omit the points suffix for them.
	if !mat.BuildPassing {
		add("high", "Fix the build/test baseline: the newest cached unit phase is not passing (run `test-genie execute <scenario> --preset quick`)", phaseGapImpact(comp))
	}
	for _, d := range mat.Dimensions {
		if d.ErrorPlus > 0 {
			recs = append(recs, Recommendation{
				Priority:    "high",
				Description: fmt.Sprintf("Resolve %d error-level finding(s) in the %s dimension (blocks rung %s)", d.ErrorPlus, d.Dimension, firstNonEmpty(mat.WorkingRung, "R0")),
			})
		}
	}

	if m, gap := metricGap("quality", "requirement_pass_rate"); gap > 0 && rateFromObserved(snap.Requirements.Passing, snap.Requirements.Total) < recommendedPassRate {
		add("high", fmt.Sprintf("Raise the requirement pass rate (%s)", m.Observed), gap)
	}
	if m, gap := metricGap("quality", "target_pass_rate"); gap > 0 && rateFromObserved(snap.Requirements.TargetsPassing, snap.Requirements.TargetsTotal) < recommendedPassRate {
		add("high", fmt.Sprintf("Close failing operational targets (%s)", m.Observed), gap)
	}
	if m, gap := metricGap("quality", "phase_pass_rate"); gap > 0 {
		add("high", fmt.Sprintf("Fix failing test phases (%s)", m.Observed), gap)
	}

	if m, gap := metricGap("ui", "template_check"); gap > 0 {
		add("medium", fmt.Sprintf("Replace the template UI with a scenario-specific experience (%s)", m.Observed), gap)
	}
	if m, gap := metricGap("ui", "api_integration"); gap >= 3 {
		add("medium", fmt.Sprintf("Wire the UI to more API surface (%s)", m.Observed), gap)
	}
	if m, gap := metricGap("coverage", "validation_coverage"); gap > 0 {
		add("medium", fmt.Sprintf("Declare validations for unvalidated requirements (%s)", m.Observed), gap)
	}

	if m, gap := metricGap("coverage", "requirement_depth"); gap > 0 {
		add("low", fmt.Sprintf("Decompose flat requirements into sub-requirements (%s)", m.Observed), gap)
	}
	for _, id := range []string{"requirements_count", "targets_count", "phases_count"} {
		if m, gap := metricGap("quantity", id); gap > 0 && (m.Threshold == "below" || m.Threshold == "ok") {
			add("low", fmt.Sprintf("Grow %s beyond the %s band (currently %s)", m.Label, m.Threshold, m.Observed), gap)
		}
	}
	for _, id := range []string{"component_complexity", "routing", "code_volume"} {
		if m, gap := metricGap("ui", id); gap > 0 && m.Threshold == "below" {
			add("low", fmt.Sprintf("Expand the UI (%s: %s)", m.Label, m.Observed), gap)
		}
	}

	sort.SliceStable(recs, func(i, j int) bool {
		pi, pj := priorityRank(recs[i].Priority), priorityRank(recs[j].Priority)
		if pi != pj {
			return pi < pj
		}
		return recs[i].Impact > recs[j].Impact
	})
	return recs
}

// buildActionPlan groups the recommendations into ordered phases with
// estimated point totals and a projected score.
func buildActionPlan(comp Composite, recs []Recommendation) []ActionPhase {
	if len(recs) == 0 {
		return nil
	}

	byPriority := map[string][]Recommendation{}
	for _, r := range recs {
		byPriority[r.Priority] = append(byPriority[r.Priority], r)
	}

	titles := []struct{ priority, title string }{
		{"high", "Restore quality gates"},
		{"medium", "Deepen product surface"},
		{"low", "Round out coverage and scale"},
	}

	var phases []ActionPhase
	running := float64(comp.Score)
	for _, t := range titles {
		group := byPriority[t.priority]
		if len(group) == 0 {
			continue
		}
		phase := ActionPhase{Title: t.title}
		for _, r := range group {
			phase.Actions = append(phase.Actions, r.Description)
			phase.EstimatedPoints += r.Impact
		}
		phase.EstimatedPoints = round1(math.Min(phase.EstimatedPoints, 100-running))
		running = math.Min(100, running+phase.EstimatedPoints)
		phases = append(phases, phase)
	}
	return phases
}

// phaseGapImpact is the quality points currently lost to failing phases.
func phaseGapImpact(comp Composite) float64 {
	for _, g := range comp.Groups {
		if g.ID != "quality" {
			continue
		}
		for _, m := range g.Metrics {
			if m.ID == "phase_pass_rate" {
				return m.MaxPoints - m.Points
			}
		}
	}
	return 0
}

func rateFromObserved(passing, total int) float64 {
	if total <= 0 {
		// No data: treat as below the bar so the recommendation fires only
		// when there is a points gap to recover.
		return 0
	}
	return float64(passing) / float64(total)
}

func priorityRank(p string) int {
	switch p {
	case "high":
		return 0
	case "medium":
		return 1
	default:
		return 2
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
