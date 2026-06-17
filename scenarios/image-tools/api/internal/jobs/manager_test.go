package jobs

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	testdb "image-tools/internal/testutil/db"
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
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (string, error) {
			emit(50, "halfway")
			return "out/result.png", nil
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

func TestSubmitWaitFailure(t *testing.T) {
	m := startManager(t, Config{
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (string, error) {
			return "", errors.New("boom")
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
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (string, error) {
			close(started)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-release:
				return "done", nil
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

func TestCancelQueuedJob(t *testing.T) {
	// Occupy the single GPU worker so the second GPU job stays queued.
	block := make(chan struct{})
	m := startManager(t, Config{
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (string, error) {
			if j.Operation == "blocker" {
				<-block
			}
			return "ok", nil
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
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (string, error) {
			c := atomic.AddInt32(&concurrent, 1)
			mu.Lock()
			if c > maxConcurrent {
				maxConcurrent = c
			}
			mu.Unlock()
			<-gate
			atomic.AddInt32(&concurrent, -1)
			return "ok", nil
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
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (string, error) {
			atomic.AddInt32(&running, 1)
			reached <- struct{}{}
			<-gate
			return "ok", nil
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
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (string, error) {
			<-release
			return "survived", nil
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
		Runner: func(ctx context.Context, j Job, emit func(int, string)) (string, error) {
			emit(25, "quarter")
			<-gate
			emit(75, "three-quarter")
			return "ok", nil
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

	m := New(d, Config{Runner: func(context.Context, Job, func(int, string)) (string, error) { return "", nil }})
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
	m := New(newJobsDB(t), Config{Runner: func(context.Context, Job, func(int, string)) (string, error) { return "", nil }})
	if _, err := m.Submit(context.Background(), Spec{Operation: "x"}); !errors.Is(err, ErrNotStarted) {
		t.Fatalf("want ErrNotStarted, got %v", err)
	}
}
