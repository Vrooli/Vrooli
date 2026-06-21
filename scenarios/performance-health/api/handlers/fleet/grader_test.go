package fleet

import (
	"context"
	"testing"

	internalbudgets "performance-health/internal/budgets"
	"performance-health/internal/trend"
)

type fakeBudgets struct {
	byScenario map[string]internalbudgets.Budget
}

func (f fakeBudgets) Get(_ context.Context, scenario string) (internalbudgets.Budget, bool, error) {
	b, ok := f.byScenario[scenario]
	if !ok {
		return internalbudgets.Budget{Scenario: scenario}, false, nil
	}
	return b, true, nil
}

type fakeTrend struct {
	byScenario map[string]trend.Sample
}

func (f fakeTrend) Latest(_ context.Context, scenario string) (trend.Sample, bool, error) {
	s, ok := f.byScenario[scenario]
	return s, ok, nil
}

// [REQ:PH-FLEET-001] The grader marks a scenario with no declared budget as
// no-budget; build times come from the latest persisted sample.
func TestGraderNoBudget(t *testing.T) {
	g := newGrader(fakeBudgets{}, fakeTrend{byScenario: map[string]trend.Sample{
		"demo": {Scenario: "demo", GoBuildMs: 40000, UIBuildMs: 30000},
	}}, nil)
	entry, err := g.Grade(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Grade: %v", err)
	}
	if entry.HasBudget {
		t.Fatal("expected no-budget scenario")
	}
	if entry.GoBuildMs != 40000 || entry.UIBuildMs != 30000 {
		t.Fatalf("expected build times from latest sample, got %#v", entry)
	}
	if entry.Regressed {
		t.Fatal("a no-budget scenario can never be regressed-vs-budget")
	}
}

// [REQ:PH-FLEET-001] The grader marks a scenario whose latest sample breaches
// its declared budget as regressed, with the breaching axis in the reason.
func TestGraderRegressedAgainstBudget(t *testing.T) {
	g := newGrader(
		fakeBudgets{byScenario: map[string]internalbudgets.Budget{
			"demo": {Scenario: "demo", GoBuildMaxMs: 50000},
		}},
		fakeTrend{byScenario: map[string]trend.Sample{
			"demo": {Scenario: "demo", GoBuildMs: 90000},
		}},
		nil,
	)
	entry, err := g.Grade(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Grade: %v", err)
	}
	if !entry.HasBudget {
		t.Fatal("expected declared budget")
	}
	if !entry.Regressed {
		t.Fatal("expected regression: latest sample breaches the go-build budget")
	}
	if entry.DegradedReason == "" {
		t.Fatal("expected a degraded reason naming the breaching axis")
	}
}

// [REQ:PH-FLEET-001] A within-budget latest sample is not a regression.
func TestGraderWithinBudget(t *testing.T) {
	g := newGrader(
		fakeBudgets{byScenario: map[string]internalbudgets.Budget{
			"demo": {Scenario: "demo", GoBuildMaxMs: 100000},
		}},
		fakeTrend{byScenario: map[string]trend.Sample{
			"demo": {Scenario: "demo", GoBuildMs: 40000},
		}},
		nil,
	)
	entry, err := g.Grade(context.Background(), "demo")
	if err != nil {
		t.Fatalf("Grade: %v", err)
	}
	if entry.Regressed {
		t.Fatalf("within-budget sample must not be regressed: %#v", entry)
	}
}
