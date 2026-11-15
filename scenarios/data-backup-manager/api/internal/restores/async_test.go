package restores_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"data-backup-manager/internal/platformreceipts"
	"data-backup-manager/internal/restores"
	restoresmocks "data-backup-manager/internal/restores/mocks"
	"data-backup-manager/internal/sources"
	sourcesmocks "data-backup-manager/internal/sources/mocks"
	"data-backup-manager/internal/testutil/mocks"

	"github.com/vrooli/api-core/scheduletest"
)

type receiptCollector struct{ rows chan platformreceipts.Input }

func (c receiptCollector) Record(_ context.Context, row platformreceipts.Input) error {
	c.rows <- row
	return nil
}

type changesWhileQueued struct{ calls int }

func (l *changesWhileQueued) DestinationForRestore(_ context.Context, id string) (restores.DestinationForRestore, error) {
	l.calls++
	if l.calls == 1 {
		return restores.DestinationForRestore{ID: id, Name: "nightly"}, nil
	}
	return restores.DestinationForRestore{}, fmt.Errorf("volume readiness changed")
}

func (l *changesWhileQueued) WithResolvedDestination(ctx context.Context, id string, use func(restores.DestinationForRestore) error) error {
	destination, err := l.DestinationForRestore(ctx, id)
	if err != nil {
		return err
	}
	return use(destination)
}

type changesBeforeUse struct{}

func (changesBeforeUse) DestinationForRestore(_ context.Context, id string) (restores.DestinationForRestore, error) {
	return restores.DestinationForRestore{ID: id, Name: "nightly"}, nil
}

func (changesBeforeUse) WithResolvedDestination(context.Context, string, func(restores.DestinationForRestore) error) error {
	return fmt.Errorf("volume readiness changed immediately before repository use")
}

func TestVerifyTargetRechecksDestinationBeforeQueuedRepositoryUse(t *testing.T) {
	ctx := context.Background()
	repo := restores.NewSQLiteRepository(newRestoresDB(t), scheduletest.New(time.Time{}))
	eng := &mocks.FakeKopiaEngine{}
	destinations := &changesWhileQueued{}
	svc := restores.NewService(restores.Deps{
		Repo:         repo,
		Targets:      &restoresmocks.FakeTargetLookup{Targets: map[string]restores.TargetForRestore{"t1": {ID: "t1", Kind: sources.KindFilesystem, Locator: "loc"}}},
		Destinations: destinations,
		Engine:       eng,
		Sources:      sources.NewRegistry(&sourcesmocks.FakeCapturer{SourceKind: sources.KindFilesystem}),
		Clock:        scheduletest.New(time.Time{}), ScratchRoot: t.TempDir(), Executor: restoresmocks.NewSyncExecutor(),
	})
	defer func() { _ = svc.Shutdown(ctx) }()
	result, err := svc.VerifyTarget(ctx, "t1", "dst-1", "snap-1")
	if err != nil {
		t.Fatalf("VerifyTarget: %v", err)
	}
	if result.Status != restores.RestoreFailed || result.Error == "" {
		t.Fatalf("result = %+v, want failed after worker re-resolution", result)
	}
	if destinations.calls != 2 {
		t.Fatalf("destination resolutions = %d, want request + worker", destinations.calls)
	}
	if len(eng.Calls) != 0 {
		t.Fatalf("engine touched after destination readiness changed: %v", eng.Calls)
	}
}

func TestVerifyTargetRechecksDestinationAtRepositoryUseBoundary(t *testing.T) {
	ctx := context.Background()
	repo := restores.NewSQLiteRepository(newRestoresDB(t), scheduletest.New(time.Time{}))
	eng := &mocks.FakeKopiaEngine{}
	svc := restores.NewService(restores.Deps{
		Repo:         repo,
		Targets:      &restoresmocks.FakeTargetLookup{Targets: map[string]restores.TargetForRestore{"t1": {ID: "t1", Kind: sources.KindFilesystem, Locator: "loc"}}},
		Destinations: changesBeforeUse{},
		Engine:       eng,
		Sources:      sources.NewRegistry(&sourcesmocks.FakeCapturer{SourceKind: sources.KindFilesystem}),
		Clock:        scheduletest.New(time.Time{}), ScratchRoot: t.TempDir(), Executor: restoresmocks.NewSyncExecutor(),
	})
	defer func() { _ = svc.Shutdown(ctx) }()
	result, err := svc.VerifyTarget(ctx, "t1", "dst-1", "snap-1")
	if err != nil {
		t.Fatalf("VerifyTarget: %v", err)
	}
	if result.Status != restores.RestoreFailed || result.Error == "" {
		t.Fatalf("result = %+v, want failed after final destination gate", result)
	}
	for _, call := range eng.Calls {
		if call == "SnapshotRestore(nightly,snap-1)" || call == "SnapshotVerify(nightly,snap-1,100)" {
			t.Fatalf("engine used a destination after its final gate failed: %v", eng.Calls)
		}
	}
}

// TestVerifyTarget_AsyncLifecycleIsPersisted proves the async contract with the
// real background executor: VerifyTarget returns a non-terminal record
// immediately, the status advances to verifying while the engine works, and the
// record lands verified with a checksum + last_verified_at. The verify hook
// blocks so the test observes the mid-flight persisted transition.
func TestVerifyTarget_AsyncLifecycleIsPersisted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := restores.NewSQLiteRepository(newRestoresDB(t), scheduletest.New(time.Time{}))
	eng := &mocks.FakeKopiaEngine{}
	capt := &sourcesmocks.FakeCapturer{SourceKind: sources.KindFilesystem}
	receipts := make(chan platformreceipts.Input, 1)

	verifyReached := make(chan struct{})
	releaseVerify := make(chan struct{})
	eng.SnapshotVerifyFn = func(_ context.Context, _, _ string, _ int) error {
		close(verifyReached)
		<-releaseVerify
		return nil
	}

	svc := restores.NewService(restores.Deps{
		Repo:             repo,
		Targets:          &restoresmocks.FakeTargetLookup{Targets: map[string]restores.TargetForRestore{"t1": {ID: "t1", Kind: sources.KindFilesystem, Locator: "loc"}}},
		Destinations:     &restoresmocks.FakeDestinationLookup{Destinations: map[string]restores.DestinationForRestore{"dst-1": {ID: "dst-1", Name: "nightly"}}},
		Engine:           eng,
		Sources:          sources.NewRegistry(capt),
		Clock:            scheduletest.New(time.Time{}),
		ScratchRoot:      t.TempDir(),
		BaseContext:      ctx,
		Executor:         restores.NewAsyncExecutor(1),
		PlatformReceipts: receiptCollector{rows: receipts},
	})
	defer func() { _ = svc.Shutdown(ctx) }()

	pending, err := svc.VerifyTarget(ctx, "t1", "dst-1", "snap-1")
	if err != nil {
		t.Fatalf("VerifyTarget: %v", err)
	}
	if pending.Status == restores.RestoreVerified || pending.Status == restores.RestoreFailed {
		t.Fatalf("returned status = %s, want non-terminal (async returns immediately)", pending.Status)
	}
	id := pending.ID

	waitSig(t, verifyReached, "verify to begin")
	if got := mustStatus(t, repo, id); got != restores.RestoreVerifying {
		t.Fatalf("status during verify = %s, want verifying", got)
	}
	close(releaseVerify)

	final := waitTerminalRestore(t, repo, id)
	if final.Status != restores.RestoreVerified {
		t.Fatalf("final status = %s, want verified", final.Status)
	}
	if final.Checksum == "" || final.LastVerifiedAt.IsZero() {
		t.Fatalf("verified record missing checksum/last_verified_at: %+v", final)
	}
	if final.FinishedAt.IsZero() {
		t.Fatalf("verified record missing finished_at")
	}
	select {
	case receipt := <-receipts:
		if receipt.Capability != "scratch_restore" || receipt.Outcome != "passed" || receipt.TargetID != "t1" || receipt.DestinationID != "dst-1" {
			t.Fatalf("scratch restore receipt = %+v", receipt)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("successful native scratch restore did not emit a platform receipt")
	}
}

// TestReconcile_FailsOrphanedRestores proves a restore left non-terminal by a
// crash is closed as failed (fail-not-resume) and never falsely verified.
func TestReconcile_FailsOrphanedRestores(t *testing.T) {
	ctx := context.Background()
	repo := restores.NewSQLiteRepository(newRestoresDB(t), scheduletest.New(time.Time{}))

	orphan, err := repo.CreateRestore(ctx, restores.Restore{
		TargetID: "t1", DestinationID: "dst-1", SnapshotID: "snap-1",
		Mode: restores.ModeVerify, Status: restores.RestoreVerifying,
	})
	if err != nil {
		t.Fatalf("seed orphan: %v", err)
	}

	svc := restores.NewService(restores.Deps{
		Repo:     repo,
		Clock:    scheduletest.New(time.Time{}),
		Executor: restoresmocks.NewSyncExecutor(),
	})
	if err := svc.Reconcile(ctx); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	got, err := repo.GetRestore(ctx, orphan.ID)
	if err != nil {
		t.Fatalf("GetRestore: %v", err)
	}
	if got.Status != restores.RestoreFailed {
		t.Fatalf("orphan status = %s, want failed", got.Status)
	}
	if !got.LastVerifiedAt.IsZero() {
		t.Fatalf("reconciled orphan must never be falsely verified: %+v", got)
	}
}

func waitSig(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func mustStatus(t *testing.T, repo restores.Repository, id string) restores.RestoreStatus {
	t.Helper()
	r, err := repo.GetRestore(context.Background(), id)
	if err != nil {
		t.Fatalf("GetRestore: %v", err)
	}
	return r.Status
}

func waitTerminalRestore(t *testing.T, repo restores.Repository, id string) restores.Restore {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		r, err := repo.GetRestore(context.Background(), id)
		if err != nil {
			t.Fatalf("GetRestore: %v", err)
		}
		switch r.Status {
		case restores.RestoreVerified, restores.RestoreRestored, restores.RestoreFailed:
			return r
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("restore %s did not reach a terminal state in time", id)
	return restores.Restore{}
}
