// Package database provides database connection helpers with automatic retry
// and environment-based configuration for Vrooli scenarios.
//
// The package automatically reads connection parameters from environment variables
// set by the Vrooli lifecycle system, eliminating boilerplate configuration code.
//
// Basic usage (reads POSTGRES_* from environment):
//
//	db, err := database.Connect(ctx, database.Config{
//	    Driver: "postgres",
//	})
//	if err != nil {
//	    log.Fatalf("Database connection failed: %v", err)
//	}
//	defer db.Close()
//
// With custom pool settings:
//
//	db, err := database.Connect(ctx, database.Config{
//	    Driver:       "postgres",
//	    MaxOpenConns: 50,
//	})
//
// With explicit DSN (bypasses environment):
//
//	db, err := database.Connect(ctx, database.Config{
//	    Driver: "postgres",
//	    DSN:    "postgres://user:pass@host:5432/db",
//	})
package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/retry"
)

// Supported drivers with auto-configuration from environment variables.
const (
	// DriverPostgres auto-reads POSTGRES_* environment variables.
	// Required: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_DB
	// Optional: POSTGRES_PASSWORD, POSTGRES_SSLMODE (default: disable)
	// Also checks: POSTGRES_URL, DATABASE_URL (used directly if set)
	DriverPostgres = "postgres"

	// DriverSQLite reads SQLITE_PATH or SQLITE_DB environment variables and
	// matches the modernc.org/sqlite driver name used by cross-platform scenarios.
	DriverSQLite = "sqlite"

	// DriverSQLiteLegacy remains supported for scenarios that still open the CGO
	// sqlite3 driver explicitly.
	DriverSQLiteLegacy = "sqlite3"
)

// Config controls database connection behavior.
type Config struct {
	// Driver specifies the database driver (e.g., "postgres", "sqlite").
	// For known drivers, connection parameters are auto-read from environment.
	// Required.
	Driver string

	// DSN overrides automatic environment-based configuration.
	// If set, environment variables are ignored for connection string.
	// Optional.
	DSN string

	// MaxOpenConns sets the maximum number of open connections.
	// If zero, reads from POSTGRES_POOL_SIZE or defaults to 25.
	MaxOpenConns int

	// MaxIdleConns sets the maximum number of idle connections.
	// If zero, defaults to MaxOpenConns/5 (minimum 2).
	MaxIdleConns int

	// ConnMaxLifetime sets the maximum lifetime of connections.
	// If zero, defaults to 5 minutes.
	ConnMaxLifetime time.Duration

	// Retry configures retry behavior for connection attempts.
	// If nil, uses retry.DefaultConfig() (10 attempts, 500ms base, 30s max).
	Retry *retry.Config

	// Opener overrides sql.Open for testing.
	// If nil, uses sql.Open.
	Opener func(driver, dsn string) (*sql.DB, error)

	// EnvGetter overrides os.Getenv for testing.
	// If nil, uses os.Getenv.
	EnvGetter func(key string) string

	// Logger receives connection status messages.
	// Optional. If nil, no logging occurs.
	Logger func(format string, args ...interface{})
}

// Connect opens a database connection with retry and automatic configuration.
//
// For known drivers (postgres, sqlite, sqlite3), connection parameters are automatically
// read from environment variables set by the Vrooli lifecycle system.
//
// The function retries failed connections using exponential backoff with jitter
// to prevent thundering herd when multiple services restart simultaneously.
//
// Returns a configured *sql.DB with connection pooling applied.
func Connect(ctx context.Context, cfg Config) (*sql.DB, error) {
	cfg = withDefaults(cfg)

	// Build DSN from environment if not provided
	if cfg.DSN == "" {
		dsn, err := buildDSNFromEnv(cfg)
		if err != nil {
			return nil, fmt.Errorf("build connection string: %w", err)
		}
		cfg.DSN = dsn
	}

	// Resolve pool settings from environment
	resolvePoolSettings(&cfg)

	// Configure retry
	retryCfg := retry.DefaultConfig()
	if cfg.Retry != nil {
		retryCfg = *cfg.Retry
	}
	if retryCfg.Retryable == nil {
		retryCfg.Retryable = isRetryableConnectionError
	}

	// Add logging callback if logger provided
	if cfg.Logger != nil {
		originalOnRetry := retryCfg.OnRetry
		retryCfg.OnRetry = func(attempt int, err error, delay time.Duration) {
			cfg.Logger("database connection attempt %d failed: %v (retrying in %v)", attempt, err, delay)
			if originalOnRetry != nil {
				originalOnRetry(attempt, err, delay)
			}
		}
	}

	var db *sql.DB
	err := retry.Do(ctx, retryCfg, func(attempt int) error {
		var openErr error
		db, openErr = cfg.open()
		if openErr != nil {
			return fmt.Errorf("open: %w", openErr)
		}

		// Ping to verify the connection is actually working
		if pingErr := db.PingContext(ctx); pingErr != nil {
			db.Close() // Clean up failed connection
			return fmt.Errorf("ping: %w", pingErr)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if cfg.Logger != nil {
		cfg.Logger("database connected (pool: %d open, %d idle, %v lifetime)",
			cfg.MaxOpenConns, cfg.MaxIdleConns, cfg.ConnMaxLifetime)
	}

	return db, nil
}

// isRetryableConnectionError rejects configuration failures that no retry can
// repair. database/sql exposes unknown-driver failures as an error string
// rather than a public sentinel, so match its stable diagnostic while allowing
// operational open and ping failures to use the normal retry policy.
func isRetryableConnectionError(err error) bool {
	return !strings.Contains(err.Error(), "sql: unknown driver ")
}

// MustConnect is like Connect but panics on error.
// Useful for initialization in main() where failure is unrecoverable.
func MustConnect(ctx context.Context, cfg Config) *sql.DB {
	db, err := Connect(ctx, cfg)
	if err != nil {
		panic(fmt.Sprintf("database.MustConnect: %v", err))
	}
	return db
}

// buildDSNFromEnv constructs a DSN from environment variables based on driver.
func buildDSNFromEnv(cfg Config) (string, error) {
	getenv := cfg.getenv

	switch cfg.Driver {
	case DriverPostgres:
		return buildPostgresDSN(getenv)

	case DriverSQLite, DriverSQLiteLegacy:
		return buildSQLiteDSN(getenv)

	default:
		return "", fmt.Errorf(
			"driver %q has no auto-configuration; provide DSN explicitly or use a supported driver (%s, %s, %s)",
			cfg.Driver, DriverPostgres, DriverSQLite, DriverSQLiteLegacy,
		)
	}
}

// buildPostgresDSN constructs a PostgreSQL connection string from environment.
//
// Shadow isolation: POSTGRES_DB is injected by the lifecycle from
// scenarioruntime.InstanceKey.Namespace().PostgresDB — "vrooli_<scenario>" for
// the live instance, "vrooli_<scenario>_<variant>" for a shadow. Postgres
// therefore inherits variant isolation for free; do NOT hardcode the scenario's
// database name anywhere — that would point a shadow at live's database. See the
// storage package's variant-aware namespace helpers for the Redis/Qdrant
// equivalent.
func buildPostgresDSN(getenv func(string) string) (string, error) {
	// First check if a complete URL is already set
	if url := getenv("POSTGRES_URL"); url != "" {
		return url, nil
	}
	if url := getenv("DATABASE_URL"); url != "" {
		return url, nil
	}

	// Build from individual components
	host := getenv("POSTGRES_HOST")
	port := getenv("POSTGRES_PORT")
	user := getenv("POSTGRES_USER")
	pass := getenv("POSTGRES_PASSWORD")
	dbname := getenv("POSTGRES_DB")
	sslmode := getenv("POSTGRES_SSLMODE")

	// Validate required fields
	var missing []string
	if host == "" {
		missing = append(missing, "POSTGRES_HOST")
	}
	if port == "" {
		missing = append(missing, "POSTGRES_PORT")
	}
	if user == "" {
		missing = append(missing, "POSTGRES_USER")
	}
	if dbname == "" {
		missing = append(missing, "POSTGRES_DB")
	}

	if len(missing) > 0 {
		return "", fmt.Errorf(
			"postgres connection requires environment variables: %v (are you running through the Vrooli lifecycle system?)",
			missing,
		)
	}

	// Default sslmode
	if sslmode == "" {
		sslmode = "disable"
	}

	// Build URL
	// Note: password may be empty for local development
	if pass != "" {
		return fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			user, pass, host, port, dbname, sslmode,
		), nil
	}

	return fmt.Sprintf(
		"postgres://%s@%s:%s/%s?sslmode=%s",
		user, host, port, dbname, sslmode,
	), nil
}

// buildSQLiteDSN returns the SQLite database file path from environment.
//
// Shadow isolation: SQLITE_PATH (or the storage resolver's variant-scoped data
// dir behind SQLITE_DB) is rooted under the lifecycle's per-variant data
// directory, so a shadow's SQLite file is physically distinct from live's. Read
// the path from the environment as below; never hardcode a per-scenario path
// that would alias the two variants.
func buildSQLiteDSN(getenv func(string) string) (string, error) {
	if path := getenv("SQLITE_PATH"); path != "" {
		return path, nil
	}
	if path := getenv("SQLITE_DB"); path != "" {
		return path, nil
	}
	return "", fmt.Errorf(
		"sqlite connection requires SQLITE_PATH or SQLITE_DB environment variable",
	)
}

// resolvePoolSettings reads pool configuration from environment if not set.
func resolvePoolSettings(cfg *Config) {
	getenv := cfg.getenv

	// MaxOpenConns
	if cfg.MaxOpenConns == 0 {
		if v := getenv("POSTGRES_POOL_SIZE"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				cfg.MaxOpenConns = n
			}
		}
		if cfg.MaxOpenConns == 0 {
			cfg.MaxOpenConns = 25
		}
	}

	// MaxIdleConns (derived from MaxOpenConns if not set)
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = cfg.MaxOpenConns / 5
		if cfg.MaxIdleConns < 2 {
			cfg.MaxIdleConns = 2
		}
	}

	// ConnMaxLifetime
	if cfg.ConnMaxLifetime == 0 {
		cfg.ConnMaxLifetime = 5 * time.Minute
	}
}

// getenv returns environment variable value using custom getter or os.Getenv.
func (cfg Config) getenv(key string) string {
	if cfg.EnvGetter != nil {
		return cfg.EnvGetter(key)
	}
	return os.Getenv(key)
}

// open opens a database connection using custom opener or sql.Open.
func (cfg Config) open() (*sql.DB, error) {
	if cfg.Opener != nil {
		return cfg.Opener(cfg.Driver, cfg.DSN)
	}
	return sql.Open(cfg.Driver, cfg.DSN)
}

// withDefaults fills in zero values with defaults.
func withDefaults(cfg Config) Config {
	if cfg.Driver == "" {
		cfg.Driver = DriverPostgres
	}
	return cfg
}
