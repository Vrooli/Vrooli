package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	testdb "github.com/vrooli/api-core/databasetest"
)

func newJobsDB(t *testing.T) *sql.DB {
	t.Helper()
	d := testdb.NewSQLite(t)
	if _, err := d.ExecContext(context.Background(), schemaSQL); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	return d
}

func startManager(t *testing.T, cfg Config) *Manager {
	t.Helper()
	m := New(newJobsDB(t), cfg)
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(m.Close)
	return m
}

func TestSubmitWaitSuccess(t *testing.T) {
	m := startManager(t, Config{
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (Result, error) {
			emit(50, "halfway")
			return Result{Ref: "out/result.png"}, nil
		},
	})
	job, err := m.Submit(context.Background(), Spec{Operation: "resize", Lane: LaneCPU, EstimatedSeconds: 3})
	if err != nil {
		t.Fatal(err)
	}
	if job.State != StateQueued || job.EstimatedSeconds != 3 {
		t.Fatalf("submit returned %+v", job)
	}
	done, err := m.Wait(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.State != StateSucceeded || done.ResultRef != "out/result.png" || done.Progress != 100 {
		t.Fatalf("final job %+v", done)
	}
}

func TestGPUAdmissionLeaseCoversSuccessAndMergesMetadata(t *testing.T) {
	released := make(chan struct{})
	m := startManager(t, Config{
		Admit: func(context.Context, context.CancelCauseFunc, Job) (func(), map[string]string, error) {
			return func() { close(released) }, map[string]string{"applied_rung": "offload-dit"}, nil
		},
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (Result, error) {
			return Result{Meta: map[string]string{"model_id": "ace-step"}}, nil
		},
	})
	job, err := m.Submit(context.Background(), Spec{Operation: "compose", Lane: LaneGPU})
	if err != nil {
		t.Fatal(err)
	}
	done, err := m.Wait(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.Meta["applied_rung"] != "offload-dit" || done.Meta["model_id"] != "ace-step" {
		t.Fatalf("metadata=%v", done.Meta)
	}
	select {
	case <-released:
	case <-time.After(time.Second):
		t.Fatal("admission lease was not released")
	}
}

func TestSubmitWaitFailure(t *testing.T) {
	m := startManager(t, Config{
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (Result, error) {
			return Result{}, errors.New("boom")
		},
	})
	job, _ := m.Submit(context.Background(), Spec{Operation: "upscale", Lane: LaneCPU})
	done, err := m.Wait(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.State != StateFailed || done.Error != "boom" {
		t.Fatalf("final job %+v", done)
	}
}

func TestCancelRunningJob(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	m := startManager(t, Config{
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (Result, error) {
			close(started)
			select {
			case <-ctx.Done():
				return Result{}, ctx.Err()
			case <-release:
				return Result{Ref: "done"}, nil
			}
		},
	})
	job, _ := m.Submit(context.Background(), Spec{Operation: "generate", Lane: LaneGPU})
	<-started
	if err := m.Cancel(job.ID); err != nil {
		t.Fatal(err)
	}
	done, err := m.Wait(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.State != StateCanceled {
		t.Fatalf("want canceled, got %+v", done)
	}
	close(release)
}

func TestGPUAdmissionLeaseReleasesOnCancel(t *testing.T) {
	started := make(chan struct{})
	released := make(chan struct{})
	m := startManager(t, Config{
		Admit: func(context.Context, context.CancelCauseFunc, Job) (func(), map[string]string, error) {
			return func() { close(released) }, nil, nil
		},
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (Result, error) {
			close(started)
			<-ctx.Done()
			return Result{}, ctx.Err()
		},
	})
	job, _ := m.Submit(context.Background(), Spec{Operation: "compose", Lane: LaneGPU})
	<-started
	if err := m.Cancel(job.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Wait(context.Background(), job.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-released:
	case <-time.After(time.Second):
		t.Fatal("admission lease was not released on cancel")
	}
}

func TestCancelQueuedJob(t *testing.T) {
	// Occupy the single GPU worker so the second GPU job stays queued.
	block := make(chan struct{})
	m := startManager(t, Config{
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (Result, error) {
			if j.Operation == "blocker" {
				<-block
			}
			return Result{Ref: "ok"}, nil
		},
	})
	blocker, _ := m.Submit(context.Background(), Spec{Operation: "blocker", Lane: LaneGPU})
	queued, _ := m.Submit(context.Background(), Spec{Operation: "queued", Lane: LaneGPU})

	if err := m.Cancel(queued.ID); err != nil {
		t.Fatal(err)
	}
	done, err := m.Wait(context.Background(), queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.State != StateCanceled {
		t.Fatalf("queued cancel: %+v", done)
	}
	close(block)
	if _, err := m.Wait(context.Background(), blocker.ID); err != nil {
		t.Fatal(err)
	}
}

// GPU lane must run at most one job at a time.
func TestGPUSerialized(t *testing.T) {
	var concurrent, maxConcurrent int32
	var mu sync.Mutex
	gate := make(chan struct{})
	m := startManager(t, Config{
		CPUWorkers: 4,
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (Result, error) {
			c := atomic.AddInt32(&concurrent, 1)
			mu.Lock()
			if c > maxConcurrent {
				maxConcurrent = c
			}
			mu.Unlock()
			<-gate
			atomic.AddInt32(&concurrent, -1)
			return Result{Ref: "ok"}, nil
		},
	})
	ids := make([]string, 3)
	for i := range ids {
		j, _ := m.Submit(context.Background(), Spec{Operation: "gpu", Lane: LaneGPU})
		ids[i] = j.ID
	}
	// Give the worker a moment to pick up; then release all.
	time.Sleep(50 * time.Millisecond)
	close(gate)
	for _, id := range ids {
		if _, err := m.Wait(context.Background(), id); err != nil {
			t.Fatal(err)
		}
	}
	if maxConcurrent != 1 {
		t.Fatalf("GPU lane ran %d concurrently, want 1", maxConcurrent)
	}
}

// CPU lane must run multiple jobs concurrently.
func TestCPUConcurrent(t *testing.T) {
	const n = 4
	var running int32
	reached := make(chan struct{}, n)
	gate := make(chan struct{})
	m := startManager(t, Config{
		CPUWorkers: n,
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (Result, error) {
			atomic.AddInt32(&running, 1)
			reached <- struct{}{}
			<-gate
			return Result{Ref: "ok"}, nil
		},
	})
	ids := make([]string, n)
	for i := range ids {
		j, _ := m.Submit(context.Background(), Spec{Operation: "cpu", Lane: LaneCPU})
		ids[i] = j.ID
	}
	// All n should reach the runner concurrently.
	for i := 0; i < n; i++ {
		select {
		case <-reached:
		case <-time.After(2 * time.Second):
			t.Fatalf("only %d of %d CPU jobs started concurrently", atomic.LoadInt32(&running), n)
		}
	}
	close(gate)
	for _, id := range ids {
		if _, err := m.Wait(context.Background(), id); err != nil {
			t.Fatal(err)
		}
	}
}

// A waiter giving up (ctx canceled) must NOT cancel the job — it keeps running
// server-side and completes (the disconnect-survival property).
func TestWaitGiveUpDoesNotCancelJob(t *testing.T) {
	release := make(chan struct{})
	m := startManager(t, Config{
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (Result, error) {
			<-release
			return Result{Ref: "survived"}, nil
		},
	})
	job, _ := m.Submit(context.Background(), Spec{Operation: "slow", Lane: LaneCPU})

	waitCtx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, err := m.Wait(waitCtx, job.ID); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want deadline exceeded, got %v", err)
	}
	// Job still running; complete it and confirm success.
	close(release)
	done, err := m.Wait(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.State != StateSucceeded || done.ResultRef != "survived" {
		t.Fatalf("job should have survived the waiter giving up: %+v", done)
	}
}

func TestProgressSubscribe(t *testing.T) {
	gate := make(chan struct{})
	m := startManager(t, Config{
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (Result, error) {
			emit(25, "quarter")
			<-gate
			emit(75, "three-quarter")
			return Result{Ref: "ok"}, nil
		},
	})
	job, _ := m.Submit(context.Background(), Spec{Operation: "x", Lane: LaneCPU})
	ch, unsub, err := m.Subscribe(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer unsub()

	var sawTerminal bool
	var maxProgress int
	done := make(chan struct{})
	go func() {
		for ev := range ch {
			if ev.Progress > maxProgress {
				maxProgress = ev.Progress
			}
			if ev.State.Terminal() {
				sawTerminal = true
			}
		}
		close(done)
	}()
	close(gate)
	<-done
	if !sawTerminal {
		t.Fatal("expected a terminal event on the stream")
	}
	if maxProgress < 75 {
		t.Fatalf("expected to observe progress >= 75, got %d", maxProgress)
	}
}

// Durability: a job left non-terminal by a prior process is recovered as failed.
func TestRecoveryMarksOrphansFailed(t *testing.T) {
	d := newJobsDB(t)
	st := newStore(d)
	now := time.Now()
	orphan := Job{ID: "orphan-1", Operation: "generate", Lane: LaneGPU, State: StateRunning, CreatedAt: now}
	if err := st.insert(context.Background(), orphan); err != nil {
		t.Fatal(err)
	}

	m := New(d, Config{Runner: func(context.Context, Job, func(int, string)) (Result, error) { return Result{}, nil }})
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	got, err := m.Get(context.Background(), "orphan-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.State != StateFailed || got.Error == "" {
		t.Fatalf("orphan not recovered: %+v", got)
	}
}

func TestSubmitBeforeStart(t *testing.T) {
	m := New(newJobsDB(t), Config{Runner: func(context.Context, Job, func(int, string)) (Result, error) { return Result{}, nil }})
	if _, err := m.Submit(context.Background(), Spec{Operation: "x"}); !errors.Is(err, ErrNotStarted) {
		t.Fatalf("want ErrNotStarted, got %v", err)
	}
}

// A backend's record of the run has to survive the trip through the store, or
// the cost of a cloud generation is knowable only inside the goroutine that
// paid for it. This asserts the round trip rather than the in-memory hand-off:
// Get reads back through SQLite, which is where the fact would be lost.
func TestRunRecordSurvivesTheStore(t *testing.T) {
	m := startManager(t, Config{
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (Result, error) {
			return Result{Ref: "out/gen.png", Meta: map[string]string{
				"model":    "openrouter-image",
				"tier":     "byok-cloud",
				"cost_usd": "0.042",
			}}, nil
		},
	})
	job, err := m.Submit(context.Background(), Spec{Operation: "text_to_image", Lane: LaneGPU})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Wait(context.Background(), job.ID); err != nil {
		t.Fatal(err)
	}
	persisted, err := m.Get(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := persisted.Meta["cost_usd"]; got != "0.042" {
		t.Fatalf("cost_usd = %q, want 0.042 (run record: %v)", got, persisted.Meta)
	}
	if got := persisted.Meta["model"]; got != "openrouter-image" {
		t.Fatalf("model = %q, want openrouter-image", got)
	}
}

// A runner that reports nothing must leave no record, so a caller can tell
// "this route cost nothing" from "nobody reported a cost". Recording an empty
// map as {} would erase that distinction at the first read.
func TestARunnerThatReportsNothingLeavesNoRecord(t *testing.T) {
	m := startManager(t, Config{
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (Result, error) {
			return Result{Ref: "out/resize.png"}, nil
		},
	})
	job, err := m.Submit(context.Background(), Spec{Operation: "resize", Lane: LaneCPU})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Wait(context.Background(), job.ID); err != nil {
		t.Fatal(err)
	}
	persisted, err := m.Get(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Meta != nil {
		t.Fatalf("meta = %v, want nil for a runner that reported nothing", persisted.Meta)
	}
}

// TestNetworkLaneDoesNotQueueBehindGPU is the F7 regression: a remote-tier job
// must run while a local GPU job holds the single GPU worker.
func TestNetworkLaneDoesNotQueueBehindGPU(t *testing.T) {
	ctx := context.Background()
	gpuStarted := make(chan struct{})
	releaseGPU := make(chan struct{})
	m := startManager(t, Config{
		NetworkWorkers: 2,
		Runner: func(_ context.Context, j Job, _ func(int, string)) (Result, error) {
			if j.Lane == LaneNetwork {
				return Result{Ref: "out/net.png"}, nil
			}
			close(gpuStarted)
			<-releaseGPU
			return Result{Ref: "out/gpu.png"}, nil
		},
	})

	gpu, err := m.Submit(ctx, Spec{Operation: "text_to_image", Lane: LaneGPU})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-gpuStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("gpu job never started")
	}

	netJob, err := m.Submit(ctx, Spec{Operation: "text_to_image", Lane: LaneNetwork})
	if err != nil {
		t.Fatal(err)
	}
	if netJob.Lane != LaneNetwork {
		t.Fatalf("lane = %s, want network", netJob.Lane)
	}
	done, err := m.Wait(ctx, netJob.ID)
	if err != nil {
		t.Fatalf("network job did not finish behind a blocked gpu job: %v", err)
	}
	if done.State != StateSucceeded {
		t.Fatalf("network job state = %s", done.State)
	}

	close(releaseGPU)
	if _, err := m.Wait(ctx, gpu.ID); err != nil {
		t.Fatalf("gpu wait: %v", err)
	}
}

type failingJobDB struct {
	*sql.DB
	fail atomic.Bool
}

func (d *failingJobDB) ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error) {
	if d.fail.Load() && strings.Contains(q, "UPDATE jobs") && len(args) > 0 && args[0] == string(StateSucceeded) {
		return nil, errors.New("injected finalization outage")
	}
	return d.DB.ExecContext(ctx, q, args...)
}

func TestFinalizationFailureNeverReportsSuccessAndRecoversWithoutRerun(t *testing.T) {
	db := &failingJobDB{DB: newJobsDB(t)}
	db.fail.Store(true)
	var calls, completions atomic.Int32
	m := New(db, Config{Runner: func(context.Context, Job, func(int, string)) (Result, error) {
		calls.Add(1)
		return Result{Ref: "out/retained.wav"}, nil
	}, OnComplete: func(Job) { completions.Add(1) }})
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	j, err := m.Submit(context.Background(), Spec{Lane: LaneCPU})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err = m.Wait(ctx, j.ID); !errors.Is(err, ErrPersistence) {
		t.Fatalf("false success: %v", err)
	}
	if _, err = m.Get(ctx, j.ID); !errors.Is(err, ErrPersistence) {
		t.Fatalf("Get hid failure: %v", err)
	}
	listed, err := m.List(ctx, 10)
	if err != nil || len(listed) != 1 || listed[0].State == StateSucceeded {
		t.Fatalf("durable state lied: %+v %v", listed, err)
	}
	if completions.Load() != 0 {
		t.Fatal("completion notified before persistence")
	}
	db.fail.Store(false)
	done, err := m.Get(ctx, j.ID)
	if err != nil || done.State != StateSucceeded || done.ResultRef != "out/retained.wav" {
		t.Fatalf("recovery %+v %v", done, err)
	}
	if calls.Load() != 1 || completions.Load() != 1 {
		t.Fatalf("rerun or duplicate completion: %d %d", calls.Load(), completions.Load())
	}
}

func TestHistoricalReadsDistinguishMissingCanceledAndUnavailable(t *testing.T) {
	db := newJobsDB(t)
	m := New(db, Config{Runner: func(context.Context, Job, func(int, string)) (Result, error) {
		t.Fatal("read ran producer")
		return Result{}, nil
	}})
	if err := m.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	recorded, err := m.Record(Spec{Operation: "historical"}, "out/retained.wav", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, read := range []func(context.Context, string) (Job, error){m.Get, m.Wait} {
		j, err := read(t.Context(), recorded.ID)
		if err != nil || j.ID != recorded.ID {
			t.Fatalf("historical read %+v %v", j, err)
		}
		if _, err = read(t.Context(), "missing"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("missing: %v", err)
		}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		if _, err = read(ctx, recorded.ID); !errors.Is(err, context.Canceled) || errors.Is(err, ErrNotFound) {
			t.Fatalf("cancel classification: %v", err)
		}
	}
	if _, err = db.Exec("UPDATE jobs SET progress='corrupt' WHERE id=?", recorded.ID); err != nil {
		t.Fatal(err)
	}
	for _, read := range []func(context.Context, string) (Job, error){m.Get, m.Wait} {
		if _, err = read(t.Context(), recorded.ID); !errors.Is(err, ErrPersistence) || errors.Is(err, ErrNotFound) {
			t.Fatalf("corrupt scan classification: %v", err)
		}
	}
	if _, err = db.Exec("UPDATE jobs SET progress=100 WHERE id=?", recorded.ID); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	for _, read := range []func(context.Context, string) (Job, error){m.Get, m.Wait} {
		if _, err = read(t.Context(), recorded.ID); !errors.Is(err, ErrPersistence) || errors.Is(err, ErrNotFound) {
			t.Fatalf("outage classification: %v", err)
		}
	}
}

func TestSlowSubscriberAlwaysReceivesOnePersistedTerminalEvent(t *testing.T) {
	entered, proceed := make(chan struct{}), make(chan struct{})
	var completions atomic.Int32
	m := startManager(t, Config{Runner: func(ctx context.Context, j Job, emit func(int, string)) (Result, error) {
		close(entered)
		<-proceed
		for i := 0; i < 200; i++ {
			emit(i%100, "progress")
		}
		return Result{Ref: "out/complete.wav"}, nil
	}, OnComplete: func(Job) { completions.Add(1) }})
	j, err := m.Submit(t.Context(), Spec{Lane: LaneCPU})
	if err != nil {
		t.Fatal(err)
	}
	<-entered
	events, unsubscribe, err := m.Subscribe(j.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer unsubscribe()
	close(proceed)
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	done, err := m.Wait(ctx, j.ID)
	if err != nil {
		t.Fatal(err)
	}
	terminals := 0
	for event := range events {
		if event.State.Terminal() {
			terminals++
			if event.State != done.State {
				t.Fatal("terminal differed from persisted state")
			}
		}
	}
	if terminals != 1 {
		t.Fatalf("terminal count %d", terminals)
	}
}

func TestTerminalCacheIsBoundedAndDurableObserversStillWork(t *testing.T) {
	completed := make(chan string, 10)
	m := startManager(t, Config{TerminalCacheLimit: 2, Runner: func(context.Context, Job, func(int, string)) (Result, error) {
		return Result{Ref: "out/retained.wav"}, nil
	}, OnComplete: func(j Job) { completed <- j.ID }})
	var first string
	for i := 0; i < 6; i++ {
		j, err := m.Submit(t.Context(), Spec{Lane: LaneCPU})
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = j.ID
		}
		if _, err = m.Wait(t.Context(), j.ID); err != nil {
			t.Fatal(err)
		}
		<-completed
	}
	m.mu.Lock()
	size := len(m.entries)
	_, cached := m.entries[first]
	m.mu.Unlock()
	if size > 2 || cached {
		t.Fatalf("unbounded terminal cache size=%d old cached=%v", size, cached)
	}
	for _, read := range []func(context.Context, string) (Job, error){m.Get, m.Wait} {
		j, err := read(t.Context(), first)
		if err != nil || j.State != StateSucceeded || j.ResultRef != "out/retained.wav" {
			t.Fatalf("durable observer %+v %v", j, err)
		}
	}
	stream, unsub, err := m.Subscribe(first)
	if err != nil {
		t.Fatal(err)
	}
	defer unsub()
	count := 0
	for event := range stream {
		count++
		if event.State != StateSucceeded {
			t.Fatal("historical event differs")
		}
	}
	if count != 1 {
		t.Fatalf("historical terminal count %d", count)
	}
	if err = m.Cancel(first); err != nil {
		t.Fatal("terminal cancellation lost idempotence", err)
	}
}

func TestTerminalEvictionPreservesUnpersistedFinalization(t *testing.T) {
	db := &failingJobDB{DB: newJobsDB(t)}
	db.fail.Store(true)
	completed := make(chan string, 10)
	m := New(db, Config{TerminalCacheLimit: 1, Runner: func(context.Context, Job, func(int, string)) (Result, error) {
		return Result{Ref: "out/retained.wav"}, nil
	}, OnComplete: func(j Job) { completed <- j.ID }})
	if err := m.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	pending, err := m.Submit(t.Context(), Spec{Lane: LaneCPU})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.Wait(t.Context(), pending.ID); !errors.Is(err, ErrPersistence) {
		t.Fatal("missing pending result", err)
	}
	db.fail.Store(false)
	for i := 0; i < 3; i++ {
		j, err := m.Submit(t.Context(), Spec{Lane: LaneCPU})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = m.Wait(t.Context(), j.ID); err != nil {
			t.Fatal(err)
		}
		<-completed
	}
	m.mu.Lock()
	entry := m.entries[pending.ID]
	size := len(m.entries)
	m.mu.Unlock()
	if entry == nil || size != 2 {
		t.Fatalf("pending result evicted: size=%d", size)
	}
	done, err := m.Get(t.Context(), pending.ID)
	if err != nil || done.State != StateSucceeded {
		t.Fatalf("pending recovery %+v %v", done, err)
	}
}

func TestTerminalCacheDoesNotEvictActiveSubscriber(t *testing.T) {
	entered := make(chan struct{})
	completed := make(chan string, 10)
	m := startManager(t, Config{TerminalCacheLimit: 1, Runner: func(ctx context.Context, j Job, emit func(int, string)) (Result, error) {
		if string(j.Payload) == "active" {
			close(entered)
			<-ctx.Done()
			return Result{}, ctx.Err()
		}
		return Result{}, nil
	}, OnComplete: func(j Job) { completed <- j.ID }})
	active, err := m.Submit(t.Context(), Spec{Lane: LaneCPU, Payload: []byte("active")})
	if err != nil {
		t.Fatal(err)
	}
	<-entered
	stream, unsub, err := m.Subscribe(active.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer unsub()
	for i := 0; i < 3; i++ {
		j, err := m.Submit(t.Context(), Spec{Lane: LaneCPU})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = m.Wait(t.Context(), j.ID); err != nil {
			t.Fatal(err)
		}
		<-completed
	}
	m.mu.Lock()
	_, present := m.entries[active.ID]
	m.mu.Unlock()
	if !present {
		t.Fatal("active subscriber entry evicted")
	}
	if err = m.Cancel(active.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = m.Wait(t.Context(), active.ID); err != nil {
		t.Fatal(err)
	}
	terminals := 0
	for event := range stream {
		if event.State.Terminal() {
			terminals++
		}
	}
	if terminals != 1 {
		t.Fatalf("active subscriber lost terminal result: %d", terminals)
	}
}

func TestPartialRunnerMetadataSurvivesFailureAndCancellation(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelled), func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			m := startManager(t, Config{Runner: func(context.Context, Job, func(int, string)) (Result, error) {
				close(entered)
				<-release
				return Result{Ref: "must-not-publish", Meta: map[string]string{"completed_take_ids": "[one]"}}, errors.New("second take failed")
			}})
			j, err := m.Submit(t.Context(), Spec{Operation: "compose", Lane: LaneGPU})
			if err != nil {
				t.Fatal(err)
			}
			<-entered
			if cancelled {
				if err := m.Cancel(j.ID); err != nil {
					t.Fatal(err)
				}
			}
			close(release)
			done, err := m.Wait(t.Context(), j.ID)
			if err != nil {
				t.Fatal(err)
			}
			want := StateFailed
			if cancelled {
				want = StateCanceled
			}
			if done.State != want || done.ResultRef != "" || done.Meta["completed_take_ids"] != "[one]" {
				t.Fatalf("terminal lost partial facts or invented success: %+v", done)
			}
		})
	}
}

func TestRecoveredTerminalCannotBeOverwrittenByLateWorker(t *testing.T) {
	db := newJobsDB(t)
	entered, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	callback := make(chan struct{}, 1)
	m := New(db, Config{Runner: func(context.Context, Job, func(int, string)) (Result, error) {
		close(entered)
		<-release
		return Result{Ref: "late", Meta: map[string]string{"late": "must-not-win"}}, nil
	}, OnComplete: func(Job) { calls.Add(1); callback <- struct{}{} }})
	if err := m.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	j, err := m.Submit(t.Context(), Spec{Operation: "compose", Lane: LaneGPU})
	if err != nil {
		t.Fatal(err)
	}
	<-entered
	observer := New(db, Config{Runner: func(context.Context, Job, func(int, string)) (Result, error) { return Result{}, nil }})
	if err := observer.Start(t.Context()); err != nil {
		close(release)
		t.Fatal(err)
	}
	defer observer.Close()
	before, err := observer.Get(t.Context(), j.ID)
	if err != nil || before.State != StateFailed {
		close(release)
		t.Fatalf("recovery=%+v %v", before, err)
	}
	close(release)
	done, err := m.Wait(t.Context(), j.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.State != StateFailed || done.Error != before.Error || done.ResultRef != "" || done.Meta["late"] != "" {
		t.Fatalf("late worker replaced authority: %+v", done)
	}
	again, err := observer.Get(t.Context(), j.ID)
	if err != nil || again.State != before.State || again.Error != before.Error || again.ResultRef != "" {
		t.Fatalf("durable terminal changed: %+v %v", again, err)
	}
	select {
	case <-callback:
	case <-time.After(time.Second):
		t.Fatal("completion callback missing")
	}
	if calls.Load() != 1 {
		t.Fatalf("completion callbacks=%d", calls.Load())
	}
}

func TestPendingFinalizationAdoptsRecoveryWinnerWithoutRerun(t *testing.T) {
	db := &failingJobDB{DB: newJobsDB(t)}
	db.fail.Store(true)
	var runs, callbacks atomic.Int32
	m := New(db, Config{Runner: func(context.Context, Job, func(int, string)) (Result, error) {
		runs.Add(1)
		return Result{Ref: "late", Meta: map[string]string{"late": "receipt"}}, nil
	}, OnComplete: func(Job) { callbacks.Add(1) }})
	if err := m.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	j, err := m.Submit(t.Context(), Spec{Lane: LaneCPU})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Wait(t.Context(), j.ID); !errors.Is(err, ErrPersistence) {
		t.Fatalf("initial persistence=%v", err)
	}
	recovery := New(db, Config{Runner: func(context.Context, Job, func(int, string)) (Result, error) {
		t.Error("recovery reran producer")
		return Result{}, nil
	}})
	if err := recovery.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer recovery.Close()
	db.fail.Store(false)
	for _, read := range []func(context.Context, string) (Job, error){m.Get, m.Wait, m.Get} {
		done, err := read(t.Context(), j.ID)
		if err != nil || done.State != StateFailed || done.Error != "interrupted by server restart" || done.ResultRef != "" || done.Meta["late"] != "" {
			t.Fatalf("pending overwrote terminal: %+v %v", done, err)
		}
	}
	if runs.Load() != 1 || callbacks.Load() != 1 {
		t.Fatalf("runs=%d callbacks=%d", runs.Load(), callbacks.Load())
	}
}

type rowsErrorResult struct{ sql.Result }

func (rowsErrorResult) RowsAffected() (int64, error) {
	return 0, errors.New("rows receipt unavailable")
}

type rowsErrorDB struct{ *sql.DB }

func (d rowsErrorDB) ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error) {
	r, e := d.DB.ExecContext(ctx, q, args...)
	if e == nil && strings.Contains(q, "UPDATE jobs") {
		return rowsErrorResult{r}, nil
	}
	return r, e
}
func TestStoreUpdateDoesNotHideRowsAffectedFailure(t *testing.T) {
	db := newJobsDB(t)
	s := newStore(rowsErrorDB{db})
	j := Job{ID: "rows-error", State: StateQueued, CreatedAt: time.Now()}
	if err := s.insert(t.Context(), j); err != nil {
		t.Fatal(err)
	}
	j.State = StateSucceeded
	if err := s.update(t.Context(), j); err == nil || !strings.Contains(err.Error(), "rows receipt unavailable") {
		t.Fatalf("rows error hidden: %v", err)
	}
}

func TestLateProgressAdoptsTerminalAndCannotReopenSubscription(t *testing.T) {
	db := newJobsDB(t)
	entered, progress, afterProgress, release := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	m := New(db, Config{Runner: func(_ context.Context, _ Job, emit func(int, string)) (Result, error) {
		close(entered)
		<-progress
		emit(60, "late progress")
		close(afterProgress)
		<-release
		emit(90, "later progress")
		return Result{Meta: map[string]string{"late": "receipt"}}, nil
	}})
	if err := m.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	j, err := m.Submit(t.Context(), Spec{Lane: LaneGPU})
	if err != nil {
		t.Fatal(err)
	}
	<-entered
	events, unsub, err := m.Subscribe(j.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer unsub()
	winner, err := m.st.get(t.Context(), j.ID)
	if err != nil {
		t.Fatal(err)
	}
	winner.State = StateCanceled
	winner.Error = ""
	winner.Message = "external terminal"
	now := time.Now()
	winner.FinishedAt = &now
	if err := m.st.update(t.Context(), winner); err != nil {
		t.Fatal(err)
	}
	close(progress)
	<-afterProgress
	close(release)
	done, err := m.Wait(t.Context(), j.ID)
	if err != nil || done.State != StateCanceled || done.Meta["late"] != "" {
		t.Fatalf("progress authority %+v %v", done, err)
	}
	terminals := 0
	for ev := range events {
		if ev.State.Terminal() {
			terminals++
		}
		if ev.Message == "late progress" || ev.Message == "later progress" {
			t.Fatalf("late progress published: %+v", ev)
		}
	}
	if terminals != 1 {
		t.Fatalf("terminal events=%d", terminals)
	}
}
