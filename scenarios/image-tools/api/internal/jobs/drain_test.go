package jobs

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func always(string) bool { return true }

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func state(t *testing.T, m *Manager, id string) Job {
	t.Helper()
	j, err := m.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return j
}

// A job that is only waiting (for capacity, for model weights) has computed
// nothing, so a restart must not wait for it: it is returned to the queue,
// persisted, and no longer holds the drain.
func TestDrainReturnsAWaitingJobToTheQueueInsteadOfHoldingTheDrain(t *testing.T) {
	var admissions atomic.Int32
	m := startManager(t, Config{
		Resumable: always,
		Admit: func(ctx context.Context, _ Job) (Admission, error) {
			admissions.Add(1)
			AdmissionProgress(ctx)(95, "waiting for model smoke capacity")
			<-ctx.Done() // capacity never frees up
			return Admission{Context: ctx}, ctx.Err()
		},
		Runner: func(context.Context, Job, func(int, string)) (Result, error) { return Result{}, nil },
	})
	job, err := m.Submit(context.Background(), Spec{Operation: "edit_instruct", Payload: []byte(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "admission to start", func() bool { return state(t, m, job.ID).State == StateAdmitting })
	if got := state(t, m, job.ID); got.State == StateRunning {
		t.Fatalf("a job waiting on admission must not read as running: %+v", got)
	}
	if len(m.Active()) != 0 {
		t.Fatalf("a waiting resumable job must not count as drain work before the drain: %+v", m.Active())
	}

	started := time.Now()
	if err := m.Quiesce(context.Background()); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the waiting job to return to the queue", func() bool { return len(m.Active()) == 0 && state(t, m, job.ID).State == StateQueued })
	if took := time.Since(started); took > 2*time.Second {
		t.Fatalf("drain took %s for a job that only waits", took)
	}
	got := state(t, m, job.ID)
	if got.Error != "" || got.FinishedAt != nil || got.StartedAt != nil {
		t.Fatalf("paused job must be a clean queued job: %+v", got)
	}
	time.Sleep(100 * time.Millisecond)
	if n := admissions.Load(); n != 1 {
		t.Fatalf("a draining scheduler must not start the job again; admissions=%d", n)
	}
}

// Only a job that is computing holds the drain, and it reports its progress.
func TestDrainGivesTimeOnlyToComputingJobsAndReportsTheirProgress(t *testing.T) {
	release, running := make(chan struct{}), make(chan struct{})
	m := startManager(t, Config{
		Resumable: always,
		Runner: func(ctx context.Context, _ Job, emit func(int, string)) (Result, error) {
			emit(42, "denoising step 8/20")
			close(running)
			select {
			case <-release:
				return Result{Ref: "out.png"}, nil
			case <-ctx.Done():
				return Result{}, ctx.Err()
			}
		},
	})
	job, err := m.Submit(context.Background(), Spec{Operation: "denoise", Payload: []byte(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	<-running
	waitFor(t, "progress to persist", func() bool { return state(t, m, job.ID).Progress == 42 })
	if err := m.Quiesce(context.Background()); err != nil {
		t.Fatal(err)
	}
	active := m.Active()
	if len(active) != 1 || active[0].ID != job.ID || active[0].State != StateRunning || active[0].Progress != 42 || active[0].Message != "denoising step 8/20" {
		t.Fatalf("a computing job must hold the drain and report progress: %+v", active)
	}
	if got := state(t, m, job.ID); got.State != StateRunning {
		t.Fatalf("drain must not interrupt computing work: %+v", got)
	}
	close(release)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if final, err := m.Wait(ctx, job.ID); err != nil || final.State != StateSucceeded {
		t.Fatalf("final=%+v err=%v", final, err)
	}
	waitFor(t, "the drain to clear", func() bool { return len(m.Active()) == 0 })
}

// A drain that is abandoned must leave the queue exactly as it found it.
func TestResumeAfterAnAbandonedDrainRunsThePausedJob(t *testing.T) {
	var admit atomic.Bool
	m := startManager(t, Config{
		Resumable: always,
		Admit: func(ctx context.Context, _ Job) (Admission, error) {
			if !admit.Load() {
				<-ctx.Done()
				return Admission{Context: ctx}, ctx.Err()
			}
			return Admission{Context: ctx}, nil
		},
		Runner: func(context.Context, Job, func(int, string)) (Result, error) { return Result{Ref: "ok"}, nil },
	})
	job, _ := m.Submit(context.Background(), Spec{Operation: "colorize", Payload: []byte(`{}`)})
	waitFor(t, "admission to start", func() bool { return state(t, m, job.ID).State == StateAdmitting })
	_ = m.Quiesce(context.Background())
	waitFor(t, "pause", func() bool { return state(t, m, job.ID).State == StateQueued })
	admit.Store(true)
	_ = m.Resume(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if final, err := m.Wait(ctx, job.ID); err != nil || final.State != StateSucceeded || final.ResultRef != "ok" {
		t.Fatalf("final=%+v err=%v", final, err)
	}
}

// The restart itself: a waiting job survives Close, and the next process runs
// it once from its stored payload.
func TestWaitingJobsResumeInTheNextProcess(t *testing.T) {
	db := newJobsDB(t)
	blocked := Config{
		Resumable: always,
		Admit: func(ctx context.Context, _ Job) (Admission, error) {
			<-ctx.Done()
			return Admission{Context: ctx}, ctx.Err()
		},
		Runner: func(context.Context, Job, func(int, string)) (Result, error) { return Result{}, nil },
	}
	first := New(db, blocked)
	if err := first.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	waiting, _ := first.Submit(context.Background(), Spec{Operation: "edit_instruct", Payload: []byte(`{"prompt":"p"}`)})
	waitFor(t, "admission", func() bool { return state(t, first, waiting.ID).State == StateAdmitting })
	queued, _ := first.Submit(context.Background(), Spec{Operation: "edit_instruct", Payload: []byte(`{"prompt":"q"}`), Priority: PriorityBatch})
	first.Close()
	for _, id := range []string{waiting.ID, queued.ID} {
		if got := state(t, first, id); got.State.Terminal() {
			t.Fatalf("a restart must not end a job that computed nothing: %+v", got)
		}
	}

	var ran atomic.Int32
	second := New(db, Config{
		Resumable: always,
		Runner: func(_ context.Context, job Job, _ func(int, string)) (Result, error) {
			ran.Add(1)
			return Result{Ref: string(job.Payload)}, nil
		},
	})
	if err := second.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for id, want := range map[string]string{waiting.ID: `{"prompt":"p"}`, queued.ID: `{"prompt":"q"}`} {
		final, err := second.Wait(ctx, id)
		if err != nil || final.State != StateSucceeded || final.ResultRef != want {
			t.Fatalf("job %s final=%+v err=%v", id, final, err)
		}
	}
	if ran.Load() != 2 {
		t.Fatalf("each job must run exactly once, ran %d", ran.Load())
	}
}

// Work that was computing cannot be resumed (it may have billed or written
// output) and an inline task has no stored work to resume: both fail loudly.
func TestRestartStillFailsRunningJobsAndInlineTasks(t *testing.T) {
	db := newJobsDB(t)
	st := newStore(db)
	now := time.Now()
	for _, j := range []Job{
		{ID: "running", Operation: "edit_instruct", Lane: LaneGPU, State: StateRunning, CreatedAt: now, Payload: []byte(`{}`)},
		{ID: "unlisted", Operation: "nsfw_classify", Lane: LaneCPU, State: StateQueued, CreatedAt: now},
		{ID: "testmode", Operation: "edit_instruct", Lane: LaneGPU, State: StateQueued, CreatedAt: now, TestMode: true, Payload: []byte(`{}`)},
	} {
		if err := st.insert(context.Background(), j); err != nil {
			t.Fatal(err)
		}
	}
	m := New(db, Config{
		Resumable: func(op string) bool { return op == "edit_instruct" },
		Runner:    func(context.Context, Job, func(int, string)) (Result, error) { return Result{}, nil },
	})
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	for _, id := range []string{"running", "unlisted", "testmode"} {
		if got := state(t, m, id); got.State != StateFailed || got.Error != "interrupted by server restart" {
			t.Fatalf("%s: %+v", id, got)
		}
	}
}

// A download is "running" but computes nothing a restart would lose: it resumes
// from its staging directory. A restart stops it, queues it, and the next
// process continues it, instead of waiting for the whole transfer.
func TestDrainStopsRestartableRunningJobsAndTheNextProcessContinuesThem(t *testing.T) {
	db := newJobsDB(t)
	downloading := make(chan struct{})
	var runs atomic.Int32
	first := New(db, Config{
		Resumable:   always,
		Restartable: func(op string) bool { return op == "model_install" },
		Runner: func(ctx context.Context, job Job, emit func(int, string)) (Result, error) {
			runs.Add(1)
			emit(35, "downloading model.safetensors")
			if job.Operation == "model_install" {
				close(downloading)
			}
			<-ctx.Done()
			return Result{}, ctx.Err()
		},
	})
	if err := first.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	install, _ := first.Submit(context.Background(), Spec{Operation: "model_install", Payload: []byte(`{"model_id":"m"}`)})
	<-downloading
	waitFor(t, "progress", func() bool { return state(t, first, install.ID).Progress == 35 })

	if err := first.Quiesce(context.Background()); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the download to be stopped and queued", func() bool {
		return state(t, first, install.ID).State == StateQueued && len(first.Active()) == 0
	})
	got := state(t, first, install.ID)
	if got.StartedAt != nil || got.Progress != 0 || got.Error != "" {
		t.Fatalf("a stopped download must be a clean queued job: %+v", got)
	}
	first.Close()

	second := New(db, Config{
		Resumable:   always,
		Restartable: func(op string) bool { return op == "model_install" },
		Runner:      func(context.Context, Job, func(int, string)) (Result, error) { return Result{Ref: "models/m"}, nil },
	})
	if err := second.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if final, err := second.Wait(ctx, install.ID); err != nil || final.State != StateSucceeded || final.ResultRef != "models/m" {
		t.Fatalf("final=%+v err=%v", final, err)
	}
}

// A computing job that is not restartable still holds the drain and is left
// running: stopping it would waste (or double-bill) work.
func TestDrainDoesNotStopANonRestartableRunningJob(t *testing.T) {
	stopped := make(chan struct{}, 1)
	release := make(chan struct{})
	running := make(chan struct{})
	m := startManager(t, Config{
		Resumable:   always,
		Restartable: func(op string) bool { return op == "model_install" },
		Runner: func(ctx context.Context, _ Job, _ func(int, string)) (Result, error) {
			close(running)
			select {
			case <-ctx.Done():
				stopped <- struct{}{}
				return Result{}, ctx.Err()
			case <-release:
				return Result{Ref: "done"}, nil
			}
		},
	})
	job, _ := m.Submit(context.Background(), Spec{Operation: "edit_instruct", Payload: []byte(`{}`)})
	<-running
	_ = m.Quiesce(context.Background())
	select {
	case <-stopped:
		t.Fatal("a computing, non-restartable job must not be stopped by a drain")
	case <-time.After(150 * time.Millisecond):
	}
	if len(m.Active()) != 1 {
		t.Fatalf("it must hold the drain: %+v", m.Active())
	}
	close(release)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if final, err := m.Wait(ctx, job.ID); err != nil || final.State != StateSucceeded {
		t.Fatalf("final=%+v err=%v", final, err)
	}
}

// An owner that registers its handlers after Start defers recovery: jobs the
// last process left behind are settled only when Recover runs, so Resumable is
// answered with every handler known. A job submitted meanwhile is left alone.
func TestDeferredRecoveryResumesWithHandlersRegisteredAfterStart(t *testing.T) {
	db := newJobsDB(t)
	st := newStore(db)
	now := time.Now()
	if err := st.insert(context.Background(), Job{ID: "left-behind", Operation: "edit_instruct", Lane: LaneGPU, State: StateAdmitting, CreatedAt: now, Payload: []byte(`{"p":1}`)}); err != nil {
		t.Fatal(err)
	}
	registered := false
	m := New(db, Config{
		DeferRecovery: true,
		Resumable:     func(op string) bool { return registered && op == "edit_instruct" },
		Runner: func(_ context.Context, job Job, _ func(int, string)) (Result, error) {
			return Result{Ref: string(job.Payload)}, nil
		},
	})
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if got := state(t, m, "left-behind"); got.State != StateAdmitting {
		t.Fatalf("recovery must wait for Recover: %+v", got)
	}
	live, err := m.Submit(context.Background(), Spec{Operation: "edit_instruct", Payload: []byte(`{"p":2}`)})
	if err != nil {
		t.Fatal(err)
	}
	registered = true
	if err := m.Recover(); err != nil {
		t.Fatal(err)
	}
	if err := m.Recover(); err == nil {
		t.Fatal("recovery must run once")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if final, err := m.Wait(ctx, "left-behind"); err != nil || final.State != StateSucceeded || final.ResultRef != `{"p":1}` {
		t.Fatalf("left-behind job not resumed: %+v %v", final, err)
	}
	if final, err := m.Wait(ctx, live.ID); err != nil || final.State != StateSucceeded {
		t.Fatalf("live job: %+v %v", final, err)
	}
}
