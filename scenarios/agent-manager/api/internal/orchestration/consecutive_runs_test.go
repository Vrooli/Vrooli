package orchestration_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

// TestOrchestrator_ConsecutiveRuns verifies that multiple runs can be executed
// consecutively without the reconciler incorrectly marking them as stale.
// This test was added to diagnose an issue where the second task in a sequence
// was being killed by the reconciler.
//
// [REQ:ECOSYSTEM-MANAGER-INTEGRATION] Tests for consecutive task execution
func TestOrchestrator_ConsecutiveRuns(t *testing.T) {
	ctx := context.Background()

	// Setup repositories
	profileRepo := repository.NewMemoryProfileRepository()
	taskRepo := repository.NewMemoryTaskRepository()
	runRepo := repository.NewMemoryRunRepository()
	checkpointRepo := repository.NewMemoryCheckpointRepository()
	idempotencyRepo := repository.NewMemoryIdempotencyRepository()
	eventStore := event.NewMemoryStore()

	// Setup mock runner that completes after a short delay
	runnerRegistry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "mock runner available")

	// Track execution count
	var executionCount int
	var executionMu sync.Mutex

	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		executionMu.Lock()
		executionCount++
		execNum := executionCount
		executionMu.Unlock()

		t.Logf("Mock execution #%d started for run %s (tag=%s)", execNum, req.RunID, req.Tag)

		// Simulate some work - sleep for 500ms to allow heartbeat to be sent
		select {
		case <-ctx.Done():
			t.Logf("Mock execution #%d cancelled", execNum)
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}

		t.Logf("Mock execution #%d completed successfully", execNum)
		return &runner.ExecuteResult{
			ExitCode: 0,
			Summary: &domain.RunSummary{
				Description: "Mock execution completed successfully",
			},
		}, nil
	}

	if err := runnerRegistry.Register(mockRunner); err != nil {
		t.Fatalf("Failed to register mock runner: %v", err)
	}

	// Create orchestrator with in-place mode (no sandbox required)
	svc := orchestration.New(
		profileRepo,
		taskRepo,
		runRepo,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:          5 * time.Minute,
			MaxConcurrentRuns:       10,
			RequireSandboxByDefault: false, // Allow in-place execution
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(runnerRegistry),
		orchestration.WithCheckpoints(checkpointRepo),
		orchestration.WithIdempotency(idempotencyRepo),
	)

	// Create a profile
	profile := &domain.AgentProfile{
		ID:              uuid.New(),
		Name:            "consecutive-test-profile",
		RunnerType:      domain.RunnerTypeClaudeCode,
		Model:           "claude-3-opus",
		RequiresSandbox: false, // In-place execution
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	createdProfile, err := svc.CreateProfile(ctx, profile)
	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}

	// Create two tasks
	task1 := &domain.Task{
		ID:          uuid.New(),
		Title:       "Consecutive Test Task 1",
		Description: "First task in consecutive test",
		ScopePath:   "/tmp",
		ProjectRoot: "/tmp",
		Status:      domain.TaskStatusQueued,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	createdTask1, err := svc.CreateTask(ctx, task1)
	if err != nil {
		t.Fatalf("CreateTask 1 failed: %v", err)
	}

	task2 := &domain.Task{
		ID:          uuid.New(),
		Title:       "Consecutive Test Task 2",
		Description: "Second task in consecutive test",
		ScopePath:   "/tmp",
		ProjectRoot: "/tmp",
		Status:      domain.TaskStatusQueued,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	createdTask2, err := svc.CreateTask(ctx, task2)
	if err != nil {
		t.Fatalf("CreateTask 2 failed: %v", err)
	}

	// Execute first run
	t.Log("Creating first run...")
	runMode := domain.RunModeInPlace
	run1, err := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         createdTask1.ID,
		AgentProfileID: &createdProfile.ID,
		Prompt:         "Execute first test task",
		RunMode:        &runMode,
	})
	if err != nil {
		t.Fatalf("CreateRun 1 failed: %v", err)
	}
	t.Logf("First run created: %s (tag=%s)", run1.ID, run1.GetTag())

	// Wait for first run to complete
	t.Log("Waiting for first run to complete...")
	run1Completed, err := waitForRunCompletion(t, ctx, svc, run1.ID, 30*time.Second)
	if err != nil {
		t.Fatalf("First run failed to complete: %v", err)
	}
	if run1Completed.Status != domain.RunStatusComplete {
		t.Fatalf("First run expected status Complete, got %s (error: %s)",
			run1Completed.Status, run1Completed.ErrorMsg)
	}
	t.Logf("First run completed successfully with status: %s", run1Completed.Status)

	// Execute second run (this is where the bug was observed)
	t.Log("Creating second run...")
	run2, err := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         createdTask2.ID,
		AgentProfileID: &createdProfile.ID,
		Prompt:         "Execute second test task",
		RunMode:        &runMode,
	})
	if err != nil {
		t.Fatalf("CreateRun 2 failed: %v", err)
	}
	t.Logf("Second run created: %s (tag=%s)", run2.ID, run2.GetTag())

	// Wait for second run to complete
	t.Log("Waiting for second run to complete...")
	run2Completed, err := waitForRunCompletion(t, ctx, svc, run2.ID, 30*time.Second)
	if err != nil {
		t.Fatalf("Second run failed to complete: %v", err)
	}
	if run2Completed.Status != domain.RunStatusComplete {
		t.Fatalf("Second run expected status Complete, got %s (error: %s)",
			run2Completed.Status, run2Completed.ErrorMsg)
	}
	t.Logf("Second run completed successfully with status: %s", run2Completed.Status)

	// Verify execution count
	executionMu.Lock()
	finalCount := executionCount
	executionMu.Unlock()
	if finalCount != 2 {
		t.Errorf("Expected 2 executions, got %d", finalCount)
	}

	t.Log("Both consecutive runs completed successfully!")
}

// TestOrchestrator_ConsecutiveRunsWithHeartbeat tests that heartbeats are properly
// sent during execution and the reconciler doesn't incorrectly kill running tasks.
func TestOrchestrator_ConsecutiveRunsWithHeartbeat(t *testing.T) {
	ctx := context.Background()

	// Setup repositories
	profileRepo := repository.NewMemoryProfileRepository()
	taskRepo := repository.NewMemoryTaskRepository()
	runRepo := repository.NewMemoryRunRepository()
	checkpointRepo := repository.NewMemoryCheckpointRepository()
	idempotencyRepo := repository.NewMemoryIdempotencyRepository()
	eventStore := event.NewMemoryStore()

	// Setup mock runner with longer execution time to test heartbeat
	runnerRegistry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "mock runner available")

	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		t.Logf("Mock execution started for run %s (tag=%s)", req.RunID, req.Tag)

		// Simulate work that takes longer than one heartbeat interval (15s)
		// but shorter than stale threshold (5min)
		// For test speed, just sleep 2 seconds which is enough to verify heartbeat starts
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}

		t.Logf("Mock execution completed for run %s", req.RunID)
		return &runner.ExecuteResult{
			ExitCode: 0,
			Summary: &domain.RunSummary{
				Description: "Mock execution with heartbeat test completed",
			},
		}, nil
	}

	if err := runnerRegistry.Register(mockRunner); err != nil {
		t.Fatalf("Failed to register mock runner: %v", err)
	}

	// Create orchestrator
	svc := orchestration.New(
		profileRepo,
		taskRepo,
		runRepo,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:          5 * time.Minute,
			MaxConcurrentRuns:       10,
			RequireSandboxByDefault: false,
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(runnerRegistry),
		orchestration.WithCheckpoints(checkpointRepo),
		orchestration.WithIdempotency(idempotencyRepo),
	)

	// Create profile and task
	profile := &domain.AgentProfile{
		ID:              uuid.New(),
		Name:            "heartbeat-test-profile",
		RunnerType:      domain.RunnerTypeClaudeCode,
		RequiresSandbox: false,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	createdProfile, err := svc.CreateProfile(ctx, profile)
	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}

	task := &domain.Task{
		ID:          uuid.New(),
		Title:       "Heartbeat Test Task",
		ScopePath:   "/tmp",
		ProjectRoot: "/tmp",
		Status:      domain.TaskStatusQueued,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	createdTask, err := svc.CreateTask(ctx, task)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Create and execute run
	runMode := domain.RunModeInPlace
	run, err := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         createdTask.ID,
		AgentProfileID: &createdProfile.ID,
		Prompt:         "Test heartbeat during execution",
		RunMode:        &runMode,
	})
	if err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	// Wait for completion
	completedRun, err := waitForRunCompletion(t, ctx, svc, run.ID, 30*time.Second)
	if err != nil {
		t.Fatalf("Run failed to complete: %v", err)
	}

	// Verify the run completed successfully (not killed by reconciler)
	if completedRun.Status != domain.RunStatusComplete {
		t.Fatalf("Expected status Complete, got %s (error: %s)",
			completedRun.Status, completedRun.ErrorMsg)
	}

	// Verify heartbeat was set (should have at least one heartbeat)
	if completedRun.LastHeartbeat == nil {
		t.Error("Expected LastHeartbeat to be set")
	} else {
		t.Logf("Last heartbeat was at: %v", completedRun.LastHeartbeat)
	}

	t.Log("Heartbeat test completed successfully!")
}

// waitForRunCompletion polls for run completion with a timeout.
func waitForRunCompletion(t *testing.T, ctx context.Context, svc orchestration.Service, runID uuid.UUID, timeout time.Duration) (*domain.Run, error) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				run, _ := svc.GetRun(ctx, runID)
				if run != nil {
					return nil, &runTimeoutError{
						runID:  runID,
						status: run.Status,
						error:  run.ErrorMsg,
					}
				}
				return nil, &runTimeoutError{runID: runID}
			}

			run, err := svc.GetRun(ctx, runID)
			if err != nil {
				return nil, err
			}
			if run == nil {
				return nil, &runNotFoundError{runID: runID}
			}

			// Check for terminal states
			switch run.Status {
			case domain.RunStatusComplete, domain.RunStatusFailed, domain.RunStatusCancelled:
				return run, nil
			}
		}
	}
}

type runTimeoutError struct {
	runID  uuid.UUID
	status domain.RunStatus
	error  string
}

func (e *runTimeoutError) Error() string {
	if e.status != "" {
		return "run " + e.runID.String() + " timed out in status " + string(e.status) + " (error: " + e.error + ")"
	}
	return "run " + e.runID.String() + " timed out"
}

type runNotFoundError struct {
	runID uuid.UUID
}

func (e *runNotFoundError) Error() string {
	return "run " + e.runID.String() + " not found"
}
