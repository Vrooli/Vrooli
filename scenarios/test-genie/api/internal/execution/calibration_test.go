package execution

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"test-genie/internal/testsqlite"

	"github.com/google/uuid"
)

// insertCalibrationSample writes one phase row with an explicit reliability, so
// a case can describe exactly which history shape it is testing. A nil
// reliability means the provider reported its resources UNAVAILABLE and
// metricColumns therefore stored nothing — the workflow-health shape.
func insertCalibrationSample(t *testing.T, db *sql.DB, scenario, phase string, completedAt time.Time, reliability *string) {
	t.Helper()
	ctx := context.Background()
	execID := uuid.New().String()
	stamp := completedAt.UTC().Format(time.RFC3339Nano)
	if _, err := db.ExecContext(ctx, `
INSERT INTO suite_executions (id, run_id, scenario_name, success, started_at, completed_at)
VALUES (?, ?, ?, 1, ?, ?)`, execID, "run-"+execID[:8], scenario, stamp, stamp); err != nil {
		t.Fatalf("insert execution: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO suite_execution_phases (execution_id, ordinal, phase_name, status, duration_seconds, duration_ms, cpu_reliability, memory_reliability)
VALUES (?, 0, ?, 'passed', 1, 1000, ?, ?)`, execID, phase, reliability, reliability); err != nil {
		t.Fatalf("insert phase: %v", err)
	}
}

func reliable() *string   { value := "RELIABILITY_RELIABLE"; return &value }
func bestEffort() *string { value := "RELIABILITY_BEST_EFFORT"; return &value }

// A phase whose provider reports its resources unavailable stores no
// reliability at all and can never produce a reliable sample. Letting it demand
// one made this function return "calibrate" on every run of every scenario,
// forever — measured at 100% of runs on 2026-08-08, always naming `workflow`.
func TestCalibrationIgnoresPhaseThatCanNeverReportReliability(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()

	insertCalibrationSample(t, db, "demo", "unit", now.Add(-time.Hour), reliable())
	// workflow observed repeatedly, never with a reliability.
	for i := 0; i < 3; i++ {
		insertCalibrationSample(t, db, "demo", "workflow", now.Add(-time.Duration(i+1)*time.Hour), nil)
	}

	forceSerial, reason := repo.CalibrationDecision(context.Background(), "demo", []string{"unit", "workflow"}, "")
	if forceSerial {
		t.Fatalf("forced serial for an unmeasurable phase: %s", reason)
	}
}

// The exclusion must be narrow. A phase that CAN report reliability but has not
// done so recently is exactly what calibration exists for.
func TestCalibrationStillFiresForStaleMeasurablePhase(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()

	insertCalibrationSample(t, db, "demo", "unit", now.Add(-time.Hour), reliable())
	// security can measure — it has best-effort rows — but no reliable one.
	insertCalibrationSample(t, db, "demo", "security", now.Add(-2*time.Hour), bestEffort())

	forceSerial, reason := repo.CalibrationDecision(context.Background(), "demo", []string{"unit", "security"}, "")
	if !forceSerial {
		t.Fatal("did not force serial for a measurable phase with no reliable sample")
	}
	if !strings.Contains(reason, "security") {
		t.Fatalf("reason = %q, want it to name security", reason)
	}
}

// A phase that keeps reporting BEST_EFFORT is unmeasurable here, not
// uncalibrated. api-core's collector degrades to BEST_EFFORT when more than one
// collector is active inside the provider process, which serializing this run's
// phases cannot quiet, so the veto's own remedy would never clear it. Measured
// 2026-08-08: `experience` held this shape for source-ledger, vrooli-memory,
// and scenario-to-desktop, and those runs executed every phase serially.
func TestCalibrationIgnoresPhaseThatOnlyEverReportsBestEffort(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()

	insertCalibrationSample(t, db, "demo", "unit", now.Add(-time.Hour), reliable())
	for i := 0; i < minUnmeasurableEvidence; i++ {
		insertCalibrationSample(t, db, "demo", "experience", now.Add(-time.Duration(i+1)*time.Hour), bestEffort())
	}

	forceSerial, reason := repo.CalibrationDecision(context.Background(), "demo", []string{"unit", "experience"}, "")
	if forceSerial {
		t.Fatalf("forced serial for a phase that can only report best-effort: %s", reason)
	}
}

// The exclusion is earned by evidence, not assumed on the first miss. Below the
// threshold the phase is still treated as uncalibrated, because one best-effort
// sample cannot distinguish "not measured uncontended yet" from "cannot be".
func TestCalibrationStillFiresBelowUnmeasurableEvidenceThreshold(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()

	insertCalibrationSample(t, db, "demo", "unit", now.Add(-time.Hour), reliable())
	for i := 0; i < minUnmeasurableEvidence-1; i++ {
		insertCalibrationSample(t, db, "demo", "experience", now.Add(-time.Duration(i+1)*time.Hour), bestEffort())
	}

	forceSerial, reason := repo.CalibrationDecision(context.Background(), "demo", []string{"unit", "experience"}, "")
	if !forceSerial {
		t.Fatal("stopped calibrating before the evidence threshold was reached")
	}
	if !strings.Contains(reason, "experience") {
		t.Fatalf("reason = %q, want it to name experience", reason)
	}
}

// A phase with no history at all is calibrated, not skipped. Silence is not
// evidence that a phase cannot be measured.
func TestCalibrationFiresForPhaseWithNoHistory(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)

	insertCalibrationSample(t, db, "demo", "unit", time.Now().UTC().Add(-time.Hour), reliable())

	forceSerial, reason := repo.CalibrationDecision(context.Background(), "demo", []string{"unit", "brand-new"}, "")
	if !forceSerial {
		t.Fatal("did not force serial for a phase with no history")
	}
	if !strings.Contains(reason, "brand-new") {
		t.Fatalf("reason = %q, want it to name brand-new", reason)
	}
}

// A reliable sample older than the interval is stale history, and a phase whose
// only reliable sample has aged out must be re-measured.
func TestCalibrationFiresWhenReliableSampleAgedOut(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()

	// Inside the window but best-effort; the reliable one is outside it.
	insertCalibrationSample(t, db, "demo", "unit", now.Add(-time.Hour), bestEffort())
	insertCalibrationSample(t, db, "demo", "unit", now.Add(-calibrationInterval-time.Hour), reliable())

	forceSerial, reason := repo.CalibrationDecision(context.Background(), "demo", []string{"unit"}, "")
	if !forceSerial {
		t.Fatalf("did not force serial when the only reliable sample aged out: %q", reason)
	}
}

// A fresh reliable sample means there is nothing to calibrate.
func TestCalibrationQuietWhenReliableSampleIsFresh(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()

	insertCalibrationSample(t, db, "demo", "unit", now.Add(-time.Hour), reliable())
	insertCalibrationSample(t, db, "demo", "contracts", now.Add(-time.Hour), reliable())

	forceSerial, reason := repo.CalibrationDecision(context.Background(), "demo", []string{"unit", "contracts"}, "")
	if forceSerial {
		t.Fatalf("forced serial with fresh reliable samples: %s", reason)
	}
}

// A changed descriptor is still measured serially regardless of sample health.
func TestCalibrationFiresOnDescriptorChange(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()

	insertCalibrationSample(t, db, "demo", "unit", now.Add(-time.Hour), reliable())
	if _, err := db.ExecContext(context.Background(),
		`UPDATE suite_executions SET descriptor_snapshot_digest = 'ds:old' WHERE scenario_name = 'demo'`); err != nil {
		t.Fatalf("set descriptor: %v", err)
	}

	forceSerial, reason := repo.CalibrationDecision(context.Background(), "demo", []string{"unit"}, "ds:new")
	if !forceSerial || !strings.Contains(reason, "descriptor") {
		t.Fatalf("forceSerial=%t reason=%q, want a descriptor-change calibration", forceSerial, reason)
	}
}
