package selfhealth

import (
	"context"
	"testing"
	"time"

	"test-genie/internal/execution"
)

type fakeFleetSource struct {
	runs []execution.ScenarioRunRollup
	obs  []execution.PhaseObservation
}

func (f fakeFleetSource) AggregateScenarioRuns(context.Context, time.Time, int) ([]execution.ScenarioRunRollup, error) {
	return f.runs, nil
}

func (f fakeFleetSource) AggregatePhaseObservations(context.Context, time.Time, int) ([]execution.PhaseObservation, error) {
	return f.obs, nil
}

func TestFleetLedgerRanksMostErroredFirst(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	src := fakeFleetSource{
		runs: []execution.ScenarioRunRollup{
			{ScenarioName: "healthy", Runs: 4, Passed: 4, LastCompletedAt: now.Add(-24 * time.Hour), LastOutcome: "passed"},
			{ScenarioName: "flaky", Runs: 4, Passed: 1, LastCompletedAt: now.Add(-2 * time.Hour), LastOutcome: "failed"},
		},
		obs: []execution.PhaseObservation{
			{ScenarioName: "flaky", Status: "failed", FindingSource: "standards"},
			{ScenarioName: "flaky", Status: "failed", FindingSource: "standards"},
			{ScenarioName: "flaky", Status: "passed", FindingSource: "unit"},
			{ScenarioName: "healthy", Status: "passed", FindingSource: "unit"},
		},
	}
	b := NewFleetBuilder(src, 0)
	b.now = func() time.Time { return now }

	led, err := b.Build(context.Background(), nil)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if led.ScenariosTested != 2 || led.TotalRuns != 8 {
		t.Fatalf("ledger = %+v, want 2 tested / 8 runs", led)
	}
	if led.Scenarios[0].Scenario != "flaky" {
		t.Fatalf("most-errored first = %q, want flaky", led.Scenarios[0].Scenario)
	}
	flaky := led.Scenarios[0]
	if flaky.FailedRuns != 3 || flaky.Issues != 2 {
		t.Fatalf("flaky = %+v, want 3 failed runs / 2 issues", flaky)
	}
	if flaky.AgeDays <= 0 {
		t.Fatalf("flaky AgeDays = %v, want > 0 (staleness explicit)", flaky.AgeDays)
	}
	if led.TotalIssues != 2 {
		t.Fatalf("TotalIssues = %d, want 2", led.TotalIssues)
	}
	if len(led.TopFindingSources) == 0 || led.TopFindingSources[0].Source != "standards" {
		t.Fatalf("top finding source = %+v, want standards", led.TopFindingSources)
	}
	if len(led.Alerts) != 1 || led.Alerts[0].Code != "FLEET_SCENARIO_NOT_GREEN" || led.Alerts[0].NextAction == "" {
		t.Fatalf("fleet alert = %+v, want actionable failed-scenario alert", led.Alerts)
	}
	if !led.CapturedAt.Equal(now) {
		t.Fatalf("CapturedAt = %v, want as-of %v", led.CapturedAt, now)
	}
}

func TestFleetLedgerNeverTestedFromRoster(t *testing.T) {
	now := time.Now().UTC()
	src := fakeFleetSource{
		runs: []execution.ScenarioRunRollup{
			{ScenarioName: "alpha", Runs: 1, Passed: 1, LastCompletedAt: now, LastOutcome: "passed"},
		},
	}
	b := NewFleetBuilder(src, 0)
	b.now = func() time.Time { return now }

	led, err := b.Build(context.Background(), []string{"alpha", "beta", "gamma", "beta"})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if led.ScenariosTotal != 3 {
		t.Fatalf("ScenariosTotal = %d, want 3", led.ScenariosTotal)
	}
	want := []string{"beta", "gamma"}
	if len(led.NeverTestedInWindow) != 2 || led.NeverTestedInWindow[0] != want[0] || led.NeverTestedInWindow[1] != want[1] {
		t.Fatalf("NeverTestedInWindow = %v, want %v (deduped, sorted)", led.NeverTestedInWindow, want)
	}
	if len(led.Alerts) != 2 || led.Alerts[0].Code != "FLEET_COVERAGE_GAP" || led.Alerts[0].RollbackPath == "" {
		t.Fatalf("coverage alerts = %+v, want one actionable alert per gap", led.Alerts)
	}
}

func TestFleetLedgerNoRosterLeavesNeverTestedEmpty(t *testing.T) {
	now := time.Now().UTC()
	src := fakeFleetSource{runs: []execution.ScenarioRunRollup{{ScenarioName: "alpha", Runs: 1, Passed: 1, LastCompletedAt: now}}}
	b := NewFleetBuilder(src, 0)
	b.now = func() time.Time { return now }
	led, _ := b.Build(context.Background(), nil)
	if len(led.NeverTestedInWindow) != 0 || led.ScenariosTotal != 1 {
		t.Fatalf("no roster: never-tested=%v total=%d, want empty/1", led.NeverTestedInWindow, led.ScenariosTotal)
	}
}
