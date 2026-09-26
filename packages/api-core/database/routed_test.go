package database_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/vrooli/api-core/database"
)

// errorsAs is a tiny helper so test expressions stay one-liners.
func errorsAs(err error, target any) bool { return errors.As(err, target) }

// openSQLitePool opens a fresh SQLite pool at path. Tests use unique file
// paths so the two pools live in separate files even when both are "test"
// targets — modernc.org/sqlite's in-memory databases are per-connection and
// don't survive pool reuse, which would defeat the routing assertion.
func openSQLitePool(t *testing.T, path string) (*database.RoutedDB, error) {
	t.Helper()
	return database.Open(context.Background(), database.Config{
		Driver: database.DriverSQLite,
		DSN:    path,
	})
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func seedPool(t *testing.T, path, marker string) {
	t.Helper()
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open seed pool: %v", err)
	}
	defer raw.Close()
	mustExec(t, raw, `CREATE TABLE IF NOT EXISTS t (v TEXT NOT NULL)`)
	mustExec(t, raw, `INSERT INTO t (v) VALUES (?)`, marker)
}

func readMarker(t *testing.T, r *database.RoutedDB, ctx context.Context) string {
	t.Helper()
	var v string
	row := r.QueryRowContext(ctx, `SELECT v FROM t LIMIT 1`)
	if err := row.Scan(&v); err != nil {
		t.Fatalf("scan: %v", err)
	}
	return v
}

// TestRoutedDB_RoutesByContext is the core invariant: with no test pool,
// every read sees the primary; with a test pool installed, only test-mode
// contexts see the test pool.
func TestRoutedDB_RoutesByContext(t *testing.T) {
	dir := t.TempDir()
	primaryPath := filepath.Join(dir, "primary.db")
	testPath := filepath.Join(dir, "test.db")

	seedPool(t, primaryPath, "PRIMARY")
	seedPool(t, testPath, "TEST")

	r, err := openSQLitePool(t, primaryPath)
	if err != nil {
		t.Fatalf("open routed: %v", err)
	}
	defer r.Close()

	ctxPlain := context.Background()
	ctxTest := database.WithTestMode(ctxPlain)

	// No test pool installed yet: both contexts must hit primary.
	if got := readMarker(t, r, ctxPlain); got != "PRIMARY" {
		t.Fatalf("pre-install plain: got %q, want PRIMARY", got)
	}
	if got := readMarker(t, r, ctxTest); got != "PRIMARY" {
		t.Fatalf("pre-install test-mode without pool: got %q, want PRIMARY (fallback)", got)
	}

	if err := r.InstallTestPool(ctxPlain, testPath, "lease-route", 0); err != nil {
		t.Fatalf("install test pool: %v", err)
	}

	// After install: plain hits primary, test-mode hits test.
	if got := readMarker(t, r, ctxPlain); got != "PRIMARY" {
		t.Fatalf("post-install plain: got %q, want PRIMARY", got)
	}
	if got := readMarker(t, r, ctxTest); got != "TEST" {
		t.Fatalf("post-install test-mode: got %q, want TEST", got)
	}

	if err := r.ClearTestPool("lease-route"); err != nil {
		t.Fatalf("clear test pool: %v", err)
	}

	// After clear: both contexts hit primary again.
	if got := readMarker(t, r, ctxTest); got != "PRIMARY" {
		t.Fatalf("post-clear test-mode: got %q, want PRIMARY", got)
	}
}

func TestRoutedDB_InitializesNewTestPoolBeforeRouting(t *testing.T) {
	dir := t.TempDir()
	r, err := openSQLitePool(t, filepath.Join(dir, "primary.db"))
	if err != nil {
		t.Fatalf("open routed: %v", err)
	}
	defer r.Close()

	r.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		_, err := pool.ExecContext(ctx, `CREATE TABLE initialized (value TEXT NOT NULL); INSERT INTO initialized(value) VALUES ('ready')`)
		return err
	})
	if err := r.InstallTestPool(context.Background(), filepath.Join(dir, "test.db"), "lease-init", 0); err != nil {
		t.Fatalf("install initialized pool: %v", err)
	}

	var value string
	if err := r.QueryRowContext(database.WithTestMode(context.Background()), `SELECT value FROM initialized`).Scan(&value); err != nil {
		t.Fatalf("test-mode query against initialized pool: %v", err)
	}
	if value != "ready" {
		t.Fatalf("initialized value = %q, want ready", value)
	}
}

// TestRoutedDB_TransactionsArePoolBound verifies the §8.4 contract: a tx
// commits to whichever pool was picked when BeginTx ran.
func TestRoutedDB_TransactionsArePoolBound(t *testing.T) {
	dir := t.TempDir()
	primaryPath := filepath.Join(dir, "primary.db")
	testPath := filepath.Join(dir, "test.db")

	seedPool(t, primaryPath, "PRIMARY")
	seedPool(t, testPath, "TEST")

	r, err := openSQLitePool(t, primaryPath)
	if err != nil {
		t.Fatalf("open routed: %v", err)
	}
	defer r.Close()

	if err := r.InstallTestPool(context.Background(), testPath, "lease-tx", 0); err != nil {
		t.Fatalf("install test pool: %v", err)
	}

	ctxTest := database.WithTestMode(context.Background())
	tx, err := r.BeginTx(ctxTest, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	if _, err := tx.ExecContext(ctxTest, `INSERT INTO t (v) VALUES ('TX_TEST')`); err != nil {
		t.Fatalf("tx insert: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Test pool received the insert.
	rows, err := r.QueryContext(ctxTest, `SELECT v FROM t ORDER BY v`)
	if err != nil {
		t.Fatalf("query test: %v", err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, v)
	}
	if len(got) != 2 || got[0] != "TEST" || got[1] != "TX_TEST" {
		t.Fatalf("test pool rows = %v, want [TEST TX_TEST]", got)
	}

	// Primary pool unchanged.
	ctxPlain := context.Background()
	rows2, err := r.QueryContext(ctxPlain, `SELECT v FROM t`)
	if err != nil {
		t.Fatalf("query primary: %v", err)
	}
	defer rows2.Close()
	var primaryRows []string
	for rows2.Next() {
		var v string
		if err := rows2.Scan(&v); err != nil {
			t.Fatalf("scan: %v", err)
		}
		primaryRows = append(primaryRows, v)
	}
	if len(primaryRows) != 1 || primaryRows[0] != "PRIMARY" {
		t.Fatalf("primary rows = %v, want [PRIMARY]", primaryRows)
	}
}

// TestRoutedDB_InstallTestPool_SameLeaseReplaces verifies the idempotent
// retry contract: a second install with the same lease replaces the
// previous pool.
func TestRoutedDB_InstallTestPool_SameLeaseReplaces(t *testing.T) {
	dir := t.TempDir()
	primaryPath := filepath.Join(dir, "primary.db")
	firstTestPath := filepath.Join(dir, "test1.db")
	secondTestPath := filepath.Join(dir, "test2.db")

	seedPool(t, primaryPath, "PRIMARY")
	seedPool(t, firstTestPath, "TEST1")
	seedPool(t, secondTestPath, "TEST2")

	r, err := openSQLitePool(t, primaryPath)
	if err != nil {
		t.Fatalf("open routed: %v", err)
	}
	defer r.Close()

	ctx := context.Background()
	if err := r.InstallTestPool(ctx, firstTestPath, "lease-a", 0); err != nil {
		t.Fatalf("install 1: %v", err)
	}
	if err := r.InstallTestPool(ctx, secondTestPath, "lease-a", 0); err != nil {
		t.Fatalf("install 2 (same lease): %v", err)
	}

	if got := readMarker(t, r, database.WithTestMode(ctx)); got != "TEST2" {
		t.Fatalf("after replace, test pool marker = %q, want TEST2", got)
	}
}

// TestRoutedDB_InstallTestPool_RejectsDifferentLease asserts the
// concurrency guard: a second install under a different lease must be
// rejected with *ErrLeaseConflict.
func TestRoutedDB_InstallTestPool_RejectsDifferentLease(t *testing.T) {
	dir := t.TempDir()
	primaryPath := filepath.Join(dir, "primary.db")
	firstTestPath := filepath.Join(dir, "test1.db")
	secondTestPath := filepath.Join(dir, "test2.db")
	seedPool(t, primaryPath, "PRIMARY")
	seedPool(t, firstTestPath, "TEST1")
	seedPool(t, secondTestPath, "TEST2")

	r, err := openSQLitePool(t, primaryPath)
	if err != nil {
		t.Fatalf("open routed: %v", err)
	}
	defer r.Close()

	ctx := context.Background()
	if err := r.InstallTestPool(ctx, firstTestPath, "lease-a", 0); err != nil {
		t.Fatalf("install: %v", err)
	}

	err = r.InstallTestPool(ctx, secondTestPath, "lease-b", 0)
	var conflict *database.ErrLeaseConflict
	if !errorsAs(err, &conflict) {
		t.Fatalf("expected *ErrLeaseConflict, got %v", err)
	}
	if conflict.ActiveLeaseID != "lease-a" {
		t.Fatalf("active lease = %q, want lease-a", conflict.ActiveLeaseID)
	}

	// The original pool is still installed.
	if got := readMarker(t, r, database.WithTestMode(ctx)); got != "TEST1" {
		t.Fatalf("post-conflict test marker = %q, want TEST1", got)
	}
}

// TestRoutedDB_ClearTestPool_RejectsMismatchedLease asserts Clear honors leases.
func TestRoutedDB_ClearTestPool_RejectsMismatchedLease(t *testing.T) {
	dir := t.TempDir()
	primaryPath := filepath.Join(dir, "primary.db")
	testPath := filepath.Join(dir, "test.db")
	seedPool(t, primaryPath, "PRIMARY")
	seedPool(t, testPath, "TEST")

	r, err := openSQLitePool(t, primaryPath)
	if err != nil {
		t.Fatalf("open routed: %v", err)
	}
	defer r.Close()

	if err := r.InstallTestPool(context.Background(), testPath, "lease-a", 0); err != nil {
		t.Fatalf("install: %v", err)
	}

	err = r.ClearTestPool("lease-b")
	var mismatch *database.ErrLeaseMismatch
	if !errorsAs(err, &mismatch) {
		t.Fatalf("expected *ErrLeaseMismatch, got %v", err)
	}
	if mismatch.ActiveLeaseID != "lease-a" {
		t.Fatalf("active lease = %q, want lease-a", mismatch.ActiveLeaseID)
	}

	if err := r.ClearTestPool("lease-a"); err != nil {
		t.Fatalf("clear with matching lease: %v", err)
	}
}

// TestRoutedDB_ConcurrentInstallAndQuery exercises the RWMutex correctness
// under simultaneous installs/clears and queries. This is a race-detector
// stressor more than a behavior test.
func TestRoutedDB_ConcurrentInstallAndQuery(t *testing.T) {
	dir := t.TempDir()
	primaryPath := filepath.Join(dir, "primary.db")
	testPath := filepath.Join(dir, "test.db")
	seedPool(t, primaryPath, "PRIMARY")
	seedPool(t, testPath, "TEST")

	r, err := openSQLitePool(t, primaryPath)
	if err != nil {
		t.Fatalf("open routed: %v", err)
	}
	defer r.Close()

	ctxTest := database.WithTestMode(context.Background())
	stop := make(chan struct{})
	var wg sync.WaitGroup

	// Readers.
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				var v string
				_ = r.QueryRowContext(ctxTest, `SELECT v FROM t LIMIT 1`).Scan(&v)
			}
		}()
	}

	// Install/clear flapper.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 25; i++ {
			if err := r.InstallTestPool(context.Background(), testPath, "lease-flap", 0); err != nil {
				t.Errorf("install: %v", err)
				return
			}
			if err := r.ClearTestPool("lease-flap"); err != nil {
				t.Errorf("clear: %v", err)
				return
			}
		}
	}()

	// Let the flapper finish before stopping readers.
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Crude wait: readers are stopped by close(stop) below.
	}()

	// Wait briefly then stop readers.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// The flapper completes deterministically; we just need to keep readers
	// alive until then. Close stop after a short fixed iteration window by
	// observing that the flapper exits and re-counting Done's: simpler to
	// just stop after a fixed work budget.
	for i := 0; i < 100; i++ {
		var v string
		_ = r.QueryRowContext(ctxTest, `SELECT v FROM t LIMIT 1`).Scan(&v)
	}
	close(stop)
	<-done
}
