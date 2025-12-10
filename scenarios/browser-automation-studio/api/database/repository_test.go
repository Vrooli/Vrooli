package database

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/testutil/testdb"
)

var (
	testDBHandle *testdb.Handle
	// testDBMu protects concurrent access to the shared test database
	testDBMu sync.Mutex
)

func testBackend() string {
	backend := strings.ToLower(strings.TrimSpace(os.Getenv("BAS_TEST_BACKEND")))
	if backend == "" {
		backend = strings.ToLower(strings.TrimSpace(os.Getenv("BAS_DB_BACKEND")))
	}
	if backend == "" {
		backend = "postgres"
	}
	return backend
}

func TestMain(m *testing.M) {
	backend := testBackend()
	ctx := context.Background()

	if backend == "postgres" {
		handle, err := testdb.Start(ctx)
		if err != nil {
			fmt.Printf("Failed to start postgres test db: %s\n", err)
			os.Exit(0)
		}
		testDBHandle = handle

		code := m.Run()

		handle.Terminate(ctx)
		os.Exit(code)
	}

	// SQLite mode does not require external containers.
	code := m.Run()
	os.Exit(code)
}

// setupTestDB creates a test database connection using the testcontainer.
// It acquires a mutex to ensure exclusive access to the shared test database.
func setupTestDB(t *testing.T) (*DB, func()) {
	backend := testBackend()

	oldURL := os.Getenv("DATABASE_URL")
	oldSkipDemo := os.Getenv("BAS_SKIP_DEMO_SEED")
	oldHost := os.Getenv("POSTGRES_HOST")
	oldPort := os.Getenv("POSTGRES_PORT")
	oldUser := os.Getenv("POSTGRES_USER")
	oldPass := os.Getenv("POSTGRES_PASSWORD")
	oldDB := os.Getenv("POSTGRES_DB")
	oldBackend := os.Getenv("BAS_DB_BACKEND")
	oldSQLitePath := os.Getenv("BAS_SQLITE_PATH")

	restoreEnv := func() {
		restore := func(key, val string) {
			if val == "" {
				os.Unsetenv(key)
				return
			}
			os.Setenv(key, val)
		}
		restore("DATABASE_URL", oldURL)
		restore("BAS_SKIP_DEMO_SEED", oldSkipDemo)
		restore("POSTGRES_HOST", oldHost)
		restore("POSTGRES_PORT", oldPort)
		restore("POSTGRES_USER", oldUser)
		restore("POSTGRES_PASSWORD", oldPass)
		restore("POSTGRES_DB", oldDB)
		restore("BAS_DB_BACKEND", oldBackend)
		restore("BAS_SQLITE_PATH", oldSQLitePath)
	}

	if backend == "postgres" {
		if testDBHandle == nil {
			t.Fatal("Test database not initialized - TestMain should have set handle")
		}

		testDBMu.Lock()

		os.Unsetenv("POSTGRES_HOST")
		os.Unsetenv("POSTGRES_PORT")
		os.Unsetenv("POSTGRES_USER")
		os.Unsetenv("POSTGRES_PASSWORD")
		os.Unsetenv("POSTGRES_DB")

		os.Setenv("DATABASE_URL", testDBHandle.DSN)
		os.Setenv("BAS_SKIP_DEMO_SEED", "true")
		os.Setenv("BAS_DB_BACKEND", "postgres")
		os.Unsetenv("BAS_SQLITE_PATH")

		log := logrus.New()
		log.SetOutput(ioutil.Discard)

		db, err := NewConnection(log)
		if err != nil {
			testDBMu.Unlock()
			t.Fatalf("Failed to connect to test database: %v", err)
		}

		if err := truncateAllWithRetry(db, 3); err != nil {
			db.Close()
			testDBMu.Unlock()
			t.Fatalf("failed to truncate test tables: %v", err)
		}

		cleanup := func() {
			_ = truncateAll(db)
			db.Close()

			restoreEnv()
			testDBMu.Unlock()
		}

		return db, cleanup
	}

	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	// SQLite mode: use a temp file per test to isolate state.
	tmpDir := t.TempDir()
	sqlitePath := filepath.Join(tmpDir, "test.db")

	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("POSTGRES_HOST")
	os.Unsetenv("POSTGRES_PORT")
	os.Unsetenv("POSTGRES_USER")
	os.Unsetenv("POSTGRES_PASSWORD")
	os.Unsetenv("POSTGRES_DB")
	os.Setenv("BAS_DB_BACKEND", "sqlite")
	os.Setenv("BAS_SQLITE_PATH", sqlitePath)
	os.Setenv("BAS_SKIP_DEMO_SEED", "true")

	db, err := NewConnection(log)
	if err != nil {
		t.Fatalf("Failed to connect to sqlite test database: %v", err)
	}

	if err := truncateAll(db); err != nil {
		db.Close()
		t.Fatalf("failed to truncate sqlite tables: %v", err)
	}

	cleanup := func() {
		_ = truncateAll(db)
		db.Close()
		restoreEnv()
	}

	return db, cleanup
}

func truncateAll(db *DB) error {
	if db == nil {
		return nil
	}
	ctx := context.Background()
	if db.Dialect().IsPostgres() {
		query := `TRUNCATE
		exports,
		ai_generations,
		workflow_schedules,
		workflow_templates,
		execution_artifacts,
		execution_steps,
		execution_logs,
		screenshots,
		extracted_data,
		executions,
		workflow_versions,
		workflows,
		workflow_folders,
		projects
	CASCADE`
		_, err := db.ExecContext(ctx, query)
		return err
	}

	// SQLite: DELETE tables in dependency order and vacuum to reset autoincrement.
	tables := []string{
		"exports",
		"ai_generations",
		"workflow_schedules",
		"workflow_templates",
		"execution_artifacts",
		"execution_steps",
		"execution_logs",
		"screenshots",
		"extracted_data",
		"executions",
		"workflow_versions",
		"workflows",
		"workflow_folders",
		"projects",
	}
	for _, table := range tables {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "no such table") {
				continue
			}
			return err
		}
	}
	_, _ = db.ExecContext(ctx, "VACUUM")
	return nil
}

// truncateAllWithRetry attempts to truncate all tables with retries for transient errors.
func truncateAllWithRetry(db *DB, maxRetries int) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := truncateAll(db); err != nil {
			lastErr = err
			// Check if it's a deadlock error and retry
			if strings.Contains(err.Error(), "deadlock") {
				time.Sleep(time.Duration(50*(i+1)) * time.Millisecond)
				continue
			}
			return err
		}
		return nil
	}
	return lastErr
}

func TestCreateProject(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	repo := NewRepository(db, log)

	t.Run("Success", func(t *testing.T) {
		project := &Project{
			ID:          uuid.New(),
			Name:        "Test Project",
			Description: "A test project",
			FolderPath:  "/test/project1",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		ctx := context.Background()
		err := repo.CreateProject(ctx, project)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		// Verify project was created
		retrieved, err := repo.GetProject(ctx, project.ID)
		if err != nil {
			t.Fatalf("Failed to get project: %v", err)
		}

		if retrieved.Name != project.Name {
			t.Errorf("Expected name %s, got %s", project.Name, retrieved.Name)
		}
	})

	t.Run("GenerateID", func(t *testing.T) {
		project := &Project{
			Name:       "Test Project 2",
			FolderPath: "/test/project2",
		}

		ctx := context.Background()
		err := repo.CreateProject(ctx, project)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		if project.ID == uuid.Nil {
			t.Error("Expected project ID to be generated")
		}
	})

	t.Run("DuplicateName", func(t *testing.T) {
		project1 := &Project{
			ID:         uuid.New(),
			Name:       "Duplicate Project",
			FolderPath: "/test/duplicate1",
		}

		ctx := context.Background()
		if err := repo.CreateProject(ctx, project1); err != nil {
			t.Fatalf("Failed to create first project: %v", err)
		}

		project2 := &Project{
			ID:         uuid.New(),
			Name:       "Duplicate Project",
			FolderPath: "/test/duplicate2",
		}

		// This should fail because project names are unique
		err := repo.CreateProject(ctx, project2)
		if err == nil {
			t.Error("Expected error for duplicate project name")
		}
		// Check for unique constraint violation
		if err != nil {
			errMsg := strings.ToLower(err.Error())
			if !strings.Contains(errMsg, "duplicate key") && !strings.Contains(errMsg, "unique constraint") && !strings.Contains(errMsg, "constraint failed") {
				t.Errorf("Expected unique constraint error, got: %v", err)
			}
		}
	})
}

func TestGetProject(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	repo := NewRepository(db, log)

	// Create test project
	project := &Project{
		ID:         uuid.New(),
		Name:       "Test Project",
		FolderPath: "/test/get-project",
	}

	ctx := context.Background()
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		retrieved, err := repo.GetProject(ctx, project.ID)
		if err != nil {
			t.Fatalf("Failed to get project: %v", err)
		}

		if retrieved.ID != project.ID {
			t.Errorf("Expected ID %s, got %s", project.ID, retrieved.ID)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := repo.GetProject(ctx, nonExistentID)
		if err == nil {
			t.Error("Expected error for non-existent project")
		}
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Expected ErrNotFound, got: %v", err)
		}
	})
}

func TestUpdateProject(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	repo := NewRepository(db, log)

	// Create test project
	project := &Project{
		ID:         uuid.New(),
		Name:       "Original Name",
		FolderPath: "/test/update-project",
	}

	ctx := context.Background()
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		project.Name = "Updated Name"
		project.Description = "Updated description"

		err := repo.UpdateProject(ctx, project)
		if err != nil {
			t.Fatalf("Failed to update project: %v", err)
		}

		// Verify update
		retrieved, err := repo.GetProject(ctx, project.ID)
		if err != nil {
			t.Fatalf("Failed to get project: %v", err)
		}

		if retrieved.Name != "Updated Name" {
			t.Errorf("Expected name 'Updated Name', got %s", retrieved.Name)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		nonExistentProject := &Project{
			ID:         uuid.New(),
			Name:       "Non-existent",
			FolderPath: "/test/non-existent",
		}

		err := repo.UpdateProject(ctx, nonExistentProject)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Expected ErrNotFound, got: %v", err)
		}
	})
}

func TestExecutionStepAndArtifactPersistence(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	repo := NewRepository(db, log)
	ctx := context.Background()

	workflow := &Workflow{
		ID:         uuid.New(),
		Name:       "Step Test Workflow",
		FolderPath: "/test/steps",
		FlowDefinition: JSONMap{
			"nodes": []any{},
			"edges": []any{},
		},
	}
	if err := repo.CreateWorkflow(ctx, workflow); err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}

	execution := &Execution{
		ID:              uuid.New(),
		WorkflowID:      workflow.ID,
		WorkflowVersion: workflow.Version,
		Status:          "running",
		TriggerType:     "manual",
		TriggerMetadata: JSONMap{},
		Parameters:      JSONMap{},
		Progress:        0,
		CurrentStep:     "initialize",
	}
	if err := repo.CreateExecution(ctx, execution); err != nil {
		t.Fatalf("failed to create execution: %v", err)
	}

	step := &ExecutionStep{
		ExecutionID: execution.ID,
		StepIndex:   0,
		NodeID:      "node-1",
		StepType:    "navigate",
		Status:      "running",
		Input: JSONMap{
			"url": "https://example.com",
		},
	}

	if err := repo.CreateExecutionStep(ctx, step); err != nil {
		t.Fatalf("failed to create execution step: %v", err)
	}
	if step.ID == uuid.Nil {
		t.Fatal("expected execution step to have an ID after creation")
	}

	completedAt := time.Now()
	step.Status = "completed"
	step.CompletedAt = &completedAt
	step.DurationMs = 1234
	step.Output = JSONMap{
		"finalUrl": "https://example.com/dashboard",
	}

	if err := repo.UpdateExecutionStep(ctx, step); err != nil {
		t.Fatalf("failed to update execution step: %v", err)
	}

	steps, err := repo.ListExecutionSteps(ctx, execution.ID)
	if err != nil {
		t.Fatalf("failed to list execution steps: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 execution step, got %d", len(steps))
	}
	if steps[0].Status != "completed" {
		t.Errorf("expected execution step status 'completed', got %s", steps[0].Status)
	}

	size := int64(2048)
	stepIndex := step.StepIndex
	artifact := &ExecutionArtifact{
		ExecutionID:  execution.ID,
		StepID:       &step.ID,
		StepIndex:    &stepIndex,
		ArtifactType: "screenshot",
		StorageURL:   "s3://test-bucket/executions/test/screenshot.png",
		ContentType:  "image/png",
		SizeBytes:    &size,
		Payload: JSONMap{
			"width":  1280,
			"height": 720,
		},
	}

	if err := repo.CreateExecutionArtifact(ctx, artifact); err != nil {
		t.Fatalf("failed to create execution artifact: %v", err)
	}
	if artifact.ID == uuid.Nil {
		t.Fatal("expected execution artifact to have an ID")
	}

	artifacts, err := repo.ListExecutionArtifacts(ctx, execution.ID)
	if err != nil {
		t.Fatalf("failed to list execution artifacts: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("expected 1 execution artifact, got %d", len(artifacts))
	}
	if artifacts[0].ArtifactType != "screenshot" {
		t.Errorf("expected artifact type 'screenshot', got %s", artifacts[0].ArtifactType)
	}
}

func TestDeleteProject(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	repo := NewRepository(db, log)

	t.Run("Success", func(t *testing.T) {
		project := &Project{
			ID:         uuid.New(),
			Name:       "Project to Delete",
			FolderPath: "/test/delete-project",
		}

		ctx := context.Background()
		if err := repo.CreateProject(ctx, project); err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		err := repo.DeleteProject(ctx, project.ID)
		if err != nil {
			t.Fatalf("Failed to delete project: %v", err)
		}

		// Verify deletion
		_, err = repo.GetProject(ctx, project.ID)
		if err == nil {
			t.Error("Expected error when getting deleted project")
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		ctx := context.Background()
		err := repo.DeleteProject(ctx, nonExistentID)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Expected ErrNotFound, got: %v", err)
		}
	})
}

func TestListProjects(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	repo := NewRepository(db, log)

	ctx := context.Background()

	// Create test projects
	for i := 0; i < 5; i++ {
		project := &Project{
			ID:         uuid.New(),
			Name:       "Test Project " + string(rune(i+'A')),
			FolderPath: "/test/list-project-" + string(rune(i+'a')),
		}
		if err := repo.CreateProject(ctx, project); err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}
	}

	t.Run("Success", func(t *testing.T) {
		projects, err := repo.ListProjects(ctx, 10, 0)
		if err != nil {
			t.Fatalf("Failed to list projects: %v", err)
		}

		if len(projects) < 5 {
			t.Errorf("Expected at least 5 projects, got %d", len(projects))
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		projects, err := repo.ListProjects(ctx, 2, 0)
		if err != nil {
			t.Fatalf("Failed to list projects: %v", err)
		}

		if len(projects) > 2 {
			t.Errorf("Expected at most 2 projects, got %d", len(projects))
		}
	})
}

func TestWorkflowOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	repo := NewRepository(db, log)
	ctx := context.Background()

	// Create a test project
	project := &Project{
		ID:         uuid.New(),
		Name:       "Workflow Test Project",
		FolderPath: "/test/workflow-project",
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	t.Run("CreateWorkflow", func(t *testing.T) {
		workflow := &Workflow{
			ID:         uuid.New(),
			ProjectID:  &project.ID,
			Name:       "Test Workflow",
			FolderPath: "/test/workflows",
			FlowDefinition: JSONMap{
				"nodes": []any{},
				"edges": []any{},
			},
			Tags:      []string{"test"},
			Version:   1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := repo.CreateWorkflow(ctx, workflow)
		if err != nil {
			t.Fatalf("Failed to create workflow: %v", err)
		}

		// Verify creation
		retrieved, err := repo.GetWorkflow(ctx, workflow.ID)
		if err != nil {
			t.Fatalf("Failed to get workflow: %v", err)
		}

		if retrieved.Name != workflow.Name {
			t.Errorf("Expected name %s, got %s", workflow.Name, retrieved.Name)
		}
	})

	t.Run("GetWorkflowByName", func(t *testing.T) {
		workflow := &Workflow{
			ID:         uuid.New(),
			Name:       "Named Workflow",
			FolderPath: "/test/named-workflow",
			FlowDefinition: JSONMap{
				"nodes": []any{},
			},
			Tags:    []string{},
			Version: 1,
		}

		if err := repo.CreateWorkflow(ctx, workflow); err != nil {
			t.Fatalf("Failed to create workflow: %v", err)
		}

		retrieved, err := repo.GetWorkflowByName(ctx, "Named Workflow", "/test/named-workflow")
		if err != nil {
			t.Fatalf("Failed to get workflow by name: %v", err)
		}

		if retrieved.ID != workflow.ID {
			t.Errorf("Expected ID %s, got %s", workflow.ID, retrieved.ID)
		}
	})
}

func TestExecutionOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	repo := NewRepository(db, log)
	ctx := context.Background()

	// Create test workflow
	workflow := &Workflow{
		ID:         uuid.New(),
		Name:       "Execution Test Workflow",
		FolderPath: "/test/execution-workflow",
		FlowDefinition: JSONMap{
			"nodes": []any{},
		},
		Tags:    []string{},
		Version: 1,
	}
	if err := repo.CreateWorkflow(ctx, workflow); err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	t.Run("CreateExecution", func(t *testing.T) {
		execution := &Execution{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Status:     "pending",
			StartedAt:  time.Now(),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		err := repo.CreateExecution(ctx, execution)
		if err != nil {
			t.Fatalf("Failed to create execution: %v", err)
		}

		// Verify creation
		retrieved, err := repo.GetExecution(ctx, execution.ID)
		if err != nil {
			t.Fatalf("Failed to get execution: %v", err)
		}

		if retrieved.Status != "pending" {
			t.Errorf("Expected status pending, got %s", retrieved.Status)
		}
	})

	t.Run("UpdateExecution", func(t *testing.T) {
		execution := &Execution{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Status:     "pending",
			StartedAt:  time.Now(),
		}

		if err := repo.CreateExecution(ctx, execution); err != nil {
			t.Fatalf("Failed to create execution: %v", err)
		}

		// Update status
		execution.Status = "completed"
		completedAt := time.Now()
		execution.CompletedAt = &completedAt

		err := repo.UpdateExecution(ctx, execution)
		if err != nil {
			t.Fatalf("Failed to update execution: %v", err)
		}

		// Verify update
		retrieved, err := repo.GetExecution(ctx, execution.ID)
		if err != nil {
			t.Fatalf("Failed to get execution: %v", err)
		}

		if retrieved.Status != "completed" {
			t.Errorf("Expected status completed, got %s", retrieved.Status)
		}
	})

	t.Run("ListExecutions", func(t *testing.T) {
		// Create multiple executions
		for i := 0; i < 3; i++ {
			execution := &Execution{
				ID:         uuid.New(),
				WorkflowID: workflow.ID,
				Status:     "pending",
				StartedAt:  time.Now(),
			}
			if err := repo.CreateExecution(ctx, execution); err != nil {
				t.Fatalf("Failed to create execution: %v", err)
			}
		}

		executions, err := repo.ListExecutions(ctx, &workflow.ID, 10, 0)
		if err != nil {
			t.Fatalf("Failed to list executions: %v", err)
		}

		if len(executions) < 3 {
			t.Errorf("Expected at least 3 executions, got %d", len(executions))
		}
	})
}
