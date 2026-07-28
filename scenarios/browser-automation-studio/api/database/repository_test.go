package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "bas-test.db")
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_time_format=sqlite",
		dbPath,
	)

	sqlDB, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}

	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.PanicLevel)

	wrapped := &DB{
		DB:  sqlDB,
		log: log,
	}
	if err := wrapped.EnsureSchemas(); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("init schema: %v", err)
	}

	return wrapped, func() {
		_ = sqlDB.Close()
	}
}

func TestProjectCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(db, logrus.New())
	ctx := context.Background()

	project := &ProjectIndex{
		ID:         uuid.New(),
		Name:       "Test Project",
		FolderPath: "/test/project1",
	}

	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	got, err := repo.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if got.Name != project.Name {
		t.Fatalf("expected project name %q, got %q", project.Name, got.Name)
	}

	byName, err := repo.GetProjectByName(ctx, project.Name)
	if err != nil {
		t.Fatalf("GetProjectByName: %v", err)
	}
	if byName.ID != project.ID {
		t.Fatalf("expected project id %s, got %s", project.ID, byName.ID)
	}

	byFolder, err := repo.GetProjectByFolderPath(ctx, project.FolderPath)
	if err != nil {
		t.Fatalf("GetProjectByFolderPath: %v", err)
	}
	if byFolder.ID != project.ID {
		t.Fatalf("expected project id %s, got %s", project.ID, byFolder.ID)
	}

	projects, err := repo.ListProjects(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}

	project.Name = "Renamed Project"
	if err := repo.UpdateProject(ctx, project); err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}

	got, err = repo.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject after update: %v", err)
	}
	if got.Name != "Renamed Project" {
		t.Fatalf("expected updated name, got %q", got.Name)
	}

	if err := repo.DeleteProject(ctx, project.ID); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	if _, err := repo.GetProject(ctx, project.ID); err == nil {
		t.Fatalf("expected GetProject to fail after delete")
	}
}

func TestWorkflowCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(db, logrus.New())
	ctx := context.Background()

	projectID := uuid.New()
	if err := repo.CreateProject(ctx, &ProjectIndex{ID: projectID, Name: "P1", FolderPath: "/p1"}); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	wf := &WorkflowIndex{
		ID:         uuid.New(),
		ProjectID:  &projectID,
		Name:       "Workflow A",
		FolderPath: "/p1/workflows",
		FilePath:   "bas/workflows/workflow-a.json",
		Version:    1,
	}
	if err := repo.CreateWorkflow(ctx, wf); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	got, err := repo.GetWorkflow(ctx, wf.ID)
	if err != nil {
		t.Fatalf("GetWorkflow: %v", err)
	}
	if got.Name != wf.Name {
		t.Fatalf("expected workflow name %q, got %q", wf.Name, got.Name)
	}

	gotByName, err := repo.GetWorkflowByName(ctx, wf.Name, wf.FolderPath)
	if err != nil {
		t.Fatalf("GetWorkflowByName: %v", err)
	}
	if gotByName.ID != wf.ID {
		t.Fatalf("expected workflow id %s, got %s", wf.ID, gotByName.ID)
	}

	byProject, err := repo.ListWorkflowsByProject(ctx, projectID, 10, 0)
	if err != nil {
		t.Fatalf("ListWorkflowsByProject: %v", err)
	}
	if len(byProject) != 1 {
		t.Fatalf("expected 1 workflow, got %d", len(byProject))
	}

	wf.Version = 2
	if err := repo.UpdateWorkflow(ctx, wf); err != nil {
		t.Fatalf("UpdateWorkflow: %v", err)
	}

	got, err = repo.GetWorkflow(ctx, wf.ID)
	if err != nil {
		t.Fatalf("GetWorkflow after update: %v", err)
	}
	if got.Version != 2 {
		t.Fatalf("expected updated version 2, got %d", got.Version)
	}

	if err := repo.DeleteWorkflow(ctx, wf.ID); err != nil {
		t.Fatalf("DeleteWorkflow: %v", err)
	}
	if _, err := repo.GetWorkflow(ctx, wf.ID); err == nil {
		t.Fatalf("expected GetWorkflow to fail after delete")
	}
}

func TestExecutionCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(db, logrus.New())
	ctx := context.Background()

	workflowID := uuid.New()
	if err := repo.CreateWorkflow(ctx, &WorkflowIndex{ID: workflowID, Name: "W1", FolderPath: "/w1", Version: 1}); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	exec := &ExecutionIndex{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		Status:     ExecutionStatusRunning,
		StartedAt:  time.Now().UTC(),
		ResultPath: "data/recordings/execution-1/result.json",
	}
	if err := repo.CreateExecution(ctx, exec); err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}

	got, err := repo.GetExecution(ctx, exec.ID)
	if err != nil {
		t.Fatalf("GetExecution: %v", err)
	}
	if got.Status != ExecutionStatusRunning {
		t.Fatalf("expected status %q, got %q", ExecutionStatusRunning, got.Status)
	}

	completedAt := time.Now().UTC()
	exec.Status = ExecutionStatusCompleted
	exec.CompletedAt = &completedAt
	if err := repo.UpdateExecution(ctx, exec); err != nil {
		t.Fatalf("UpdateExecution: %v", err)
	}

	list, err := repo.ListExecutions(ctx, &workflowID, nil, 10, 0)
	if err != nil {
		t.Fatalf("ListExecutions: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(list))
	}

	listByStatus, err := repo.ListExecutionsByStatus(ctx, ExecutionStatusCompleted, 10, 0)
	if err != nil {
		t.Fatalf("ListExecutionsByStatus: %v", err)
	}
	if len(listByStatus) != 1 {
		t.Fatalf("expected 1 execution by status, got %d", len(listByStatus))
	}

	if err := repo.DeleteExecution(ctx, exec.ID); err != nil {
		t.Fatalf("DeleteExecution: %v", err)
	}
	if _, err := repo.GetExecution(ctx, exec.ID); err == nil {
		t.Fatalf("expected GetExecution to fail after delete")
	}
}

func TestCreateExecutionSupportsLegacyTriggerTypeSchema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "bas-legacy.db")
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_time_format=sqlite",
		dbPath,
	)

	sqlDB, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}
	defer sqlDB.Close()

	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.PanicLevel)

	wrapped := &DB{
		DB:  sqlDB,
		log: log,
	}

	ctx := context.Background()
	if _, err := wrapped.ExecContext(ctx, `
		CREATE TABLE workflows (
			id TEXT PRIMARY KEY
		);
		CREATE TABLE executions (
			id TEXT PRIMARY KEY,
			workflow_id TEXT REFERENCES workflows(id) ON DELETE CASCADE,
			status TEXT NOT NULL,
			trigger_type TEXT NOT NULL,
			started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP,
			error_message TEXT,
			result_path TEXT,
			resumed_from_id TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}

	workflowID := uuid.New()
	if _, err := wrapped.ExecContext(ctx, `INSERT INTO workflows (id) VALUES (?)`, workflowID.String()); err != nil {
		t.Fatalf("seed workflow: %v", err)
	}

	repo := NewRepository(wrapped, logrus.New())
	exec := &ExecutionIndex{
		ID:          uuid.New(),
		WorkflowID:  workflowID,
		Status:      ExecutionStatusPending,
		TriggerType: "api",
		StartedAt:   time.Now().UTC(),
	}
	if err := repo.CreateExecution(ctx, exec); err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}

	var triggerType string
	if err := wrapped.GetContext(ctx, &triggerType, `SELECT trigger_type FROM executions WHERE id = ?`, exec.ID.String()); err != nil {
		t.Fatalf("query trigger_type: %v", err)
	}
	if triggerType != "api" {
		t.Fatalf("expected trigger_type %q, got %q", "api", triggerType)
	}
}

func TestExecutionStatusUpdatePreservesResultPath(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(db, logrus.New())
	ctx := context.Background()

	workflowID := uuid.New()
	if err := repo.CreateWorkflow(ctx, &WorkflowIndex{ID: workflowID, Name: "W1", FolderPath: "/w1", Version: 1}); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	exec := &ExecutionIndex{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		Status:     ExecutionStatusRunning,
		StartedAt:  time.Now().UTC(),
		ResultPath: "data/recordings/execution-2/result.json",
	}
	if err := repo.CreateExecution(ctx, exec); err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}

	completedAt := time.Now().UTC()
	errMsg := "completed"
	if err := repo.UpdateExecutionStatus(ctx, exec.ID, ExecutionStatusCompleted, &errMsg, &completedAt, time.Now().UTC()); err != nil {
		t.Fatalf("UpdateExecutionStatus: %v", err)
	}

	got, err := repo.GetExecution(ctx, exec.ID)
	if err != nil {
		t.Fatalf("GetExecution: %v", err)
	}
	if got.ResultPath != exec.ResultPath {
		t.Fatalf("expected result_path %q, got %q", exec.ResultPath, got.ResultPath)
	}
	if got.Status != ExecutionStatusCompleted {
		t.Fatalf("expected status %q, got %q", ExecutionStatusCompleted, got.Status)
	}
}

func TestExecutionResultPathUpdate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(db, logrus.New())
	ctx := context.Background()

	workflowID := uuid.New()
	if err := repo.CreateWorkflow(ctx, &WorkflowIndex{ID: workflowID, Name: "W2", FolderPath: "/w2", Version: 1}); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	exec := &ExecutionIndex{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		Status:     ExecutionStatusRunning,
		StartedAt:  time.Now().UTC(),
	}
	if err := repo.CreateExecution(ctx, exec); err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}

	resultPath := "data/recordings/execution-3/result.json"
	if err := repo.UpdateExecutionResultPath(ctx, exec.ID, resultPath, time.Now().UTC()); err != nil {
		t.Fatalf("UpdateExecutionResultPath: %v", err)
	}

	got, err := repo.GetExecution(ctx, exec.ID)
	if err != nil {
		t.Fatalf("GetExecution: %v", err)
	}
	if got.ResultPath != resultPath {
		t.Fatalf("expected result_path %q, got %q", resultPath, got.ResultPath)
	}
}

// TestGetProjectsStats_RoundTripsLastExecution pins the regression
// behind the "ListProjects: failed to get project stats" 500 we hit
// after the proto+Connect migration. MAX(started_at) is an aggregate
// column with no declared SQL type, so the SQLite driver can't auto-
// convert its text result into *time.Time — the value must be scanned
// as a string and parsed in code. The repair script at
// /tmp/browser-automation-studio/migrate-fix-execution-timestamps.sh
// addresses the data; this test pins the read path so a future change
// to GetProjectsStats can't reintroduce the unsupported-Scan error.
func TestGetProjectsStats_RoundTripsLastExecution(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(db, logrus.New())
	ctx := context.Background()

	project := &ProjectIndex{ID: uuid.New(), Name: "Stats Project", FolderPath: "/stats/p"}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	workflow := &WorkflowIndex{
		ID:         uuid.New(),
		ProjectID:  &project.ID,
		Name:       "Stats Workflow",
		FolderPath: "/stats",
		Version:    1,
	}
	if err := repo.CreateWorkflow(ctx, workflow); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	earliest := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	middle := time.Date(2026, 5, 10, 12, 30, 0, 0, time.UTC)
	latest := time.Date(2026, 5, 20, 18, 45, 17, 123456789, time.UTC)
	for _, at := range []time.Time{earliest, middle, latest} {
		if err := repo.CreateExecution(ctx, &ExecutionIndex{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Status:     ExecutionStatusCompleted,
			StartedAt:  at,
		}); err != nil {
			t.Fatalf("CreateExecution: %v", err)
		}
	}

	stats, err := repo.GetProjectsStats(ctx, []uuid.UUID{project.ID})
	if err != nil {
		t.Fatalf("GetProjectsStats: %v", err)
	}
	got, ok := stats[project.ID]
	if !ok {
		t.Fatalf("project %s missing from stats map", project.ID)
	}
	if got.ExecutionCount != 3 {
		t.Errorf("execution_count: got %d want 3", got.ExecutionCount)
	}
	if got.WorkflowCount != 1 {
		t.Errorf("workflow_count: got %d want 1", got.WorkflowCount)
	}
	if got.LastExecution == nil {
		t.Fatal("last_execution: got nil, want latest")
	}
	if !got.LastExecution.Equal(latest) {
		t.Errorf("last_execution: got %s, want %s", got.LastExecution.Format(time.RFC3339Nano), latest.Format(time.RFC3339Nano))
	}
}

func TestGetProjectsStatsAcceptsLegacyGoTimeString(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(db, logrus.New())
	ctx := context.Background()
	project := &ProjectIndex{ID: uuid.New(), Name: "Legacy Timestamp Project", FolderPath: "/stats/legacy"}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	workflow := &WorkflowIndex{ID: uuid.New(), ProjectID: &project.ID, Name: "Legacy Timestamp Workflow", FolderPath: "/stats", Version: 1}
	if err := repo.CreateWorkflow(ctx, workflow); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	// Simulate rows persisted by the historical time.Time.String() writer.
	const rawStartedAt = "2026-07-27 20:42:27.339810567 +0000 UTC"
	if _, err := db.ExecContext(ctx, `INSERT INTO executions (id, workflow_id, status, started_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, uuid.New(), workflow.ID, ExecutionStatusCompleted, rawStartedAt, rawStartedAt, rawStartedAt); err != nil {
		t.Fatalf("insert legacy execution: %v", err)
	}

	stats, err := repo.GetProjectsStats(ctx, []uuid.UUID{project.ID})
	if err != nil {
		t.Fatalf("GetProjectsStats: %v", err)
	}
	got := stats[project.ID]
	if got == nil || got.LastExecution == nil {
		t.Fatalf("LastExecution = %#v, want parsed legacy timestamp", got)
	}
	want := time.Date(2026, 7, 27, 20, 42, 27, 339810567, time.UTC)
	if !got.LastExecution.Equal(want) {
		t.Errorf("LastExecution = %s, want %s", got.LastExecution.Format(time.RFC3339Nano), want.Format(time.RFC3339Nano))
	}
}

// TestGetProjectsStats_NoExecutionsLeavesLastExecutionNil keeps the
// nullable-projection contract honest: a project with no workflows or
// executions must surface as a populated stats row with all counters
// zero and LastExecution nil — not an absent map entry, not a 500.
func TestGetProjectsStats_NoExecutionsLeavesLastExecutionNil(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(db, logrus.New())
	ctx := context.Background()

	project := &ProjectIndex{ID: uuid.New(), Name: "Empty", FolderPath: "/empty"}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	stats, err := repo.GetProjectsStats(ctx, []uuid.UUID{project.ID})
	if err != nil {
		t.Fatalf("GetProjectsStats: %v", err)
	}
	got, ok := stats[project.ID]
	if !ok {
		t.Fatalf("project %s missing from stats map", project.ID)
	}
	if got.LastExecution != nil {
		t.Errorf("LastExecution: got %v, want nil", got.LastExecution)
	}
	if got.ExecutionCount != 0 || got.WorkflowCount != 0 {
		t.Errorf("counts should be zero: %+v", got)
	}
}

// TestParseTimestamp_AcceptsKnownLayouts pins the two text shapes
// SQLite stores time values in (RFC3339Nano from typed Go bindings,
// and SQLite's CURRENT_TIMESTAMP "YYYY-MM-DD HH:MM:SS" form). Both
// must parse cleanly because aggregate columns drop the declared
// type and force us to parse in code.
func TestParseTimestamp_AcceptsKnownLayouts(t *testing.T) {
	cases := []struct {
		in   string
		want time.Time
	}{
		// modernc.org/sqlite _time_format=sqlite shape (production default)
		{"2026-05-20 14:42:49.490183017+00:00", time.Date(2026, 5, 20, 14, 42, 49, 490183017, time.UTC)},
		// SQLite CURRENT_TIMESTAMP default
		{"2026-05-20 14:42:49", time.Date(2026, 5, 20, 14, 42, 49, 0, time.UTC)},
		// RFC3339Nano (rows repaired by the one-shot script)
		{"2026-05-20T14:42:49.490183017Z", time.Date(2026, 5, 20, 14, 42, 49, 490183017, time.UTC)},
		{"2026-05-20T14:42:49Z", time.Date(2026, 5, 20, 14, 42, 49, 0, time.UTC)},
	}
	for _, tc := range cases {
		got, err := parseTimestamp(tc.in)
		if err != nil {
			t.Errorf("parseTimestamp(%q): unexpected error %v", tc.in, err)
			continue
		}
		if !got.Equal(tc.want) {
			t.Errorf("parseTimestamp(%q) = %v; want %v", tc.in, got, tc.want)
		}
	}

	if _, err := parseTimestamp("not a timestamp"); err == nil {
		t.Error("parseTimestamp should reject unrecognized input")
	}
}
