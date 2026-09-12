package fleet

import (
	"context"
	"errors"

	internalbudgets "performance-health/internal/budgets"
	"performance-health/internal/fleet"
	"performance-health/internal/readiness"
	"performance-health/internal/trend"
)

// budgetReader is the slice of the budget store the grader needs.
type budgetReader interface {
	Get(ctx context.Context, scenario string) (internalbudgets.Budget, bool, error)
}

// trendReader is the slice of the trend store the grader needs.
type trendReader interface {
	Latest(ctx context.Context, scenario string) (trend.Sample, bool, error)
}

// tierer decides a scenario's reachable capture tier. The readiness engine
// satisfies it; it is optional (a nil tierer yields tier "unknown").
type tierer interface {
	Validate(ctx context.Context, scenario, path string) (readiness.Result, error)
}

// grader computes a scenario's performance posture deterministically from the
// declared budget and the newest persisted trend sample. A scenario is
// "regressed" when its latest sample breaches its declared budget. NO AI, NO
// semantic ranking — exact/structured grading only.
type grader struct {
	budgets budgetReader
	trend   trendReader
	tierer  tierer
}

func newGrader(budgets budgetReader, trendReader trendReader, t tierer) *grader {
	return &grader{budgets: budgets, trend: trendReader, tierer: t}
}

var _ fleet.Grader = (*grader)(nil)

// Grade builds one scenario's rollup. Store-level failures bubble up so the
// fleet service records the scenario as an error and continues over the rest.
func (g *grader) Grade(ctx context.Context, scenario string) (fleet.ScenarioEntry, error) {
	if g == nil {
		return fleet.ScenarioEntry{}, errors.New("fleet: nil grader")
	}
	entry := fleet.ScenarioEntry{Scenario: scenario, Tier: "unknown"}

	budget, declared, err := g.budgets.Get(ctx, scenario)
	if err != nil {
		return fleet.ScenarioEntry{}, err
	}
	entry.HasBudget = declared && budget.IsSet()

	// Read the newest sample fully before any further query (SQLite pool=1
	// nested-query deadlock guard).
	sample, found, err := g.trend.Latest(ctx, scenario)
	if err != nil {
		return fleet.ScenarioEntry{}, err
	}
	if found {
		entry.GoBuildMs = sample.GoBuildMs
		entry.UIBuildMs = sample.UIBuildMs
		if entry.HasBudget {
			violations := internalbudgets.Evaluate(budget, internalbudgets.SampleToMeasurement(sample))
			if len(violations) > 0 {
				entry.Regressed = true
				entry.DegradedReason = "latest sample breaches budget: " + violations[0].Axis
			}
		}
	}

	if g.tierer != nil {
		if res, terr := g.tierer.Validate(ctx, scenario, ""); terr == nil {
			entry.Tier = res.Tier.String()
		}
	}
	return entry, nil
}
