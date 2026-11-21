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

// The deadline guard must include failed executions in its observed history.
// They contribute to the p90 tail even though one isolated failure does not
// become the estimate by itself.
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
	if estimate != 10_000 {
		t.Fatalf("estimate = %d ms, want the p90 estimate to include failed-run history", estimate)
	}
}

// The statistic must be tail-aware. A phase that is fast nine times in ten and
// slow on the tenth is exactly what a timeout catches, so a middle-of-the-
// distribution estimate would wave it into a batch and let contention turn the
// slow run into a false timeout.
func TestPhaseDurationEstimateUsesP90InsteadOfOneOutlier(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)

	for i := 0; i < 19; i++ {
		insertDurationSample(t, db, "demo", "unit", "passed", 5_000, reliable())
	}
	insertDurationSample(t, db, "demo", "unit", "failed", 140_000, reliable())

	estimate, ok := repo.PhaseDurationEstimate(context.Background(), "demo", "unit")
	if !ok || estimate != 5_000 {
		t.Fatalf("estimate = %d ms ok=%t, want p90 fast-tail sample", estimate, ok)
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

	if _, ok := repo.PhaseDurationEstimate(context.Background(), "demo", "experience"); ok {
		t.Fatal("reported a p90 from fewer than ten duration samples")
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

	if _, ok := repo.PhaseDurationEstimate(context.Background(), "demo", "structure"); ok {
		t.Fatal("reported a p90 from fewer than ten executed samples")
	}
}

func insertEnvelopeRun(t *testing.T, db *sql.DB, scenario, preset string, base time.Time, phases ...[6]any) {
	t.Helper()
	ctx := context.Background()
	execID := uuid.New().String()
	stamp := base.Add(30 * time.Second).Format(time.RFC3339Nano)
	if _, err := db.ExecContext(ctx, `
INSERT INTO suite_executions (id, run_id, scenario_name, preset_used, success, started_at, completed_at)
VALUES (?, ?, ?, ?, 1, ?, ?)`, execID, "run-"+execID[:8], scenario, preset, base.Format(time.RFC3339Nano), stamp); err != nil {
		t.Fatalf("insert envelope execution: %v", err)
	}
	for ordinal, p := range phases {
		if _, err := db.ExecContext(ctx, `
INSERT INTO suite_execution_phases (execution_id, ordinal, phase_name, status, started_at, completed_at, wall_clock_ms, cpu_user_ms, peak_rss_bytes, cpu_reliability, memory_reliability)
VALUES (?, ?, ?, 'passed', ?, ?, ?, ?, ?, 'RELIABILITY_RELIABLE', 'RELIABILITY_RELIABLE')`, execID, ordinal, p[0], p[1], p[2], p[3], p[4], p[5]); err != nil {
			t.Fatalf("insert envelope phase: %v", err)
		}
	}
}

func TestSuiteEnvelopeEstimateRequiresFiveRuns(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	estimate, err := repo.SuiteEnvelopeEstimate(context.Background(), "missing", "full")
	if err != nil {
		t.Fatal(err)
	}
	if estimate.Reliable {
		t.Fatalf("estimate = %#v, want honest unknown", estimate)
	}
}

func TestSuiteEnvelopeEstimateUsesMaximumOverlapNotSum(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	base := time.Now().UTC().Add(-time.Minute)
	for i := 0; i < 5; i++ {
		start := base.Add(time.Duration(i) * time.Second)
		insertEnvelopeRun(t, db, "demo", "full", start,
			[6]any{"first", start.Format(time.RFC3339Nano), start.Add(10 * time.Second).Format(time.RFC3339Nano), int64(10_000), int64(1_000), int64(100)},
			[6]any{"second", start.Add(5 * time.Second).Format(time.RFC3339Nano), start.Add(15 * time.Second).Format(time.RFC3339Nano), int64(10_000), int64(1_000), int64(200)})
	}
	estimate, err := repo.SuiteEnvelopeEstimate(context.Background(), "demo", "full")
	if err != nil {
		t.Fatal(err)
	}
	if !estimate.Reliable || estimate.RAMBytes != 300 {
		t.Fatalf("estimate = %#v, want reliable overlapping maximum of 300 bytes", estimate)
	}
}

func TestMaxConcurrentEnvelopeUsesLargestSinglePhaseWhenDisjoint(t *testing.T) {
	base := time.Now().UTC()
	ram, _, ok := maxConcurrentEnvelope([]envelopeInterval{
		{start: base, end: base.Add(time.Second), ram: 100, cpu: 1},
		{start: base.Add(time.Second), end: base.Add(2 * time.Second), ram: 200, cpu: 2},
	})
	if !ok || ram != 200 {
		t.Fatalf("max concurrent RAM = %d ok=%t, want 200,true", ram, ok)
	}
}

// The same phase costs very different amounts on different scenarios —
// `security` measured a 54 s median on prompt-manager against 137 s on
// browser-automation-studio — so scenario history wins whenever there is
// enough of it.
func TestPhaseDurationEstimatePrefersScenarioHistory(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)

	for i := 0; i < 10; i++ {
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

	if _, ok := repo.PhaseDurationEstimate(context.Background(), "demo", "security"); ok {
		t.Fatal("reported a fleet p90 from fewer than ten duration samples")
	}
}

func TestPhaseDurationEstimateWithholdsNineSamples(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	for i := 0; i < 9; i++ {
		insertDurationSample(t, db, "demo", "unit", "passed", 5_000, reliable())
	}
	if _, ok := repo.PhaseDurationEstimate(context.Background(), "demo", "unit"); ok {
		t.Fatal("reported a deadline estimate with only nine samples")
	}
}

func TestPhaseDurationEstimateUsesTenSamples(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	for i := 0; i < 10; i++ {
		insertDurationSample(t, db, "demo", "unit", "passed", 5_000, reliable())
	}
	estimate, ok := repo.PhaseDurationEstimate(context.Background(), "demo", "unit")
	if !ok || estimate != 5_000 {
		t.Fatalf("estimate = %d ms ok=%t, want a measured p90 at ten samples", estimate, ok)
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
