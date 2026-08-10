package execution

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"test-genie/internal/testsqlite"

	"github.com/google/uuid"
)

// insertDurationSample writes one terminal phase row with an explicit measured
// duration and reliability, so a case can describe the exact history shape the
// deadline guard reads.
func insertDurationSample(t *testing.T, db *sql.DB, scenario, phase, status string, durationMs int64, reliability *string) {
	t.Helper()
	ctx := context.Background()
	execID := uuid.New().String()
	stamp := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.ExecContext(ctx, `
INSERT INTO suite_executions (id, run_id, scenario_name, success, started_at, completed_at)
VALUES (?, ?, ?, 1, ?, ?)`, execID, "run-"+execID[:8], scenario, stamp, stamp); err != nil {
		t.Fatalf("insert execution: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO suite_execution_phases (execution_id, ordinal, phase_name, status, duration_seconds, duration_ms, wall_clock_ms, cpu_reliability, memory_reliability)
VALUES (?, 0, ?, ?, 1, ?, ?, ?, ?)`, execID, phase, status, durationMs, durationMs, reliability, reliability); err != nil {
		t.Fatalf("insert phase: %v", err)
	}
}

// The deadline guard must see how long a phase actually runs, including the
// runs where it fails. Failing phases are the slow ones — measured 2026-08-08 at
// 2.2x the passing average — so an estimate drawn only from passing samples
// would understate exactly the risk the guard exists to catch.
func TestPhaseDurationEstimateIncludesFailingSamples(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)

	for i := 0; i < 9; i++ {
		insertDurationSample(t, db, "demo", "security", "passed", 10_000, reliable())
	}
	insertDurationSample(t, db, "demo", "security", "failed", 90_000, reliable())

	estimate, ok := repo.PhaseDurationEstimate(context.Background(), "demo", "security")
	if !ok {
		t.Fatal("no estimate from ten samples")
	}
	if estimate != 90_000 {
		t.Fatalf("estimate = %d ms, want the failing 90000 ms sample to be represented", estimate)
	}
}

// The statistic must be tail-aware. A phase that is fast nine times in ten and
// slow on the tenth is exactly what a timeout catches, so a middle-of-the-
// distribution estimate would wave it into a batch and let contention turn the
// slow run into a false timeout.
func TestPhaseDurationEstimateRepresentsTheSlowTail(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)

	for i := 0; i < 19; i++ {
		insertDurationSample(t, db, "demo", "unit", "passed", 5_000, reliable())
	}
	insertDurationSample(t, db, "demo", "unit", "failed", 140_000, reliable())

	estimate, ok := repo.PhaseDurationEstimate(context.Background(), "demo", "unit")
	if !ok || estimate != 140_000 {
		t.Fatalf("estimate = %d ms ok=%t, want the slow tail sample", estimate, ok)
	}
}

// Duration is timed by Test Genie around the provider call. Reliability
// qualifies the CPU and RSS readings sampled from process-wide rusage, not the
// clock, so a best-effort row is still a usable duration sample. Excluding them
// would leave the guard blind for every phase whose provider runs concurrent
// collectors.
func TestPhaseDurationEstimateUsesBestEffortSamples(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)

	insertDurationSample(t, db, "demo", "experience", "passed", 4_000, bestEffort())

	estimate, ok := repo.PhaseDurationEstimate(context.Background(), "demo", "experience")
	if !ok || estimate != 4_000 {
		t.Fatalf("estimate = %d ms ok=%t, want 4000 ms from a best-effort sample", estimate, ok)
	}
}

// A cache hit records the duration of serving a stored verdict, not of doing
// the work. Blending those into the estimate would make a phase look cheap
// enough to batch on the strength of runs that never executed it.
func TestPhaseDurationEstimateExcludesCacheHits(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)

	insertDurationSample(t, db, "demo", "structure", "passed", 30_000, reliable())
	insertDurationSample(t, db, "demo", "structure", "passed", 5, reliable())
	if _, err := db.Exec(`UPDATE suite_execution_phases SET cache_hit = 1 WHERE duration_ms = 5`); err != nil {
		t.Fatalf("mark cache hit: %v", err)
	}

	estimate, ok := repo.PhaseDurationEstimate(context.Background(), "demo", "structure")
	if !ok || estimate != 30_000 {
		t.Fatalf("estimate = %d ms ok=%t, want the executed 30000 ms sample only", estimate, ok)
	}
}

// The same phase costs very different amounts on different scenarios —
// `security` measured a 54 s median on prompt-manager against 137 s on
// browser-automation-studio — so scenario history wins whenever there is
// enough of it.
func TestPhaseDurationEstimatePrefersScenarioHistory(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)

	for i := 0; i < 5; i++ {
		insertDurationSample(t, db, "demo", "security", "passed", 10_000, reliable())
	}
	for i := 0; i < 20; i++ {
		insertDurationSample(t, db, "other", "security", "passed", 200_000, reliable())
	}

	estimate, ok := repo.PhaseDurationEstimate(context.Background(), "demo", "security")
	if !ok || estimate != 10_000 {
		t.Fatalf("estimate = %d ms ok=%t, want the scenario's own 10000 ms history", estimate, ok)
	}
}

// A phase this scenario has barely run falls back to fleet history rather than
// going unguarded.
func TestPhaseDurationEstimateFallsBackToFleetHistory(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)

	insertDurationSample(t, db, "other", "security", "passed", 120_000, reliable())

	estimate, ok := repo.PhaseDurationEstimate(context.Background(), "demo", "security")
	if !ok || estimate != 120_000 {
		t.Fatalf("estimate = %d ms ok=%t, want the fleet fallback", estimate, ok)
	}
}

// No history at all is reported as unknown, so the caller can fall back to the
// planner's prediction instead of treating silence as "fast".
func TestPhaseDurationEstimateReportsUnknownWithoutHistory(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)

	if _, ok := repo.PhaseDurationEstimate(context.Background(), "demo", "security"); ok {
		t.Fatal("reported an estimate with no history")
	}
}
