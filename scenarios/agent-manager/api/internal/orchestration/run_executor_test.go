package orchestration_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/repository"
	"agent-manager/internal/testutil"

	"github.com/google/uuid"
)

// =============================================================================
// MOCK BROADCASTER
// =============================================================================

// testBroadcaster implements orchestration.EventBroadcaster for tests.
type testBroadcaster struct {
	mu               sync.Mutex
	statusBroadcasts []*domain.Run
	eventBroadcasts  []*domain.RunEvent
}

func (b *testBroadcaster) BroadcastEvent(event *domain.RunEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.eventBroadcasts = append(b.eventBroadcasts, event)
}

func (b *testBroadcaster) BroadcastRunStatus(run *domain.Run) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// Copy status to avoid races with the executor mutating the run
	snapshot := *run
	b.statusBroadcasts = append(b.statusBroadcasts, &snapshot)
}

func (b *testBroadcaster) BroadcastProgress(runID uuid.UUID, phase domain.RunPhase, percent int, action string) {
	// no-op
}

func (b *testBroadcaster) getStatusBroadcasts() []*domain.Run {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]*domain.Run{}, b.statusBroadcasts...)
}

func (b *testBroadcaster) getEventBroadcasts() []*domain.RunEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]*domain.RunEvent{}, b.eventBroadcasts...)
}

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

// setupExecutorRepos creates SQLite-backed repos and seeds the parent entities
// (profile and task) from the given fixtures so that runs can be created with
// valid foreign keys.
func setupExecutorRepos(t *testing.T, f *testFixtures) (*database.Repositories, event.Store) {
	t.Helper()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	if err := repos.Profiles.Create(ctx, f.profile); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	if err := repos.Tasks.Create(ctx, f.task); err != nil {
		t.Fatalf("seed task: %v", err)
	}
	return repos, eventStore
}

func mustCreateRun(t *testing.T, repo repository.RunRepository, run *domain.Run) {
	t.Helper()
	if err := repo.Create(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
}

func mustRegisterRunnerForExecutor(t *testing.T, registry runner.Registry, r runner.Runner) {
	t.Helper()
	if err := registry.Register(r); err != nil {
		t.Fatalf("register runner: %v", err)
	}
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
	applyAtRunEndFunc  func(ctx context.Context, req sandbox.ApplyAtRunEndRequest) (*sandbox.ApplyAtRunEndResult, error)
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

func (m *mockSandboxProvider) ValidatePath(ctx context.Context, path string, projectRoot string) (*sandbox.PathValidationResult, error) {
	return &sandbox.PathValidationResult{Path: path, Valid: true}, nil
}

func (m *mockSandboxProvider) ApplyAtRunEnd(ctx context.Context, req sandbox.ApplyAtRunEndRequest) (*sandbox.ApplyAtRunEndResult, error) {
	if m.applyAtRunEndFunc != nil {
		return m.applyAtRunEndFunc(ctx, req)
	}
	return &sandbox.ApplyAtRunEndResult{Success: true, AppliedAt: time.Now()}, nil
}

func (m *mockSandboxProvider) ExecProcess(_ context.Context, _ sandbox.ExecProcessRequest) (*sandbox.ExecProcessResult, error) {
	return &sandbox.ExecProcessResult{ExitCode: 0}, nil
}

// =============================================================================
// EXECUTOR CONFIG TESTS
// =============================================================================

func TestDefaultExecutorConfig(t *testing.T) {
	config := orchestration.DefaultExecutorConfig()

	if config.Timeout != 60*time.Minute {
		t.Errorf("expected default timeout 60m, got %v", config.Timeout)
	}
	if config.HeartbeatInterval != 15*time.Second {
		t.Errorf("expected default heartbeat interval 15s, got %v", config.HeartbeatInterval)
	}
	if config.CheckpointInterval != 1*time.Minute {
		t.Errorf("expected default checkpoint interval 1m, got %v", config.CheckpointInterval)
	}
	if config.MaxRetries != 3 {
		t.Errorf("expected default max retries 3, got %d", config.MaxRetries)
	}
	if config.StaleThreshold != 5*time.Minute {
		t.Errorf("expected default stale threshold 5m, got %v", config.StaleThreshold)
	}
}

// =============================================================================
// EXECUTOR CREATION TESTS
// =============================================================================

func TestNewRunExecutor(t *testing.T) {
	f := newTestFixtures()
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	sandboxProvider := newMockSandboxProvider()

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		sandboxProvider,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"", // no system prompt
	)

	if executor == nil {
		t.Fatal("expected executor, got nil")
	}
}

func TestRunExecutor_WithConfig(t *testing.T) {
	f := newTestFixtures()
	repos, eventStore := setupExecutorRepos(t, f)
	registry := runner.NewRegistry()

	customConfig := orchestration.ExecutorConfig{
		Timeout:           5 * time.Minute,
		HeartbeatInterval: 10 * time.Second,
		MaxRetries:        5,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"", // no system prompt
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
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

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
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	sandboxProvider := newMockSandboxProvider()

	// Use short timeout for test
	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
		MaxRetries:        1,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		sandboxProvider,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"", // no system prompt
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
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{
			Success:  true,
			ExitCode: 0,
		}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		nil, // No sandbox provider for in-place
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"", // no system prompt
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
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	// Mock sandbox provider that fails
	sandboxProvider := &mockSandboxProvider{
		createFunc: func(ctx context.Context, req sandbox.CreateRequest) (*sandbox.Sandbox, error) {
			return nil, errors.New("sandbox service unavailable")
		},
	}

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		sandboxProvider,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"", // no system prompt
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
	updatedRun, _ := repos.Runs.Get(context.Background(), f.run.ID)
	if updatedRun.Status != domain.RunStatusFailed {
		t.Errorf("expected run status 'failed', got '%s'", updatedRun.Status)
	}
}

func TestRunExecutor_Execute_NoSandboxProvider(t *testing.T) {
	f := newTestFixtures() // Sandboxed mode requires provider
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		nil, // No sandbox provider
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"", // no system prompt
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
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(false, "resource not installed")
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"", // no system prompt
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
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	// Empty registry - no runners registered
	registry := runner.NewRegistry()

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"", // no system prompt
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
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return nil, errors.New("execution failed")
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"", // no system prompt
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
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

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
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"", // no system prompt
	).WithConfig(config)

	ctx := context.Background()
	executor.Execute(ctx)

	// Should have exit error outcome
	outcome := executor.Outcome()
	if outcome != domain.RunOutcomeExitError {
		t.Errorf("expected outcome 'exit_error', got '%s'", outcome)
	}

	// Verify run was marked failed
	updatedRun, _ := repos.Runs.Get(context.Background(), f.run.ID)
	if updatedRun.Status != domain.RunStatusFailed {
		t.Errorf("expected run status 'failed', got '%s'", updatedRun.Status)
	}
}

// =============================================================================
// CONTEXT CANCELLATION TESTS
// =============================================================================

func TestRunExecutor_Execute_ContextCancelled(t *testing.T) {
	f := newInPlaceFixtures()
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

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
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	config := orchestration.ExecutorConfig{
		Timeout:           30 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"", // no system prompt
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
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")

	// Simulate real runner behavior: returns result with session ID even on timeout.
	// The real ClaudeCodeRunner.Execute captures session ID from stream events
	// and returns (result, nil), not (nil, ctx.Err()).
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		<-ctx.Done()
		return &runner.ExecuteResult{
			Success:   false,
			ExitCode:  -1,
			SessionID: "sess-from-timeout",
		}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	// Very short timeout
	config := orchestration.ExecutorConfig{
		Timeout:           200 * time.Millisecond,
		HeartbeatInterval: 50 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"", // no system prompt
	).WithConfig(config)

	ctx := context.Background()
	executor.Execute(ctx)

	// Should timeout
	outcome := executor.Outcome()
	if outcome != domain.RunOutcomeTimeout {
		t.Errorf("expected outcome 'timeout', got '%s'", outcome)
	}
}

func TestRunExecutor_Execute_TimeoutPreservesSessionID(t *testing.T) {
	f := newInPlaceFixtures()
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")

	// Simulate real runner: returns result with session ID even on timeout
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		<-ctx.Done()
		return &runner.ExecuteResult{
			Success:   false,
			ExitCode:  -1,
			SessionID: "sess-timeout-preserve",
		}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	config := orchestration.ExecutorConfig{
		Timeout:           200 * time.Millisecond,
		HeartbeatInterval: 50 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"",
	).WithConfig(config)

	executor.Execute(context.Background())

	if outcome := executor.Outcome(); outcome != domain.RunOutcomeTimeout {
		t.Fatalf("expected outcome 'timeout', got '%s'", outcome)
	}

	// Verify session ID was persisted to the run in the DB
	updatedRun, err := repos.Runs.Get(context.Background(), f.run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if updatedRun.SessionID != "sess-timeout-preserve" {
		t.Errorf("expected session ID 'sess-timeout-preserve', got %q", updatedRun.SessionID)
	}
	if updatedRun.Status != domain.RunStatusFailed {
		t.Errorf("expected status 'failed', got %q", updatedRun.Status)
	}

	// Verify the run is now continuable
	canContinue, reason := domain.CanContinueRun(updatedRun)
	if !canContinue {
		t.Errorf("expected timed-out run with session ID to be continuable, got reason: %s", reason)
	}
}

func TestRunExecutor_Execute_TimeoutNoSessionID(t *testing.T) {
	f := newInPlaceFixtures()
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")

	// Runner returns result without session ID (session event never received)
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		<-ctx.Done()
		return &runner.ExecuteResult{
			Success:  false,
			ExitCode: -1,
			// SessionID intentionally empty
		}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	config := orchestration.ExecutorConfig{
		Timeout:           200 * time.Millisecond,
		HeartbeatInterval: 50 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"",
	).WithConfig(config)

	executor.Execute(context.Background())

	// Verify session ID remains empty — no false positive
	updatedRun, err := repos.Runs.Get(context.Background(), f.run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if updatedRun.SessionID != "" {
		t.Errorf("expected empty session ID, got %q", updatedRun.SessionID)
	}

	// Verify run is NOT continuable without session ID
	canContinue, _ := domain.CanContinueRun(updatedRun)
	if canContinue {
		t.Error("expected run without session ID to not be continuable")
	}
}

// =============================================================================
// CHECKPOINT TESTS
// =============================================================================

func TestRunExecutor_WithCheckpointRepository(t *testing.T) {
	f := newInPlaceFixtures()
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	config := orchestration.ExecutorConfig{
		Timeout:            5 * time.Second,
		HeartbeatInterval:  100 * time.Millisecond,
		CheckpointInterval: 50 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"", // no system prompt
	).WithConfig(config).WithCheckpointRepository(repos.Checkpoints)

	ctx := context.Background()
	executor.Execute(ctx)

	// Verify checkpoint was saved
	checkpoint, err := repos.Checkpoints.Get(context.Background(), f.run.ID)
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

	// Run already has sandbox ID (was created in previous attempt)
	f.run.SandboxID = &sandboxID
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

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

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		sandboxProvider,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"", // no system prompt
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
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"", // no system prompt
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
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return nil, errors.New("execution error")
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"", // no system prompt
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
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"", // no system prompt
	).WithConfig(config)

	ctx := context.Background()
	executor.Execute(ctx)

	// In-place runs auto-complete (no sandbox to diff/approve against)
	updatedRun, _ := repos.Runs.Get(context.Background(), f.run.ID)
	if updatedRun.Status != domain.RunStatusComplete {
		t.Errorf("expected in-place run status 'complete', got '%s'", updatedRun.Status)
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
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"", // no system prompt
	).WithConfig(config)

	ctx := context.Background()
	executor.Execute(ctx)

	// In-place runs skip the approval workflow entirely (no sandbox to
	// diff/approve against), so approval state should be "none".
	updatedRun, _ := repos.Runs.Get(context.Background(), f.run.ID)
	if updatedRun.ApprovalState != domain.ApprovalStateNone {
		t.Errorf("expected approval state 'none' for in-place run, got '%s'", updatedRun.ApprovalState)
	}
}

// =============================================================================
// IN-PLACE MODE VALIDATION TESTS
// =============================================================================

func TestRunExecutor_InPlaceMode_MissingProjectRoot(t *testing.T) {
	f := newInPlaceFixtures()
	f.task.ProjectRoot = "" // Missing project root

	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs,
		registry,
		nil,
		eventStore,
		f.run,
		f.task,
		f.profile,
		"test prompt",
		"", // no system prompt
	).WithConfig(config)

	ctx := context.Background()
	executor.Execute(ctx)

	// Should fail due to missing project root
	updatedRun, _ := repos.Runs.Get(context.Background(), f.run.ID)
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
			f.run.TaskID = f.task.ID
			f.profile.ID = uuid.New()
			profileID := f.profile.ID
			f.run.AgentProfileID = &profileID

			repos, eventStore := setupExecutorRepos(t, f)
			mustCreateRun(t, repos.Runs, f.run)

			registry := runner.NewRegistry()
			mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
			mockRunner.SetAvailable(true, "ready")
			mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
				time.Sleep(10 * time.Millisecond) // Simulate some work
				return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
			}
			mustRegisterRunnerForExecutor(t, registry, mockRunner)

			config := orchestration.ExecutorConfig{
				Timeout:           5 * time.Second,
				HeartbeatInterval: 50 * time.Millisecond,
			}

			executor := orchestration.NewRunExecutor(
				repos.Runs,
				registry,
				nil,
				eventStore,
				f.run,
				f.task,
				f.profile,
				fmt.Sprintf("test prompt %d", idx),
				"", // no system prompt
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

// =============================================================================
// SANDBOX ENVIRONMENT VARIABLE INJECTION
//
// These tests verify that SandboxEnvVars() correctly produces the environment
// variables that enable sandbox-aware scenario lifecycle commands. The Vrooli
// CLI (scripts/lib/scenario/runner.sh) reads these variables to transparently
// redirect scenario path resolution to the sandbox's merged/ directory, so
// agents can restart scenarios and see their own file changes.
// =============================================================================

// TestSandboxEnvVars_Sandboxed verifies that a fully-configured sandboxed run
// produces all three environment variables: VROOLI_SANDBOX_ID (for logging),
// VROOLI_SANDBOX_MERGED (the overlay path the CLI redirects to), and
// VROOLI_SANDBOX_SCOPE (which scenarios to redirect).
func TestSandboxEnvVars_Sandboxed(t *testing.T) {
	f := newTestFixtures()
	f.task.ScopePath = "scenarios/my-scenario"
	f.run.RunMode = domain.RunModeSandboxed

	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	executor := orchestration.NewRunExecutor(
		repos.Runs, nil, nil, eventStore, f.run, f.task, f.profile, "test prompt", "",
	)

	sandboxID := uuid.New()
	executor.WithExistingSandbox(sandboxID, "/tmp/sandbox/abc123/merged")

	vars := executor.SandboxEnvVars()
	if vars == nil {
		t.Fatal("expected non-nil env vars for sandboxed run")
	}
	if got := vars["VROOLI_SANDBOX_ID"]; got != sandboxID.String() {
		t.Errorf("VROOLI_SANDBOX_ID = %q, want %q", got, sandboxID.String())
	}
	if got := vars["VROOLI_SANDBOX_MERGED"]; got != "/tmp/sandbox/abc123/merged" {
		t.Errorf("VROOLI_SANDBOX_MERGED = %q, want %q", got, "/tmp/sandbox/abc123/merged")
	}
	if got := vars["VROOLI_SANDBOX_SCOPE"]; got != "scenarios/my-scenario" {
		t.Errorf("VROOLI_SANDBOX_SCOPE = %q, want %q", got, "scenarios/my-scenario")
	}
}

// TestSandboxEnvVars_InPlace verifies that in-place (non-sandboxed) runs produce
// no sandbox env vars, since the agent is working directly on the real repo and
// the CLI should use normal path resolution.
func TestSandboxEnvVars_InPlace(t *testing.T) {
	f := newInPlaceFixtures()

	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	executor := orchestration.NewRunExecutor(
		repos.Runs, nil, nil, eventStore, f.run, f.task, f.profile, "test prompt", "",
	)

	vars := executor.SandboxEnvVars()
	if vars != nil {
		t.Errorf("expected nil env vars for in-place run, got %v", vars)
	}
}

// TestSandboxEnvVars_NoSandboxID verifies that sandboxed runs return nil before
// sandbox creation completes (sandboxID not yet assigned). This prevents the CLI
// from attempting redirection when there's no sandbox to redirect to.
func TestSandboxEnvVars_NoSandboxID(t *testing.T) {
	f := newTestFixtures()
	f.run.RunMode = domain.RunModeSandboxed

	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	executor := orchestration.NewRunExecutor(
		repos.Runs, nil, nil, eventStore, f.run, f.task, f.profile, "test prompt", "",
	)
	// Don't call WithExistingSandbox — sandboxID stays nil

	vars := executor.SandboxEnvVars()
	if vars != nil {
		t.Errorf("expected nil env vars when sandboxID is nil, got %v", vars)
	}
}

// TestSandboxEnvVars_EmptyScopePath verifies that when ScopePath is empty,
// VROOLI_SANDBOX_SCOPE is omitted from the env vars but the other two are still
// present. This handles edge cases where the sandbox covers the entire project
// root — the CLI will interpret the missing scope as "everything is in scope".
func TestSandboxEnvVars_EmptyScopePath(t *testing.T) {
	f := newTestFixtures()
	f.task.ScopePath = "" // Empty scope — whole project
	f.run.RunMode = domain.RunModeSandboxed

	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	executor := orchestration.NewRunExecutor(
		repos.Runs, nil, nil, eventStore, f.run, f.task, f.profile, "test prompt", "",
	)

	sandboxID := uuid.New()
	executor.WithExistingSandbox(sandboxID, "/tmp/sandbox/abc123/merged")

	vars := executor.SandboxEnvVars()
	if vars == nil {
		t.Fatal("expected non-nil env vars for sandboxed run with empty scope")
	}
	if got := vars["VROOLI_SANDBOX_ID"]; got != sandboxID.String() {
		t.Errorf("VROOLI_SANDBOX_ID = %q, want %q", got, sandboxID.String())
	}
	if got := vars["VROOLI_SANDBOX_MERGED"]; got != "/tmp/sandbox/abc123/merged" {
		t.Errorf("VROOLI_SANDBOX_MERGED = %q, want %q", got, "/tmp/sandbox/abc123/merged")
	}
	if _, exists := vars["VROOLI_SANDBOX_SCOPE"]; exists {
		t.Error("VROOLI_SANDBOX_SCOPE should be omitted when ScopePath is empty")
	}
}

// =============================================================================
// MERGED ENV VARS TESTS
// =============================================================================

func TestMergedEnvVars_CustomOnly(t *testing.T) {
	f := newInPlaceFixtures()
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	executor := orchestration.NewRunExecutor(
		repos.Runs, nil, nil, eventStore, f.run, f.task, f.profile, "test", "",
	)
	executor.WithCustomEnvironment(map[string]string{
		"VROOLI_SPAWN_SOURCE": "research/my-research",
	})

	vars := executor.MergedEnvVars()
	if vars == nil {
		t.Fatal("expected non-nil env vars")
	}
	if got := vars["VROOLI_SPAWN_SOURCE"]; got != "research/my-research" {
		t.Errorf("VROOLI_SPAWN_SOURCE = %q, want %q", got, "research/my-research")
	}
}

func TestMergedEnvVars_SandboxOnly(t *testing.T) {
	f := newTestFixtures()
	f.run.RunMode = domain.RunModeSandboxed
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	executor := orchestration.NewRunExecutor(
		repos.Runs, nil, nil, eventStore, f.run, f.task, f.profile, "test", "",
	)
	sandboxID := uuid.New()
	executor.WithExistingSandbox(sandboxID, "/tmp/sandbox/merged")

	vars := executor.MergedEnvVars()
	if vars == nil {
		t.Fatal("expected non-nil env vars")
	}
	if _, exists := vars["VROOLI_SANDBOX_ID"]; !exists {
		t.Error("expected VROOLI_SANDBOX_ID")
	}
}

func TestMergedEnvVars_SandboxOverridesCustom(t *testing.T) {
	f := newTestFixtures()
	f.run.RunMode = domain.RunModeSandboxed
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	executor := orchestration.NewRunExecutor(
		repos.Runs, nil, nil, eventStore, f.run, f.task, f.profile, "test", "",
	)
	sandboxID := uuid.New()
	executor.WithExistingSandbox(sandboxID, "/tmp/sandbox/merged")
	executor.WithCustomEnvironment(map[string]string{
		"VROOLI_SANDBOX_MERGED": "attacker-path",
		"VROOLI_SPAWN_SOURCE":   "research/my-research",
	})

	vars := executor.MergedEnvVars()
	if vars == nil {
		t.Fatal("expected non-nil env vars")
	}
	// Sandbox var must win over custom var
	if got := vars["VROOLI_SANDBOX_MERGED"]; got != "/tmp/sandbox/merged" {
		t.Errorf("VROOLI_SANDBOX_MERGED = %q, want sandbox value %q", got, "/tmp/sandbox/merged")
	}
	// Custom var should still be present
	if got := vars["VROOLI_SPAWN_SOURCE"]; got != "research/my-research" {
		t.Errorf("VROOLI_SPAWN_SOURCE = %q, want %q", got, "research/my-research")
	}
}

func TestMergedEnvVars_BothNil(t *testing.T) {
	f := newInPlaceFixtures()
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	executor := orchestration.NewRunExecutor(
		repos.Runs, nil, nil, eventStore, f.run, f.task, f.profile, "test", "",
	)
	// No custom env, no sandbox (in-place mode)
	vars := executor.MergedEnvVars()
	if vars != nil {
		t.Errorf("expected nil env vars, got %v", vars)
	}
}

// =============================================================================
// BROADCAST ON COMPLETION TESTS (Bug 1 fix validation)
// =============================================================================

func TestRunExecutor_BroadcastsStatusOnSuccess(t *testing.T) {
	f := newInPlaceFixtures()
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	broadcaster := &testBroadcaster{}

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, nil, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithConfig(config).WithBroadcaster(broadcaster)

	executor.Execute(context.Background())

	broadcasts := broadcaster.getStatusBroadcasts()
	if len(broadcasts) == 0 {
		t.Fatal("expected at least one status broadcast on successful completion")
	}

	// The last broadcast should have the terminal status.
	// In-place runs auto-complete (no sandbox to diff/approve against).
	last := broadcasts[len(broadcasts)-1]
	if last.Status != domain.RunStatusComplete {
		t.Errorf("expected final broadcast status complete for in-place run, got %s", last.Status)
	}
}

func TestRunExecutor_BroadcastsStatusOnFailure(t *testing.T) {
	f := newInPlaceFixtures()
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return nil, errors.New("execution failed")
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	broadcaster := &testBroadcaster{}

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, nil, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithConfig(config).WithBroadcaster(broadcaster)

	executor.Execute(context.Background())

	broadcasts := broadcaster.getStatusBroadcasts()
	if len(broadcasts) == 0 {
		t.Fatal("expected at least one status broadcast on failure")
	}

	last := broadcasts[len(broadcasts)-1]
	if last.Status != domain.RunStatusFailed {
		t.Errorf("expected final broadcast status failed, got %s", last.Status)
	}
}

func TestRunExecutor_BroadcastsStatusOnCancellation(t *testing.T) {
	f := newInPlaceFixtures()
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")

	var wg sync.WaitGroup
	wg.Add(1)
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		wg.Done()
		<-ctx.Done()
		return nil, ctx.Err()
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	broadcaster := &testBroadcaster{}

	config := orchestration.ExecutorConfig{
		Timeout:           30 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, nil, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithConfig(config).WithBroadcaster(broadcaster)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		executor.Execute(ctx)
		close(done)
	}()

	wg.Wait()
	cancel()
	<-done

	broadcasts := broadcaster.getStatusBroadcasts()
	if len(broadcasts) == 0 {
		t.Fatal("expected at least one status broadcast on cancellation")
	}

	last := broadcasts[len(broadcasts)-1]
	if last.Status != domain.RunStatusCancelled {
		t.Errorf("expected final broadcast status cancelled, got %s", last.Status)
	}
}

func TestRunExecutor_NoBroadcaster_NoPanic(t *testing.T) {
	// Verify that nil broadcaster doesn't cause a panic
	f := newInPlaceFixtures()
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	// No WithBroadcaster call — broadcaster stays nil
	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, nil, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithConfig(config)

	executor.Execute(context.Background())
	// If we get here without panic, the nil guard works
}

// =============================================================================
// IN-PLACE RUN APPROVAL BYPASS TESTS
// =============================================================================
// In-place runs apply changes directly to the working tree (no sandbox).
// Since there is no sandbox to diff against or merge from, the approval
// workflow is skipped entirely and the run auto-completes.

func TestRunExecutor_InPlace_SkipsApproval(t *testing.T) {
	// An in-place run should auto-complete because there is no sandbox to
	// diff against — the approval / apply workflow doesn't apply.
	f := newInPlaceFixtures()
	f.run.ResolvedConfig = &domain.RunConfig{
		RunnerType: domain.RunnerTypeClaudeCode,
	}
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, nil, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithConfig(config)

	executor.Execute(context.Background())

	updatedRun, err := repos.Runs.Get(context.Background(), f.run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if updatedRun.Status != domain.RunStatusComplete {
		t.Errorf("expected in-place run status 'complete', got '%s'", updatedRun.Status)
	}
	if updatedRun.ApprovalState != domain.ApprovalStateNone {
		t.Errorf("expected approval state 'none' for in-place run, got '%s'", updatedRun.ApprovalState)
	}
}

func TestRunExecutor_Sandboxed_ManualReviewDefersApply(t *testing.T) {
	// Per the auditability contract, the only way a sandboxed run lands
	// in NeedsReview/Pending after success is ManualReview=true. The
	// contract is "auto-apply by default unless operator opts into manual
	// review".
	f := newTestFixtures() // sandboxed mode
	manualReviewCfg := domain.DefaultSandboxConfig()
	manualReviewCfg.ManualReview = true
	f.run.SandboxConfig = manualReviewCfg
	f.run.ResolvedConfig = &domain.RunConfig{
		RunnerType:    domain.RunnerTypeClaudeCode,
		SandboxConfig: manualReviewCfg,
	}
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	sandboxProvider := newMockSandboxProvider()
	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, sandboxProvider, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithConfig(config)

	executor.Execute(context.Background())

	updatedRun, err := repos.Runs.Get(context.Background(), f.run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if updatedRun.Status != domain.RunStatusNeedsReview {
		t.Errorf("expected sandboxed run status 'needs_review', got '%s'", updatedRun.Status)
	}
	if updatedRun.ApprovalState != domain.ApprovalStatePending {
		t.Errorf("expected approval state 'pending' for sandboxed run, got '%s'", updatedRun.ApprovalState)
	}
}

func TestRunExecutor_Sandboxed_DefaultAutoApplies_Completes(t *testing.T) {
	// Sandboxed runs with the contract defaults (AutoApply=true,
	// ManualReview=false) should auto-apply at run end and land in Complete
	// with ApprovalState=Approved.
	f := newTestFixtures() // sandboxed mode
	f.run.SandboxConfig = domain.DefaultSandboxConfig()
	f.run.ResolvedConfig = &domain.RunConfig{
		RunnerType:    domain.RunnerTypeClaudeCode,
		SandboxConfig: f.run.SandboxConfig,
	}
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	sandboxProvider := newMockSandboxProvider()
	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, sandboxProvider, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithConfig(config)

	executor.Execute(context.Background())

	updatedRun, err := repos.Runs.Get(context.Background(), f.run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if updatedRun.Status != domain.RunStatusComplete {
		t.Errorf("expected status 'complete', got '%s'", updatedRun.Status)
	}
	if updatedRun.ApprovalState != domain.ApprovalStateApproved {
		t.Errorf("expected approval state 'approved' (auto-applied), got '%s'", updatedRun.ApprovalState)
	}
}

func TestRunExecutor_InPlace_EmitsSkipApplyEvent(t *testing.T) {
	// Verify that in-place runs emit a system event explaining the
	// approval skip, so operators can trace the decision.
	f := newInPlaceFixtures()
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, nil, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithConfig(config)

	executor.Execute(context.Background())

	// Check that a system event was emitted explaining the approval skip
	events, err := eventStore.Get(context.Background(), f.run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatalf("get events: %v", err)
	}

	found := false
	for _, evt := range events {
		if evt.EventType == domain.EventTypeLog {
			if logData, ok := evt.Data.(*domain.LogEventData); ok {
				if logData.Level == "info" &&
					strings.Contains(logData.Message, "in-place run completed") &&
					strings.Contains(logData.Message, "skipping apply") {
					found = true
					break
				}
			}
		}
	}
	if !found {
		t.Error("expected system event explaining in-place approval skip, but none found")
	}
}

func TestRunExecutor_BroadcastsPostRunnerEvents(t *testing.T) {
	// Verify that system events emitted after the runner finishes
	// (phase changes, completion messages) are broadcast via WebSocket,
	// not just stored to the database. This ensures real-time UI updates
	// for post-execution events.
	f := newInPlaceFixtures()
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	broadcaster := &testBroadcaster{}

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, nil, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithConfig(config).WithBroadcaster(broadcaster)

	executor.Execute(context.Background())

	eventBroadcasts := broadcaster.getEventBroadcasts()
	if len(eventBroadcasts) == 0 {
		t.Fatal("expected post-runner events to be broadcast via WebSocket")
	}

	// Verify that at least one log event (system event) was broadcast.
	// These are the events emitted by emitSystemEvent (phase changes, etc.)
	foundLogBroadcast := false
	for _, evt := range eventBroadcasts {
		if evt.EventType == domain.EventTypeLog {
			foundLogBroadcast = true
			break
		}
	}
	if !foundLogBroadcast {
		t.Error("expected at least one log event to be broadcast, but none found")
	}
}

func TestRunExecutor_BroadcastsErrorEventsOnFailure(t *testing.T) {
	// Verify that error events emitted when a runner fails are broadcast
	// via WebSocket so the UI can show failure details in real-time.
	f := newInPlaceFixtures()
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return nil, fmt.Errorf("agent crashed unexpectedly")
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	broadcaster := &testBroadcaster{}

	config := orchestration.ExecutorConfig{
		Timeout:           5 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
	}

	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, nil, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithConfig(config).WithBroadcaster(broadcaster)

	executor.Execute(context.Background())

	eventBroadcasts := broadcaster.getEventBroadcasts()

	// Verify that an error event was broadcast
	foundErrorBroadcast := false
	for _, evt := range eventBroadcasts {
		if evt.EventType == domain.EventTypeError {
			foundErrorBroadcast = true
			break
		}
	}
	if !foundErrorBroadcast {
		t.Error("expected error event to be broadcast on runner failure, but none found")
	}
}
