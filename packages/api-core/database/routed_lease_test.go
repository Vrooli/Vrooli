package database_test

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
)

// fakeClock is a minimal Clock fake for lease-TTL tests.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
func (c *fakeClock) NewTimer(d time.Duration) schedule.Timer   { return schedule.System().NewTimer(d) }
func (c *fakeClock) NewTicker(d time.Duration) schedule.Ticker { return schedule.System().NewTicker(d) }
func (c *fakeClock) Sleep(d time.Duration)                     { c.Advance(d) }

func openWithClock(t *testing.T, clock database.Clock) *database.RoutedDB {
	t.Helper()
	r, err := database.OpenWithClock(context.Background(), database.Config{
		Driver: database.DriverSQLite,
		DSN:    filepath.Join(t.TempDir(), "primary.db"),
	}, clock)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r
}

func TestRoutedDB_Lease_DefaultTTL(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	r := openWithClock(t, clock)

	testPath := filepath.Join(t.TempDir(), "test.db")
	seedPool(t, testPath, "TEST")

	if err := r.InstallTestPool(context.Background(), testPath, "lease-a", 0); err != nil {
		t.Fatalf("install: %v", err)
	}
	if !r.HasTestPool() {
		t.Fatalf("expected pool installed")
	}

	// Just before TTL — still routed.
	clock.Advance(database.DefaultLeaseTTL - time.Second)
	if !r.HasTestPool() {
		t.Fatalf("expected pool live before TTL")
	}

	// Past TTL — lease expires.
	clock.Advance(2 * time.Second)
	if r.HasTestPool() {
		t.Fatalf("expected pool expired after TTL")
	}
}

func TestRoutedDB_Lease_PickFallsBackToPrimaryAfterExpiry(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	r := openWithClock(t, clock)

	primaryDB := r.Primary()
	if _, err := primaryDB.ExecContext(context.Background(), `CREATE TABLE t (v TEXT)`); err != nil {
		t.Fatalf("create primary table: %v", err)
	}
	if _, err := primaryDB.ExecContext(context.Background(), `INSERT INTO t (v) VALUES ('PRIMARY')`); err != nil {
		t.Fatalf("seed primary: %v", err)
	}

	testPath := filepath.Join(t.TempDir(), "test.db")
	seedPool(t, testPath, "TEST")

	if err := r.InstallTestPool(context.Background(), testPath, "lease-a", 1*time.Minute); err != nil {
		t.Fatalf("install: %v", err)
	}

	ctxTest := database.WithTestMode(context.Background())
	if got := readMarker(t, r, ctxTest); got != "TEST" {
		t.Fatalf("pre-expiry test marker = %q, want TEST", got)
	}

	clock.Advance(2 * time.Minute)

	// pick() should now fall back to primary even with test-mode context.
	if got := readMarker(t, r, ctxTest); got != "PRIMARY" {
		t.Fatalf("post-expiry test marker = %q, want PRIMARY (lease expired)", got)
	}
	// Pool reference is cleared too.
	if r.HasTestPool() {
		t.Fatalf("expected expired lease to clear the pool")
	}
}

func TestRoutedDB_Heartbeat_ExtendsExpiry(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	r := openWithClock(t, clock)

	testPath := filepath.Join(t.TempDir(), "test.db")
	seedPool(t, testPath, "TEST")

	if err := r.InstallTestPool(context.Background(), testPath, "lease-a", 1*time.Minute); err != nil {
		t.Fatalf("install: %v", err)
	}

	clock.Advance(50 * time.Second)
	if _, err := r.HeartbeatTestPool("lease-a", 1*time.Minute); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	clock.Advance(50 * time.Second)
	// Without heartbeat we'd be 100s past install, expired. With heartbeat
	// the new expiry is at 1m50s past install, so we're 10s before expiry.
	if !r.HasTestPool() {
		t.Fatalf("expected lease alive after heartbeat")
	}
}

func TestRoutedDB_Heartbeat_RejectsWrongLease(t *testing.T) {
	clock := &fakeClock{now: time.Now()}
	r := openWithClock(t, clock)

	testPath := filepath.Join(t.TempDir(), "test.db")
	seedPool(t, testPath, "TEST")

	if err := r.InstallTestPool(context.Background(), testPath, "lease-a", 0); err != nil {
		t.Fatalf("install: %v", err)
	}

	_, err := r.HeartbeatTestPool("lease-b", 0)
	var mismatch *database.ErrLeaseMismatch
	if !errorsAs(err, &mismatch) {
		t.Fatalf("expected ErrLeaseMismatch, got %v", err)
	}
	if mismatch.ActiveLeaseID != "lease-a" {
		t.Fatalf("active lease = %q, want lease-a", mismatch.ActiveLeaseID)
	}
}

func TestRoutedDB_LeaseStats_CountsTestPoolRequests(t *testing.T) {
	clock := &fakeClock{now: time.Now()}
	r := openWithClock(t, clock)

	testPath := filepath.Join(t.TempDir(), "test.db")
	seedPool(t, testPath, "TEST")

	if err := r.InstallTestPool(context.Background(), testPath, "lease-a", 0); err != nil {
		t.Fatalf("install: %v", err)
	}

	ctxTest := database.WithTestMode(context.Background())
	for i := 0; i < 3; i++ {
		_ = readMarker(t, r, ctxTest)
	}

	got := r.LeaseStats()
	if got.TestPoolRequests != 3 {
		t.Fatalf("test_pool_requests = %d, want 3", got.TestPoolRequests)
	}
	if got.PrimaryDuringTestModeRequests != 0 {
		t.Fatalf("primary_during_test_mode_requests = %d, want 0", got.PrimaryDuringTestModeRequests)
	}
}

func TestRoutedDB_LeaseStats_CountsBypassAfterExpiry(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	r := openWithClock(t, clock)

	// Seed primary so the post-expiry test-mode read can return a row.
	if _, err := r.Primary().ExecContext(context.Background(), `CREATE TABLE t (v TEXT)`); err != nil {
		t.Fatalf("create primary table: %v", err)
	}
	if _, err := r.Primary().ExecContext(context.Background(), `INSERT INTO t (v) VALUES ('PRIMARY')`); err != nil {
		t.Fatalf("seed primary: %v", err)
	}

	testPath := filepath.Join(t.TempDir(), "test.db")
	seedPool(t, testPath, "TEST")

	if err := r.InstallTestPool(context.Background(), testPath, "lease-a", 1*time.Minute); err != nil {
		t.Fatalf("install: %v", err)
	}

	ctxTest := database.WithTestMode(context.Background())
	_ = readMarker(t, r, ctxTest) // test_pool_requests++

	clock.Advance(2 * time.Minute) // lease expires
	_ = readMarker(t, r, ctxTest)  // primary_during_test_mode_requests++

	got := r.LeaseStats()
	if got.TestPoolRequests != 1 {
		t.Fatalf("test_pool_requests = %d, want 1", got.TestPoolRequests)
	}
	if got.PrimaryDuringTestModeRequests != 1 {
		t.Fatalf("primary_during_test_mode_requests = %d, want 1", got.PrimaryDuringTestModeRequests)
	}
}
