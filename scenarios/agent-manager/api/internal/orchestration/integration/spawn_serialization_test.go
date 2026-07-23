// Spawn-dispatcher burst integration test.
//
// This test boots the full orchestrator with the spawn dispatcher
// configured with a dispatcher, fires N CreateRun calls concurrently, and
// asserts that:
//
//  1. All N requests are accepted and reach a terminal state.
//  2. Dispatcher accounting drains to zero after the burst.
//
// Startup-slot serialization itself is deliberately asserted in spawn's unit
// suite. The slot is released at RunStatusRunning (before Execute), so treating
// runner execution as the slot window here would assert the wrong contract.
package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/testutil"

	"github.com/google/uuid"
)

func TestSpawnDispatcher_SerializesBurst(t *testing.T) {
	const burst = 6

	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	// The runner completes immediately. This test owns the orchestrator's burst
	// lifecycle; spawn/dispatcher_test.go owns the lower-level slot gate.
	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "mock runner available")

	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return &runner.ExecuteResult{
			ExitCode: 0,
			Summary:  &domain.RunSummary{Description: "ok"},
		}, nil
	}
	if err := registry.Register(mockRunner); err != nil {
		t.Fatalf("register runner: %v", err)
	}

	// Queue capacity must accommodate the entire burst so no Enqueue rejects.
	dispatcher := spawn.New(spawn.Config{
		MaxStartingConcurrency: 1,
		QueueCapacity:          burst * 2,
	})
	t.Cleanup(dispatcher.Close)

	svc := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:          time.Minute,
			MaxConcurrentRuns:       burst * 2,
			RequireSandboxByDefault: false,
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithCheckpoints(repos.Checkpoints),
		orchestration.WithIdempotency(repos.Idempotency),
		newTestRolePolicyOption(t),
		orchestration.WithSpawnDispatcher(dispatcher),
	)

	profile, err := svc.CreateProfile(ctx, &domain.AgentProfile{
		ID:   uuid.New(),
		Name: "spawn-burst",

		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}, RoleRef: "code.default",
	})
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	tasks := make([]*domain.Task, burst)
	for i := 0; i < burst; i++ {
		task, err := svc.CreateTask(ctx, &domain.Task{
			ID:          uuid.New(),
			Title:       "burst",
			Description: "burst task",
			ScopePath:   "/tmp",
			ProjectRoot: "/tmp",
			Status:      domain.TaskStatusQueued,
		})
		if err != nil {
			t.Fatalf("CreateTask %d: %v", i, err)
		}
		tasks[i] = task
	}

	// Fire all CreateRun calls concurrently — the heartbeat-burst
	// scenario the dispatcher is designed to handle.
	runMode := domain.RunModeInPlace
	runs := make([]*domain.Run, burst)
	var wg sync.WaitGroup
	wg.Add(burst)
	for i := 0; i < burst; i++ {
		i := i
		go func() {
			defer wg.Done()
			run, err := svc.CreateRun(ctx, orchestration.CreateRunRequest{
				TaskID:         tasks[i].ID,
				AgentProfileID: &profile.ID,
				Prompt:         "burst",
				RunMode:        &runMode,
			})
			if err != nil {
				t.Errorf("CreateRun %d: %v", i, err)
				return
			}
			runs[i] = run
		}()
	}
	wg.Wait()

	// Wait for every run to reach a terminal state.
	for i, run := range runs {
		if run == nil {
			continue
		}
		if _, err := waitForTerminal(t, ctx, svc, run.ID, 15*time.Second); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}

	// Final state: dispatcher must drain to zero.
	awaitTrue(t, 3*time.Second, "Stats to drain", func() bool {
		s := svc.SpawnStats()
		return s.QueueDepth == 0 && s.ActiveCount == 0 && s.StartingCount == 0
	})
}

// awaitTrue polls cond up to timeout. Fails the test when cond never
// returns true.
func awaitTrue(t *testing.T, timeout time.Duration, msg string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting: %s", msg)
}

// waitForTerminal polls the run status until it leaves Running/Starting.
func waitForTerminal(t *testing.T, ctx context.Context, svc *orchestration.Orchestrator, runID uuid.UUID, timeout time.Duration) (*domain.Run, error) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		run, err := svc.GetRun(ctx, runID)
		if err != nil {
			return nil, err
		}
		switch run.Status {
		case domain.RunStatusRunning, domain.RunStatusStarting, domain.RunStatusPending:
			time.Sleep(20 * time.Millisecond)
			continue
		default:
			return run, nil
		}
	}
	return nil, context.DeadlineExceeded
}
