package integration

import (
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	base "agent-manager/internal/testutil"
	"agent-manager/internal/testutil/mocks"
)

// fakeClock provides deterministic time for integration lifecycles without
// introducing a production dependency on the test fixture package.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock(now time.Time) *fakeClock { return &fakeClock{now: now} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

type orchestratorHarness struct {
	Orchestrator *orchestration.Orchestrator
	Repos        *database.Repositories
	Events       event.Store
	Runner       *mocks.TranscriptReplayRunner
	Sandbox      *mocks.FakeSandboxProvider
	Broadcaster  *mocks.FakeBroadcaster
	Clock        *fakeClock
	Cleanup      func()
}

func newOrchestratorHarness(t *testing.T) *orchestratorHarness {
	t.Helper()
	repos, events, cleanup := base.SetupTestRepos(t)
	clock := newFakeClock(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	fakeRunner := mocks.NewTranscriptReplayRunner(domain.RunnerTypeCodex)
	registry := runner.NewRegistry()
	if err := registry.Register(fakeRunner); err != nil {
		cleanup()
		t.Fatalf("register transcript replay runner: %v", err)
	}
	sandbox := mocks.NewFakeSandboxProvider()
	broadcaster := mocks.NewFakeBroadcaster()
	orch := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithWorkflowRepository(repos.Workflows), orchestration.WithWorkflowExecutionRepository(repos.WorkflowExecutions),
		orchestration.WithCheckpoints(repos.Checkpoints), orchestration.WithIdempotency(repos.Idempotency),
		orchestration.WithEvents(events), orchestration.WithRunners(registry), orchestration.WithSandbox(sandbox),
		orchestration.WithBroadcaster(broadcaster), orchestration.WithClock(clock.Now),
		orchestration.WithConfig(orchestration.OrchestratorConfig{DefaultTimeout: time.Minute, MaxConcurrentRuns: 4, RequireSandboxByDefault: false}),
	)
	return &orchestratorHarness{Orchestrator: orch, Repos: repos, Events: events, Runner: fakeRunner, Sandbox: sandbox, Broadcaster: broadcaster, Clock: clock, Cleanup: cleanup}
}
