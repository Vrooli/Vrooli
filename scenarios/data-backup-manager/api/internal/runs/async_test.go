package runs_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/runs"
	runsmocks "data-backup-manager/internal/runs/mocks"
	"data-backup-manager/internal/sources"
	sourcesmocks "data-backup-manager/internal/sources/mocks"
	"data-backup-manager/internal/testutil/mocks"
)

// TestTriggerRun_AsyncLifecycleIsPersisted proves the async contract end to end
// with the real background executor: TriggerRun returns a non-terminal run
// immediately, and the run's status is persisted as it advances (capturing
// while the source is captured, snapshotting while the snapshot is written,
// completed at the end). The capture/snapshot hooks block so the test can
// observe each persisted transition through the real sqlite repo mid-flight.
func TestTriggerRun_AsyncLifecycleIsPersisted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repo := runs.NewSQLiteRepository(newRunsDB(t), mocks.NewFakeClock(time.Time{}))

	eng := &mocks.FakeKopiaEngine{}
	capt := &sourcesmocks.FakeCapturer{SourceKind: sources.KindFilesystem}
	registry := sources.NewRegistry(capt)

	captureReached := make(chan struct{})
	releaseCapture := make(chan struct{})
	snapshotReached := make(chan struct{})
	releaseSnapshot := make(chan struct{})

	capt.CaptureFn = func(_ context.Context, spec sources.CaptureSpec) (sources.Artifact, error) {
		close(captureReached)
		<-releaseCapture
		return sources.Artifact{Path: spec.StageDir, Bytes: 10}, nil
	}
	eng.SnapshotCreateFn = func(_ context.Context, _, _ string, _ engine.SnapshotMetadata) (engine.Snapshot, error) {
		close(snapshotReached)
		<-releaseSnapshot
		return engine.Snapshot{ID: "snap-1"}, nil
	}

	plan := runs.PlanForRun{ID: "plan-1", TargetIDs: []string{"t1"}, DestinationIDs: []string{"dst-1"}}
	svc := runs.NewService(runs.Deps{
		Repo:         repo,
		Plans:        &runsmocks.FakePlanLookup{Plans: map[string]runs.PlanForRun{plan.ID: plan}},
		Targets:      &runsmocks.FakeTargetLookup{Targets: map[string]runs.TargetForRun{"t1": {ID: "t1", Kind: sources.KindFilesystem, Locator: "a"}}},
		Destinations: &runsmocks.FakeDestinationLookup{Destinations: map[string]runs.DestinationForRun{"dst-1": {ID: "dst-1", Name: "nightly"}}},
		Engine:       eng,
		Sources:      registry,
		Clock:        mocks.NewFakeClock(time.Time{}),
		StagingRoot:  t.TempDir(),
		BaseContext:  ctx,
		Executor:     runs.NewAsyncExecutor(1),
	})
	defer func() { _ = svc.Shutdown(ctx) }()

	pending, err := svc.TriggerRun(ctx, "plan-1", runs.TriggerManual)
	if err != nil {
		t.Fatalf("TriggerRun: %v", err)
	}
	if pending.Status != runs.RunPending {
		t.Fatalf("returned status = %s, want pending (async returns immediately)", pending.Status)
	}
	runID := pending.ID

	waitSignal(t, captureReached, "capture to begin")
	if got := mustStatus(t, repo, runID); got != runs.RunCapturing {
		t.Fatalf("status during capture = %s, want capturing", got)
	}
	close(releaseCapture)

	waitSignal(t, snapshotReached, "snapshot to begin")
	if got := mustStatus(t, repo, runID); got != runs.RunSnapshotting {
		t.Fatalf("status during snapshot = %s, want snapshotting", got)
	}
	close(releaseSnapshot)

	final := waitTerminal(t, repo, runID)
	if final.Status != runs.RunCompleted {
		t.Fatalf("final status = %s, want completed", final.Status)
	}
	if final.FinishedAt.IsZero() {
		t.Fatalf("completed run missing finished_at")
	}
	if len(final.Outcomes) != 1 || final.Outcomes[0].SnapshotID != "snap-1" {
		t.Fatalf("outcome = %+v, want one succeeded snapshot", final.Outcomes)
	}
}

func waitSignal(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func mustStatus(t *testing.T, repo runs.Repository, id string) runs.RunStatus {
	t.Helper()
	r, err := repo.GetRun(context.Background(), id)
	if err != nil {
		t.Fatalf("GetRun %s: %v", id, err)
	}
	return r.Status
}

func waitTerminal(t *testing.T, repo runs.Repository, id string) runs.Run {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		r, err := repo.GetRun(context.Background(), id)
		if err != nil {
			t.Fatalf("GetRun %s: %v", id, err)
		}
		switch r.Status {
		case runs.RunCompleted, runs.RunPartialFailed, runs.RunFailed:
			return r
		}
		select {
		case <-deadline:
			t.Fatalf("run %s did not reach terminal status (last=%s)", id, r.Status)
		case <-time.After(5 * time.Millisecond):
		}
	}
}

// TestTriggerRun_BoundedConcurrentFanOut proves target×destination units run in
// parallel but never exceed DBM_RUN_CONCURRENCY. With 6 targets and a limit of
// 3, exactly 3 captures are in flight at the barrier and the 4th cannot start
// until one releases — so the observed peak is the bound, not more, not one.
func TestTriggerRun_BoundedConcurrentFanOut(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repo := runs.NewSQLiteRepository(newRunsDB(t), mocks.NewFakeClock(time.Time{}))

	const limit = 3
	eng := &mocks.FakeKopiaEngine{}
	capt := &sourcesmocks.FakeCapturer{SourceKind: sources.KindFilesystem}
	registry := sources.NewRegistry(capt)

	var (
		mu       sync.Mutex
		inflight int
		maxSeen  int
	)
	reachedLimit := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	capt.CaptureFn = func(_ context.Context, spec sources.CaptureSpec) (sources.Artifact, error) {
		mu.Lock()
		inflight++
		if inflight > maxSeen {
			maxSeen = inflight
		}
		if inflight == limit {
			once.Do(func() { close(reachedLimit) })
		}
		mu.Unlock()
		<-release
		mu.Lock()
		inflight--
		mu.Unlock()
		return sources.Artifact{Path: spec.StageDir, Bytes: 1}, nil
	}

	targetIDs := []string{"t1", "t2", "t3", "t4", "t5", "t6"}
	targets := map[string]runs.TargetForRun{}
	for _, id := range targetIDs {
		targets[id] = runs.TargetForRun{ID: id, Kind: sources.KindFilesystem, Locator: id}
	}
	plan := runs.PlanForRun{ID: "plan-1", TargetIDs: targetIDs, DestinationIDs: []string{"dst-1"}}

	svc := runs.NewService(runs.Deps{
		Repo:              repo,
		Plans:             &runsmocks.FakePlanLookup{Plans: map[string]runs.PlanForRun{plan.ID: plan}},
		Targets:           &runsmocks.FakeTargetLookup{Targets: targets},
		Destinations:      &runsmocks.FakeDestinationLookup{Destinations: map[string]runs.DestinationForRun{"dst-1": {ID: "dst-1", Name: "nightly"}}},
		Engine:            eng,
		Sources:           registry,
		Clock:             mocks.NewFakeClock(time.Time{}),
		StagingRoot:       t.TempDir(),
		BaseContext:       ctx,
		TargetConcurrency: limit,
		Executor:          runs.NewAsyncExecutor(1),
	})
	defer func() { _ = svc.Shutdown(ctx) }()

	pending, err := svc.TriggerRun(ctx, "plan-1", runs.TriggerManual)
	if err != nil {
		t.Fatalf("TriggerRun: %v", err)
	}

	waitSignal(t, reachedLimit, "concurrency limit to be reached")
	mu.Lock()
	if inflight != limit {
		t.Errorf("in-flight at barrier = %d, want exactly %d (bound)", inflight, limit)
	}
	mu.Unlock()
	close(release)

	final := waitTerminal(t, repo, pending.ID)
	if final.Status != runs.RunCompleted {
		t.Fatalf("final status = %s, want completed", final.Status)
	}
	if len(final.Outcomes) != len(targetIDs) {
		t.Fatalf("outcomes = %d, want %d", len(final.Outcomes), len(targetIDs))
	}
	mu.Lock()
	defer mu.Unlock()
	if maxSeen != limit {
		t.Fatalf("peak concurrency = %d, want exactly %d (parallel but bounded)", maxSeen, limit)
	}
}

// TestReconcile_ClosesOrphanedRuns proves startup reconciliation marks every
// run left non-terminal by a crash/restart as failed with a reason, and leaves
// already-terminal runs untouched.
func TestReconcile_ClosesOrphanedRuns(t *testing.T) {
	ctx := context.Background()
	repo := runs.NewSQLiteRepository(newRunsDB(t), mocks.NewFakeClock(time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)))

	// Two orphans (pending + capturing) and one already-completed run.
	mustCreate(t, repo, "orphan-pending", runs.RunPending)
	mustCreate(t, repo, "orphan-capturing", runs.RunCapturing)
	mustCreate(t, repo, "done", runs.RunCompleted)

	svc := runs.NewService(runs.Deps{
		Repo:     repo,
		Clock:    mocks.NewFakeClock(time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)),
		Executor: runsmocks.NewSyncExecutor(),
	})

	if err := svc.Reconcile(ctx); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	for _, id := range []string{"orphan-pending", "orphan-capturing"} {
		r, err := repo.GetRun(ctx, id)
		if err != nil {
			t.Fatalf("GetRun %s: %v", id, err)
		}
		if r.Status != runs.RunFailed {
			t.Errorf("%s status = %s, want failed", id, r.Status)
		}
		if r.Error == "" {
			t.Errorf("%s missing reconciliation reason", id)
		}
		if r.FinishedAt.IsZero() {
			t.Errorf("%s missing finished_at after reconcile", id)
		}
	}

	done, err := repo.GetRun(ctx, "done")
	if err != nil {
		t.Fatalf("GetRun done: %v", err)
	}
	if done.Status != runs.RunCompleted {
		t.Errorf("already-terminal run was altered: status = %s", done.Status)
	}
	if done.Error != "" {
		t.Errorf("already-terminal run got an error reason: %q", done.Error)
	}

	// Reconcile is idempotent: a second pass finds no orphans.
	if err := svc.Reconcile(ctx); err != nil {
		t.Fatalf("Reconcile (second pass): %v", err)
	}
}

func mustCreate(t *testing.T, repo runs.Repository, id string, status runs.RunStatus) {
	t.Helper()
	if _, err := repo.CreateRun(context.Background(), runs.Run{
		ID:        id,
		PlanID:    "plan-x",
		Trigger:   runs.TriggerManual,
		Status:    status,
		StartedAt: time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("create run %s: %v", id, err)
	}
}
