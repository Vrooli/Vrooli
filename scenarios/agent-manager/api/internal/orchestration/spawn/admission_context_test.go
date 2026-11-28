package spawn

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

// A disconnected HTTP producer must not move admitted SQL/file effects back
// into primary storage, nor may a cleared lease be replaced by a newer lease.
func TestAdmissionRetainsPinnedStorageAfterRequestCancellation(t *testing.T) {
	db, err := coredb.Open(t.Context(), coredb.Config{Driver: coredb.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(t.Context(), "CREATE TABLE effects(value TEXT)"); err != nil {
		t.Fatal(err)
	}
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		_, err := pool.ExecContext(ctx, "CREATE TABLE effects(value TEXT)")
		return err
	})
	if err := db.InstallTestPool(t.Context(), "file:"+filepath.Join(t.TempDir(), "test.db"), "lease-a", time.Minute); err != nil {
		t.Fatal(err)
	}
	paths := func(root string) storage.Paths {
		return storage.Paths{ConfigDir: filepath.Join(root, "config"), DataDir: filepath.Join(root, "data"), CacheDir: filepath.Join(root, "cache"), LogsDir: filepath.Join(root, "logs"), StateDir: filepath.Join(root, "state")}
	}
	roots := filerouting.New(paths(t.TempDir()))
	leased := paths(t.TempDir())
	if err := roots.InstallTestRoots(leased, "lease-a", time.Minute); err != nil {
		t.Fatal(err)
	}
	request, err := db.PinTestPool(t.Context(), "lease-a")
	if err != nil {
		t.Fatal(err)
	}
	request, err = roots.PinTestRoots(request, "lease-a")
	if err != nil {
		t.Fatal(err)
	}
	request, cancelRequest := context.WithCancel(request)
	defer cancelRequest()
	d := New(Config{MaxStartingConcurrency: 1, QueueCapacity: 1})
	defer d.Close()
	id := uuid.New()
	admission, err := d.ReserveAdmission(request, id)
	if err != nil {
		t.Fatal(err)
	}
	defer d.ReleaseAdmission(id, admission)
	cancelRequest()
	if err := admission.Context().Err(); err != nil {
		t.Fatalf("request cancellation discarded admitted work: %v", err)
	}
	completed := make(chan error, 1)
	if err := d.EnqueueAdmitted(&Job{RunID: id, Fn: func(started StartedFn) {
		started()
		ctx := admission.Context()
		if _, err := db.ExecContext(ctx, "INSERT INTO effects VALUES ('admitted')"); err != nil {
			completed <- err
			return
		}
		root, err := roots.Pick(ctx, storage.ClassState)
		if err == nil && root != leased.StateDir {
			err = fmt.Errorf("admitted file effect selected primary root: %s", root)
		}
		if err == nil {
			roots.RecordWrite(ctx)
		}
		completed <- err
	}}, admission); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-completed:
		if err != nil {
			t.Error(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("admitted executor did not complete")
	}
	for _, sample := range []struct {
		ctx  context.Context
		want int
	}{{t.Context(), 0}, {admission.Context(), 1}} {
		var count int
		if err := db.QueryRowContext(sample.ctx, "SELECT COUNT(*) FROM effects").Scan(&count); err != nil || count != sample.want {
			t.Fatalf("wrong SQL storage: effects=%d want=%d err=%v", count, sample.want, err)
		}
	}
	if stats := roots.LeaseStats(); stats.TestRootWrites != 1 || stats.PrimaryWritesDuringTestMode != 0 {
		t.Fatalf("wrong file routing evidence: %+v", stats)
	}
	if err := db.ClearTestPool("lease-a"); err != nil {
		t.Fatal(err)
	}
	if err := roots.ClearTestRoots("lease-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(admission.Context(), "INSERT INTO effects VALUES ('after-clear')"); err == nil {
		t.Fatal("cleared pinned SQL context fell back to primary")
	}
	if _, err := roots.Pick(admission.Context(), storage.ClassState); err == nil {
		t.Fatal("cleared pinned file context fell back to primary")
	}
	d.CancelAdmission(id)
	if admission.Context().Err() != context.Canceled {
		t.Fatal("owner cancellation no longer cancels admitted execution")
	}
}
