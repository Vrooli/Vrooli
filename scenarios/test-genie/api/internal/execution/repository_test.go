package execution

import (
	"context"
	"testing"
	"time"

	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/storage/sqliteutil"
	"test-genie/internal/testsqlite"

	"github.com/google/uuid"
)

func TestSuiteExecutionRepositoryCreate(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()
	execID := uuid.New()
	record := &SuiteExecutionRecord{
		ID:                  execID,
		RunID:               "20260711-000000-evidence",
		ScenarioName:        "demo",
		PresetUsed:          "quick",
		RequestedPreset:     "quick",
		RequestedPhases:     []string{"structure", "unit"},
		RequestedSkipPhases: []string{"performance"},
		PlannedPhases:       []string{"structure", "unit"},
		FailFast:            true,
		Success:             true,
		Phases: []phases.ExecutionResult{
			{Name: "structure", Status: "passed", DurationSeconds: 1},
		},
		StartedAt:   now.Add(-time.Minute),
		CompletedAt: now,
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
}

func TestSuiteExecutionRepositoryListRecent(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()

	if _, err := db.Exec(`
INSERT INTO suite_executions (
	id, scenario_name, preset_used, requested_preset, requested_phases,
	requested_skip_phases, planned_phases, fail_fast, success, phases, started_at, completed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		`[{"name":"structure","status":"passed","durationSeconds":1}]`,
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
	if len(results[0].Phases) != 1 {
		t.Fatalf("expected phases to be unmarshaled: %#v", results[0])
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
	requested_skip_phases, planned_phases, fail_fast, success, phases, started_at, completed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		`[{"name":"structure","status":"failed","durationSeconds":2}]`,
		sqliteutil.FormatTimestamp(now.Add(-time.Minute)),
		sqliteutil.FormatTimestamp(now),
	); err != nil {
		t.Fatalf("seed execution: %v", err)
	}

	record, err := repo.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("expected get to succeed: %v", err)
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

	insert := func(scenario, outcome string, success int, phasesJSON string, age time.Duration) {
		t.Helper()
		if _, err := db.Exec(`
INSERT INTO suite_executions (
	id, scenario_name, requested_phases, requested_skip_phases, planned_phases,
	fail_fast, success, terminal_outcome, phases, started_at, completed_at
) VALUES (?, ?, '[]', '[]', '[]', 0, ?, ?, ?, ?, ?)`,
			uuid.NewString(), scenario, success, outcome, phasesJSON,
			sqliteutil.FormatTimestamp(now.Add(-age-time.Minute)),
			sqliteutil.FormatTimestamp(now.Add(-age)),
		); err != nil {
			t.Fatalf("seed execution: %v", err)
		}
	}

	// A completed run with a passing + failing phase (the failing one carries metrics).
	insert("demo", "failed", 0, `[
		{"name":"proto","status":"passed","durationSeconds":12},
		{"name":"unit","status":"failed","durationSeconds":7,"metrics":{"wall_clock_ms":7000}}
	]`, time.Hour)
	// A passing run.
	insert("demo", "passed", 1, `[
		{"name":"proto","status":"passed","durationSeconds":10}
	]`, 2*time.Hour)
	// A catastrophic run: no phases blob, errored outcome (the B4a case).
	insert("demo", "errored", 0, `[]`, 3*time.Hour)
	// An out-of-window run that must be excluded.
	insert("demo", "passed", 1, `[{"name":"proto","status":"passed","durationSeconds":99}]`, 90*24*time.Hour)

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

	if _, err := db.Exec(`
INSERT INTO suite_executions (
	id, scenario_name, requested_phases, requested_skip_phases, planned_phases, fail_fast, success, phases, started_at, completed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
		uuid.NewString(),
		"demo",
		`["unit"]`,
		`[]`,
		`["unit"]`,
		0,
		1,
		`[
			{"name":"unit","status":"passed","durationSeconds":42},
			{"name":"integration","status":"failed","durationSeconds":12}
		]`,
		sqliteutil.FormatTimestamp(now.Add(-time.Minute)),
		sqliteutil.FormatTimestamp(now),
	); err != nil {
		t.Fatalf("seed execution: %v", err)
	}

	samples, err := repo.ListPhaseSamples(context.Background(), []string{"unit", "unit"}, now.Add(-time.Hour), 100)
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
