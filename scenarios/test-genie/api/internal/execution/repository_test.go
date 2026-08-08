package execution

import (
	"context"
	"testing"
	"time"

	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/storage/sqliteutil"
	"test-genie/internal/testsqlite"

	"github.com/google/uuid"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestSuiteExecutionRepositoryCreate(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()
	execID := uuid.New()
	record := &SuiteExecutionRecord{
		ID:                       execID,
		RunID:                    "20260711-000000-evidence",
		ScenarioName:             "demo",
		PresetUsed:               "quick",
		RequestedPreset:          "quick",
		RequestedPhases:          []string{"structure", "unit"},
		RequestedSkipPhases:      []string{"performance"},
		PlannedPhases:            []string{"structure", "unit"},
		PhaseSetDigest:           "phase-set:demo",
		DescriptorSnapshotDigest: "ds:demo",
		ConfigurationFingerprint: "execution-config:demo",
		FailFast:                 true,
		Success:                  true,
		Phases: []phases.ExecutionResult{
			{Name: "structure", Status: "passed", DurationSeconds: 1},
		},
		PreparationStages: []orchestrator.PreparationStage{{Name: "provider_readiness", Status: "completed", DurationMilliseconds: 1234}, {Name: "provider_check", Parent: "provider_readiness", Subject: "unit-health", Status: "ready", DurationMilliseconds: 45}},
		StartedAt:         now.Add(-time.Minute),
		CompletedAt:       now,
	}

	if err := repo.Create(context.Background(), record); err != nil {
		t.Fatalf("expected create to succeed: %v", err)
	}

	stored, err := repo.GetByID(context.Background(), execID)
	if err != nil {
		t.Fatalf("expected execution to be readable: %v", err)
	}
	if stored == nil || stored.ID != execID {
		t.Fatalf("unexpected execution: %#v", stored)
	}
	if stored.RunID != record.RunID {
		t.Fatalf("expected run ID to round-trip, got %q", stored.RunID)
	}
	if len(stored.PlannedPhases) != 2 || stored.PlannedPhases[1] != "unit" {
		t.Fatalf("expected planned phases to round-trip: %#v", stored)
	}
	if stored.TerminalOutcome != TerminalOutcomePassed {
		t.Fatalf("expected terminal_outcome derived as passed, got %q", stored.TerminalOutcome)
	}
	if len(stored.PreparationStages) != 2 || stored.PreparationStages[1].Subject != "unit-health" || stored.PreparationStages[0].DurationMilliseconds != 1234 {
		t.Fatalf("expected preparation stages to round-trip: %#v", stored.PreparationStages)
	}
	if stored.PhaseSetDigest != record.PhaseSetDigest || stored.DescriptorSnapshotDigest != record.DescriptorSnapshotDigest || stored.ConfigurationFingerprint != record.ConfigurationFingerprint {
		t.Fatalf("expected comparability metadata to round-trip: %#v", stored)
	}
}

func TestSuiteExecutionRepositoryDeleteByRunID(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()
	retainedExecutionID := uuid.New()
	for _, runID := range []string{"retained-run", "other-run"} {
		id := uuid.New()
		if runID == "retained-run" {
			id = retainedExecutionID
		}
		if err := repo.Create(context.Background(), &SuiteExecutionRecord{ID: id, RunID: runID, ScenarioName: "demo", Success: true, Phases: []phases.ExecutionResult{{Name: "unit", Status: "passed"}}, StartedAt: now, CompletedAt: now}); err != nil {
			t.Fatalf("seed %s: %v", runID, err)
		}
	}
	if err := repo.DeleteByRunID(context.Background(), "retained-run"); err != nil {
		t.Fatalf("DeleteByRunID: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM suite_executions WHERE run_id = ?`, "retained-run").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("deleted run still has %d rows", count)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM suite_execution_phases WHERE execution_id = ?`, retainedExecutionID.String()).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("deleted run retained %d compact phase rows", count)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM suite_execution_stages WHERE execution_id = ?`, retainedExecutionID.String()).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("deleted run retained %d compact stage rows", count)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM suite_executions WHERE run_id = ?`, "other-run").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("other run rows = %d, want 1", count)
	}
}

func TestSuiteExecutionRepositoryListPlanSamplesPreservesComparabilityKey(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()
	record := &SuiteExecutionRecord{
		ID: uuid.New(), ScenarioName: "demo", Success: false, TerminalOutcome: TerminalOutcomeTimeout,
		PhaseSetDigest: "phase-set:demo", DescriptorSnapshotDigest: "ds:demo", ConfigurationFingerprint: "execution-config:demo",
		Phases: []phases.ExecutionResult{}, StartedAt: now.Add(-7 * time.Minute), CompletedAt: now,
	}
	if err := repo.Create(context.Background(), record); err != nil {
		t.Fatalf("create: %v", err)
	}
	samples, err := repo.ListPlanSamples(context.Background(), "demo", now.Add(-time.Hour), 10)
	if err != nil {
		t.Fatalf("list plan samples: %v", err)
	}
	if len(samples) != 1 || samples[0].DurationSeconds != 420 || samples[0].TerminalOutcome != TerminalOutcomeTimeout.String() {
		t.Fatalf("unexpected plan sample: %#v", samples)
	}
}

func TestSuiteExecutionRepositoryListPlanSamplesAcceptsLegacyNullComparabilityMetadata(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()
	if _, err := db.Exec(`INSERT INTO suite_executions (id, scenario_name, success, started_at, completed_at) VALUES (?, ?, ?, ?, ?)`,
		uuid.NewString(), "demo", 1, sqliteutil.FormatTimestamp(now.Add(-time.Minute)), sqliteutil.FormatTimestamp(now)); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}
	samples, err := repo.ListPlanSamples(context.Background(), "demo", now.Add(-time.Hour), 10)
	if err != nil {
		t.Fatalf("legacy rows must remain readable: %v", err)
	}
	if len(samples) != 1 || samples[0].PhaseSetDigest != "" || samples[0].DurationSeconds != 60 {
		t.Fatalf("unexpected legacy sample: %#v", samples)
	}
}

func TestSuiteExecutionRepositoryListRecent(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()

	if _, err := db.Exec(`
INSERT INTO suite_executions (
	id, scenario_name, preset_used, requested_preset, requested_phases,
	requested_skip_phases, planned_phases, fail_fast, success, started_at, completed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
		"11111111-1111-1111-1111-111111111111",
		"demo",
		"quick",
		"quick",
		`["structure","unit"]`,
		`["performance"]`,
		`["structure","unit"]`,
		1,
		1,
		sqliteutil.FormatTimestamp(now.Add(-time.Minute)),
		sqliteutil.FormatTimestamp(now),
	); err != nil {
		t.Fatalf("seed execution: %v", err)
	}

	results, err := repo.ListRecent(context.Background(), "demo", 5, 0)
	if err != nil {
		t.Fatalf("expected list to succeed: %v", err)
	}
	if len(results) != 1 || results[0].ScenarioName != "demo" {
		t.Fatalf("unexpected list response: %#v", results)
	}
	if len(results[0].Phases) != 0 {
		t.Fatalf("list must not hydrate phase detail: %#v", results[0])
	}
	if results[0].RequestedPreset != "quick" || !results[0].FailFast {
		t.Fatalf("expected execution metadata to round-trip: %#v", results[0])
	}
	if len(results[0].PlannedPhases) != 2 || results[0].PlannedPhases[1] != "unit" {
		t.Fatalf("expected planned phases to round-trip: %#v", results[0])
	}
}

func TestSuiteExecutionRepositoryGetByID(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()
	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	if _, err := db.Exec(`
INSERT INTO suite_executions (
	id, scenario_name, preset_used, requested_preset, requested_phases,
	requested_skip_phases, planned_phases, fail_fast, success, started_at, completed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
		id.String(),
		"demo",
		nil,
		nil,
		`["structure"]`,
		`["performance"]`,
		`["structure","integration"]`,
		0,
		0,
		sqliteutil.FormatTimestamp(now.Add(-time.Minute)),
		sqliteutil.FormatTimestamp(now),
	); err != nil {
		t.Fatalf("seed execution: %v", err)
	}

	record, err := repo.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("expected get to succeed: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO suite_execution_phases (execution_id, ordinal, phase_name, status, duration_seconds) VALUES (?, 0, 'structure', 'failed', 2)`, id.String()); err != nil {
		t.Fatalf("seed compact phase history: %v", err)
	}
	if record == nil || record.ID != id {
		t.Fatalf("unexpected record: %#v", record)
	}
	if record.Success {
		t.Fatalf("expected success=false")
	}
	if len(record.RequestedPhases) != 1 || record.RequestedPhases[0] != "structure" {
		t.Fatalf("expected requested phases to round-trip: %#v", record)
	}
	if len(record.RequestedSkipPhases) != 1 || record.RequestedSkipPhases[0] != "performance" {
		t.Fatalf("expected requested skip phases to round-trip: %#v", record)
	}
}

func TestSuiteExecutionRepositoryAggregation(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()

	insert := func(scenario, outcome string, success bool, phaseResults []phases.ExecutionResult, age time.Duration) {
		t.Helper()
		if err := repo.Create(context.Background(), &SuiteExecutionRecord{
			ID: uuid.New(), ScenarioName: scenario, Success: success, TerminalOutcome: TerminalOutcome(outcome), Phases: phaseResults,
			StartedAt: now.Add(-age - time.Minute), CompletedAt: now.Add(-age),
		}); err != nil {
			t.Fatalf("seed execution: %v", err)
		}
	}

	// A completed run with a passing + failing phase (the failing one carries metrics).
	insert("demo", "failed", false, []phases.ExecutionResult{
		{Name: "proto", Status: "passed", DurationSeconds: 12},
		{Name: "unit", Status: "failed", DurationSeconds: 7, Metrics: &commonv1.ExecutionMetrics{}},
	}, time.Hour)
	// A passing run.
	insert("demo", "passed", true, []phases.ExecutionResult{{Name: "proto", Status: "passed", DurationSeconds: 10}}, 2*time.Hour)
	// A catastrophic run has no phase rows but remains in the denominator.
	insert("demo", "errored", false, nil, 3*time.Hour)
	// An out-of-window run that must be excluded.
	insert("demo", "passed", true, []phases.ExecutionResult{{Name: "proto", Status: "passed", DurationSeconds: 99}}, 90*24*time.Hour)

	since := now.Add(-30 * 24 * time.Hour)

	outcomes, err := repo.CountRunOutcomes(context.Background(), since, 0)
	if err != nil {
		t.Fatalf("CountRunOutcomes: %v", err)
	}
	got := map[string]int{}
	for _, o := range outcomes {
		got[o.TerminalOutcome] = o.Count
	}
	if got["passed"] != 1 || got["failed"] != 1 || got["errored"] != 1 {
		t.Fatalf("unexpected outcome histogram: %#v", got)
	}

	observations, err := repo.AggregatePhaseObservations(context.Background(), since, 0)
	if err != nil {
		t.Fatalf("AggregatePhaseObservations: %v", err)
	}
	// 2 phases (failed run) + 1 phase (passed run) = 3; catastrophic run yields 0.
	if len(observations) != 3 {
		t.Fatalf("expected 3 phase observations, got %d: %#v", len(observations), observations)
	}
	var unitMetrics bool
	for _, o := range observations {
		if o.PhaseName == "unit" {
			unitMetrics = o.MetricsPresent
			if o.Status != "failed" || o.DurationSeconds != 7 {
				t.Fatalf("unexpected unit observation: %#v", o)
			}
		}
	}
	if !unitMetrics {
		t.Fatalf("expected unit observation to report metrics present")
	}
}

func TestSuiteExecutionRepositoryListPhaseSamples(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()

	if err := repo.Create(context.Background(), &SuiteExecutionRecord{
		ID: uuid.New(), ScenarioName: "demo", RequestedPhases: []string{"unit"}, PlannedPhases: []string{"unit"}, Success: true,
		Phases:    []phases.ExecutionResult{{Name: "unit", Status: "passed", DurationSeconds: 42}, {Name: "integration", Status: "failed", DurationSeconds: 12}},
		StartedAt: now.Add(-time.Minute), CompletedAt: now,
	}); err != nil {
		t.Fatalf("seed execution: %v", err)
	}

	samples, err := repo.ListPhaseSamples(context.Background(), "demo", []string{"unit", "unit"}, now.Add(-time.Hour), 100)
	if err != nil {
		t.Fatalf("expected list phase samples to succeed: %v", err)
	}
	if len(samples) != 1 {
		t.Fatalf("expected one sample, got %#v", samples)
	}
	if samples[0].ScenarioName != "demo" || samples[0].PhaseName != "unit" || samples[0].DurationSeconds != 42 {
		t.Fatalf("unexpected sample: %#v", samples[0])
	}
}

func TestSuiteExecutionRepositoryCostReportIncludesPredictionError(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()
	id := uuid.New()
	if err := repo.Create(context.Background(), &SuiteExecutionRecord{
		ID: id, ScenarioName: "demo", Success: true,
		Phases:    []phases.ExecutionResult{{Name: "unit", Status: "passed", DurationMilliseconds: 10000, PredictedDurationMilliseconds: 8000}},
		StartedAt: now.Add(-time.Second), CompletedAt: now,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := db.Exec(`UPDATE suite_execution_phases SET wall_clock_ms = 10000, cpu_user_ms = 100, peak_rss_bytes = 20, cpu_reliability = 'RELIABILITY_RELIABLE', memory_reliability = 'RELIABILITY_RELIABLE' WHERE execution_id = ?`, id.String()); err != nil {
		t.Fatalf("seed reliable metrics: %v", err)
	}
	report, err := repo.CostReport(context.Background(), "demo", now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("cost report: %v", err)
	}
	if len(report) != 1 || report[0].PredictionSampleCount != 1 || report[0].PredictionMeanAbsoluteErrorMs != 2000 || report[0].PredictionErrorTotalMs != 2000 {
		t.Fatalf("unexpected prediction report: %#v", report)
	}
}

func TestSuiteExecutionRepositoryCostReportSeparatesPassingAndFailingDurations(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()
	insert := func(status string, duration int64, offset time.Duration) {
		t.Helper()
		id := uuid.New()
		if err := repo.Create(context.Background(), &SuiteExecutionRecord{
			ID: id, ScenarioName: "demo", Success: status == "passed",
			Phases:    []phases.ExecutionResult{{Name: "unit", Status: status, DurationMilliseconds: duration}},
			StartedAt: now.Add(-offset - time.Second), CompletedAt: now.Add(-offset),
		}); err != nil {
			t.Fatalf("create %s: %v", status, err)
		}
		if _, err := db.Exec(`UPDATE suite_execution_phases SET wall_clock_ms = ?, cpu_reliability = 'RELIABILITY_RELIABLE', memory_reliability = 'RELIABILITY_RELIABLE' WHERE execution_id = ?`, duration, id.String()); err != nil {
			t.Fatalf("seed %s metrics: %v", status, err)
		}
	}
	insert("passed", 1000, time.Minute)
	insert("passed", 3000, 2*time.Minute)
	insert("failed", 9000, 3*time.Minute)

	report, err := repo.CostReport(context.Background(), "demo", now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("cost report: %v", err)
	}
	if len(report) != 1 {
		t.Fatalf("expected one phase summary, got %#v", report)
	}
	summary := report[0]
	if summary.PassingSampleCount != 2 || summary.FailingSampleCount != 1 {
		t.Fatalf("unexpected status counts: %#v", summary)
	}
	if summary.PassingMedianWallClockMs != 1000 || summary.PassingP90WallClockMs != 3000 || summary.FailingMedianWallClockMs != 9000 || summary.FailingP90WallClockMs != 9000 {
		t.Fatalf("unexpected status distributions: %#v", summary)
	}
}

func TestSuiteExecutionRepositoryCostReportIncludesCacheEconomics(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()
	insert := func(duration int64, hit, audit bool, offset time.Duration) {
		t.Helper()
		id := uuid.New()
		phase := phases.ExecutionResult{Name: "unit", Status: "passed", DurationMilliseconds: duration, CacheHit: hit, CacheAudit: audit}
		if err := repo.Create(context.Background(), &SuiteExecutionRecord{ID: id, ScenarioName: "demo", Success: true, Phases: []phases.ExecutionResult{phase}, StartedAt: now.Add(-offset - time.Second), CompletedAt: now.Add(-offset)}); err != nil {
			t.Fatalf("create: %v", err)
		}
		if _, err := db.Exec(`UPDATE suite_execution_phases SET wall_clock_ms = ?, cpu_reliability = 'RELIABILITY_RELIABLE', memory_reliability = 'RELIABILITY_RELIABLE' WHERE execution_id = ?`, duration, id.String()); err != nil {
			t.Fatalf("seed metrics: %v", err)
		}
	}
	insert(1000, false, false, time.Minute)
	insert(1000, true, false, 2*time.Minute)
	insert(1200, false, true, 3*time.Minute)

	report, err := repo.CostReport(context.Background(), "demo", now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("cost report: %v", err)
	}
	if len(report) != 1 {
		t.Fatalf("expected one phase summary, got %#v", report)
	}
	summary := report[0]
	if summary.CacheHitCount != 1 || summary.ExecutedSampleCount != 2 || summary.CacheAuditCount != 1 || summary.CacheHitRatePercent < 33.33 || summary.CacheHitRatePercent > 33.34 {
		t.Fatalf("unexpected cache counts: %#v", summary)
	}
	if summary.EstimatedGrossSavedWallClockMs != 1000 || summary.CacheAuditWallClockMs != 1200 || summary.EstimatedNetSavedWallClockMs != -200 {
		t.Fatalf("net saving did not account for audit cost: %#v", summary)
	}
}
