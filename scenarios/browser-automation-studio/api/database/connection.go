package database

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/constants"
	repocontract "github.com/vrooli/repo-contract-go"

	_ "modernc.org/sqlite" // registers the pure-Go sqlite driver for database/sql
)

// DB wraps sqlx.DB with additional functionality
type DB struct {
	*sqlx.DB
	log *logrus.Logger
}

// Note: As of Go 1.20, the global random generator is automatically seeded.
// No explicit seeding is needed.

// NewConnection creates a new database connection with exponential backoff
func NewConnection(log *logrus.Logger) (*DB, error) {
	var db *sqlx.DB
	var err error

	// Load configuration from control surface
	cfg := config.Load()

	// Exponential backoff configuration
	maxRetries := cfg.Database.MaxRetries
	baseDelay := cfg.Database.BaseRetryDelay
	maxDelay := cfg.Database.MaxRetryDelay
	jitterFactor := cfg.Database.RetryJitterFactor

	for attempt := 0; attempt < maxRetries; attempt++ {
		db, err = connectSQLite(log)

		if err == nil {
			// Test the connection
			ctx, cancel := context.WithTimeout(context.Background(), constants.DatabaseQueryTimeout)
			err = db.PingContext(ctx)
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
		jitter := time.Duration(float64(delay) * jitterFactor * rand.Float64())
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

	// SQLite supports a single writer; pool of one avoids busy-locks under contention.
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	dbWrapper := &DB{
		DB:  db,
		log: log,
	}

	// Initialize database schema
	if err := dbWrapper.initSchema(); err != nil {
		log.WithError(err).Error("Failed to initialize database schema")
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return dbWrapper, nil
}

func connectSQLite(log *logrus.Logger) (*sqlx.DB, error) {
	dsn, err := sqliteDSN(log)
	if err != nil {
		return nil, err
	}

	return sqlx.Connect("sqlite", dsn)
}

func sqliteDSN(log *logrus.Logger) (string, error) {
	root := strings.TrimSpace(os.Getenv("BAS_SQLITE_PATH"))
	if root == "" {
		if custom := strings.TrimSpace(os.Getenv("DATABASE_URL")); strings.HasPrefix(custom, "file:") {
			return custom, nil
		}
		path, err := scenarioDBPath()
		if err != nil {
			return "", fmt.Errorf("resolve canonical sqlite path: %w", err)
		}
		root = path
	}

	if err := os.MkdirAll(filepath.Dir(root), 0o755); err != nil {
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}

	if log != nil {
		log.WithField("path", root).Info("Using SQLite database")
	}

	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=page_size(4096)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)&_pragma=mmap_size(268435456)",
		root,
	), nil
}

func scenarioDBPath() (string, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", err
	}
	return resolver.Path(storage.Options{ScenarioID: "browser-automation-studio"}, storage.ClassData, "browser-automation-studio.db")
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
	tx, err := db.Beginx()
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
func (db *DB) initSchema() error {
	scenarioRoot, err := resolveScenarioRoot()
	if err != nil {
		return fmt.Errorf("resolve browser-automation-studio scenario root: %w", err)
	}
	schemaPath := filepath.Join(scenarioRoot, "initialization", "storage", "sqlite", "schema.sql")

	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		db.log.WithError(err).WithField("path", schemaPath).Error("Failed to read schema file")
		return fmt.Errorf("failed to read schema file at %s: %w", schemaPath, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DatabaseMigrationTimeout)
	defer cancel()

	if _, err := db.ExecContext(ctx, string(schemaBytes)); err != nil {
		db.log.WithError(err).Error("Failed to execute schema initialization")
		return err
	}

	db.log.Info("Database schema initialized successfully")
	return nil
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
