package database_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
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

func TestPrimaryFingerprintUsesResolvedSQLiteTargetAndSurvivesWorkingDirectoryChange(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalDir) }()
	dirA, dirB := t.TempDir(), t.TempDir()
	if err := os.Chdir(dirA); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: "primary.db"})
	if err != nil {
		t.Fatalf("open relative primary database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	fingerprint := db.PrimaryFingerprint()
	if fingerprint == "" {
		t.Fatal("resolved primary fingerprint is empty")
	}
	if err := os.Chdir(dirB); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirB, "primary.db"), []byte("different primary target"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := db.PrimaryFingerprint(); got != fingerprint {
		t.Fatalf("primary fingerprint changed after cwd change: got %q want %q", got, fingerprint)
	}
}

func TestPrimaryFingerprintCachesEnvironmentResolvedSQLiteTargetAtOpen(t *testing.T) {
	dirA, dirB := t.TempDir(), t.TempDir()
	dataDir := dirA
	env := func(key string) string {
		switch key {
		case "SCENARIO_NAME", "VROOLI_SCENARIO", "VROOLI_STORAGE_NAMESPACE":
			return "fingerprint-test"
		case "SCENARIO_DATA_DIR":
			return dataDir
		default:
			return ""
		}
	}
	openResolved := func() *database.RoutedDB {
		db, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, Scenario: "fingerprint-test", EnvGetter: env})
		if err != nil {
			t.Fatalf("open environment-resolved primary database: %v", err)
		}
		t.Cleanup(func() { _ = db.Close() })
		return db
	}
	first := openResolved()
	fingerprintA := first.PrimaryFingerprint()
	if fingerprintA == "" {
		t.Fatal("environment-resolved primary fingerprint is empty")
	}
	dataDir = dirB
	second := openResolved()
	fingerprintB := second.PrimaryFingerprint()
	if fingerprintB == "" || fingerprintA == fingerprintB {
		t.Fatalf("resolved primary targets were not distinguished: A=%q B=%q", fingerprintA, fingerprintB)
	}
	if got := first.PrimaryFingerprint(); got != fingerprintA {
		t.Fatalf("first owner's cached fingerprint changed after environment update: got %q want %q", got, fingerprintA)
	}
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

func TestRoutedDB_InstallDifferentLeaseReclaimsExpiredPoolWithoutRoutingRead(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	r := openWithClock(t, clock)
	if _, err := r.Primary().ExecContext(context.Background(), `CREATE TABLE t (v TEXT)`); err != nil {
		t.Fatalf("create primary table: %v", err)
	}
	if _, err := r.Primary().ExecContext(context.Background(), `INSERT INTO t(v) VALUES ('PRIMARY')`); err != nil {
		t.Fatalf("seed primary: %v", err)
	}

	firstPath := filepath.Join(t.TempDir(), "first.db")
	secondPath := filepath.Join(t.TempDir(), "second.db")
	seedPool(t, firstPath, "FIRST")
	seedPool(t, secondPath, "SECOND")
	if err := r.InstallTestPool(context.Background(), firstPath, "lease-a", time.Minute); err != nil {
		t.Fatalf("install first lease: %v", err)
	}
	pinned, err := r.PinTestPool(context.Background(), "lease-a")
	if err != nil {
		t.Fatalf("pin first lease: %v", err)
	}
	if got := readMarker(t, r, database.WithTestMode(context.Background())); got != "FIRST" {
		t.Fatalf("old lease routed to %q, want FIRST", got)
	}

	// No routing read or explicit clear occurs between expiry and replacement.
	clock.Advance(time.Minute + time.Second)
	if err := r.InstallTestPool(context.Background(), secondPath, "lease-b", time.Minute); err != nil {
		t.Fatalf("install replacement after expiry: %v", err)
	}
	if got := r.ActiveLeaseID(); got != "lease-b" {
		t.Fatalf("active lease = %q, want lease-b", got)
	}
	if got := r.LeaseStats(); got.TestPoolRequests != 0 || got.PrimaryDuringTestModeRequests != 0 {
		t.Fatalf("replacement did not reset lease stats: %#v", got)
	}
	if got := readMarker(t, r, context.Background()); got != "PRIMARY" {
		t.Fatalf("plain request routed to %q, want PRIMARY", got)
	}
	if got := readMarker(t, r, database.WithTestMode(context.Background())); got != "SECOND" {
		t.Fatalf("new lease routed to %q, want SECOND", got)
	}

	var staleValue string
	if err := r.QueryRowContext(pinned, `SELECT v FROM t LIMIT 1`).Scan(&staleValue); err == nil {
		t.Fatalf("old pinned request crossed into another pool and read %q", staleValue)
	}
	if _, err := r.PoolForContext(pinned); err == nil {
		t.Fatal("old pinned request rebound to the replacement pool")
	}
}

func TestRoutedDB_InstallDifferentLeaseRechecksRenewedOwnerAfterPoolPreparation(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	r := openWithClock(t, clock)
	firstPath := filepath.Join(t.TempDir(), "first.db")
	secondPath := filepath.Join(t.TempDir(), "second.db")
	seedPool(t, firstPath, "FIRST")
	seedPool(t, secondPath, "SECOND")
	if err := r.InstallTestPool(context.Background(), firstPath, "lease-a", time.Minute); err != nil {
		t.Fatalf("install expiring owner: %v", err)
	}
	clock.Advance(time.Minute + time.Second)

	preparing := make(chan struct{})
	continuePreparation := make(chan struct{})
	r.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		if _, err := pool.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS initialized (v INTEGER)`); err != nil {
			return err
		}
		var marker string
		if err := pool.QueryRowContext(ctx, `SELECT v FROM t LIMIT 1`).Scan(&marker); err != nil {
			return err
		}
		if marker == "SECOND" {
			close(preparing)
			<-continuePreparation
		}
		return nil
	})

	installErr := make(chan error, 1)
	go func() {
		installErr <- r.InstallTestPool(context.Background(), secondPath, "lease-b", time.Minute)
	}()
	select {
	case <-preparing:
	case err := <-installErr:
		t.Fatalf("replacement did not prepare past the expired lease: %v", err)
	}
	if _, err := r.HeartbeatTestPool("lease-a", time.Minute); err != nil {
		close(continuePreparation)
		t.Fatalf("renew owner while replacement prepares: %v", err)
	}
	close(continuePreparation)
	if err := <-installErr; err == nil {
		t.Fatal("replacement overwrote a renewed live lease during preparation")
	} else {
		var conflict *database.ErrLeaseConflict
		if !errors.As(err, &conflict) || conflict.ActiveLeaseID != "lease-a" {
			t.Fatalf("install error = %v, want conflict with renewed lease-a", err)
		}
	}
	if got := r.ActiveLeaseID(); got != "lease-a" {
		t.Fatalf("active lease = %q, want lease-a", got)
	}
}

func TestResumeTestPoolReplacesExpiredBindingAndKeepsOldPinsIsolated(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	r := openWithClock(t, clock)
	if _, err := r.Primary().ExecContext(context.Background(), `CREATE TABLE t (v TEXT)`); err != nil {
		t.Fatalf("create primary table: %v", err)
	}
	if _, err := r.Primary().ExecContext(context.Background(), `INSERT INTO t(v) VALUES ('PRIMARY')`); err != nil {
		t.Fatalf("seed primary: %v", err)
	}

	oldPath := filepath.Join(t.TempDir(), "old.db")
	retainedPath := filepath.Join(t.TempDir(), "retained.db")
	seedPool(t, oldPath, "OLD")
	seedPool(t, retainedPath, "RETAINED")
	if err := r.InstallTestPool(context.Background(), oldPath, "expired", time.Minute); err != nil {
		t.Fatalf("install old lease: %v", err)
	}
	pinned, err := r.PinTestPool(context.Background(), "expired")
	if err != nil {
		t.Fatalf("pin old lease: %v", err)
	}
	clock.Advance(time.Minute + time.Second)

	state := database.TestPoolRestartState{
		LeaseID: "resumed", ExpiresAt: clock.Now().Add(time.Minute), DSN: retainedPath,
	}
	if err := r.ResumeTestPool(context.Background(), state); err != nil {
		t.Fatalf("resume over expired binding: %v", err)
	}
	if got := readMarker(t, r, database.WithTestMode(context.Background())); got != "RETAINED" {
		t.Fatalf("resumed lease routed to %q, want RETAINED", got)
	}
	if got := readMarker(t, r, context.Background()); got != "PRIMARY" {
		t.Fatalf("plain request routed to %q, want PRIMARY", got)
	}
	var staleValue string
	if err := r.QueryRowContext(pinned, `SELECT v FROM t LIMIT 1`).Scan(&staleValue); err == nil {
		t.Fatalf("old pin crossed to resumed pool and read %q", staleValue)
	}
	if _, err := r.PoolForContext(pinned); err == nil {
		t.Fatal("old pin rebound to resumed pool")
	}
}

func TestResumeTestPoolRefusesSameIDLiveBinding(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	r := openWithClock(t, clock)
	activePath := filepath.Join(t.TempDir(), "active.db")
	resumePath := filepath.Join(t.TempDir(), "resume.db")
	seedPool(t, activePath, "ACTIVE")
	seedPool(t, resumePath, "RESUMED")
	if err := r.InstallTestPool(context.Background(), activePath, "lease-a", time.Minute); err != nil {
		t.Fatalf("install active lease: %v", err)
	}
	state := database.TestPoolRestartState{
		LeaseID: "lease-a", ExpiresAt: clock.Now().Add(time.Minute), DSN: resumePath,
	}
	if err := r.ResumeTestPool(context.Background(), state); err == nil {
		t.Fatal("resume replaced a live binding with the same lease ID")
	}
	if got := readMarker(t, r, database.WithTestMode(context.Background())); got != "ACTIVE" {
		t.Fatalf("live binding changed to %q, want ACTIVE", got)
	}
}

func TestResumeTestPoolRejectsBindingThatExpiresDuringInitialization(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	r := openWithClock(t, clock)
	retainedPath := filepath.Join(t.TempDir(), "retained.db")
	seedPool(t, retainedPath, "RETAINED")
	r.SetTestPoolInitializer(func(context.Context, *sql.DB) error {
		clock.Advance(time.Minute)
		return nil
	})
	state := database.TestPoolRestartState{
		LeaseID: "resumed", ExpiresAt: clock.Now().Add(time.Second), DSN: retainedPath,
	}
	if err := r.ResumeTestPool(context.Background(), state); err == nil {
		t.Fatal("resume installed a binding that expired during pool initialization")
	}
	if got := r.ActiveLeaseID(); got != "" {
		t.Fatalf("expired restart binding became active under lease %q", got)
	}
	if _, err := r.PoolForContext(database.WithTestMode(context.Background())); err == nil {
		t.Fatal("expired restart binding was accepted for test mode")
	}
}

func TestPinnedTestPoolReplacementFailsClosedAcrossMethods(t *testing.T) {
	dir := t.TempDir()
	primaryPath := filepath.Join(dir, "primary.db")
	firstPath := filepath.Join(dir, "first.db")
	secondPath := filepath.Join(dir, "second.db")
	seedPool(t, primaryPath, "PRIMARY")
	seedPool(t, firstPath, "FIRST")
	seedPool(t, secondPath, "SECOND")
	r, err := openSQLitePool(t, primaryPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := r.InstallTestPool(context.Background(), firstPath, "lease-a", time.Minute); err != nil {
		t.Fatal(err)
	}
	pinned, err := r.PinTestPool(context.Background(), "lease-a")
	if err != nil {
		t.Fatal(err)
	}
	// Same-ID installation is allowed for ordinary routing, but must retire
	// the exact pool captured by the pin rather than retarget its context.
	if err := r.InstallTestPool(context.Background(), secondPath, "lease-a", time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PinTestPool(pinned, "lease-a"); err == nil {
		t.Fatal("stale context rebound to a same-ID replacement")
	}
	if _, err := r.PinTestPool(pinned, "lease-b"); err == nil {
		t.Fatal("lease-A context rebound to lease B")
	}
	if _, err := r.PoolForContext(pinned); err == nil {
		t.Fatal("PoolForContext selected a same-ID replacement for a stale pin")
	}

	if rows, err := r.QueryContext(pinned, `SELECT v FROM t`); err == nil {
		_ = rows.Close()
		t.Fatal("QueryContext crossed to a same-ID replacement")
	}
	if _, err := r.ExecContext(pinned, `INSERT INTO t(v) VALUES ('CROSSED')`); err == nil {
		t.Fatal("ExecContext crossed to a same-ID replacement")
	}
	if tx, err := r.BeginTx(pinned, nil); err == nil {
		_ = tx.Rollback()
		t.Fatal("BeginTx crossed to a same-ID replacement")
	}
	var value string
	if err := r.QueryRowContext(pinned, `SELECT v FROM t LIMIT 1`).Scan(&value); err == nil {
		t.Fatalf("QueryRowContext crossed to replacement and scanned %q", value)
	}
	if got := readMarker(t, r, database.WithTestMode(context.Background())); got != "SECOND" {
		t.Fatalf("unbound routing changed: got %q, want SECOND", got)
	}
	if err := r.ClearTestPool("lease-a"); err != nil {
		t.Fatal(err)
	}
	if err := r.InstallTestPool(context.Background(), firstPath, "lease-b", time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PinTestPool(pinned, "lease-b"); err == nil {
		t.Fatal("lease-A context rebound to a different active lease")
	}
	if err := r.QueryRowContext(pinned, `SELECT v FROM t LIMIT 1`).Scan(&value); err == nil {
		t.Fatalf("old lease-A context crossed to lease B and scanned %q", value)
	}
	if got := readMarker(t, r, database.WithTestMode(context.Background())); got != "FIRST" {
		t.Fatalf("unbound routing did not follow lease B: got %q, want FIRST", got)
	}
}

func TestPinnedTestPoolFromAnotherOwnerFailsWithoutClosingEitherPool(t *testing.T) {
	makeRouted := func(t *testing.T, name, marker string) *database.RoutedDB {
		t.Helper()
		path := filepath.Join(t.TempDir(), name+"-primary.db")
		seedPool(t, path, marker)
		r, err := openSQLitePool(t, path)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = r.Close() })
		leased := filepath.Join(t.TempDir(), name+"-leased.db")
		seedPool(t, leased, marker+"-LEASED")
		if err := r.InstallTestPool(context.Background(), leased, "lease-shared", time.Minute); err != nil {
			t.Fatal(err)
		}
		return r
	}
	first := makeRouted(t, "first", "FIRST")
	second := makeRouted(t, "second", "SECOND")
	pinned, err := first.PinTestPool(context.Background(), "lease-shared")
	if err != nil {
		t.Fatal(err)
	}
	var value string
	if err := second.QueryRowContext(pinned, `SELECT v FROM t LIMIT 1`).Scan(&value); err == nil {
		t.Fatalf("foreign owner pin unexpectedly scanned %q", value)
	}
	if _, err := second.PoolForContext(pinned); err == nil {
		t.Fatal("foreign owner PoolForContext accepted a different owner's pin")
	}
	if got := readMarker(t, first, pinned); got != "FIRST-LEASED" {
		t.Fatalf("foreign pin attempt closed or changed its owner: got %q", got)
	}
	if got := readMarker(t, second, database.WithTestMode(context.Background())); got != "SECOND-LEASED" {
		t.Fatalf("foreign pin attempt changed the target owner: got %q", got)
	}
}

func TestPinnedTestPoolExpiryFailsClosedWithoutPrimaryFallback(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	r := openWithClock(t, clock)
	primary := r.Primary()
	if _, err := primary.ExecContext(context.Background(), `CREATE TABLE t (v TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := primary.ExecContext(context.Background(), `INSERT INTO t(v) VALUES ('PRIMARY')`); err != nil {
		t.Fatal(err)
	}
	testPath := filepath.Join(t.TempDir(), "leased.db")
	seedPool(t, testPath, "LEASED")
	if err := r.InstallTestPool(context.Background(), testPath, "lease-expiring", time.Minute); err != nil {
		t.Fatal(err)
	}
	pinned, err := r.PinTestPool(context.Background(), "lease-expiring")
	if err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Minute + time.Second)
	var value string
	if err := r.QueryRowContext(pinned, `SELECT v FROM t LIMIT 1`).Scan(&value); err == nil {
		t.Fatalf("expired pin unexpectedly scanned %q", value)
	}
	if _, err := r.PoolForContext(pinned); err == nil {
		t.Fatal("PoolForContext accepted an expired pin")
	}
	if got := readMarker(t, r, database.WithTestMode(context.Background())); got != "PRIMARY" {
		t.Fatalf("unbound expiry fallback changed: got %q, want PRIMARY", got)
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

func TestPoolForContextRefusesMissingAndExpiredTestLease(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	r := openWithClock(t, clock)
	testCtx := database.WithTestMode(context.Background())
	if pool, err := r.PoolForContext(testCtx); err == nil || pool != nil {
		t.Fatal("missing test lease fell back to primary")
	}
	live, err := r.PoolForContext(context.Background())
	if err != nil || live != r.Primary() {
		t.Fatal("normal request did not bind primary", err)
	}
	testPath := filepath.Join(t.TempDir(), "strict-test.db")
	seedPool(t, testPath, "TEST")
	if err := r.InstallTestPool(context.Background(), testPath, "strict", time.Minute); err != nil {
		t.Fatal(err)
	}
	pinned, err := r.PoolForContext(testCtx)
	if err != nil || pinned == live {
		t.Fatal("test request did not bind isolated pool", err)
	}
	clock.Advance(2 * time.Minute)
	if pool, err := r.PoolForContext(testCtx); err == nil || pool != nil {
		t.Fatal("expired test lease fell back to primary")
	}
	// The already-bound repository remains tied to its original pool object.
	if pinned == r.Primary() {
		t.Fatal("bound test pool changed to primary")
	}
}
