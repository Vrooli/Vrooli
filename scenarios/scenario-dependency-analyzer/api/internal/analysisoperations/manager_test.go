package analysisoperations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func testRepository(t *testing.T) *SQLRepository {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	return NewRepository(db)
}

type testRunner struct {
	names   []string
	analyze func(context.Context, string) (ScenarioOutcome, error)
	graph   func(context.Context, bool) (GraphOutcome, error)
}

func (r testRunner) Scenarios(context.Context) ([]string, error) { return r.names, nil }
func (r testRunner) Analyze(ctx context.Context, name string) (ScenarioOutcome, error) {
	if r.analyze != nil {
		return r.analyze(ctx, name)
	}
	return ScenarioOutcome{DetectedResources: 2}, nil
}

func (r testRunner) RebuildGraph(ctx context.Context, apply bool) (GraphOutcome, error) {
	if r.graph != nil {
		return r.graph(ctx, apply)
	}
	return GraphOutcome{Applied: apply}, nil
}

func startManager(t *testing.T, repo Repository, runner Runner) *Manager {
	t.Helper()
	m := NewManager(repo, runner, Config{Logf: t.Logf})
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.Close(ctx); err != nil {
			t.Error(err)
		}
	})
	return m
}

func waitOperation(t *testing.T, m *Manager, id string) Operation {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	op, err := m.Wait(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return op
}

func TestDisconnectedObserverAndRepeatedCallerNeverRepeatWork(t *testing.T) {
	repo := testRepository(t)
	entered, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	m := startManager(t, repo, testRunner{names: []string{"alpha"}, analyze: func(ctx context.Context, name string) (ScenarioOutcome, error) {
		calls.Add(1)
		close(entered)
		select {
		case <-release:
			return ScenarioOutcome{}, nil
		case <-ctx.Done():
			return ScenarioOutcome{}, ctx.Err()
		}
	}})
	op, reused, err := m.Submit(context.Background(), "lost-response", Spec{})
	if err != nil || reused {
		t.Fatalf("submit: reused=%v err=%v", reused, err)
	}
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.Wait(ctx, op.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("observer: %v", err)
	}
	alias, reused, err := m.Submit(context.Background(), "another-observer", Spec{})
	if err != nil || !reused || alias.ID != op.ID {
		t.Fatalf("coalescing: %+v %v %v", alias, reused, err)
	}
	close(release)
	finished := waitOperation(t, m, op.ID)
	if finished.State != Succeeded || finished.Completed != 1 {
		t.Fatalf("terminal: %+v", finished)
	}
	for _, caller := range []string{"lost-response", "another-observer"} {
		replay, reused, err := m.Submit(context.Background(), caller, Spec{})
		if err != nil || !reused || replay.ID != op.ID || replay.State != Succeeded {
			t.Fatalf("replay: %+v %v %v", replay, reused, err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("work repeated %d times", calls.Load())
	}
}

func TestSingleWorkerPartialResultsDoNotPublishGraph(t *testing.T) {
	var active, peak, graphs atomic.Int32
	m := startManager(t, testRepository(t), testRunner{names: []string{"zeta", "alpha", "alpha"}, analyze: func(ctx context.Context, name string) (ScenarioOutcome, error) {
		n := active.Add(1)
		defer active.Add(-1)
		if n > peak.Load() {
			peak.Store(n)
		}
		if name == "zeta" {
			return ScenarioOutcome{}, errors.New("source unavailable")
		}
		return ScenarioOutcome{DetectedResources: 3}, nil
	}, graph: func(context.Context, bool) (GraphOutcome, error) { graphs.Add(1); return GraphOutcome{}, nil }})
	a, _, err := m.Submit(context.Background(), "analysis", Spec{Kind: FleetAnalysis})
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := m.Submit(context.Background(), "graph", Spec{Kind: GraphRebuild, ApplyGraph: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{a.ID, b.ID} {
		op := waitOperation(t, m, id)
		if op.State != Partial || op.Total != 2 || op.Completed != 2 || op.Failures != 1 || op.Scenarios[0].Scenario != "alpha" {
			t.Fatalf("partial: %+v", op)
		}
	}
	if graphs.Load() != 0 || peak.Load() != 1 {
		t.Fatalf("graphs=%d concurrency=%d", graphs.Load(), peak.Load())
	}
}

func TestRecoveryInterruptsRunningAndResumesQueuedWithoutReexecution(t *testing.T) {
	repo := testRepository(t)
	ctx := context.Background()
	now := time.Now()
	running, _, err := repo.Submit(ctx, "running", Spec{Kind: FleetAnalysis}, []string{"alpha"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ClaimNext(ctx, now); err != nil {
		t.Fatal(err)
	}
	queued, _, err := repo.Submit(ctx, "queued", Spec{Kind: GraphRebuild}, []string{"beta"}, now)
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	m := startManager(t, repo, testRunner{analyze: func(ctx context.Context, name string) (ScenarioOutcome, error) {
		if name != "beta" {
			t.Errorf("reexecuted interrupted source %s", name)
		}
		calls.Add(1)
		return ScenarioOutcome{}, nil
	}})
	if op := waitOperation(t, m, running.ID); op.State != Interrupted {
		t.Fatalf("recovery: %+v", op)
	}
	if op := waitOperation(t, m, queued.ID); op.State != Succeeded {
		t.Fatalf("queue: %+v", op)
	}
	replay, reused, err := m.Submit(ctx, "running", Spec{Kind: FleetAnalysis})
	if err != nil || !reused || replay.State != Interrupted || calls.Load() != 1 {
		t.Fatalf("replay: %+v %v %v calls=%d", replay, reused, err, calls.Load())
	}
	if err := repo.StartScenario(ctx, running.ID, "alpha"); !errors.Is(err, ErrState) {
		t.Fatalf("late write allowed: %v", err)
	}
}

func TestClaimReadFailureRollsBackBeforeAnyExecution(t *testing.T) {
	repo := testRepository(t)
	ctx := context.Background()
	at := time.Now()
	op, _, err := repo.Submit(ctx, "claim", Spec{Kind: FleetAnalysis}, []string{"alpha"}, at)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.db.ExecContext(ctx, `ALTER TABLE analysis_operation_scenarios RENAME TO unavailable_items`); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ClaimNext(ctx, at); err == nil {
		t.Fatal("claim succeeded without its immutable work snapshot")
	}
	var state State
	if err := repo.db.QueryRowContext(ctx, `SELECT state FROM analysis_operations WHERE id=?`, op.ID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != Queued {
		t.Fatalf("read failure stranded accepted work in %s", state)
	}
	if _, err := repo.db.ExecContext(ctx, `ALTER TABLE unavailable_items RENAME TO analysis_operation_scenarios`); err != nil {
		t.Fatal(err)
	}
	claimed, err := repo.ClaimNext(ctx, at)
	if err != nil || claimed.ID != op.ID || len(claimed.Scenarios) != 1 {
		t.Fatalf("claim recovery: %+v %v", claimed, err)
	}
}

func TestDetailRetentionKeepsCallerBindingAndOutcome(t *testing.T) {
	repo := testRepository(t)
	ctx := context.Background()
	now := time.Now()
	var first Operation
	for n := 0; n < RetainedDetails+1; n++ {
		op, _, err := repo.Submit(ctx, fmt.Sprint(n), Spec{Kind: FleetAnalysis}, []string{"alpha"}, now)
		if err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			first = op
		}
		if _, err := repo.ClaimNext(ctx, now); err != nil {
			t.Fatal(err)
		}
		if err := repo.Finish(ctx, op.ID, Failed, "fixture failed", nil, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.Prune(ctx); err != nil {
		t.Fatal(err)
	}
	op, reused, err := repo.Submit(ctx, "0", Spec{Kind: FleetAnalysis}, nil, now)
	if err != nil || !reused || op.ID != first.ID || op.State != Failed || !op.DetailsPruned || len(op.Scenarios) != 0 {
		t.Fatalf("retained receipt: %+v %v %v", op, reused, err)
	}
	if _, _, err := repo.Submit(ctx, "0", Spec{Kind: GraphRebuild}, nil, now); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed input: %v", err)
	}
}

type failingFinish struct {
	Repository
	attempts atomic.Int32
}

func (r *failingFinish) Finish(ctx context.Context, id string, state State, msg string, graph *GraphOutcome, at time.Time) error {
	if r.attempts.Add(1) == 1 {
		return errors.New("temporary write failure")
	}
	return r.Repository.Finish(ctx, id, state, msg, graph, at)
}

func TestPublicationRetryDoesNotRepeatExecution(t *testing.T) {
	repo := &failingFinish{Repository: testRepository(t)}
	var calls atomic.Int32
	m := startManager(t, repo, testRunner{names: []string{"alpha"}, analyze: func(context.Context, string) (ScenarioOutcome, error) { calls.Add(1); return ScenarioOutcome{}, nil }})
	op, _, err := m.Submit(context.Background(), "publish", Spec{})
	if err != nil {
		t.Fatal(err)
	}
	if result := waitOperation(t, m, op.ID); result.State != Succeeded || calls.Load() != 1 || repo.attempts.Load() != 2 {
		t.Fatalf("result=%+v calls=%d attempts=%d", result, calls.Load(), repo.attempts.Load())
	}
}

type unavailableFinish struct {
	Repository
	unavailable atomic.Bool
}

func (r *unavailableFinish) Finish(ctx context.Context, id string, state State, message string, graph *GraphOutcome, at time.Time) error {
	if r.unavailable.Load() {
		return errors.New("database temporarily unavailable")
	}
	return r.Repository.Finish(ctx, id, state, message, graph, at)
}

func TestResumeNotifiesExistingWaiterAfterShutdownPublicationFailure(t *testing.T) {
	repo := &unavailableFinish{Repository: testRepository(t)}
	repo.unavailable.Store(true)
	entered := make(chan struct{})
	m := startManager(t, repo, testRunner{names: []string{"alpha"}, analyze: func(ctx context.Context, _ string) (ScenarioOutcome, error) {
		close(entered)
		<-ctx.Done()
		return ScenarioOutcome{}, ctx.Err()
	}})
	op, _, err := m.Submit(context.Background(), "resume-waiter", Spec{})
	if err != nil {
		t.Fatal(err)
	}
	<-entered
	ready, release := m.coord.RegisterWaiter(op.ID)
	defer release()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := m.Close(ctx); err != nil {
		t.Fatal(err)
	}
	repo.unavailable.Store(false)
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal("maintenance resume stranded an existing observer")
	}
	if result := waitOperation(t, m, op.ID); result.State != Interrupted {
		t.Fatalf("recovered result: %+v", result)
	}
}

func TestPanicAndShutdownAreExplicitTerminalOutcomes(t *testing.T) {
	t.Run("panic", func(t *testing.T) {
		m := startManager(t, testRepository(t), testRunner{names: []string{"alpha"}, analyze: func(context.Context, string) (ScenarioOutcome, error) { panic("broken runner") }})
		op, _, err := m.Submit(context.Background(), "panic", Spec{})
		if err != nil {
			t.Fatal(err)
		}
		if result := waitOperation(t, m, op.ID); result.State != Failed || result.Scenarios[0].State != Interrupted {
			t.Fatalf("panic result: %+v", result)
		}
	})
	t.Run("shutdown", func(t *testing.T) {
		entered := make(chan struct{})
		m := startManager(t, testRepository(t), testRunner{names: []string{"alpha"}, analyze: func(ctx context.Context, _ string) (ScenarioOutcome, error) {
			close(entered)
			<-ctx.Done()
			return ScenarioOutcome{}, ctx.Err()
		}})
		op, _, err := m.Submit(context.Background(), "shutdown", Spec{})
		if err != nil {
			t.Fatal(err)
		}
		<-entered
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := m.Close(ctx); err != nil {
			t.Fatal(err)
		}
		if result := waitOperation(t, m, op.ID); result.State != Interrupted {
			t.Fatalf("shutdown result: %+v", result)
		}
		if err := m.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		if replay, _, err := m.Submit(context.Background(), "shutdown", Spec{}); err != nil || replay.State != Interrupted {
			t.Fatalf("restarted replay: %+v %v", replay, err)
		}
	})
}
