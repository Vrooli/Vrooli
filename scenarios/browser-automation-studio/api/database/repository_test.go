package database

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
	"github.com/sirupsen/logrus"
)

func testBackend() string {
	backend := strings.ToLower(strings.TrimSpace(os.Getenv("BAS_TEST_BACKEND")))
	if backend == "" {
		backend = strings.ToLower(strings.TrimSpace(os.Getenv("BAS_DB_BACKEND")))
	}
	if backend == "" {
		backend = "sqlite"
	}
	return backend
}

func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	backend := testBackend()
	if backend != "sqlite" {
		t.Skipf("unsupported test backend %q (set BAS_TEST_BACKEND=sqlite)", backend)
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "bas-test.db")
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)",
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
		DB:      sqlDB,
		log:     log,
		dialect: DialectSQLite,
	}
	if err := wrapped.initSchema(); err != nil {
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

	list, err := repo.ListExecutions(ctx, &workflowID, 10, 0)
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

