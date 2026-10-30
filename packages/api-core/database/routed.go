package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vrooli/api-core/schedule"
)

// validateDSNMatchesDriver rejects DSNs whose scheme is clearly meant for
// a different driver than the pool's configured one. Catches the common
// mistake of handing a postgres URL to a sqlite-driven RoutedDB (which
// otherwise hangs inside PingContext) and vice versa.
func validateDSNMatchesDriver(driver, dsn string) error {
	lower := strings.ToLower(strings.TrimSpace(dsn))
	looksLikePostgres := strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://")
	switch driver {
	case DriverPostgres:
		if !looksLikePostgres {
			return fmt.Errorf("dsn does not look like a postgres URL (driver=%q): %s", driver, dsn)
		}
	case DriverSQLite, DriverSQLiteLegacy:
		if looksLikePostgres {
			return fmt.Errorf("dsn looks like a postgres URL but pool driver is %q: %s", driver, dsn)
		}
	}
	return nil
}

// DefaultLeaseTTL is the per-lease expiry used when InstallTestPool is
// called without an explicit TTL. test-genie's playbooks heartbeat keeps
// the lease alive on a 30s cadence; 90s is ≈ 3× that, matching the
// claim-TTL ratio used by internal/playbooksclaims.
const DefaultLeaseTTL = 90 * time.Second

// Clock is the time seam used by RoutedDB for lease expiry. Production
// wires nowFunc = time.Now; tests pass a fake.
//
// seam: ambient time. The interface is single-method to keep substitution
// trivial.
type Clock = schedule.Clock

// systemClock is the default Clock backed by time.Now.
// SystemClock returns the default time.Now-backed Clock.
func SystemClock() Clock { return schedule.System() }

// RoutedDB wraps a primary *sql.DB pool with an optional, runtime-installable
// test pool. Per-request routing is driven by IsTestMode(ctx): when the
// context is marked test-mode AND a test pool has been installed, the call is
// served from the test pool. Every other case is served from the primary
// pool.
//
// RoutedDB is the persistence seam scenarios depend on instead of *sql.DB. It
// exposes the subset of the *sql.DB method surface that handlers and
// repositories use in practice. *sql.DB is a struct rather than an interface,
// so *RoutedDB is not a type-level drop-in; callers update their field types
// once and the body of their handlers stays unchanged.
//
// seam: RoutedDB is the persistence substrate seam. Production wires it via
// database.Open. Test-genie installs and clears a runtime test pool through
// the dev-only RoutingService (see packages/api-core/devrouting). Tests
// substitute the entire seam by passing a different *RoutedDB constructed
// against in-memory drivers.
type RoutedDB struct {
	mu                 sync.RWMutex
	primary            *sql.DB
	test               *sql.DB
	lease              leaseState
	cfg                Config
	primaryFingerprint string
	clock              Clock
	// testPoolInitializer prepares each newly installed pool before it becomes
	// visible to test-mode requests (for example, by applying scenario schemas).
	// It is deliberately process-owned, not part of the devrouting request.
	testPoolInitializer func(context.Context, *sql.DB) error
	// stats are per-lease atomic counters reset on each install.
	stats leaseStats
	// expiryWarned is reset on install/clear; ensures the slog.Warn that
	// fires when a pick falls through to primary because the lease expired
	// only logs once per lease.
	expiryWarned atomic.Bool
}

// leaseState holds the current install's ownership data.
type leaseState struct {
	id        string
	expiresAt time.Time
	dsn       string
}

// TestPoolRestartState is an owner-only snapshot used by devrouting to persist
// an explicit graceful restart checkpoint. DSN is intentionally available only
// through this in-process API; devrouting must keep it in its private record.
type TestPoolRestartState struct {
	LeaseID            string
	ExpiresAt          time.Time
	DSN                string
	PrimaryFingerprint string
	Stats              LeaseStatsSnapshot
}

type testPoolPinKey struct{}

type testPoolPin struct {
	owner   *RoutedDB
	leaseID string
	pool    *sql.DB
}

var errForeignTestPoolPin = errors.New("database.RoutedDB: test pool pin belongs to another owner")

type foreignPinConnector struct{}

func (foreignPinConnector) Connect(context.Context) (driver.Conn, error) {
	return nil, errForeignTestPoolPin
}
func (foreignPinConnector) Driver() driver.Driver { return foreignPinDriver{} }

type foreignPinDriver struct{}

func (foreignPinDriver) Open(string) (driver.Conn, error) { return nil, errForeignTestPoolPin }

var foreignPinPool = sql.OpenDB(foreignPinConnector{})

// leaseStats is the atomic snapshot of LeaseStats counters.
type leaseStats struct {
	testPoolRequests             atomic.Int64
	primaryDuringTestModeRequest atomic.Int64
}

// LeaseStatsSnapshot is the read-only projection of leaseStats returned to
// callers (e.g. devrouting.Service.ClearTestPool's response builder).
type LeaseStatsSnapshot struct {
	TestPoolRequests              int64
	PrimaryDuringTestModeRequests int64
}

// ErrLeaseConflict is returned by InstallTestPool when a test pool is
// already installed under a different lease_id. ActiveLeaseID exposes the
// current owner so callers can surface a useful conflict message.
type ErrLeaseConflict struct {
	ActiveLeaseID string
}

func (e *ErrLeaseConflict) Error() string {
	if e.ActiveLeaseID == "" {
		return "database.RoutedDB: test pool already installed"
	}
	return fmt.Sprintf("database.RoutedDB: test pool already installed under lease %q", e.ActiveLeaseID)
}

// ErrLeaseMismatch is returned by ClearTestPool / HeartbeatTestPool when the
// caller's lease_id does not match the active install.
type ErrLeaseMismatch struct {
	ActiveLeaseID string
}

func (e *ErrLeaseMismatch) Error() string {
	if e.ActiveLeaseID == "" {
		return "database.RoutedDB: lease mismatch (no active install)"
	}
	return fmt.Sprintf("database.RoutedDB: lease mismatch (active lease %q)", e.ActiveLeaseID)
}

// Open opens a database connection following the same DSN-resolution and
// retry rules as Connect, and wraps the resulting *sql.DB in a RoutedDB.
//
// The returned RoutedDB has no test pool until InstallTestPool is called.
// Production code paths see exactly the behavior of the underlying *sql.DB.
func Open(ctx context.Context, cfg Config) (*RoutedDB, error) {
	cfg, err := resolvePrimaryDSN(cfg)
	if err != nil {
		return nil, err
	}
	primary, err := Connect(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &RoutedDB{primary: primary, cfg: cfg, primaryFingerprint: primaryFingerprintForConfig(cfg), clock: schedule.System()}, nil
}

// OpenWithClock is Open with an explicit clock seam. Tests pass a fake
// clock; production callers use Open.
func OpenWithClock(ctx context.Context, cfg Config, clock Clock) (*RoutedDB, error) {
	cfg, err := resolvePrimaryDSN(cfg)
	if err != nil {
		return nil, err
	}
	primary, err := Connect(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if clock == nil {
		clock = schedule.System()
	}
	return &RoutedDB{primary: primary, cfg: cfg, primaryFingerprint: primaryFingerprintForConfig(cfg), clock: clock}, nil
}

func resolvePrimaryDSN(cfg Config) (Config, error) {
	cfg = withDefaults(cfg)
	if cfg.DSN == "" {
		dsn, err := buildDSNFromEnv(cfg)
		if err != nil {
			return Config{}, fmt.Errorf("build connection string: %w", err)
		}
		cfg.DSN = dsn
	}
	return cfg, nil
}

// NewFromPrimary wraps an already-open *sql.DB as a RoutedDB whose primary pool
// is that handle. Intended for tests — including sqlmock-backed handles — that
// need a *RoutedDB but manage the underlying connection themselves; production
// code should use Open so connection retry/backoff is applied. The returned
// RoutedDB has no test pool until InstallTestPool is called, so it behaves
// exactly like the wrapped handle until then.
func NewFromPrimary(primary *sql.DB) *RoutedDB {
	return &RoutedDB{primary: primary, cfg: Config{}, clock: schedule.System()}
}

// MustOpen is like Open but panics on error. Useful for main().
func MustOpen(ctx context.Context, cfg Config) *RoutedDB {
	r, err := Open(ctx, cfg)
	if err != nil {
		panic(fmt.Sprintf("database.MustOpen: %v", err))
	}
	return r
}

// SetClock replaces the clock seam. Intended for tests; production callers
// should pass the clock via OpenWithClock.
func (r *RoutedDB) SetClock(clock Clock) {
	if clock == nil {
		clock = schedule.System()
	}
	r.mu.Lock()
	r.clock = clock
	r.mu.Unlock()
}

// SetTestPoolInitializer configures setup run once for every newly installed
// test pool. A failing initializer rejects the install and leaves any prior
// lease untouched, preventing requests from reaching an uninitialized pool.
func (r *RoutedDB) SetTestPoolInitializer(initializer func(context.Context, *sql.DB) error) {
	r.mu.Lock()
	r.testPoolInitializer = initializer
	r.mu.Unlock()
}

// testPoolIsLiveLocked reports whether the installed pool's lease is live.
// The caller must hold r.mu.
func (r *RoutedDB) testPoolIsLiveLocked() bool {
	return r.test != nil && (r.lease.expiresAt.IsZero() || !r.clock.Now().After(r.lease.expiresAt))
}

// hasDifferentLiveLeaseLocked reports whether another unexpired test pool
// currently owns the router. The caller must hold r.mu.
func (r *RoutedDB) hasDifferentLiveLeaseLocked(leaseID string) bool {
	return r.testPoolIsLiveLocked() && r.lease.id != leaseID
}

// pick returns the pool that should serve a request carrying ctx.
// It takes a read lock so installs/clears can occur concurrently with
// in-flight requests without tearing. pick also enforces lease expiry: an
// expired lease behaves as if no pool were installed.
func (r *RoutedDB) pick(ctx context.Context) *sql.DB {
	if pin, ok := ctx.Value(testPoolPinKey{}).(testPoolPin); ok {
		if pin.owner != r {
			return foreignPinPool
		}
		r.mu.RLock()
		matches := r.test == pin.pool && r.lease.id == pin.leaseID
		expired := matches && !r.lease.expiresAt.IsZero() && r.clock.Now().After(r.lease.expiresAt)
		r.mu.RUnlock()
		if expired {
			r.expireLeaseIfStale(pin.leaseID, pin.pool)
			return pin.pool
		}
		if !matches {
			_ = pin.pool.Close()
			return pin.pool
		}
		r.stats.testPoolRequests.Add(1)
		return pin.pool
	}
	testMode := IsTestMode(ctx)

	r.mu.RLock()
	test := r.test
	expiresAt := r.lease.expiresAt
	leaseID := r.lease.id
	clock := r.clock
	r.mu.RUnlock()

	if test == nil {
		if testMode {
			r.stats.primaryDuringTestModeRequest.Add(1)
		}
		return r.primaryPool()
	}

	// Expired lease — revert to primary and (best-effort) clear once.
	if !expiresAt.IsZero() && clock.Now().After(expiresAt) {
		r.expireLeaseIfStale(leaseID, test)
		if testMode {
			r.stats.primaryDuringTestModeRequest.Add(1)
		}
		return r.primaryPool()
	}

	if testMode {
		r.stats.testPoolRequests.Add(1)
		return test
	}
	return r.primaryPool()
}

// primaryPool returns r.primary under a read lock; isolates the
// concurrency contract for pick().
func (r *RoutedDB) primaryPool() *sql.DB {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.primary
}

// expireLeaseIfStale closes and clears the test pool if it is still owned
// by leaseID. Idempotent under racing pick() callers.
func (r *RoutedDB) expireLeaseIfStale(leaseID string, expected *sql.DB) {
	r.mu.Lock()
	if r.test == nil || r.test != expected || r.lease.id != leaseID || r.lease.expiresAt.IsZero() || !r.clock.Now().After(r.lease.expiresAt) {
		r.mu.Unlock()
		return
	}
	if !r.expiryWarned.Swap(true) {
		slog.Warn("database.RoutedDB: test pool lease expired; reverting to primary pool", "lease_id", leaseID)
	}
	old := r.test
	r.test = nil
	r.lease = leaseState{}
	r.mu.Unlock()
	if old != nil {
		_ = old.Close()
	}
}

// Primary returns the underlying primary pool. Use sparingly — most call sites
// should depend on the *RoutedDB surface so they participate in routing.
func (r *RoutedDB) Primary() *sql.DB {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.primary
}

// PoolForContext binds a repository to the pool selected for this request.
// Test-mode requests require a live test lease and never fall back to primary.
// The returned pool can close when its lease ends; callers must propagate that
// error instead of resolving a replacement pool during the same operation.
func (r *RoutedDB) PoolForContext(ctx context.Context) (*sql.DB, error) {
	if ctx != nil {
		if pin, ok := ctx.Value(testPoolPinKey{}).(testPoolPin); ok {
			if pin.owner != r {
				return nil, errForeignTestPoolPin
			}
			r.mu.RLock()
			valid := r.test == pin.pool && r.lease.id == pin.leaseID
			expired := valid && !r.lease.expiresAt.IsZero() && r.clock.Now().After(r.lease.expiresAt)
			r.mu.RUnlock()
			if expired {
				r.expireLeaseIfStale(pin.leaseID, pin.pool)
				return nil, fmt.Errorf("test database lease is expired")
			}
			if !valid {
				return nil, &ErrLeaseMismatch{ActiveLeaseID: r.ActiveLeaseID()}
			}
			return pin.pool, nil
		}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !IsTestMode(ctx) {
		return r.primary, nil
	}
	if r.test == nil || (!r.lease.expiresAt.IsZero() && r.clock.Now().After(r.lease.expiresAt)) {
		return nil, fmt.Errorf("test database lease is missing or expired")
	}
	return r.test, nil
}

// PinTestPool binds subsequent RoutedDB calls made with the returned context
// to the exact pool installed under expectedLeaseID. If that pool is cleared,
// replaced (including a same-ID reinstall), or expires, calls fail against the
// closed pinned pool instead of selecting a new pool or primary storage.
func (r *RoutedDB) PinTestPool(ctx context.Context, expectedLeaseID string) (context.Context, error) {
	if r == nil {
		return nil, &ErrLeaseMismatch{}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if expectedLeaseID == "" {
		return nil, &ErrLeaseMismatch{}
	}
	if pin, ok := ctx.Value(testPoolPinKey{}).(testPoolPin); ok {
		if pin.owner != r || pin.leaseID != expectedLeaseID {
			return nil, &ErrLeaseMismatch{ActiveLeaseID: r.ActiveLeaseID()}
		}
		r.mu.RLock()
		valid := r.test == pin.pool && r.lease.id == pin.leaseID && (r.lease.expiresAt.IsZero() || !r.clock.Now().After(r.lease.expiresAt))
		r.mu.RUnlock()
		if !valid {
			return nil, &ErrLeaseMismatch{ActiveLeaseID: r.ActiveLeaseID()}
		}
		return ctx, nil
	}
	r.mu.RLock()
	pool := r.test
	valid := pool != nil && r.lease.id == expectedLeaseID && (r.lease.expiresAt.IsZero() || !r.clock.Now().After(r.lease.expiresAt))
	activeLeaseID := r.lease.id
	r.mu.RUnlock()
	if !valid {
		return nil, &ErrLeaseMismatch{ActiveLeaseID: activeLeaseID}
	}
	ctx = WithTestMode(ctx)
	return context.WithValue(ctx, testPoolPinKey{}, testPoolPin{owner: r, leaseID: expectedLeaseID, pool: pool}), nil
}

// HasTestPool reports whether a test pool is currently installed (and not
// expired).
func (r *RoutedDB) HasTestPool() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.test == nil {
		return false
	}
	if !r.lease.expiresAt.IsZero() && r.clock.Now().After(r.lease.expiresAt) {
		return false
	}
	return true
}

// InstallTestPool opens a new pool against dsn and installs it as the
// active test pool under leaseID with the supplied TTL. A TTL of zero uses
// DefaultLeaseTTL. Lease semantics:
//
//   - No pool installed: install under leaseID (empty string is allowed —
//     it represents an un-claimed ad-hoc install used only by direct
//     operator tooling).
//   - Pool installed under same leaseID: idempotent — close the old pool,
//     install the new one under the same lease, refresh the TTL. Retry-safe.
//   - Pool installed under a different, unexpired leaseID: reject with
//     *ErrLeaseConflict carrying the active lease. An expired lease can be
//     replaced without a routing read first.
//
// On a successful install the per-lease counters are reset.
func (r *RoutedDB) InstallTestPool(ctx context.Context, dsn, leaseID string, ttl time.Duration) error {
	if dsn == "" {
		return errors.New("database.RoutedDB.InstallTestPool: dsn is empty")
	}
	testDriver := r.cfg.TestDriver
	if testDriver == "" {
		testDriver = r.cfg.Driver
	}
	if err := validateDSNMatchesDriver(testDriver, dsn); err != nil {
		return fmt.Errorf("install test pool: %w", err)
	}
	if ttl <= 0 {
		ttl = DefaultLeaseTTL
	}

	r.mu.Lock()
	if r.hasDifferentLiveLeaseLocked(leaseID) {
		active := r.lease.id
		r.mu.Unlock()
		return &ErrLeaseConflict{ActiveLeaseID: active}
	}
	r.mu.Unlock()

	testCfg := r.cfg
	testCfg.DSN = dsn
	testCfg.Driver = testDriver
	pool, err := Connect(ctx, testCfg)
	if err != nil {
		return fmt.Errorf("install test pool: %w", err)
	}
	r.mu.RLock()
	initializer := r.testPoolInitializer
	r.mu.RUnlock()
	if initializer != nil {
		if err := initializer(ctx, pool); err != nil {
			_ = pool.Close()
			return fmt.Errorf("initialize test pool: %w", err)
		}
	}

	r.mu.Lock()
	// Re-check under exclusive lock — another caller may have installed
	// between the read above and here.
	if r.hasDifferentLiveLeaseLocked(leaseID) {
		active := r.lease.id
		r.mu.Unlock()
		_ = pool.Close()
		return &ErrLeaseConflict{ActiveLeaseID: active}
	}
	old := r.test
	r.test = pool
	r.lease = leaseState{id: leaseID, expiresAt: r.clock.Now().Add(ttl), dsn: dsn}
	r.stats.testPoolRequests.Store(0)
	r.stats.primaryDuringTestModeRequest.Store(0)
	r.expiryWarned.Store(false)
	r.mu.Unlock()

	if old != nil {
		_ = old.Close()
	}
	return nil
}

// TestPoolRestartState returns the private binding for an active, unexpired
// test pool. The returned DSN must not be logged or exposed outside the owner.
func (r *RoutedDB) TestPoolRestartState(leaseID string) (TestPoolRestartState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.test == nil || r.lease.id != leaseID {
		return TestPoolRestartState{}, &ErrLeaseMismatch{ActiveLeaseID: r.lease.id}
	}
	dsn := r.lease.dsn
	driver := r.cfg.TestDriver
	if driver == "" {
		driver = r.cfg.Driver
	}
	if driver == DriverSQLite || driver == DriverSQLiteLegacy {
		var err error
		dsn, err = nonCreatingSQLiteDSN(driver, dsn)
		if err != nil {
			return TestPoolRestartState{}, err
		}
	}
	fingerprint := r.primaryFingerprint
	if fingerprint == "" {
		fingerprint = primaryFingerprintForConfig(r.cfg)
	}
	return TestPoolRestartState{LeaseID: leaseID, ExpiresAt: r.lease.expiresAt, DSN: dsn, PrimaryFingerprint: fingerprint, Stats: r.LeaseStats()}, nil
}

// PrimaryFingerprint identifies the configured primary database target without
// exposing credentials or DSNs outside the owner.
func (r *RoutedDB) PrimaryFingerprint() string {
	if r == nil {
		return ""
	}
	r.mu.RLock()
	cfg := r.cfg
	fingerprint := r.primaryFingerprint
	r.mu.RUnlock()
	if fingerprint != "" {
		return fingerprint
	}
	return primaryFingerprintForConfig(cfg)
}

func primaryFingerprintForConfig(cfg Config) string {
	primaryDSN := cfg.DSN
	if primaryDSN == "" {
		var err error
		primaryDSN, err = buildDSNFromEnv(withDefaults(cfg))
		if err != nil {
			return ""
		}
	}
	if cfg.Driver == DriverSQLite || cfg.Driver == DriverSQLiteLegacy {
		if canonical, err := nonCreatingSQLiteDSN(cfg.Driver, primaryDSN); err == nil {
			primaryDSN = canonical
		} else {
			return ""
		}
	}
	body, _ := json.Marshal([]string{cfg.Driver, cfg.TestDriver, cfg.Scenario, primaryDSN})
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

// ResumeTestPool reopens the exact owner-retained database binding after a
// graceful process restart. Callers must obtain state from a private owner
// checkpoint; this method does not accept state from a public request body.
func (r *RoutedDB) ResumeTestPool(ctx context.Context, state TestPoolRestartState) error {
	if state.LeaseID == "" || state.DSN == "" || !state.ExpiresAt.After(r.clock.Now()) {
		return errors.New("database.RoutedDB: restart binding is invalid or expired")
	}
	testDriver := r.cfg.TestDriver
	if testDriver == "" {
		testDriver = r.cfg.Driver
	}
	if err := validateDSNMatchesDriver(testDriver, state.DSN); err != nil {
		return errors.New("database.RoutedDB: retained restart binding does not match configured driver")
	}
	testCfg := r.cfg
	testCfg.Driver = testDriver
	dsn, err := nonCreatingSQLiteDSN(testDriver, state.DSN)
	if err != nil {
		return errors.New("database.RoutedDB: retained database file is unavailable")
	}
	testCfg.DSN = dsn
	pool, err := Connect(ctx, testCfg)
	if err != nil {
		return fmt.Errorf("resume retained test pool: %w", err)
	}
	r.mu.RLock()
	initializer := r.testPoolInitializer
	r.mu.RUnlock()
	if initializer != nil {
		if err := initializer(ctx, pool); err != nil {
			_ = pool.Close()
			return fmt.Errorf("initialize resumed test pool: %w", err)
		}
	}
	r.mu.Lock()
	if !state.ExpiresAt.After(r.clock.Now()) {
		r.mu.Unlock()
		_ = pool.Close()
		return errors.New("database.RoutedDB: restart binding is invalid or expired")
	}
	if r.testPoolIsLiveLocked() {
		active := r.lease.id
		r.mu.Unlock()
		_ = pool.Close()
		return &ErrLeaseConflict{ActiveLeaseID: active}
	}
	old := r.test
	r.test = pool
	r.lease = leaseState{id: state.LeaseID, expiresAt: state.ExpiresAt, dsn: state.DSN}
	r.stats.testPoolRequests.Store(0)
	r.stats.primaryDuringTestModeRequest.Store(0)
	r.expiryWarned.Store(false)
	r.mu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	return nil
}

// nonCreatingSQLiteDSN verifies the retained database exists, then forces
// SQLite's read/write-existing mode so a deleted lease file cannot be silently
// recreated during resume. Memory databases cannot satisfy persistence proof.
func nonCreatingSQLiteDSN(driver, dsn string) (string, error) {
	if driver != DriverSQLite && driver != DriverSQLiteLegacy {
		return dsn, nil
	}
	if dsn == "" || dsn == ":memory:" {
		return "", errors.New("in-memory database cannot resume")
	}
	var path string
	var query url.Values
	isURI := strings.HasPrefix(dsn, "file:")
	if isURI {
		parsed, err := url.Parse(dsn)
		if err != nil || parsed.Scheme != "file" || parsed.Host != "" {
			return "", errors.New("invalid SQLite URI")
		}
		path = parsed.Path
		if parsed.Opaque != "" {
			path = parsed.Opaque
		}
		query = parsed.Query()
	} else {
		filePath, rawQuery, found := strings.Cut(dsn, "?")
		path = filePath
		query = url.Values{}
		if found {
			var err error
			query, err = url.ParseQuery(rawQuery)
			if err != nil {
				return "", err
			}
		}
	}
	if path == "" || path == ":memory:" || strings.EqualFold(query.Get("mode"), "memory") {
		return "", errors.New("in-memory database cannot resume")
	}
	info, err := os.Stat(filepath.FromSlash(path))
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("retained SQLite file is unavailable")
	}
	query.Set("mode", "rw")
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	parsed := url.URL{Scheme: "file", Path: filepath.ToSlash(abs), RawQuery: query.Encode()}
	return parsed.String(), nil
}

// HeartbeatTestPool extends the active lease's expiry by ttl (or by
// DefaultLeaseTTL if ttl <= 0). The caller's leaseID must match the active
// install. Returns the new absolute expiry timestamp.
func (r *RoutedDB) HeartbeatTestPool(leaseID string, ttl time.Duration) (time.Time, error) {
	if ttl <= 0 {
		ttl = DefaultLeaseTTL
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.test == nil {
		return time.Time{}, &ErrLeaseMismatch{ActiveLeaseID: ""}
	}
	if r.lease.id != leaseID {
		return time.Time{}, &ErrLeaseMismatch{ActiveLeaseID: r.lease.id}
	}
	r.lease.expiresAt = r.clock.Now().Add(ttl)
	return r.lease.expiresAt, nil
}

// ActiveLeaseID returns the lease_id currently owning the installed test
// pool, or empty string if no pool is installed.
func (r *RoutedDB) ActiveLeaseID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lease.id
}

// LeaseStats returns a snapshot of the per-lease counters tracked since the
// last InstallTestPool. Safe to call concurrently with pick().
func (r *RoutedDB) LeaseStats() LeaseStatsSnapshot {
	return LeaseStatsSnapshot{
		TestPoolRequests:              r.stats.testPoolRequests.Load(),
		PrimaryDuringTestModeRequests: r.stats.primaryDuringTestModeRequest.Load(),
	}
}

// ClearTestPool closes the installed test pool and reverts routing to the
// primary pool. Lease semantics:
//
//   - No pool installed: no-op success regardless of leaseID.
//   - Pool installed under same leaseID: clear.
//   - Pool installed under a different leaseID: reject with *ErrLeaseMismatch.
//
// The LeaseStats counters are *not* reset by Clear; callers should snapshot
// LeaseStats() before invoking Clear if they need the values.
func (r *RoutedDB) ClearTestPool(leaseID string) error {
	r.mu.Lock()
	if r.test == nil {
		r.mu.Unlock()
		return nil
	}
	if r.lease.id != leaseID {
		active := r.lease.id
		r.mu.Unlock()
		return &ErrLeaseMismatch{ActiveLeaseID: active}
	}
	old := r.test
	r.test = nil
	r.lease = leaseState{}
	r.mu.Unlock()

	if old != nil {
		return old.Close()
	}
	return nil
}

// Close closes both the primary and (if present) the test pool. The first
// error encountered is returned; both pools are always attempted.
func (r *RoutedDB) Close() error {
	r.mu.Lock()
	primary := r.primary
	test := r.test
	r.primary = nil
	r.test = nil
	r.lease = leaseState{}
	r.mu.Unlock()

	var errPrimary, errTest error
	if primary != nil {
		errPrimary = primary.Close()
	}
	if test != nil {
		errTest = test.Close()
	}
	if errPrimary != nil {
		return errPrimary
	}
	return errTest
}

// QueryContext executes a query that returns rows, against the routed pool.
func (r *RoutedDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return r.pick(ctx).QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query that returns at most one row, against the
// routed pool.
func (r *RoutedDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return r.pick(ctx).QueryRowContext(ctx, query, args...)
}

// ExecContext executes a query without returning rows, against the routed
// pool.
func (r *RoutedDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return r.pick(ctx).ExecContext(ctx, query, args...)
}

// BeginTx starts a transaction on the routed pool. The returned *sql.Tx is
// bound to whichever pool was picked at this call; a transaction cannot span
// the primary and test pools.
func (r *RoutedDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return r.pick(ctx).BeginTx(ctx, opts)
}

// PrepareContext creates a prepared statement on the routed pool. The
// statement is bound to whichever pool was picked at this call.
func (r *RoutedDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return r.pick(ctx).PrepareContext(ctx, query)
}

// Conn returns a single connection from the routed pool.
func (r *RoutedDB) Conn(ctx context.Context) (*sql.Conn, error) {
	return r.pick(ctx).Conn(ctx)
}

// PingContext verifies a connection to the primary pool. The test pool, if
// installed, is intentionally not pinged here; callers that care about the
// test pool's health should query it explicitly.
func (r *RoutedDB) PingContext(ctx context.Context) error {
	r.mu.RLock()
	primary := r.primary
	r.mu.RUnlock()
	if primary == nil {
		return errors.New("database.RoutedDB: primary pool is closed")
	}
	return primary.PingContext(ctx)
}

// Ping is shorthand for PingContext(context.Background()).
func (r *RoutedDB) Ping() error {
	return r.PingContext(context.Background())
}

// Stats returns the primary pool's stats. The test pool's stats are not
// included; in-flight test runs are short-lived and querying them through the
// routing surface would obscure production telemetry.
func (r *RoutedDB) Stats() sql.DBStats {
	r.mu.RLock()
	primary := r.primary
	r.mu.RUnlock()
	if primary == nil {
		return sql.DBStats{}
	}
	return primary.Stats()
}
