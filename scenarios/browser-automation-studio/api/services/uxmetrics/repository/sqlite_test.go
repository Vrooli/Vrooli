package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	repocontract "github.com/vrooli/repo-contract-go"

	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics/contracts"

	_ "modernc.org/sqlite"
)

// newRepoTestDB opens an in-memory SQLite DB, applies the canonical
// scenario schema (the same file the API loads at startup), and seeds the
// FK targets that the UX metrics tables require.
func newRepoTestDB(t *testing.T) (*sqlx.DB, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "uxmetrics-test.db")
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_time_format=sqlite",
		dbPath,
	)
	db, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repoRoot, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}
	scenarioRoot, err := repocontract.ResolveScenarioPath(repoRoot, "browser-automation-studio")
	if err != nil {
		t.Fatalf("resolve scenario root: %v", err)
	}
	schemaPath := filepath.Join(scenarioRoot, "initialization", "storage", "sqlite", "schema.sql")
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if _, err := db.Exec(string(schemaBytes)); err != nil {
		t.Fatalf("exec schema: %v", err)
	}

	projectID := uuid.New()
	workflowID := uuid.New()
	executionID := uuid.New()

	if _, err := db.Exec(
		`INSERT INTO projects (id, name, folder_path) VALUES (?, ?, ?)`,
		projectID.String(), "uxm-test", "/tmp/uxm",
	); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO workflows (id, project_id, name, folder_path) VALUES (?, ?, ?, ?)`,
		workflowID.String(), projectID.String(), "wf", "/tmp/uxm/wf",
	); err != nil {
		t.Fatalf("seed workflow: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO executions (id, workflow_id, status) VALUES (?, ?, 'completed')`,
		executionID.String(), workflowID.String(),
	); err != nil {
		t.Fatalf("seed execution: %v", err)
	}

	return db, executionID, workflowID, projectID
}

func TestRepository_SaveAndListInteractionTraces(t *testing.T) {
	db, executionID, _, _ := newRepoTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Millisecond)
	trace := &contracts.InteractionTrace{
		ID:          uuid.New(),
		ExecutionID: executionID,
		StepIndex:   3,
		ActionType:  contracts.ActionClick,
		ElementID:   "btn-submit",
		Selector:    "#btn-submit",
		Position:    &autocontracts.Point{X: 120, Y: 240},
		Timestamp:   now,
		DurationMs:  150,
		Success:     true,
		Metadata:    map[string]any{"foo": "bar"},
	}

	if err := repo.SaveInteractionTrace(ctx, trace); err != nil {
		t.Fatalf("SaveInteractionTrace: %v", err)
	}

	got, err := repo.ListInteractionTraces(ctx, executionID)
	if err != nil {
		t.Fatalf("ListInteractionTraces: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(got))
	}
	if got[0].ID != trace.ID || got[0].StepIndex != 3 || got[0].ActionType != contracts.ActionClick {
		t.Fatalf("trace mismatch: %+v", got[0])
	}
	if got[0].Position == nil || got[0].Position.X != 120 || got[0].Position.Y != 240 {
		t.Fatalf("position mismatch: %+v", got[0].Position)
	}
	if v, _ := got[0].Metadata["foo"].(string); v != "bar" {
		t.Fatalf("metadata mismatch: %+v", got[0].Metadata)
	}
}

func TestRepository_SaveAndGetCursorPath(t *testing.T) {
	db, executionID, _, _ := newRepoTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	first := &contracts.CursorPath{
		StepIndex:       2,
		Points:          []contracts.TimedPoint{{X: 1, Y: 1, Timestamp: time.Now()}},
		TotalDistancePx: 100,
		DirectDistance:  90,
		DurationMs:      500,
		Directness:      0.9,
		ZigZagScore:     0.1,
		AverageSpeed:    0.2,
		MaxSpeed:        0.5,
		Hesitations:     1,
	}
	if err := repo.SaveCursorPath(ctx, executionID, first); err != nil {
		t.Fatalf("first SaveCursorPath: %v", err)
	}

	// Save again with different metrics for the same (execution, step) — must upsert.
	second := *first
	second.TotalDistancePx = 250
	second.Hesitations = 4
	if err := repo.SaveCursorPath(ctx, executionID, &second); err != nil {
		t.Fatalf("second SaveCursorPath (upsert): %v", err)
	}

	got, err := repo.GetCursorPath(ctx, executionID, 2)
	if err != nil {
		t.Fatalf("GetCursorPath: %v", err)
	}
	if got == nil {
		t.Fatalf("expected cursor path, got nil")
	}
	if got.TotalDistancePx != 250 || got.Hesitations != 4 {
		t.Fatalf("upsert did not apply: %+v", got)
	}
}

func TestRepository_SaveAndGetExecutionMetrics(t *testing.T) {
	db, executionID, workflowID, _ := newRepoTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	metrics := &contracts.ExecutionMetrics{
		ExecutionID:       executionID,
		WorkflowID:        workflowID,
		ComputedAt:        time.Now().UTC(),
		TotalDurationMs:   12345,
		StepCount:         8,
		SuccessfulSteps:   7,
		FailedSteps:       1,
		TotalRetries:      2,
		AvgStepDurationMs: 1543.125,
		TotalCursorDist:   450.5,
		OverallFriction:   25.5,
		FrictionSignals:   []contracts.FrictionSignal{{Type: contracts.FrictionZigZagPath, StepIndex: 4, Severity: contracts.SeverityMedium, Score: 60}},
		StepMetrics:       []contracts.StepMetrics{{StepIndex: 0, FrictionScore: 10}},
		Summary:           &contracts.MetricsSummary{HighFrictionSteps: []int{4}},
	}
	if err := repo.SaveExecutionMetrics(ctx, metrics); err != nil {
		t.Fatalf("first SaveExecutionMetrics: %v", err)
	}

	// Save again with mutated values — must upsert by execution_id.
	updated := *metrics
	updated.OverallFriction = 75
	updated.StepCount = 9
	if err := repo.SaveExecutionMetrics(ctx, &updated); err != nil {
		t.Fatalf("second SaveExecutionMetrics (upsert): %v", err)
	}

	got, err := repo.GetExecutionMetrics(ctx, executionID)
	if err != nil {
		t.Fatalf("GetExecutionMetrics: %v", err)
	}
	if got == nil {
		t.Fatalf("expected execution metrics, got nil")
	}
	if got.OverallFriction != 75 || got.StepCount != 9 {
		t.Fatalf("upsert did not apply: %+v", got)
	}
	if len(got.FrictionSignals) != 1 || got.FrictionSignals[0].Type != contracts.FrictionZigZagPath {
		t.Fatalf("friction signals JSON did not roundtrip: %+v", got.FrictionSignals)
	}
	if got.Summary == nil || len(got.Summary.HighFrictionSteps) != 1 || got.Summary.HighFrictionSteps[0] != 4 {
		t.Fatalf("summary JSON did not roundtrip: %+v", got.Summary)
	}

	// Sanity check that we still have only one row (upsert, not insert).
	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM ux_execution_metrics WHERE execution_id = ?`, executionID); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row, got %d (upsert created duplicate?)", count)
	}

	// And that GetStepMetrics finds the seeded step.
	step, err := repo.GetStepMetrics(ctx, executionID, 0)
	if err != nil {
		t.Fatalf("GetStepMetrics: %v", err)
	}
	if step == nil || step.StepIndex != 0 {
		t.Fatalf("expected step 0, got %+v", step)
	}
	_ = strings.TrimSpace // keep imports stable if test variants are added later
}
