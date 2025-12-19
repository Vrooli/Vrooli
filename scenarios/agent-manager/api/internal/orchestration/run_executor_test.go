package orchestration_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

// =============================================================================
// TEST FIXTURES
// =============================================================================

// testFixtures holds shared test data.
type testFixtures struct {
	profile *domain.AgentProfile
	task    *domain.Task
	run     *domain.Run
}

// newTestFixtures creates a consistent set of test fixtures.
func newTestFixtures() *testFixtures {
	profileID := uuid.New()
	taskID := uuid.New()
	runID := uuid.New()

	return &testFixtures{
		profile: &domain.AgentProfile{
			ID:         profileID,
			Name:       "test-profile",
			RunnerType: domain.RunnerTypeClaudeCode,
			Model:      "claude-3-opus",
			MaxTurns:   100,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		task: &domain.Task{
			ID:          taskID,
			Title:       "Test Task",
			Description: "A test task for executor tests",
			ScopePath:   "src/",
			ProjectRoot: "/project",
			Status:      domain.TaskStatusQueued,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		run: &domain.Run{
			ID:             runID,
			TaskID:         taskID,
			AgentProfileID: &profileID,
			Status:         domain.RunStatusPending,
			Phase:          domain.RunPhaseQueued,
			RunMode:        domain.RunModeSandboxed,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
	}
}

// newInPlaceFixtures creates fixtures for in-place execution.
func newInPlaceFixtures() *testFixtures {
	f := newTestFixtures()
	f.run.RunMode = domain.RunModeInPlace
	return f
}

// =============================================================================
// MOCK SANDBOX PROVIDER
// =============================================================================

// mockSandboxProvider is a test double for sandbox.Provider.
type mockSandboxProvider struct {
	createFunc         func(ctx context.Context, req sandbox.CreateRequest) (*sandbox.Sandbox, error)
	getFunc            func(ctx context.Context, id uuid.UUID) (*sandbox.Sandbox, error)
	deleteFunc         func(ctx context.Context, id uuid.UUID) error
	getWorkspacePathFn func(ctx context.Context, id uuid.UUID) (string, error)
	getDiffFunc        func(ctx context.Context, id uuid.UUID) (*sandbox.DiffResult, error)
	approveFunc        func(ctx context.Context, req sandbox.ApproveRequest) (*sandbox.ApproveResult, error)
	rejectFunc         func(ctx context.Context, id uuid.UUID, actor string) error
	partialApproveFunc func(ctx context.Context, req sandbox.PartialApproveRequest) (*sandbox.ApproveResult, error)
	stopFunc           func(ctx context.Context, id uuid.UUID) error
	startFunc          func(ctx context.Context, id uuid.UUID) error
	isAvailableFunc    func(ctx context.Context) (bool, string)
}

func newMockSandboxProvider() *mockSandboxProvider {
	sandboxID := uuid.New()
	return &mockSandboxProvider{
		createFunc: func(ctx context.Context, req sandbox.CreateRequest) (*sandbox.Sandbox, error) {
			return &sandbox.Sandbox{
				ID:          sandboxID,
				ScopePath:   req.ScopePath,
				ProjectRoot: req.ProjectRoot,
				Status:      sandbox.SandboxStatusActive,
				WorkDir:     "/tmp/sandbox/" + sandboxID.String(),
				CreatedAt:   time.Now(),
			}, nil
		},
		getWorkspacePathFn: func(ctx context.Context, id uuid.UUID) (string, error) {
			return "/tmp/sandbox/" + id.String() + "/merged", nil
		},
	}
}

func (m *mockSandboxProvider) Create(ctx context.Context, req sandbox.CreateRequest) (*sandbox.Sandbox, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockSandboxProvider) Get(ctx context.Context, id uuid.UUID) (*sandbox.Sandbox, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockSandboxProvider) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockSandboxProvider) GetWorkspacePath(ctx context.Context, id uuid.UUID) (string, error) {
	if m.getWorkspacePathFn != nil {
		return m.getWorkspacePathFn(ctx, id)
	}
	return "", nil
}

func (m *mockSandboxProvider) IsAvailable(ctx context.Context) (bool, string) {
	if m.isAvailableFunc != nil {
		return m.isAvailableFunc(ctx)
	}
	return true, ""
}

func (m *mockSandboxProvider) GetDiff(ctx context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
	if m.getDiffFunc != nil {
		return m.getDiffFunc(ctx, id)
	}
	return &sandbox.DiffResult{}, nil
}

func (m *mockSandboxProvider) Approve(ctx context.Context, req sandbox.ApproveRequest) (*sandbox.ApproveResult, error) {
	if m.approveFunc != nil {
		return m.approveFunc(ctx, req)
	}
	return &sandbox.ApproveResult{Success: true}, nil
}

func (m *mockSandboxProvider) Reject(ctx context.Context, id uuid.UUID, actor string) error {
	if m.rejectFunc != nil {
		return m.rejectFunc(ctx, id, actor)
	}
	return nil
}

func (m *mockSandboxProvider) PartialApprove(ctx context.Context, req sandbox.PartialApproveRequest) (*sandbox.ApproveResult, error) {
	if m.partialApproveFunc != nil {
		return m.partialApproveFunc(ctx, req)
	}
	return &sandbox.ApproveResult{Success: true}, nil
}

func (m *mockSandboxProvider) Stop(ctx context.Context, id uuid.UUID) error {
	if m.stopFunc != nil {
		return m.stopFunc(ctx, id)
	}
	return nil
}

func (m *mockSandboxProvider) Start(ctx context.Context, id uuid.UUID) error {
	if m.startFunc != nil {
		return m.startFunc(ctx, id)
	}
	return nil
}

// =============================================================================
// EXECUTOR CONFIG TESTS
// =============================================================================

func TestDefaultExecutorConfig(t *testing.T) {
	config := orchestration.DefaultExecutorConfig()

	if config.Timeout != 30*time.Minute {
		t.Errorf("expected default timeout 30m, got %v", config.Timeout)
	}
	if config.HeartbeatInterval != 30*time.Second {
		t.Errorf("expected default heartbeat interval 30s, got %v", config.HeartbeatInterval)
	}
	if config.CheckpointInterval != 1*time.Minute {
		t.Errorf("expected default checkpoint interval 1m, got %v", config.CheckpointInterval)
	}
	if config.MaxRetries != 3 {
		t.Errorf("expected default max retries 3, got %d", config.MaxRetries)
	}
	if config.StaleThreshold != 2*time.Minute {
		t.Errorf("expected default stale threshold 2m, got %v", config.StaleThreshold)
	}
}

// =============================================================================
// EXECUTOR CREATION TESTS
// =============================================================================

func TestNewRunExecutor(t *testing.T) {
	f := newTestFixtures()
	runRepo := repository.NewMemoryRunRepository()
	runRepo.Create(context.Background(), f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	registry.Register(mockRunner)

	sandboxProvider := newMockSandboxProvider()
	eventStore := event.NewMemoryStore()

	executor := orchestration.NewRunExecutor(
		runRepo,
		registry,
		sandboxProvider,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
	)

	if executor == nil {
		t.Fatal("expected executor, got nil")
	}
}

func TestRunExecutor_WithConfig(t *testing.T) {
	f := newTestFixtures()
	runRepo := repository.NewMemoryRunRepository()
	registry := runner.NewRegistry()
	eventStore := event.NewMemoryStore()

	customConfig := orchestration.ExecutorConfig{
		Timeout:           5 * time.Minute,
		HeartbeatInterval: 10 * time.Second,
		MaxRetries:        5,
	}

	executor := orchestration.NewRunExecutor(
		runRepo,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
	).WithConfig(customConfig)

	if executor == nil {
		t.Fatal("expected executor, got nil")
	}
}

// =============================================================================
// SANDBOX WORKSPACE SETUP TESTS
// =============================================================================

func TestRunExecutor_Execute_SandboxedMode_Success(t *testing.T) {
	f := newTestFixtures()
	runRepo := repository.NewMemoryRunRepository()
	runRepo.Create(context.Background(), f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	// Set up mock to return successful result
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{
			Success:  true,
			ExitCode: 0,
			Summary:  &domain.RunSummary{},
		}, nil
	}
	registry.Register(mockRunner)

	sandboxProvider := newMockSandboxProvider()
	eventStore := event.NewMemoryStore()

	// Use short timeout for test
	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
		MaxRetries:        1,
	}

	executor := orchestration.NewRunExecutor(
		runRepo,
		registry,
		sandboxProvider,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
	).WithConfig(config)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	executor.Execute(ctx)

	// Verify sandbox was created
	if executor.SandboxID() == nil {
		t.Error("expected sandbox ID to be set")
	}

	// Verify work directory was set
	if executor.WorkDir() == "" {
		t.Error("expected work directory to be set")
	}

	// Verify outcome - should require review on success
	outcome := executor.Outcome()
	if outcome != domain.RunOutcomeSuccess {
		t.Errorf("expected outcome 'success', got '%s'", outcome)
	}
}

func TestRunExecutor_Execute_InPlaceMode_Success(t *testing.T) {
	f := newInPlaceFixtures()
	runRepo := repository.NewMemoryRunRepository()
	runRepo.Create(context.Background(), f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{
			Success:  true,
			ExitCode: 0,
		}, nil
	}
	registry.Register(mockRunner)

	eventStore := event.NewMemoryStore()

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		runRepo,
		registry,
		nil, // No sandbox provider for in-place
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
	).WithConfig(config)

	ctx := context.Background()
	executor.Execute(ctx)

	// Verify no sandbox was created
	if executor.SandboxID() != nil {
		t.Error("expected no sandbox ID for in-place mode")
	}

	// Verify work directory uses project root
	if executor.WorkDir() != f.task.ProjectRoot {
		t.Errorf("expected work dir '%s', got '%s'", f.task.ProjectRoot, executor.WorkDir())
	}
}

func TestRunExecutor_Execute_SandboxCreationFailure(t *testing.T) {
	f := newTestFixtures()
	runRepo := repository.NewMemoryRunRepository()
	runRepo.Create(context.Background(), f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	registry.Register(mockRunner)

	// Mock sandbox provider that fails
	sandboxProvider := &mockSandboxProvider{
		createFunc: func(ctx context.Context, req sandbox.CreateRequest) (*sandbox.Sandbox, error) {
			return nil, errors.New("sandbox service unavailable")
		},
	}

	eventStore := event.NewMemoryStore()

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		runRepo,
		registry,
		sandboxProvider,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
	).WithConfig(config)

	ctx := context.Background()
	executor.Execute(ctx)

	// Verify sandbox ID is nil
	if executor.SandboxID() != nil {
		t.Error("expected no sandbox ID on failure")
	}

	// Verify outcome indicates sandbox failure
	outcome := executor.Outcome()
	if outcome != domain.RunOutcomeSandboxFail {
		t.Errorf("expected outcome 'sandbox_fail', got '%s'", outcome)
	}

	// Verify run was marked failed
	updatedRun, _ := runRepo.Get(context.Background(), f.run.ID)
	if updatedRun.Status != domain.RunStatusFailed {
		t.Errorf("expected run status 'failed', got '%s'", updatedRun.Status)
	}
}

func TestRunExecutor_Execute_NoSandboxProvider(t *testing.T) {
	f := newTestFixtures() // Sandboxed mode requires provider
	runRepo := repository.NewMemoryRunRepository()
	runRepo.Create(context.Background(), f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	registry.Register(mockRunner)

	eventStore := event.NewMemoryStore()

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		runRepo,
		registry,
		nil, // No sandbox provider
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
	).WithConfig(config)

	ctx := context.Background()
	executor.Execute(ctx)

	// Should fail because sandboxed mode needs a provider
	outcome := executor.Outcome()
	if outcome != domain.RunOutcomeSandboxFail {
		t.Errorf("expected outcome 'sandbox_fail', got '%s'", outcome)
	}
}

// =============================================================================
// RUNNER ACQUISITION TESTS
// =============================================================================

func TestRunExecutor_Execute_RunnerNotAvailable(t *testing.T) {
	f := newInPlaceFixtures() // Skip sandbox issues
	runRepo := repository.NewMemoryRunRepository()
	runRepo.Create(context.Background(), f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(false, "resource not installed")
	registry.Register(mockRunner)

	eventStore := event.NewMemoryStore()

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		runRepo,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
	).WithConfig(config)

	ctx := context.Background()
	executor.Execute(ctx)

	// Should fail because runner is not available
	outcome := executor.Outcome()
	if outcome != domain.RunOutcomeRunnerFail {
		t.Errorf("expected outcome 'runner_fail', got '%s'", outcome)
	}
}

func TestRunExecutor_Execute_RunnerNotRegistered(t *testing.T) {
	f := newInPlaceFixtures()
	runRepo := repository.NewMemoryRunRepository()
	runRepo.Create(context.Background(), f.run)

	// Empty registry - no runners registered
	registry := runner.NewRegistry()

	eventStore := event.NewMemoryStore()

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		runRepo,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
	).WithConfig(config)

	ctx := context.Background()
	executor.Execute(ctx)

	// Should fail because runner type is not registered
	outcome := executor.Outcome()
	if outcome != domain.RunOutcomeRunnerFail {
		t.Errorf("expected outcome 'runner_fail', got '%s'", outcome)
	}
}

// =============================================================================
// EXECUTION RESULT HANDLING TESTS
// =============================================================================

func TestRunExecutor_Execute_RunnerReturnsError(t *testing.T) {
	f := newInPlaceFixtures()
	runRepo := repository.NewMemoryRunRepository()
	runRepo.Create(context.Background(), f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return nil, errors.New("execution failed")
	}
	registry.Register(mockRunner)

	eventStore := event.NewMemoryStore()

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		runRepo,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
	).WithConfig(config)

	ctx := context.Background()
	executor.Execute(ctx)

	// Should fail with exception outcome
	outcome := executor.Outcome()
	if outcome != domain.RunOutcomeException {
		t.Errorf("expected outcome 'exception', got '%s'", outcome)
	}
}

func TestRunExecutor_Execute_RunnerReturnsNonZeroExit(t *testing.T) {
	f := newInPlaceFixtures()
	runRepo := repository.NewMemoryRunRepository()
	runRepo.Create(context.Background(), f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{
			Success:      false,
			ExitCode:     1,
			ErrorMessage: "agent encountered an error",
		}, nil
	}
	registry.Register(mockRunner)

	eventStore := event.NewMemoryStore()

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		runRepo,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
	).WithConfig(config)

	ctx := context.Background()
	executor.Execute(ctx)

	// Should have exit error outcome
	outcome := executor.Outcome()
	if outcome != domain.RunOutcomeExitError {
		t.Errorf("expected outcome 'exit_error', got '%s'", outcome)
	}

	// Verify run was marked failed
	updatedRun, _ := runRepo.Get(context.Background(), f.run.ID)
	if updatedRun.Status != domain.RunStatusFailed {
		t.Errorf("expected run status 'failed', got '%s'", updatedRun.Status)
	}
}

// =============================================================================
// CONTEXT CANCELLATION TESTS
// =============================================================================

func TestRunExecutor_Execute_ContextCancelled(t *testing.T) {
	f := newInPlaceFixtures()
	runRepo := repository.NewMemoryRunRepository()
	runRepo.Create(context.Background(), f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")

	// Execution that takes a while
	var wg sync.WaitGroup
	wg.Add(1)
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		wg.Done()
		// Wait for context cancellation
		<-ctx.Done()
		return nil, ctx.Err()
	}
	registry.Register(mockRunner)

	eventStore := event.NewMemoryStore()

	config := orchestration.ExecutorConfig{
		Timeout:           30 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		runRepo,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
	).WithConfig(config)

	ctx, cancel := context.WithCancel(context.Background())

	// Run in goroutine
	done := make(chan struct{})
	go func() {
		executor.Execute(ctx)
		close(done)
	}()

	// Wait for execution to start, then cancel
	wg.Wait()
	cancel()

	// Wait for executor to finish
	select {
	case <-done:
		// Good
	case <-time.After(5 * time.Second):
		t.Fatal("executor did not complete after cancellation")
	}

	// Should be cancelled
	outcome := executor.Outcome()
	if outcome != domain.RunOutcomeCancelled {
		t.Errorf("expected outcome 'cancelled', got '%s'", outcome)
	}
}

func TestRunExecutor_Execute_ContextTimeout(t *testing.T) {
	f := newInPlaceFixtures()
	runRepo := repository.NewMemoryRunRepository()
	runRepo.Create(context.Background(), f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")

	// Execution that takes longer than timeout
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Second):
			return &runner.ExecuteResult{Success: true}, nil
		}
	}
	registry.Register(mockRunner)

	eventStore := event.NewMemoryStore()

	// Very short timeout
	config := orchestration.ExecutorConfig{
		Timeout:           200 * time.Millisecond,
		HeartbeatInterval: 50 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		runRepo,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
	).WithConfig(config)

	ctx := context.Background()
	executor.Execute(ctx)

	// Should timeout
	outcome := executor.Outcome()
	if outcome != domain.RunOutcomeTimeout {
		t.Errorf("expected outcome 'timeout', got '%s'", outcome)
	}
}

// =============================================================================
// CHECKPOINT TESTS
// =============================================================================

func TestRunExecutor_WithCheckpointRepository(t *testing.T) {
	f := newInPlaceFixtures()
	runRepo := repository.NewMemoryRunRepository()
	runRepo.Create(context.Background(), f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	registry.Register(mockRunner)

	eventStore := event.NewMemoryStore()
	checkpointRepo := repository.NewMemoryCheckpointRepository()

	config := orchestration.ExecutorConfig{
		Timeout:            5 * time.Second,
		HeartbeatInterval:  100 * time.Millisecond,
		CheckpointInterval: 50 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		runRepo,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
	).WithConfig(config).WithCheckpointRepository(checkpointRepo)

	ctx := context.Background()
	executor.Execute(ctx)

	// Verify checkpoint was saved
	checkpoint, err := checkpointRepo.Get(context.Background(), f.run.ID)
	if err != nil {
		t.Fatalf("failed to get checkpoint: %v", err)
	}
	if checkpoint == nil {
		t.Error("expected checkpoint to be saved")
	}
}

func TestRunExecutor_WithResumeFrom(t *testing.T) {
	f := newTestFixtures()
	sandboxID := uuid.New()
	workDir := "/tmp/sandbox/" + sandboxID.String()

	// Create checkpoint at runner_acquiring phase (past sandbox creation)
	checkpoint := domain.NewCheckpoint(f.run.ID, domain.RunPhaseRunnerAcquiring)
	checkpoint = checkpoint.WithSandbox(sandboxID, workDir)

	runRepo := repository.NewMemoryRunRepository()
	// Run already has sandbox ID (was created in previous attempt)
	f.run.SandboxID = &sandboxID
	runRepo.Create(context.Background(), f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	registry.Register(mockRunner)

	// Mock sandbox that allows retrieval and provides workspace path
	sandboxProvider := &mockSandboxProvider{
		createFunc: func(ctx context.Context, req sandbox.CreateRequest) (*sandbox.Sandbox, error) {
			// Should not be called when resuming past sandbox phase
			t.Error("create should not be called when resuming past sandbox phase")
			return nil, errors.New("should not create")
		},
		getFunc: func(ctx context.Context, id uuid.UUID) (*sandbox.Sandbox, error) {
			return &sandbox.Sandbox{
				ID:      id,
				Status:  sandbox.SandboxStatusActive,
				WorkDir: workDir,
			}, nil
		},
		getWorkspacePathFn: func(ctx context.Context, id uuid.UUID) (string, error) {
			return workDir, nil
		},
	}

	eventStore := event.NewMemoryStore()

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		runRepo,
		registry,
		sandboxProvider,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
	).WithConfig(config).WithResumeFrom(checkpoint)

	ctx := context.Background()
	executor.Execute(ctx)

	// Verify sandbox ID was restored from checkpoint
	if executor.SandboxID() == nil {
		t.Error("expected sandbox ID to be restored")
	}
	if *executor.SandboxID() != sandboxID {
		t.Errorf("expected sandbox ID %s, got %s", sandboxID, *executor.SandboxID())
	}

	// Verify work dir was restored
	if executor.WorkDir() != workDir {
		t.Errorf("expected work dir %s, got %s", workDir, executor.WorkDir())
	}
}

// =============================================================================
// EVENT EMISSION TESTS
// =============================================================================

func TestRunExecutor_EmitsEvents(t *testing.T) {
	f := newInPlaceFixtures()
	runRepo := repository.NewMemoryRunRepository()
	runRepo.Create(context.Background(), f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	registry.Register(mockRunner)

	eventStore := event.NewMemoryStore()

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		runRepo,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
	).WithConfig(config)

	ctx := context.Background()
	executor.Execute(ctx)

	// Verify events were emitted
	events, err := eventStore.Get(ctx, f.run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatalf("failed to get events: %v", err)
	}

	if len(events) == 0 {
		t.Error("expected events to be emitted")
	}

	// Look for phase change events
	foundPhaseEvent := false
	for _, evt := range events {
		if evt.EventType == domain.EventTypeLog {
			foundPhaseEvent = true
			break
		}
	}
	if !foundPhaseEvent {
		t.Error("expected at least one log event for phase changes")
	}
}

func TestRunExecutor_EmitsErrorEventOnFailure(t *testing.T) {
	f := newInPlaceFixtures()
	runRepo := repository.NewMemoryRunRepository()
	runRepo.Create(context.Background(), f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return nil, errors.New("execution error")
	}
	registry.Register(mockRunner)

	eventStore := event.NewMemoryStore()

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		runRepo,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
	).WithConfig(config)

	ctx := context.Background()
	executor.Execute(ctx)

	// Verify error events were emitted
	events, _ := eventStore.Get(ctx, f.run.ID, event.GetOptions{AfterSequence: -1})

	// Check for error-related events
	hasEvents := len(events) > 0
	if !hasEvents {
		t.Error("expected events to be emitted on failure")
	}
}

// =============================================================================
// RUN STATUS UPDATE TESTS
// =============================================================================

func TestRunExecutor_UpdatesRunStatus(t *testing.T) {
	f := newInPlaceFixtures()
	runRepo := repository.NewMemoryRunRepository()
	runRepo.Create(context.Background(), f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	registry.Register(mockRunner)

	eventStore := event.NewMemoryStore()

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		runRepo,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
	).WithConfig(config)

	ctx := context.Background()
	executor.Execute(ctx)

	// Verify run was updated to needs_review (success requires review)
	updatedRun, _ := runRepo.Get(context.Background(), f.run.ID)
	if updatedRun.Status != domain.RunStatusNeedsReview {
		t.Errorf("expected run status 'needs_review', got '%s'", updatedRun.Status)
	}

	// Verify StartedAt was set
	if updatedRun.StartedAt == nil {
		t.Error("expected StartedAt to be set")
	}

	// Verify EndedAt was set
	if updatedRun.EndedAt == nil {
		t.Error("expected EndedAt to be set")
	}
}

func TestRunExecutor_SetsApprovalStateOnSuccess(t *testing.T) {
	f := newInPlaceFixtures()
	runRepo := repository.NewMemoryRunRepository()
	runRepo.Create(context.Background(), f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	registry.Register(mockRunner)

	eventStore := event.NewMemoryStore()

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		runRepo,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
	).WithConfig(config)

	ctx := context.Background()
	executor.Execute(ctx)

	// Verify approval state was set to pending
	updatedRun, _ := runRepo.Get(context.Background(), f.run.ID)
	if updatedRun.ApprovalState != domain.ApprovalStatePending {
		t.Errorf("expected approval state 'pending', got '%s'", updatedRun.ApprovalState)
	}
}

// =============================================================================
// IN-PLACE MODE VALIDATION TESTS
// =============================================================================

func TestRunExecutor_InPlaceMode_MissingProjectRoot(t *testing.T) {
	f := newInPlaceFixtures()
	f.task.ProjectRoot = "" // Missing project root

	runRepo := repository.NewMemoryRunRepository()
	runRepo.Create(context.Background(), f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	registry.Register(mockRunner)

	eventStore := event.NewMemoryStore()

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		runRepo,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
	).WithConfig(config)

	ctx := context.Background()
	executor.Execute(ctx)

	// Should fail due to missing project root
	updatedRun, _ := runRepo.Get(context.Background(), f.run.ID)
	if updatedRun.Status != domain.RunStatusFailed {
		t.Errorf("expected run status 'failed', got '%s'", updatedRun.Status)
	}
}

// =============================================================================
// CONCURRENT EXECUTION TESTS
// =============================================================================

func TestRunExecutor_ConcurrentExecutions(t *testing.T) {
	// Test that multiple executors can run concurrently without issues
	const numExecutors = 5
	var wg sync.WaitGroup
	errors := make(chan error, numExecutors)

	for i := 0; i < numExecutors; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			f := newInPlaceFixtures()
			f.run.ID = uuid.New() // Unique run ID
			f.task.ID = uuid.New()
			f.profile.ID = uuid.New()

			runRepo := repository.NewMemoryRunRepository()
			runRepo.Create(context.Background(), f.run)

			registry := runner.NewRegistry()
			mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
			mockRunner.SetAvailable(true, "ready")
			mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
				time.Sleep(10 * time.Millisecond) // Simulate some work
				return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
			}
			registry.Register(mockRunner)

			eventStore := event.NewMemoryStore()

			config := orchestration.ExecutorConfig{
				Timeout:           5 * time.Second,
				HeartbeatInterval: 50 * time.Millisecond,
			}

			executor := orchestration.NewRunExecutor(
				runRepo,
				registry,
				nil,
				eventStore,
				f.run,
				f.task,
				f.profile,
				fmt.Sprintf("test prompt %d", idx),
			).WithConfig(config)

			ctx := context.Background()
			executor.Execute(ctx)

			if executor.Outcome() != domain.RunOutcomeSuccess {
				errors <- fmt.Errorf("executor %d: expected success, got %s", idx, executor.Outcome())
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}
