package orchestration_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	cfgpkg "agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/fixtures"
	"agent-manager/internal/orchestration/testutil/mocks"
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
func newTestFixtures(t *testing.T) *testFixtures {
	t.Helper()
	profile := fixtures.NewAgentProfile(t,
		fixtures.WithAgentProfileName("test-profile"),
	)
	task := fixtures.NewTask(t,
		fixtures.WithTaskDescription("A test task for executor tests"),
	)

	return &testFixtures{
		profile: profile,
		task:    task,
		run:     fixtures.NewRun(t, task.ID, profile.ID),
	}
}

// newInPlaceFixtures creates fixtures for in-place execution.
func newInPlaceFixtures(t *testing.T) *testFixtures {
	t.Helper()
	f := newTestFixtures(t)
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

func newMockSandboxProvider() *mocks.FakeSandboxProvider {
	return mocks.NewFakeSandboxProvider()
}

type testBroadcaster struct {
	*mocks.FakeBroadcaster
}

func (b *testBroadcaster) ensure() *mocks.FakeBroadcaster {
	if b.FakeBroadcaster == nil {
		b.FakeBroadcaster = mocks.NewFakeBroadcaster()
	}
	return b.FakeBroadcaster
}

func (b *testBroadcaster) BroadcastEvent(event *domain.RunEvent) {
	b.ensure().BroadcastEvent(event)
}

func (b *testBroadcaster) BroadcastRunStatus(run *domain.Run) {
	b.ensure().BroadcastRunStatus(run)
}

func (b *testBroadcaster) BroadcastProgress(runID uuid.UUID, phase domain.RunPhase, percent int, action string) {
	b.ensure().BroadcastProgress(runID, phase, percent, action)
}

func (b *testBroadcaster) getStatusBroadcasts() []*domain.Run {
	return b.ensure().StatusBroadcasts()
}

// =============================================================================
// EXECUTOR CREATION TESTS
// =============================================================================

func TestNewRunExecutor(t *testing.T) {
	f := newTestFixtures(t)
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

func TestRunExecutor_WithLevers(t *testing.T) {
	f := newTestFixtures(t)
	repos, eventStore := setupExecutorRepos(t, f)
	registry := runner.NewRegistry()

	customLevers := cfgpkg.DefaultLevers()
	customLevers.Execution.DefaultTimeout = 5 * time.Minute
	customLevers.Heartbeat.RunHeartbeatInterval = 10 * time.Second
	customLevers.Heartbeat.MaxRetriesPerPhase = 5

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
	).WithLevers(customLevers)

	if executor == nil {
		t.Fatal("expected executor, got nil")
	}
}

// =============================================================================
// SANDBOX WORKSPACE SETUP TESTS
// =============================================================================

func TestRunExecutor_Execute_SandboxedMode_Success(t *testing.T) {
	f := newTestFixtures(t)
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
	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond
	levers.Heartbeat.MaxRetriesPerPhase = 1

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
	).WithLevers(levers)

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
	f := newInPlaceFixtures(t)
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

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

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
	).WithLevers(levers)

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
	f := newTestFixtures(t)
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	// Mock sandbox provider that fails
	sandboxProvider := &mocks.FakeSandboxProvider{
		CreateFunc: func(ctx context.Context, req sandbox.CreateRequest) (*sandbox.Sandbox, error) {
			return nil, errors.New("sandbox service unavailable")
		},
	}

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

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
	).WithLevers(levers)

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
	f := newTestFixtures(t) // Sandboxed mode requires provider
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

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
	).WithLevers(levers)

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
	f := newInPlaceFixtures(t) // Skip sandbox issues
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(false, "resource not installed")
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

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
	).WithLevers(levers)

	ctx := context.Background()
	executor.Execute(ctx)

	// Should fail because runner is not available
	outcome := executor.Outcome()
	if outcome != domain.RunOutcomeRunnerFail {
		t.Errorf("expected outcome 'runner_fail', got '%s'", outcome)
	}
}

func TestRunExecutor_Execute_RunnerNotRegistered(t *testing.T) {
	f := newInPlaceFixtures(t)
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	// Empty registry - no runners registered
	registry := runner.NewRegistry()

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

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
	).WithLevers(levers)

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
	f := newInPlaceFixtures(t)
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return nil, errors.New("execution failed")
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

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
	).WithLevers(levers)

	ctx := context.Background()
	executor.Execute(ctx)

	// Should fail with exception outcome
	outcome := executor.Outcome()
	if outcome != domain.RunOutcomeException {
		t.Errorf("expected outcome 'exception', got '%s'", outcome)
	}
}

func TestRunExecutor_Execute_RunnerReturnsNonZeroExit(t *testing.T) {
	f := newInPlaceFixtures(t)
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

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

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
	).WithLevers(levers)

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
	f := newInPlaceFixtures(t)
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

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 30 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

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
	).WithLevers(levers)

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
	f := newInPlaceFixtures(t)
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")

	// Simulate real runner behavior: returns result with session ID even on timeout.
	// The real claude codec captures session ID from stream events and the core
	// Runner returns (result, nil), not (nil, ctx.Err()).
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
	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 200 * time.Millisecond
	levers.Heartbeat.RunHeartbeatInterval = 50 * time.Millisecond

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
	).WithLevers(levers)

	ctx := context.Background()
	executor.Execute(ctx)

	// Should timeout
	outcome := executor.Outcome()
	if outcome != domain.RunOutcomeTimeout {
		t.Errorf("expected outcome 'timeout', got '%s'", outcome)
	}
}

func TestRunExecutor_Execute_UsesResolvedRunTimeout(t *testing.T) {
	f := newInPlaceFixtures(t)
	f.run.ResolvedConfig = &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode, Timeout: 100 * time.Millisecond}
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 25 * time.Millisecond
	executor := orchestration.NewRunExecutor(repos.Runs, registry, nil, eventStore, f.run, f.task, f.profile, "test prompt", "").WithLevers(levers).WithRunStateRoot(t.TempDir())

	started := time.Now()
	executor.Execute(context.Background())
	if elapsed := time.Since(started); elapsed >= time.Second {
		t.Fatalf("executor ignored resolved timeout: ran for %s", elapsed)
	}
	if got := executor.Outcome(); got != domain.RunOutcomeTimeout {
		t.Fatalf("outcome = %q, want timeout", got)
	}
}

func TestRunExecutor_Execute_TimeoutPreservesSessionID(t *testing.T) {
	f := newInPlaceFixtures(t)
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

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 200 * time.Millisecond
	levers.Heartbeat.RunHeartbeatInterval = 50 * time.Millisecond

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
	).WithLevers(levers)

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
	f := newInPlaceFixtures(t)
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

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 200 * time.Millisecond
	levers.Heartbeat.RunHeartbeatInterval = 50 * time.Millisecond

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
	).WithLevers(levers)

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
	f := newInPlaceFixtures(t)
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond
	levers.Heartbeat.CheckpointInterval = 50 * time.Millisecond

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
	).WithLevers(levers).WithCheckpointRepository(repos.Checkpoints)

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
	f := newTestFixtures(t)
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
	sandboxProvider := &mocks.FakeSandboxProvider{
		CreateFunc: func(ctx context.Context, req sandbox.CreateRequest) (*sandbox.Sandbox, error) {
			// Should not be called when resuming past sandbox phase
			t.Error("create should not be called when resuming past sandbox phase")
			return nil, errors.New("should not create")
		},
		GetFunc: func(ctx context.Context, id uuid.UUID) (*sandbox.Sandbox, error) {
			return &sandbox.Sandbox{
				ID:      id,
				Status:  sandbox.SandboxStatusActive,
				WorkDir: workDir,
			}, nil
		},
		GetWorkspacePathFn: func(ctx context.Context, id uuid.UUID) (string, error) {
			return workDir, nil
		},
	}

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

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
	).WithLevers(levers).WithResumeFrom(checkpoint)

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
	f := newInPlaceFixtures(t)
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

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
	).WithLevers(levers)

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
	f := newInPlaceFixtures(t)
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return nil, errors.New("execution error")
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

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
	).WithLevers(levers)

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
	f := newInPlaceFixtures(t)
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

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
	).WithLevers(levers)
	terminal := make(chan *domain.Run, 1)
	executor.WithTerminalObserver(func(run *domain.Run) { terminal <- run })

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

	select {
	case observed := <-terminal:
		if observed.ID != f.run.ID || observed.Status != domain.RunStatusComplete {
			t.Fatalf("terminal observer received %+v", observed)
		}
	case <-time.After(time.Second):
		t.Fatal("terminal observer was not called")
	}
}

func TestRunExecutor_SetsApprovalStateOnSuccess(t *testing.T) {
	f := newInPlaceFixtures(t)
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

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
	).WithLevers(levers)

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
	f := newInPlaceFixtures(t)
	f.task.ProjectRoot = "" // Missing project root

	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

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
	).WithLevers(levers)

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

			f := newInPlaceFixtures(t)
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

			levers := cfgpkg.DefaultLevers()
			levers.Execution.DefaultTimeout = 5 * time.Second
			levers.Heartbeat.RunHeartbeatInterval = 50 * time.Millisecond

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
			).WithLevers(levers)

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

// MERGED ENV VARS TESTS
// =============================================================================

func TestMergedEnvVars_CustomOnly(t *testing.T) {
	f := newInPlaceFixtures(t)
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
	f := newTestFixtures(t)
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
	f := newTestFixtures(t)
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
	f := newInPlaceFixtures(t)
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
	f := newInPlaceFixtures(t)
	repos, eventStore := setupExecutorRepos(t, f)
	mustCreateRun(t, repos.Runs, f.run)

	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "ready")
	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{Success: true, ExitCode: 0}, nil
	}
	mustRegisterRunnerForExecutor(t, registry, mockRunner)

	broadcaster := mocks.NewFakeBroadcaster()

	levers := cfgpkg.DefaultLevers()
	levers.Execution.DefaultTimeout = 5 * time.Second
	levers.Heartbeat.RunHeartbeatInterval = 100 * time.Millisecond

	executor := orchestration.NewRunExecutor(
		repos.Runs, registry, nil, eventStore,
		f.run, f.task, f.profile, "test prompt", "",
	).WithLevers(levers).WithBroadcaster(broadcaster)

	executor.Execute(context.Background())

	broadcasts := broadcaster.StatusBroadcasts()
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
