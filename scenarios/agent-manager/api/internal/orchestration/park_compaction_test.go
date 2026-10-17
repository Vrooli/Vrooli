package orchestration_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"
)

// compactingRunner is a Codex runner that records out-of-band compactions.
type compactingRunner struct {
	*runner.MockRunner
	compact func(runner.CompactSessionRequest) (*runner.CompactSessionResult, error)
	calls   atomic.Int32
}

func (r *compactingRunner) CompactSession(_ context.Context, req runner.CompactSessionRequest) (*runner.CompactSessionResult, error) {
	r.calls.Add(1)
	if r.compact != nil {
		return r.compact(req)
	}
	return &runner.CompactSessionResult{Compacted: true, ContextTokens: 120_000}, nil
}

type compactionFixture struct {
	svc       *orchestration.Orchestrator
	repos     *database.Repositories
	runner    *compactingRunner
	continued chan struct{}
}

func newCompactionFixture(t *testing.T, delay time.Duration) *compactionFixture {
	return newCompactionFixtureWith(t, delay, true)
}

func newCompactionFixtureWith(t *testing.T, delay time.Duration, declaresCompaction bool) *compactionFixture {
	t.Helper()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	f := &compactionFixture{repos: repos, continued: make(chan struct{}, 4)}
	mock := runner.NewMockRunner(domain.RunnerTypeCodex)
	mock.SetAvailable(true, "available")
	mock.SetCapabilities(runner.Capabilities{SupportsMessages: true, SupportsContinuation: true, SupportsSessionCompaction: declaresCompaction, MaxTurns: 100, SupportedModels: []string{"mock-model"}})
	mock.ContinueFunc = func(_ context.Context, req runner.ContinueRequest) (*runner.ExecuteResult, error) {
		f.continued <- struct{}{}
		return &runner.ExecuteResult{Success: true, SessionID: req.SessionID}, nil
	}
	f.runner = &compactingRunner{MockRunner: mock}
	registry := runner.NewRegistry()
	if err := registry.Register(f.runner); err != nil {
		t.Fatalf("register runner: %v", err)
	}
	f.svc = orchestration.New(
		repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithRunStateRoot(t.TempDir()),
		orchestration.WithIdentitySecret(parkFromAgentSecret),
		orchestration.WithParkCompaction(delay, 40_000),
		newTestRolePolicyOption(t),
	)
	return f
}

// parkCodexRun parks a running codec-pipe Codex run on its children.
func (f *compactionFixture) parkCodexRun(t *testing.T, mode domain.ExecutionMode) *domain.Run {
	t.Helper()
	ctx := context.Background()
	run := newParkableRun(t, ctx, f.svc, f.repos)
	run.ResolvedConfig = &domain.RunConfig{RunnerType: domain.RunnerTypeCodex}
	run.ExecutionMode = mode
	if err := f.repos.Runs.Update(ctx, run); err != nil {
		t.Fatalf("update run: %v", err)
	}
	token := activateToken(t, ctx, f.repos, run)
	if _, err := f.svc.ParkRunFromAgent(ctx, orchestration.ParkRunFromAgentRequest{
		RunID: run.ID, Producer: orchestration.ProducerChildren, Key: run.ID.String(), IdentityToken: token,
	}); err != nil {
		t.Fatalf("ParkRunFromAgent: %v", err)
	}
	return run
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestParkCompaction_CompactsARunThatStaysParked(t *testing.T) {
	f := newCompactionFixture(t, 100*time.Millisecond)
	var got runner.CompactSessionRequest
	f.runner.compact = func(req runner.CompactSessionRequest) (*runner.CompactSessionResult, error) {
		got = req
		return &runner.CompactSessionResult{Compacted: true, ContextTokens: 120_000}, nil
	}
	run := f.parkCodexRun(t, domain.ExecutionModeCodecPipe)

	waitFor(t, "the parked session to be compacted", func() bool { return f.runner.calls.Load() == 1 })
	if got.SessionID != run.SessionID || got.MinContextTokens != 40_000 || got.Env["CODEX_HOME"] == "" {
		t.Fatalf("compaction request = %+v, want the run's session, threshold and run-scoped CODEX_HOME", got)
	}
}

func TestParkCompaction_SkipsARunWokenBeforeTheDelay(t *testing.T) {
	// The delay outlasts the park turn-end grace (2s) that every wake waits for.
	f := newCompactionFixture(t, 3*time.Second)
	run := f.parkCodexRun(t, domain.ExecutionModeCodecPipe)
	if _, err := f.svc.WakeRun(context.Background(), orchestration.WakeRunInput{RunID: run.ID, Result: "child ended"}); err != nil {
		t.Fatalf("WakeRun: %v", err)
	}
	time.Sleep(1500 * time.Millisecond)
	if n := f.runner.calls.Load(); n != 0 {
		t.Fatalf("compacted %d times after an early wake; short parks keep their full context", n)
	}
}

func TestParkCompaction_NeverCompactsInteractiveRuns(t *testing.T) {
	// Resuming an interactive thread in the app-server clears its native goal.
	f := newCompactionFixture(t, 50*time.Millisecond)
	f.parkCodexRun(t, domain.ExecutionModeInteractive)
	time.Sleep(400 * time.Millisecond)
	if n := f.runner.calls.Load(); n != 0 {
		t.Fatalf("compacted an interactive run %d times", n)
	}
}

func TestParkCompaction_SkipsRunnersThatDoNotDeclareIt(t *testing.T) {
	// Support comes from the runner's declared capability, never its type.
	f := newCompactionFixtureWith(t, 50*time.Millisecond, false)
	f.parkCodexRun(t, domain.ExecutionModeCodecPipe)
	time.Sleep(400 * time.Millisecond)
	if n := f.runner.calls.Load(); n != 0 {
		t.Fatalf("compacted %d times through a runner without the capability", n)
	}
}

func TestWakeRun_WaitsForAnInFlightCompaction(t *testing.T) {
	// Codex refuses a second writer on a thread, and a continuation started
	// mid-compaction would resume the pre-compaction history.
	f := newCompactionFixture(t, 50*time.Millisecond)
	release := make(chan struct{})
	var finished atomic.Bool
	f.runner.compact = func(runner.CompactSessionRequest) (*runner.CompactSessionResult, error) {
		<-release
		finished.Store(true)
		return &runner.CompactSessionResult{Compacted: true}, nil
	}
	run := f.parkCodexRun(t, domain.ExecutionModeCodecPipe)
	waitFor(t, "the compaction to start", func() bool { return f.runner.calls.Load() == 1 })

	woke := make(chan error, 1)
	go func() {
		_, err := f.svc.WakeRun(context.Background(), orchestration.WakeRunInput{RunID: run.ID, Result: "child ended"})
		woke <- err
	}()
	select {
	case <-f.continued:
		t.Fatal("the continuation started while the session was being compacted")
	case <-time.After(300 * time.Millisecond):
	}
	close(release)
	if err := <-woke; err != nil {
		t.Fatalf("WakeRun: %v", err)
	}
	select {
	case <-f.continued:
		if !finished.Load() {
			t.Fatal("the continuation started before the compaction finished")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the run never continued after the compaction")
	}
}
