package renderjobs

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"device-studio/internal/jobs"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

type replacingProgressService struct {
	jobs.Service
	onList        func()
	replaceBefore bool
}

type scheduledProgressService struct {
	jobs.Service
	list func(context.Context) ([]jobs.Job, error)
}

func (s scheduledProgressService) ListProgress(ctx context.Context, _ jobs.Cursor, _ int) ([]jobs.Job, error) {
	return s.list(ctx)
}

func TestPumpSubmitWakesIdleProgressWithoutActivatingATestRoute(t *testing.T) {
	listed := make(chan struct{}, 4)
	pump := NewPump(nil, scheduledProgressService{list: func(context.Context) ([]jobs.Job, error) {
		listed <- struct{}{}
		return nil, nil
	}}, nil, nil, nil)
	pump.interval = time.Hour
	pump.Start(context.Background())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := pump.Stop(ctx); err != nil {
			t.Error(err)
		}
	})
	awaitProgressList(t, listed)
	if err := pump.ActivateTestRoute(context.Background()); err != nil {
		t.Fatal(err)
	}
	awaitProgressList(t, listed)
	if pump.testLease != nil {
		t.Fatal("primary wake installed a test lease")
	}
}

func TestPumpFollowsActiveWorkThenReturnsToIdleCadence(t *testing.T) {
	listed := make(chan struct{}, 4)
	var calls atomic.Int32
	pump := NewPump(nil, scheduledProgressService{list: func(context.Context) ([]jobs.Job, error) {
		listed <- struct{}{}
		if calls.Add(1) == 1 {
			return []jobs.Job{{ID: "active-job"}}, nil
		}
		return nil, nil
	}}, nil, nil, nil)
	pump.interval = time.Hour
	pump.advance = func(context.Context, string) error { return nil }
	pump.Start(context.Background())
	awaitProgressList(t, listed)
	awaitProgressList(t, listed)
	select {
	case <-listed:
		t.Fatal("empty progress listing kept the active cadence")
	case <-time.After(400 * time.Millisecond):
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := pump.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-listed:
		t.Fatal("progress continued after Stop joined the owner")
	default:
	}
}

func awaitProgressList(t *testing.T, listed <-chan struct{}) {
	t.Helper()
	select {
	case <-listed:
	case <-time.After(2 * time.Second):
		t.Fatal("progress did not respond while idle interval was one hour")
	}
}

func (s replacingProgressService) ListProgress(ctx context.Context, cursor jobs.Cursor, limit int) ([]jobs.Job, error) {
	if s.replaceBefore && s.onList != nil {
		s.onList()
	}
	rows, err := s.Service.ListProgress(ctx, cursor, limit)
	if !s.replaceBefore && err == nil && s.onList != nil {
		s.onList()
	}
	return rows, err
}

func TestPumpStopsBeforeAdvancingWhenTestLeaseChangesAfterList(t *testing.T) {
	t.Run("after listing", func(t *testing.T) { assertPumpStopsOnLeaseReplacement(t, false) })
	t.Run("before listing", func(t *testing.T) { assertPumpStopsOnLeaseReplacement(t, true) })
	t.Run("same ID reinstall after listing", func(t *testing.T) { assertPumpStopsOnSameIDReinstall(t, false) })
	t.Run("same ID reinstall before listing", func(t *testing.T) { assertPumpStopsOnSameIDReinstall(t, true) })
}

func assertPumpStopsOnSameIDReinstall(t *testing.T, replaceBeforeList bool) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		_, err := pool.ExecContext(ctx, jobs.Schema())
		return err
	})
	rootPath := filepath.Join(t.TempDir(), "same-route-root")
	roots := filerouting.New(storage.Paths{DataDir: t.TempDir()})
	install := func() {
		t.Helper()
		if err := db.InstallTestPool(ctx, filepath.Join(t.TempDir(), "lease-same.db"), "lease-same", time.Minute); err != nil {
			t.Fatal(err)
		}
		if err := roots.InstallTestRoots(storage.Paths{DataDir: rootPath}, "lease-same", time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	install()
	service := jobs.NewService(jobs.NewSQLiteRepository(db), nil)
	testCtx := database.WithTestMode(ctx)
	if _, _, err := service.Submit(testCtx, jobs.SubmitInput{IdempotencyKey: "same-id-pump-job", Input: pumpTestInput()}); err != nil {
		t.Fatal(err)
	}
	var advances atomic.Int32
	pump := NewPump(nil, replacingProgressService{Service: service, replaceBefore: replaceBeforeList, onList: func() {
		if err := roots.ClearTestRoots("lease-same"); err != nil {
			t.Error(err)
		}
		if err := db.ClearTestPool("lease-same"); err != nil {
			t.Error(err)
		}
		install()
	}}, db, roots, nil)
	pump.advance = func(context.Context, string) error { advances.Add(1); return nil }
	if err := pump.ActivateTestRoute(testCtx); err != nil {
		t.Fatal(err)
	}
	lifecycle := context.Background()
	activeCtx, activation, ok := pump.activeTestContext(lifecycle)
	if !ok {
		t.Fatal("expected activation before replacement")
	}
	pump.advanceRoute(lifecycle, activeCtx, jobs.Cursor{}, "test", activation)
	if got := advances.Load(); got != 0 {
		t.Fatalf("advanced %d jobs after same-ID reinstall, want none", got)
	}
	if pump.testLease != nil {
		t.Fatalf("stale activation remained after reinstall: %#v", pump.testLease)
	}
	if got := db.LeaseStats().PrimaryDuringTestModeRequests; got != 0 {
		t.Fatalf("test work fell through to primary %d times", got)
	}
}

func assertPumpStopsOnLeaseReplacement(t *testing.T, replaceBeforeList bool) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	// InstallTestPool's initializer receives *sql.DB. Keep its schema setup
	// explicit here so the list comes from the leased pool, not primary.
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		_, err := pool.ExecContext(ctx, jobs.Schema())
		return err
	})
	roots := filerouting.New(storage.Paths{DataDir: t.TempDir()})
	install := func(id string) string {
		t.Helper()
		poolPath := filepath.Join(t.TempDir(), id+".db")
		if err := db.InstallTestPool(ctx, poolPath, id, time.Minute); err != nil {
			t.Fatal(err)
		}
		paths := storage.Paths{DataDir: filepath.Join(t.TempDir(), "data-"+id)}
		if err := roots.InstallTestRoots(paths, id, time.Minute); err != nil {
			t.Fatal(err)
		}
		return paths.DataDir
	}
	firstRoot := install("lease-a")
	service := jobs.NewService(jobs.NewSQLiteRepository(db), nil)
	testCtx := database.WithTestMode(ctx)
	_, _, err = service.Submit(testCtx, jobs.SubmitInput{IdempotencyKey: "pump-test-job", Input: pumpTestInput()})
	if err != nil {
		t.Fatal(err)
	}
	var advances atomic.Int32
	pump := NewPump(nil, replacingProgressService{Service: service, replaceBefore: replaceBeforeList, onList: func() {
		if err := roots.ClearTestRoots("lease-a"); err != nil {
			t.Error(err)
		}
		if err := db.ClearTestPool("lease-a"); err != nil {
			t.Error(err)
		}
		install("lease-b")
	}}, db, roots, nil)
	pump.advance = func(context.Context, string) error { advances.Add(1); return nil }
	if err := pump.ActivateTestRoute(testCtx); err != nil {
		t.Fatal(err)
	}
	if pump.testLease == nil || pump.testLease.leaseID != "lease-a" || pump.testLease.dataRoot != firstRoot {
		t.Fatalf("activation did not pin original route: %#v", pump.testLease)
	}
	lifecycle := context.Background()
	activeCtx, activation, ok := pump.activeTestContext(lifecycle)
	if !ok {
		t.Fatal("expected original lease activation before list")
	}
	pump.advanceRoute(lifecycle, activeCtx, jobs.Cursor{}, "test", activation)
	if got := advances.Load(); got != 0 {
		t.Fatalf("advanced %d jobs after lease replacement, want none", got)
	}
	if pump.testLease != nil {
		t.Fatalf("stale activation remained after replacement: %#v", pump.testLease)
	}
	if got := db.LeaseStats().PrimaryDuringTestModeRequests; got != 0 {
		t.Fatalf("test work fell through to primary %d times", got)
	}
}

func pumpTestInput() jobs.Input {
	return jobs.Input{
		ProjectID: "project-a", ShotRevisionID: "shot-rev-1", ShotRevision: 1,
		Bindings:          []jobs.ArtifactBinding{{Role: "plate", ArtifactID: "plate", RevisionID: "plate-v1", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, {Role: "screen", ArtifactID: "screen", RevisionID: "screen-v1", SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},
		SourceClockDigest: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", ToolProfileDigest: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", RendererVersion: "renderer-v1", OutputProfile: "png-v1",
	}
}
