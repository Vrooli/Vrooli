package orchestration_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"

	agentconfig "agent-manager/internal/config"

	"github.com/google/uuid"
)

// newTestOrchestrationSettings creates an OrchestrationSettingsStore backed by a
// temp file with a custom RunTimeoutMinutes value. Tests can then distinguish
// the persisted fallback from a workflow node's per-turn override.
func newTestOrchestrationSettings(t *testing.T, runTimeoutMinutes int) *agentconfig.OrchestrationSettingsStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "orchestration-settings.json")
	store, err := agentconfig.NewOrchestrationSettingsStore(path)
	if err != nil {
		t.Fatalf("create orchestration settings store: %v", err)
	}
	settings := store.Get()
	settings.RunExecution.RunTimeoutMinutes = runTimeoutMinutes
	if err := store.Update(settings); err != nil {
		t.Fatalf("update orchestration settings: %v", err)
	}
	return store
}

func TestContinuation_HasPerTurnTimeout(t *testing.T) {
	ctx := context.Background()

	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "available")
	mockRunner.SetCapabilities(runner.Capabilities{
		SupportsMessages:     true,
		SupportsStreaming:    true,
		SupportsCancellation: true,
		SupportsContinuation: true,
		MaxTurns:             100,
		SupportedModels:      []string{"mock-model"},
	})

	var hadDeadline bool
	var deadlineDuration time.Duration
	var resolvedMaxTurns int
	var resolvedTimeout time.Duration
	continueDone := make(chan struct{})
	mockRunner.ContinueFunc = func(ctx context.Context, req runner.ContinueRequest) (*runner.ExecuteResult, error) {
		defer close(continueDone)
		// Verify that the context has a deadline (from the per-turn timeout)
		if deadline, ok := ctx.Deadline(); ok {
			hadDeadline = true
			deadlineDuration = time.Until(deadline)
		}
		if req.ResolvedConfig != nil {
			resolvedMaxTurns = req.ResolvedConfig.MaxTurns
			resolvedTimeout = req.ResolvedConfig.Timeout
		}
		// Return immediately — no need to wait for the actual timeout.
		// The executor-level timeout tests already cover the timeout path.
		return &runner.ExecuteResult{
			Success:   true,
			ExitCode:  0,
			SessionID: "sess-continued",
		}, nil
	}

	registry := runner.NewRegistry()
	if err := registry.Register(mockRunner); err != nil {
		t.Fatalf("register runner: %v", err)
	}

	// Configure the persisted fallback to one minute, then verify that the
	// workflow node's two-second override wins for this continuation turn.
	settingsStore := newTestOrchestrationSettings(t, 1)

	svc := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:          5 * time.Minute,
			MaxConcurrentRuns:       10,
			RequireSandboxByDefault: false,
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithCheckpoints(repos.Checkpoints),
		orchestration.WithIdempotency(repos.Idempotency),
		orchestration.WithRunners(registry),
		orchestration.WithOrchestrationSettings(settingsStore),
		orchestration.WithRunStateRoot(t.TempDir()),
	)

	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:       "timeout-test-profile",
		ProfileKey: "timeout-test-" + uuid.New().String()[:8],

		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}, RoleRef: "code.default",
	})
	task := mustCreateTask(t, svc, ctx, &domain.Task{
		Title:       "timeout-test-task",
		Description: "task for continuation timeout test",
		ScopePath:   "src/",
	})

	now := time.Now()
	runID := uuid.New()
	directRun := &domain.Run{
		ID:             runID,
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Tag:            runID.String(),
		RunMode:        domain.RunModeInPlace,
		Status:         domain.RunStatusComplete,
		Phase:          domain.RunPhaseCompleted,
		SessionID:      "sess-original",
		StartedAt:      &now,
		EndedAt:        &now,
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeClaudeCode,
		},
		ApprovalState: domain.ApprovalStateNone,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repos.Runs.Create(ctx, directRun); err != nil {
		t.Fatalf("create run: %v", err)
	}

	nodeTimeout := 2 * time.Second
	nodeMaxTurns := 3
	_, err := svc.ContinueRun(ctx, orchestration.ContinueRunRequest{
		RunID:    runID,
		Message:  "please continue",
		MaxTurns: &nodeMaxTurns,
		Timeout:  &nodeTimeout,
	})
	if err != nil {
		t.Fatalf("ContinueRun: %v", err)
	}

	select {
	case <-continueDone:
	case <-time.After(10 * time.Second):
		t.Fatal("continuation runner was never called")
	}
	waitForStatusEvents(t, ctx, eventStore, runID, 2)

	if !hadDeadline {
		t.Error("expected continuation context to have a deadline from per-turn timeout")
	}
	if deadlineDuration < time.Second || deadlineDuration > 3*time.Second {
		t.Errorf("expected workflow node deadline ~2 seconds in the future, got %v", deadlineDuration)
	}
	if resolvedMaxTurns != nodeMaxTurns || resolvedTimeout != nodeTimeout {
		t.Errorf("continuation resolved limits=(%d,%s), want (%d,%s)", resolvedMaxTurns, resolvedTimeout, nodeMaxTurns, nodeTimeout)
	}
}

func TestContinuation_FailurePreservesSessionID(t *testing.T) {
	ctx := context.Background()

	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	continueDone := make(chan struct{})
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "available")
	mockRunner.SetCapabilities(runner.Capabilities{
		SupportsMessages:     true,
		SupportsStreaming:    true,
		SupportsCancellation: true,
		SupportsContinuation: true,
		MaxTurns:             100,
		SupportedModels:      []string{"mock-model"},
	})
	mockRunner.ContinueFunc = func(ctx context.Context, req runner.ContinueRequest) (*runner.ExecuteResult, error) {
		defer close(continueDone)
		return &runner.ExecuteResult{
			Success:      false,
			ExitCode:     1,
			SessionID:    "sess-updated-after-failure",
			ErrorMessage: "something went wrong",
		}, nil
	}

	registry := runner.NewRegistry()
	if err := registry.Register(mockRunner); err != nil {
		t.Fatalf("register runner: %v", err)
	}

	svc := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:          5 * time.Minute,
			MaxConcurrentRuns:       10,
			RequireSandboxByDefault: false,
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithCheckpoints(repos.Checkpoints),
		orchestration.WithIdempotency(repos.Idempotency),
		orchestration.WithRunners(registry),
		orchestration.WithRunStateRoot(t.TempDir()),
	)

	// Create profile and task
	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:       "fail-session-profile",
		ProfileKey: "fail-session-" + uuid.New().String()[:8],

		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}, RoleRef: "code.default",
	})
	task := mustCreateTask(t, svc, ctx, &domain.Task{
		Title:       "fail-session-task",
		Description: "task for continuation failure session ID test",
		ScopePath:   "src/",
	})

	// Insert a continuable run
	now := time.Now()
	runID := uuid.New()
	directRun := &domain.Run{
		ID:             runID,
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Tag:            runID.String(),
		RunMode:        domain.RunModeInPlace,
		Status:         domain.RunStatusComplete,
		Phase:          domain.RunPhaseCompleted,
		SessionID:      "sess-original",
		StartedAt:      &now,
		EndedAt:        &now,
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeClaudeCode,
		},
		ApprovalState: domain.ApprovalStateNone,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repos.Runs.Create(ctx, directRun); err != nil {
		t.Fatalf("create run: %v", err)
	}

	// Trigger continuation
	_, err := svc.ContinueRun(ctx, orchestration.ContinueRunRequest{
		RunID:   runID,
		Message: "please continue",
	})
	if err != nil {
		t.Fatalf("ContinueRun: %v", err)
	}

	// Wait for the continuation to finish
	select {
	case <-continueDone:
	case <-time.After(10 * time.Second):
		t.Fatal("continuation never completed")
	}

	// Give the orchestrator time to persist
	time.Sleep(200 * time.Millisecond)

	// Verify session ID was preserved despite failure
	updatedRun, err := repos.Runs.Get(ctx, runID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if updatedRun.SessionID != "sess-updated-after-failure" {
		t.Errorf("expected session ID 'sess-updated-after-failure', got %q", updatedRun.SessionID)
	}
	if updatedRun.Status != domain.RunStatusFailed {
		t.Errorf("expected status 'failed', got %q", updatedRun.Status)
	}

	// Verify the run is still continuable
	canContinue, reason := domain.CanContinueRun(updatedRun)
	if !canContinue {
		t.Errorf("expected failed run with session ID to be continuable, got reason: %s", reason)
	}
}
