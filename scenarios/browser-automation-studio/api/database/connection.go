package database

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/constants"
	repocontract "github.com/vrooli/repo-contract-go"

	_ "modernc.org/sqlite" // registers the pure-Go sqlite driver for database/sql
)

// DB wraps sqlx.DB with additional functionality
type DB struct {
	*sqlx.DB
	// Routed is the request-context-aware persistence seam used by domains that
	// participate in Test Genie lease isolation. The embedded sqlx handle remains
	// temporarily for repository code that has not yet completed its sqlx-to-
	// domain-repository migration.
	Routed *coredb.RoutedDB
	log    *logrus.Logger
}

// Note: As of Go 1.20, the global random generator is automatically seeded.
// No explicit seeding is needed.

// NewConnection creates a new database connection with exponential backoff
func NewConnection(log *logrus.Logger) (*DB, error) {
	var routed *coredb.RoutedDB
	var err error

	// Load configuration from control surface
	cfg := config.Load()

	// Exponential backoff configuration
	maxRetries := cfg.Database.MaxRetries
	baseDelay := cfg.Database.BaseRetryDelay
	maxDelay := cfg.Database.MaxRetryDelay
	jitterFactor := cfg.Database.RetryJitterFactor

	for attempt := 0; attempt < maxRetries; attempt++ {
		routed, err = connectRoutedSQLite(log)

		if err == nil {
			// Test the connection
			ctx, cancel := context.WithTimeout(context.Background(), constants.DatabaseQueryTimeout)
			err = routed.PingContext(ctx)
			cancel()

			if err == nil {
				log.Info("Successfully connected to database")
				break
			}
		}

		// Calculate delay with exponential backoff and random jitter
		delay := baseDelay * time.Duration(1<<attempt)
		if delay > maxDelay {
			delay = maxDelay
		}
		jitterMax := int64(float64(delay) * jitterFactor)
		jitter := time.Duration(0)
		if jitterMax > 0 {
			value, randomErr := cryptorand.Int(cryptorand.Reader, big.NewInt(jitterMax+1))
			if randomErr != nil {
				return nil, fmt.Errorf("generate retry jitter: %w", randomErr)
			}
			jitter = time.Duration(value.Int64())
		}
		actualDelay := delay + jitter

		log.WithFields(logrus.Fields{
			"attempt":    attempt + 1,
			"maxRetries": maxRetries,
			"delay":      actualDelay,
			"error":      err.Error(),
		}).Warn("Failed to connect to database, retrying...")

		time.Sleep(actualDelay)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
	}

	if routed == nil {
		return nil, fmt.Errorf("database connection did not initialize a routed pool")
	}

	// Keep the existing sqlx repository surface on top of the routed primary
	// pool while domain-owned consumers migrate to DB.Routed. The one-writer
	// SQLite policy is configured in connectRoutedSQLite.
	db := sqlx.NewDb(routed.Primary(), "sqlite")
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	dbWrapper := &DB{
		DB:     db,
		Routed: routed,
		log:    log,
	}

	// Initialize database schema
	if err := dbWrapper.EnsureSchemas(); err != nil {
		log.WithError(err).Error("Failed to initialize database schema")
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}
	// A leased Test Genie pool must receive the same schema before requests are
	// allowed to route to it. The initializer runs before installation becomes
	// visible, so no request can observe an uninitialized test database.
	routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		testDB := &DB{DB: sqlx.NewDb(pool, "sqlite"), log: log}
		return testDB.EnsureSchemas()
	})

	return dbWrapper, nil
}

func connectRoutedSQLite(log *logrus.Logger) (*coredb.RoutedDB, error) {
	dsn, err := sqliteDSN(log)
	if err != nil {
		return nil, err
	}
	return coredb.Open(context.Background(), coredb.Config{
		Driver:       coredb.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
}

// connectSQLite is retained for focused database tests that exercise the
// sqlx-specific compatibility helpers. Production startup uses
// connectRoutedSQLite so request-scoped test routing remains the sole runtime
// connection path.
func connectSQLite(log *logrus.Logger) (*sqlx.DB, error) {
	routed, err := connectRoutedSQLite(log)
	if err != nil {
		return nil, err
	}
	return sqlx.NewDb(routed.Primary(), "sqlite"), nil
}

// sqliteTuning is the driver configuration this scenario needs on top of the
// shared defaults api-core/storage already applies (foreign_keys, WAL,
// busy_timeout, cache_size, synchronous, temp_store).
//
// _time_format=sqlite makes the modernc.org/sqlite driver bind time.Time values
// as "2006-01-02 15:04:05.999999999-07:00" (one of its built-in parseable
// layouts). Without it the driver falls back to time.Time.String()
// ("2026-05-20 14:42:49.49 +0000 UTC"), which the same driver cannot read back
// into *time.Time — every SELECT into a timestamp field 500s with "unsupported
// Scan". See modernc.org/sqlite@v1.40.1/sqlite.go formatTime() for the upstream
// default.
var sqliteTuning = storage.SQLiteTuning{
	PageSizeBytes: 4096,
	MMapSizeBytes: 268435456,
	TimeFormat:    "sqlite",
}

// sqliteDSN resolves this scenario's database through api-core/storage.
//
// The path comes from the scenario naming ITSELF, never from a generic
// environment variable. A variable that does not identify a scenario cannot be
// scoped to one: any process exporting it redirects the database of every
// scenario it goes on to start. BAS_SQLITE_PATH remains because it names this
// scenario and the desktop bundle sets it deliberately (bundle/bundle.json);
// it is passed as an argument rather than read as ambient configuration.
func sqliteDSN(log *logrus.Logger) (string, error) {
	var (
		dsn string
		err error
	)
	if root := strings.TrimSpace(os.Getenv("BAS_SQLITE_PATH")); root != "" {
		if log != nil {
			log.WithField("path", root).Info("Using SQLite database")
		}
		dsn, err = storage.SQLiteDSNAt(root, sqliteTuning)
	} else {
		dsn, err = storage.SQLiteDSN(storage.SQLiteConfig{
			Scenario: "browser-automation-studio",
			Tuning:   sqliteTuning,
		})
		if err == nil && log != nil {
			log.WithField("dsn", dsn).Info("Using SQLite database")
		}
	}
	if err != nil {
		return "", fmt.Errorf("resolve sqlite dsn: %w", err)
	}
	return dsn, nil
}

// RawDB returns the underlying *sql.DB for direct database access
func (db *DB) RawDB() *sql.DB {
	return db.DB.DB
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// HealthCheck performs a health check on the database
func (db *DB) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), constants.DatabasePingTimeout)
	defer cancel()
	return db.PingContext(ctx)
}

// WithTransaction executes a function within a database transaction
func (db *DB) WithTransaction(fn func(*sqlx.Tx) error) error {
	// sqlx does not expose a safe constructor for wrapping a transaction
	// selected by RoutedDB. This compatibility helper is not used by the
	// request-facing repository; migrate its remaining callers before routing
	// transaction-based domain work through this method.
	tx, err := db.DB.BeginTxx(context.Background(), nil)
	if err != nil {
		return err
	}

	rollback := func(reason string) {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
			if db.log != nil {
				db.log.WithError(rollbackErr).WithField("reason", reason).Warn("Failed to rollback transaction")
			}
		}
	}

	defer func() {
		if r := recover(); r != nil {
			rollback("panic")
			// Log the panic but don't re-panic - let the error propagate gracefully
			if db.log != nil {
				db.log.WithField("panic", r).Error("Panic recovered during database transaction")
			}
		}
	}()

	if err := fn(tx); err != nil {
		rollback("error")
		return err
	}

	return tx.Commit()
}

// initSchema initializes the database schema
// EnsureSchemas creates or updates the schema for this database pool. It is
// used for both the primary pool and each leased Test Genie pool before that
// pool is made available to request routing.
func (db *DB) EnsureSchemas() error {
	scenarioRoot, err := resolveScenarioRoot()
	if err != nil {
		return fmt.Errorf("resolve browser-automation-studio scenario root: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.DatabaseMigrationTimeout)
	defer cancel()

	providers, err := SchemaProviders(scenarioRoot)
	if err != nil {
		return fmt.Errorf("load domain schema registry: %w", err)
	}
	if err := coredb.EnsureSchemas(ctx, db.DB, providers...); err != nil {
		return fmt.Errorf("initialize domain schema registry: %w", err)
	}

	if err := db.ensureSchemaCompatibility(ctx); err != nil {
		return err
	}

	db.log.Info("Database schema initialized successfully")
	return nil
}

func (db *DB) ensureSchemaCompatibility(ctx context.Context) error {
	if err := db.ensureWorkflowSchemaCompatibility(ctx); err != nil {
		return err
	}
	if err := db.ensureExecutionSchemaCompatibility(ctx); err != nil {
		return err
	}
	return nil
}

func (db *DB) ensureWorkflowSchemaCompatibility(ctx context.Context) error {
	hasWorkflows, err := db.tableExists(ctx, "workflows")
	if err != nil {
		return fmt.Errorf("check workflows table: %w", err)
	}
	if !hasWorkflows {
		return nil
	}

	hasFilePath, err := db.columnExists(ctx, "workflows", "file_path")
	if err != nil {
		return fmt.Errorf("check workflows.file_path column: %w", err)
	}
	if hasFilePath {
		return nil
	}

	if err := db.ensureColumnCompatibility(ctx, "workflows", "file_path", "TEXT"); err != nil {
		return fmt.Errorf("ensure workflows.file_path column: %w", err)
	}
	return nil
}

func (db *DB) ensureExecutionSchemaCompatibility(ctx context.Context) error {
	hasExecutions, err := db.tableExists(ctx, "executions")
	if err != nil {
		return fmt.Errorf("check executions table: %w", err)
	}
	if !hasExecutions {
		return nil
	}

	requiredColumns := map[string]string{
		"error_message":   "TEXT",
		"result_path":     "TEXT",
		"resumed_from_id": "TEXT",
	}
	for columnName, columnDef := range requiredColumns {
		if err := db.ensureColumnCompatibility(ctx, "executions", columnName, columnDef); err != nil {
			return fmt.Errorf("ensure executions.%s column: %w", columnName, err)
		}
	}
	return nil
}

func (db *DB) ensureColumnCompatibility(ctx context.Context, tableName, columnName, columnDefinition string) error {
	hasColumn, err := db.columnExists(ctx, tableName, columnName)
	if err != nil {
		return fmt.Errorf("check %s.%s column: %w", tableName, columnName, err)
	}
	if hasColumn {
		return nil
	}

	statement := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, columnName, columnDefinition)
	if _, err := db.ExecContext(ctx, statement); err != nil {
		if db.log != nil {
			db.log.WithError(err).Warnf("Failed to add %s.%s compatibility column", tableName, columnName)
		}
		return fmt.Errorf("add %s.%s column: %w", tableName, columnName, err)
	}
	if db.log != nil {
		db.log.Warnf("Added missing %s.%s column for SQLite compatibility", tableName, columnName)
	}
	return nil
}

func (db *DB) tableExists(ctx context.Context, tableName string) (bool, error) {
	var count int
	if err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, tableName); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *DB) columnExists(ctx context.Context, tableName, columnName string) (bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			dataType   string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultVal, &primaryKey); err != nil {
			return false, err
		}
		if name == columnName {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return false, nil
}

func resolveScenarioRoot() (string, error) {
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		repoRoot, err := repocontract.FindRepoRootFromPath(root)
		if err != nil {
			return "", err
		}
		return repocontract.ResolveScenarioPath(repoRoot, "browser-automation-studio")
	}

	repoRoot, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return "", err
	}
	return repocontract.ResolveScenarioPath(repoRoot, "browser-automation-studio")
}
