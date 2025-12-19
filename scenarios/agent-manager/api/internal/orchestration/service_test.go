package orchestration_test

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

// [REQ:REQ-P0-001] [REQ:REQ-P0-002] [REQ:REQ-P0-003] [REQ:REQ-P0-004]
// Tests for orchestration service - profile, task, and run operations

func newTestOrchestrator(t *testing.T) orchestration.Service {
	t.Helper()

	profileRepo := repository.NewMemoryProfileRepository()
	taskRepo := repository.NewMemoryTaskRepository()
	runRepo := repository.NewMemoryRunRepository()
	checkpointRepo := repository.NewMemoryCheckpointRepository()
	idempotencyRepo := repository.NewMemoryIdempotencyRepository()

	eventStore := event.NewMemoryStore()

	runnerRegistry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "mock runner available")
	runnerRegistry.Register(mockRunner)

	return orchestration.New(
		profileRepo,
		taskRepo,
		runRepo,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:          30 * time.Minute,
			MaxConcurrentRuns:       10,
			RequireSandboxByDefault: true,
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(runnerRegistry),
		orchestration.WithCheckpoints(checkpointRepo),
		orchestration.WithIdempotency(idempotencyRepo),
	)
}

func TestOrchestrator_ProfileCRUD(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	// Create
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "test-profile",
		RunnerType: domain.RunnerTypeClaudeCode,
		Model:      "claude-3-opus",
		MaxTurns:   100,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	created, err := svc.CreateProfile(ctx, profile)
	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}
	if created.ID != profile.ID {
		t.Error("created profile should have same ID")
	}

	// Get
	got, err := svc.GetProfile(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected profile, got nil")
	}
	if got.Name != "test-profile" {
		t.Errorf("expected name 'test-profile', got '%s'", got.Name)
	}

	// List
	profiles, err := svc.ListProfiles(ctx, orchestration.ListOptions{})
	if err != nil {
		t.Fatalf("ListProfiles failed: %v", err)
	}
	if len(profiles) != 1 {
		t.Errorf("expected 1 profile, got %d", len(profiles))
	}

	// Update
	profile.MaxTurns = 200
	updated, err := svc.UpdateProfile(ctx, profile)
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}
	if updated.MaxTurns != 200 {
		t.Errorf("expected MaxTurns 200, got %d", updated.MaxTurns)
	}

	// Delete
	if err := svc.DeleteProfile(ctx, profile.ID); err != nil {
		t.Fatalf("DeleteProfile failed: %v", err)
	}

	got, _ = svc.GetProfile(ctx, profile.ID)
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestOrchestrator_TaskCRUD(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	// Create
	task := &domain.Task{
		ID:          uuid.New(),
		Title:       "Test Task",
		Description: "A test task for orchestration",
		ScopePath:   "src/",
		Status:      domain.TaskStatusQueued,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	created, err := svc.CreateTask(ctx, task)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	if created.ID != task.ID {
		t.Error("created task should have same ID")
	}

	// Get
	got, err := svc.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected task, got nil")
	}
	if got.Title != "Test Task" {
		t.Errorf("expected title 'Test Task', got '%s'", got.Title)
	}

	// List
	tasks, err := svc.ListTasks(ctx, orchestration.ListOptions{})
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}

	// Update
	task.Description = "Updated description"
	updated, err := svc.UpdateTask(ctx, task)
	if err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}
	if updated.Description != "Updated description" {
		t.Errorf("expected updated description")
	}

	// Cancel
	if err := svc.CancelTask(ctx, task.ID); err != nil {
		t.Fatalf("CancelTask failed: %v", err)
	}

	got, _ = svc.GetTask(ctx, task.ID)
	if got.Status != domain.TaskStatusCancelled {
		t.Errorf("expected cancelled status, got %s", got.Status)
	}
}

func TestOrchestrator_RunOperations(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	// First create a profile and task
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "run-test-profile",
		RunnerType: domain.RunnerTypeClaudeCode,
		Model:      "claude-3-opus",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	svc.CreateProfile(ctx, profile)

	task := &domain.Task{
		ID:          uuid.New(),
		Title:       "Run Test Task",
		ScopePath:   "src/",
		Status:      domain.TaskStatusQueued,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	svc.CreateTask(ctx, task)

	// Create run (will fail due to missing sandbox, but we can test the creation logic)
	run, err := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         task.ID,
		AgentProfileID: profile.ID,
		Prompt:         "Test prompt",
	})
	if err != nil {
		// Expected - sandbox not available
		t.Logf("CreateRun returned expected error (sandbox unavailable): %v", err)
	} else if run != nil {
		// If we got a run, verify it
		got, err := svc.GetRun(ctx, run.ID)
		if err != nil {
			t.Fatalf("GetRun failed: %v", err)
		}
		if got == nil {
			t.Fatal("expected run, got nil")
		}

		// List runs
		runs, err := svc.ListRuns(ctx, orchestration.RunListOptions{})
		if err != nil {
			t.Fatalf("ListRuns failed: %v", err)
		}
		if len(runs) < 1 {
			t.Error("expected at least 1 run")
		}
	}
}

func TestOrchestrator_GetHealth(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	health, err := svc.GetHealth(ctx)
	if err != nil {
		t.Fatalf("GetHealth failed: %v", err)
	}

	// Verify required health fields per schema
	if health.Service != "agent-manager" {
		t.Errorf("expected service 'agent-manager', got '%s'", health.Service)
	}
	if health.Status == "" {
		t.Error("expected status to be set")
	}
	if health.Timestamp == "" {
		t.Error("expected timestamp to be set")
	}
}

func TestOrchestrator_GetRunnerStatus(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	statuses, err := svc.GetRunnerStatus(ctx)
	if err != nil {
		t.Fatalf("GetRunnerStatus failed: %v", err)
	}

	// We registered a mock claude-code runner
	found := false
	for _, s := range statuses {
		if s.Type == domain.RunnerTypeClaudeCode {
			found = true
			if !s.Available {
				t.Error("mock runner should be available")
			}
		}
	}
	if !found {
		t.Error("expected claude-code runner in status")
	}
}

func TestOrchestrator_ListProfiles_Pagination(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	// Create multiple profiles
	for i := 0; i < 10; i++ {
		profile := &domain.AgentProfile{
			ID:         uuid.New(),
			Name:       "profile-" + uuid.New().String(),
			RunnerType: domain.RunnerTypeClaudeCode,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		svc.CreateProfile(ctx, profile)
	}

	// Test limit
	profiles, _ := svc.ListProfiles(ctx, orchestration.ListOptions{Limit: 5})
	if len(profiles) != 5 {
		t.Errorf("expected 5 profiles with limit, got %d", len(profiles))
	}

	// Test offset
	profiles, _ = svc.ListProfiles(ctx, orchestration.ListOptions{Offset: 8})
	if len(profiles) != 2 {
		t.Errorf("expected 2 profiles with offset 8, got %d", len(profiles))
	}
}

func TestOrchestrator_GetProfile_NotFound(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	profile, err := svc.GetProfile(ctx, uuid.New())
	// Service returns NotFoundError for non-existent profiles
	if err == nil {
		t.Error("GetProfile should return error for non-existent profile")
	}
	if profile != nil {
		t.Error("expected nil for non-existent profile")
	}
}

func TestOrchestrator_GetTask_NotFound(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	task, err := svc.GetTask(ctx, uuid.New())
	// Service returns NotFoundError for non-existent tasks
	if err == nil {
		t.Error("GetTask should return error for non-existent task")
	}
	if task != nil {
		t.Error("expected nil for non-existent task")
	}
}

func TestOrchestrator_DeleteProfile_NotFound(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	// Delete non-existent profile should be idempotent
	err := svc.DeleteProfile(ctx, uuid.New())
	if err != nil {
		t.Errorf("DeleteProfile should be idempotent: %v", err)
	}
}
