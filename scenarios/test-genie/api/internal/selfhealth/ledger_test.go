package selfhealth

import (
	"context"
	"testing"
	"time"

	"test-genie/internal/execution"
)

type fakeSource struct {
	observations []execution.PhaseObservation
	outcomes     []execution.RunOutcomeCount
}

func (f fakeSource) AggregatePhaseObservations(_ context.Context, _ time.Time, _ int) ([]execution.PhaseObservation, error) {
	return f.observations, nil
}

func (f fakeSource) CountRunOutcomes(_ context.Context, _ time.Time, _ int) ([]execution.RunOutcomeCount, error) {
	return f.outcomes, nil
}

func obs(scenario, outcome, phase, status string, dur int, metrics bool) execution.PhaseObservation {
	return execution.PhaseObservation{
		ScenarioName:    scenario,
		TerminalOutcome: outcome,
		PhaseName:       phase,
		Status:          status,
		DurationSeconds: dur,
		MetricsPresent:  metrics,
	}
}

func phaseByName(t *testing.T, l *Ledger, name string) PhaseReliability {
	t.Helper()
	for _, p := range l.Phases {
		if p.Phase == name {
			return p
		}
	}
	t.Fatalf("phase %q not found in ledger", name)
	return PhaseReliability{}
}

// TestBuildLedgerExcludesLegacyPhases asserts the rollup stays ⊆ catalog: phases
// whose name is absent from phaseMeta (legacy pseudo-phases like coverage/lint
// from historical runs) are dropped so they no longer masquerade as live phases.
func TestBuildLedgerExcludesLegacyPhases(t *testing.T) {
	src := fakeSource{
		observations: []execution.PhaseObservation{
			obs("a", "passed", "proto", "passed", 10, true),
			obs("a", "passed", "coverage", "passed", 5, false), // legacy
			obs("b", "failed", "lint", "failed", 7, false),     // legacy
		},
		outcomes: []execution.RunOutcomeCount{{TerminalOutcome: "passed", Count: 1}},
	}
	meta := map[string]PhaseMeta{"proto": {Provider: "proto-health", Delegated: true}}

	l, err := NewBuilder(src, 0).Build(context.Background(), meta)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(l.Phases) != 1 || l.Phases[0].Phase != "proto" {
		t.Fatalf("expected only the catalog phase 'proto', got %+v", l.Phases)
	}
	for _, p := range l.Phases {
		if _, ok := meta[p.Phase]; !ok {
			t.Errorf("ledger surfaced non-catalog phase %q", p.Phase)
		}
	}
}

func TestBuildLedgerRollups(t *testing.T) {
	src := fakeSource{
		observations: []execution.PhaseObservation{
			// proto phase: 3 passed, 1 failed, 1 skipped across scenarios.
			obs("a", "passed", "proto", "passed", 10, true),
			obs("a", "passed", "proto", "passed", 20, true),
			obs("b", "failed", "proto", "failed", 30, true),
			obs("b", "passed", "proto", "passed", 40, false), // metrics absent
			func() execution.PhaseObservation {
				o := obs("c", "passed", "proto", "skipped", 0, false)
				o.RunnabilityVerdict = "skip"
				o.RunnabilityReason = "no ui surface"
				return o
			}(),
			// unit phase: native (no provider), one degraded run.
			func() execution.PhaseObservation {
				o := obs("a", "passed", "unit", "passed", 5, false)
				o.RunnabilityVerdict = "run_degraded"
				o.Classification = "flaky"
				return o
			}(),
		},
		outcomes: []execution.RunOutcomeCount{
			{TerminalOutcome: "passed", Count: 7},
			{TerminalOutcome: "failed", Count: 2},
			{TerminalOutcome: "errored", Count: 1},
		},
	}

	meta := map[string]PhaseMeta{
		"proto": {Provider: "proto-health", FindingSource: "proto", Delegated: true},
		"unit":  {FindingSource: "coverage"},
	}

	b := NewBuilder(src, 0)
	l, err := b.Build(context.Background(), meta)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if l.WindowDays != 30 {
		t.Fatalf("windowDays = %d, want 30", l.WindowDays)
	}
	// Run availability: (7+2)/(7+2+1) = 9/10.
	if l.RunCount != 10 {
		t.Fatalf("runCount = %d, want 10", l.RunCount)
	}
	if l.Availability < 0.89 || l.Availability > 0.91 {
		t.Fatalf("availability = %v, want ~0.9", l.Availability)
	}

	proto := phaseByName(t, l, "proto")
	if proto.Provider != "proto-health" {
		t.Fatalf("proto provider = %q", proto.Provider)
	}
	if proto.TotalObservations != 5 || proto.Passed != 3 || proto.Failed != 1 || proto.Skipped != 1 {
		t.Fatalf("proto counts unexpected: %+v", proto)
	}
	// Availability = executed/total = 4/5.
	if proto.Availability < 0.79 || proto.Availability > 0.81 {
		t.Fatalf("proto availability = %v, want 0.8", proto.Availability)
	}
	// FailureRate = failed/executed = 1/4.
	if proto.FailureRate < 0.24 || proto.FailureRate > 0.26 {
		t.Fatalf("proto failureRate = %v, want 0.25", proto.FailureRate)
	}
	if proto.MetricsAdopted != 3 {
		t.Fatalf("proto metricsAdopted = %d, want 3", proto.MetricsAdopted)
	}
	// Duration over executed (10,20,30,40) → min 10, max 40, avg 25.
	if proto.Duration.Samples != 4 || proto.Duration.Min != 10 || proto.Duration.Max != 40 || proto.Duration.Avg != 25 {
		t.Fatalf("proto duration unexpected: %+v", proto.Duration)
	}
	if len(proto.SkipReasons) != 1 || proto.SkipReasons[0].Label != "no ui surface" {
		t.Fatalf("proto skipReasons unexpected: %+v", proto.SkipReasons)
	}

	unit := phaseByName(t, l, "unit")
	if unit.Provider != "" {
		t.Fatalf("unit should be native, got provider %q", unit.Provider)
	}
	if unit.Degraded != 1 {
		t.Fatalf("unit degraded = %d, want 1", unit.Degraded)
	}
	if len(unit.Classifications) != 1 || unit.Classifications[0].Label != "flaky" {
		t.Fatalf("unit classifications unexpected: %+v", unit.Classifications)
	}

	// Provider rollup: only proto-health (unit is native).
	if len(l.Providers) != 1 || l.Providers[0].Provider != "proto-health" {
		t.Fatalf("providers unexpected: %+v", l.Providers)
	}
	if l.Providers[0].TotalObservations != 5 {
		t.Fatalf("provider observations = %d, want 5", l.Providers[0].TotalObservations)
	}
}

func TestBuildLedgerEmptyAndMetricsAbsent(t *testing.T) {
	// No metrics anywhere, no outcomes — must not panic and must yield zeroed
	// availability rather than NaN.
	src := fakeSource{
		observations: []execution.PhaseObservation{
			obs("a", "passed", "structure", "passed", 0, false),
		},
	}
	l, err := NewBuilder(src, 0).Build(context.Background(), map[string]PhaseMeta{"structure": {}})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if l.RunCount != 0 || l.Availability != 0 {
		t.Fatalf("empty outcomes should yield zero availability, got count=%d avail=%v", l.RunCount, l.Availability)
	}
	s := phaseByName(t, l, "structure")
	if s.MetricsAdopted != 0 {
		t.Fatalf("metricsAdopted should be 0, got %d", s.MetricsAdopted)
	}
	// Duration with no positive samples degrades to zero stats.
	if s.Duration.Samples != 0 {
		t.Fatalf("duration samples = %d, want 0", s.Duration.Samples)
	}
}

func TestSecurityFrictionTracksRecurrenceAndTimeToGreen(t *testing.T) {
	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	failed := obs("demo", "failed", "security", "failed", 8, false)
	failed.CompletedAt = base
	recur := obs("demo", "failed", "security", "failed", 9, false)
	recur.CompletedAt = base.Add(2 * time.Minute)
	passed := obs("demo", "passed", "security", "passed", 7, true)
	passed.CompletedAt = base.Add(5 * time.Minute)

	l, err := NewBuilder(fakeSource{observations: []execution.PhaseObservation{failed, recur, passed}}, 0).
		Build(context.Background(), map[string]PhaseMeta{"security": {FindingSource: "security", Delegated: true}})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	security := phaseByName(t, l, "security")
	friction := security.SecurityFriction
	if friction.FailedAttempts != 2 || friction.RecurringFailures != 1 || friction.GreenTransitions != 1 {
		t.Fatalf("friction counts = %+v", friction)
	}
	if friction.TimeToGreenSamples != 1 || friction.TimeToGreen.P50 != 300 || friction.TimeToGreen.P95 != 300 {
		t.Fatalf("time to green = %+v", friction.TimeToGreen)
	}

	if got := phaseByName(t, l, "security").SecurityFriction; got.TimeToGreenSamples != 1 {
		t.Fatalf("security friction should remain attached to security phase: %+v", got)
	}
}

func TestWorstScenariosRanking(t *testing.T) {
	var observations []execution.PhaseObservation
	// scenario "bad": 3 executed, 2 failed → 0.667
	observations = append(observations,
		obs("bad", "failed", "proto", "failed", 1, false),
		obs("bad", "failed", "proto", "failed", 1, false),
		obs("bad", "passed", "proto", "passed", 1, false),
	)
	// scenario "ok": 3 executed, 1 failed → 0.333
	observations = append(observations,
		obs("ok", "failed", "proto", "failed", 1, false),
		obs("ok", "passed", "proto", "passed", 1, false),
		obs("ok", "passed", "proto", "passed", 1, false),
	)
	// scenario "rare": only 1 executed → excluded (below minRunsForWorstRanking)
	observations = append(observations, obs("rare", "failed", "proto", "failed", 1, false))

	l, err := NewBuilder(fakeSource{observations: observations}, 0).Build(context.Background(), map[string]PhaseMeta{"proto": {Provider: "proto-health", Delegated: true}})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	proto := phaseByName(t, l, "proto")
	if len(proto.WorstScenarios) != 2 {
		t.Fatalf("expected 2 ranked scenarios, got %+v", proto.WorstScenarios)
	}
	if proto.WorstScenarios[0].Scenario != "bad" {
		t.Fatalf("worst scenario should be 'bad', got %q", proto.WorstScenarios[0].Scenario)
	}
}
