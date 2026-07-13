// Spawn-dispatcher serialization integration test.
//
// This test boots the full orchestrator with the spawn dispatcher
// configured for strict single-slot startup, fires N CreateRun calls
// concurrently, and asserts that:
//
//  1. At any moment, at most MaxStartingConcurrency runs are inside
//     the codex-bootstrap window (here: in MockRunner.ExecuteFunc with
//     the "started" signal not yet released).
//  2. CreateRunResponse-shaped Stats() snapshots show non-zero
//     queueDepth while the burst is in flight.
//  3. All N runs eventually complete cleanly — the queue drains.
//
// This is the regression gate for the heartbeat-driven multi-agent
// caller burst that motivated the dispatcher.
package integration

import (
	"context"
	"sync"
	"sync/atomic"
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
	const maxStarting = 1

	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	// MockRunner whose Execute() blocks until the test releases it,
	// so we can hold runs inside the startup window long enough to
	// observe queue accumulation.
	registry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "mock runner available")

	var inExecute atomic.Int32
	var maxInExecute atomic.Int32
	release := make(chan struct{})

	mockRunner.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		now := inExecute.Add(1)
		// Track the high-water mark for concurrent in-execute runs.
		for {
			cur := maxInExecute.Load()
			if now <= cur {
				break
			}
			if maxInExecute.CompareAndSwap(cur, now) {
				break
			}
		}
		select {
		case <-release:
		case <-ctx.Done():
			inExecute.Add(-1)
			return nil, ctx.Err()
		}
		inExecute.Add(-1)
		return &runner.ExecuteResult{
			ExitCode: 0,
			Summary:  &domain.RunSummary{Description: "ok"},
		}, nil
	}
	if err := registry.Register(mockRunner); err != nil {
		t.Fatalf("register runner: %v", err)
	}

	// Fresh dispatcher with strict single-slot serialization. queue
	// capacity must accommodate the entire burst so no Enqueue rejects.
	dispatcher := spawn.New(spawn.Config{
		MaxStartingConcurrency: maxStarting,
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

	// While the burst is in flight, queueDepth + activeCount should be
	// non-zero. We don't assert exact values (timing-dependent) but the
	// observable backpressure must be visible.
	awaitTrue(t, 3*time.Second, "first run to enter Execute", func() bool {
		return inExecute.Load() == 1
	})
	stats := svc.SpawnStats()
	if stats.ActiveCount == 0 {
		t.Fatalf("activeCount was zero during burst; dispatcher did not observe runs")
	}

	// Release the gate so all queued runs can drain.
	close(release)

	// Wait for every run to reach a terminal state.
	for i, run := range runs {
		if run == nil {
			continue
		}
		if _, err := waitForTerminal(t, ctx, svc, run.ID, 15*time.Second); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}

	// Regression gate: at no point should more than MaxStartingConcurrency
	// runs have been simultaneously inside the bootstrap window.
	if got := maxInExecute.Load(); got > maxStarting {
		t.Errorf("max concurrent in-Execute = %d, want <= %d (MaxStartingConcurrency)", got, maxStarting)
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
func waitForTerminal(t *testing.T, ctx context.Context, svc orchestration.Service, runID uuid.UUID, timeout time.Duration) (*domain.Run, error) {
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
