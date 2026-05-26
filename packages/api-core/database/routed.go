package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"
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
type Clock interface {
	Now() time.Time
}

// systemClock is the default Clock backed by time.Now.
type systemClock struct{}

// Now reports the current wall-clock time.
func (systemClock) Now() time.Time { return time.Now() }

// SystemClock returns the default time.Now-backed Clock. Convenience for
// callers that want to be explicit about which clock they're injecting.
func SystemClock() Clock { return systemClock{} }

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
	mu        sync.RWMutex
	primary   *sql.DB
	test      *sql.DB
	lease     leaseState
	cfg       Config
	clock     Clock
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
}

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
	primary, err := Connect(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &RoutedDB{primary: primary, cfg: cfg, clock: systemClock{}}, nil
}

// OpenWithClock is Open with an explicit clock seam. Tests pass a fake
// clock; production callers use Open.
func OpenWithClock(ctx context.Context, cfg Config, clock Clock) (*RoutedDB, error) {
	primary, err := Connect(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &RoutedDB{primary: primary, cfg: cfg, clock: clock}, nil
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
		clock = systemClock{}
	}
	r.mu.Lock()
	r.clock = clock
	r.mu.Unlock()
}

// pick returns the pool that should serve a request carrying ctx.
// It takes a read lock so installs/clears can occur concurrently with
// in-flight requests without tearing. pick also enforces lease expiry: an
// expired lease behaves as if no pool were installed.
func (r *RoutedDB) pick(ctx context.Context) *sql.DB {
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
		r.expireLeaseIfStale(leaseID)
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
func (r *RoutedDB) expireLeaseIfStale(leaseID string) {
	r.mu.Lock()
	if r.test == nil || r.lease.id != leaseID {
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
//   - Pool installed under a different leaseID: reject with
//     *ErrLeaseConflict carrying the active lease.
//
// On a successful install the per-lease counters are reset.
func (r *RoutedDB) InstallTestPool(ctx context.Context, dsn, leaseID string, ttl time.Duration) error {
	if dsn == "" {
		return errors.New("database.RoutedDB.InstallTestPool: dsn is empty")
	}
	if err := validateDSNMatchesDriver(r.cfg.Driver, dsn); err != nil {
		return fmt.Errorf("install test pool: %w", err)
	}
	if ttl <= 0 {
		ttl = DefaultLeaseTTL
	}

	r.mu.Lock()
	if r.test != nil && r.lease.id != leaseID {
		active := r.lease.id
		r.mu.Unlock()
		return &ErrLeaseConflict{ActiveLeaseID: active}
	}
	clock := r.clock
	r.mu.Unlock()

	testCfg := r.cfg
	testCfg.DSN = dsn
	pool, err := Connect(ctx, testCfg)
	if err != nil {
		return fmt.Errorf("install test pool: %w", err)
	}

	r.mu.Lock()
	// Re-check under exclusive lock — another caller may have installed
	// between the read above and here.
	if r.test != nil && r.lease.id != leaseID {
		active := r.lease.id
		r.mu.Unlock()
		_ = pool.Close()
		return &ErrLeaseConflict{ActiveLeaseID: active}
	}
	old := r.test
	r.test = pool
	r.lease = leaseState{id: leaseID, expiresAt: clock.Now().Add(ttl)}
	r.stats.testPoolRequests.Store(0)
	r.stats.primaryDuringTestModeRequest.Store(0)
	r.expiryWarned.Store(false)
	r.mu.Unlock()

	if old != nil {
		_ = old.Close()
	}
	return nil
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
