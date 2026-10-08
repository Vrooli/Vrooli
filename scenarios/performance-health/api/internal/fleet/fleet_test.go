package fleet

import (
	"context"
	"errors"
	"testing"
)

type fakeGrader struct {
	byScenario map[string]ScenarioEntry
	errFor     map[string]bool
}

func (f fakeGrader) Grade(_ context.Context, scenario string) (ScenarioEntry, error) {
	if f.errFor[scenario] {
		return ScenarioEntry{}, errors.New("ungradable")
	}
	return f.byScenario[scenario], nil
}

type fakeEnumerator struct{ scenarios []string }

func (f fakeEnumerator) List(context.Context) ([]string, error) { return f.scenarios, nil }

// [REQ:PH-FLEET-001] Scan rolls up offender counts (no-budget, regressed) and a
// tier distribution across the requested scenarios.
func TestScanRollsUpOffenders(t *testing.T) {
	grader := fakeGrader{byScenario: map[string]ScenarioEntry{
		"a": {Scenario: "a", Tier: "1", HasBudget: true},
		"b": {Scenario: "b", Tier: "0", HasBudget: false, Regressed: true},
		"c": {Scenario: "c", Tier: "0", HasBudget: false},
	}}
	svc := NewService(grader, nil)
	res, err := svc.Scan(context.Background(), []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.ScenarioCount != 3 || res.NoBudgetCount != 2 || res.RegressedCount != 1 {
		t.Fatalf("unexpected rollup: %#v", res)
	}
	if len(res.TierDistribution) == 0 || res.TierDistribution[0].Tier != "0" {
		t.Fatalf("expected tier 0 most common, got %#v", res.TierDistribution)
	}
}

// [REQ:PH-FLEET-001] An ungradable scenario is recorded as an error, not a
// crash; the rest of the fleet still grades.
func TestScanRecordsErrors(t *testing.T) {
	grader := fakeGrader{
		byScenario: map[string]ScenarioEntry{"a": {Scenario: "a", Tier: "1", HasBudget: true}},
		errFor:     map[string]bool{"b": true},
	}
	svc := NewService(grader, nil)
	res, err := svc.Scan(context.Background(), []string{"a", "b"})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.ScenarioCount != 1 || len(res.Errors) != 1 {
		t.Fatalf("expected 1 graded + 1 error, got %#v", res)
	}
}

// [REQ:PH-FLEET-001] The deterministic offender queries select the right
// scenarios from a scan result: no-budget, regressed, and slowest builds.
func TestOffenderQueries(t *testing.T) {
	grader := fakeGrader{byScenario: map[string]ScenarioEntry{
		"a": {Scenario: "a", Tier: "1", HasBudget: true, GoBuildMs: 30000, UIBuildMs: 20000},
		"b": {Scenario: "b", Tier: "0", HasBudget: false, GoBuildMs: 90000, UIBuildMs: 60000, Regressed: true},
		"c": {Scenario: "c", Tier: "0", HasBudget: false}, // no measured build
	}}
	svc := NewService(grader, nil)
	res, err := svc.Scan(context.Background(), []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	noBudget := res.NoBudget()
	if len(noBudget) != 2 || noBudget[0].Scenario != "b" || noBudget[1].Scenario != "c" {
		t.Fatalf("unexpected no-budget offenders: %#v", noBudget)
	}

	regressed := res.Regressed()
	if len(regressed) != 1 || regressed[0].Scenario != "b" {
		t.Fatalf("unexpected regressed offenders: %#v", regressed)
	}

	// b (150s) is slowest, a (50s) next, c excluded (no measured build).
	slowest := res.SlowestBuilds(0)
	if len(slowest) != 2 || slowest[0].Scenario != "b" || slowest[1].Scenario != "a" {
		t.Fatalf("unexpected slowest-builds order: %#v", slowest)
	}
	if top := res.SlowestBuilds(1); len(top) != 1 || top[0].Scenario != "b" {
		t.Fatalf("expected top-1 slowest=b, got %#v", top)
	}
}

// Scan falls back to the enumerator when no scenarios are requested.
func TestScanUsesEnumerator(t *testing.T) {
	grader := fakeGrader{byScenario: map[string]ScenarioEntry{"x": {Scenario: "x", Tier: "none", HasBudget: true}}}
	svc := NewService(grader, fakeEnumerator{scenarios: []string{"x"}})
	res, err := svc.Scan(context.Background(), nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.ScenarioCount != 1 {
		t.Fatalf("expected enumerated scan of 1, got %d", res.ScenarioCount)
	}
}
