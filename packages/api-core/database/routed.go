package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
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
	testLease string
	cfg       Config
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

// ErrLeaseMismatch is returned by ClearTestPool when the caller's
// lease_id does not match the active install.
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
	return &RoutedDB{primary: primary, cfg: cfg}, nil
}

// MustOpen is like Open but panics on error. Useful for main().
func MustOpen(ctx context.Context, cfg Config) *RoutedDB {
	r, err := Open(ctx, cfg)
	if err != nil {
		panic(fmt.Sprintf("database.MustOpen: %v", err))
	}
	return r
}

// pick returns the pool that should serve a request carrying ctx.
// It takes a read lock so installs/clears can occur concurrently with
// in-flight requests without tearing.
func (r *RoutedDB) pick(ctx context.Context) *sql.DB {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.test != nil && IsTestMode(ctx) {
		return r.test
	}
	return r.primary
}

// Primary returns the underlying primary pool. Use sparingly — most call sites
// should depend on the *RoutedDB surface so they participate in routing.
func (r *RoutedDB) Primary() *sql.DB {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.primary
}

// HasTestPool reports whether a test pool is currently installed.
func (r *RoutedDB) HasTestPool() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.test != nil
}

// InstallTestPool opens a new pool against dsn and installs it as the
// active test pool under leaseID. Lease semantics:
//
//   - No pool installed: install under leaseID (empty string is allowed —
//     it represents an un-claimed ad-hoc install used only by direct
//     operator tooling).
//   - Pool installed under same leaseID: idempotent — close the old pool,
//     install the new one under the same lease. Retry-safe.
//   - Pool installed under a different leaseID: reject with
//     *ErrLeaseConflict carrying the active lease.
func (r *RoutedDB) InstallTestPool(ctx context.Context, dsn, leaseID string) error {
	if dsn == "" {
		return errors.New("database.RoutedDB.InstallTestPool: dsn is empty")
	}
	if err := validateDSNMatchesDriver(r.cfg.Driver, dsn); err != nil {
		return fmt.Errorf("install test pool: %w", err)
	}

	r.mu.Lock()
	if r.test != nil && r.testLease != leaseID {
		active := r.testLease
		r.mu.Unlock()
		return &ErrLeaseConflict{ActiveLeaseID: active}
	}
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
	if r.test != nil && r.testLease != leaseID {
		active := r.testLease
		r.mu.Unlock()
		_ = pool.Close()
		return &ErrLeaseConflict{ActiveLeaseID: active}
	}
	old := r.test
	r.test = pool
	r.testLease = leaseID
	r.mu.Unlock()

	if old != nil {
		_ = old.Close()
	}
	return nil
}

// ActiveLeaseID returns the lease_id currently owning the installed test
// pool, or empty string if no pool is installed.
func (r *RoutedDB) ActiveLeaseID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.testLease
}

// ClearTestPool closes the installed test pool and reverts routing to the
// primary pool. Lease semantics:
//
//   - No pool installed: no-op success regardless of leaseID.
//   - Pool installed under same leaseID: clear.
//   - Pool installed under a different leaseID: reject with *ErrLeaseMismatch.
func (r *RoutedDB) ClearTestPool(leaseID string) error {
	r.mu.Lock()
	if r.test == nil {
		r.mu.Unlock()
		return nil
	}
	if r.testLease != leaseID {
		active := r.testLease
		r.mu.Unlock()
		return &ErrLeaseMismatch{ActiveLeaseID: active}
	}
	old := r.test
	r.test = nil
	r.testLease = ""
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
	r.testLease = ""
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
