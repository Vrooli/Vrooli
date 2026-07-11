package database

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-manager/internal/domain"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite" // SQLite driver
)

const (
	defaultQueryTimeout  = 30 * time.Second
	defaultSchemaTimeout = 60 * time.Second
	defaultPingTimeout   = 5 * time.Second
)

//go:embed schema.sql
var schema string

// DB wraps sqlx.DB with additional functionality for agent-manager.
type DB struct {
	*sqlx.DB
	log *logrus.Logger
}

// NewDB creates a DB wrapper from an existing sqlx.DB and logger.
// This is primarily intended for test setup where the caller manages
// the connection lifecycle and schema initialization directly.
func NewDB(db *sqlx.DB, log *logrus.Logger) *DB {
	return &DB{DB: db, log: log}
}

// NewConnection creates a new SQLite database connection.
func NewConnection(log *logrus.Logger) (*DB, error) {
	dsn, err := sqliteDSN(log)
	if err != nil {
		return nil, err
	}

	db, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		return nil, &domain.DatabaseError{
			Operation:   "connect",
			EntityType:  "Database",
			Cause:       err,
			IsTransient: true,
		}
	}
	db.SetMaxOpenConns(1)

	dbWrapper := &DB{DB: db, log: log}
	if err := dbWrapper.initSchema(); err != nil {
		_ = db.Close()
		log.WithError(err).Error("Failed to initialize database schema")
		return nil, err
	}

	log.Info("Successfully connected to database")
	return dbWrapper, nil
}

// DataDir returns the directory where the SQLite database file is stored.
// This follows the same resolution priority as sqliteDSN.
func DataDir() string {
	root := strings.TrimSpace(os.Getenv("AM_SQLITE_PATH"))
	if root != "" {
		return filepath.Dir(root)
	}
	if path, err := scenarioDBPath(); err == nil {
		return filepath.Dir(path)
	}
	return "."
}

func sqliteDSN(log *logrus.Logger) (string, error) {
	root := strings.TrimSpace(os.Getenv("AM_SQLITE_PATH"))
	if root == "" {
		if custom := strings.TrimSpace(os.Getenv("DATABASE_URL")); strings.HasPrefix(custom, "file:") {
			return custom, nil
		}
		path, err := scenarioDBPath()
		if err != nil {
			return "", domain.NewConfigInvalidError("AM_SQLITE_PATH", "resolve canonical sqlite path", err)
		}
		root = path
	}

	if err := os.MkdirAll(filepath.Dir(root), 0o755); err != nil {
		return "", domain.NewConfigInvalidError("AM_SQLITE_PATH", "prepare sqlite directory", err)
	}
	if log != nil {
		log.WithField("path", root).Info("Using SQLite database")
	}
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=page_size(4096)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		root,
	), nil
}

func scenarioDBPath() (string, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return "", err
	}
	scenarioID, err := storage.ScenarioNamespace("agent-manager")
	if err != nil {
		return "", err
	}
	return resolver.Path(storage.Options{ScenarioID: scenarioID}, storage.ClassData, "agent-manager.db")
}

// Close closes the database connection.
func (db *DB) Close() error { return db.DB.Close() }

// HealthCheck performs a health check on the database.
func (db *DB) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultPingTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return &domain.DatabaseError{Operation: "health_check", EntityType: "Database", Cause: err, IsTransient: true}
	}
	return nil
}

// WithTransaction executes a function within a database transaction.
func (db *DB) WithTransaction(fn func(*sqlx.Tx) error) error {
	tx, err := db.Beginx()
	if err != nil {
		return &domain.DatabaseError{Operation: "transaction_begin", EntityType: "Database", Cause: err}
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return &domain.DatabaseError{Operation: "transaction_commit", EntityType: "Database", Cause: err}
	}
	return nil
}

// initSchema applies the current declarative schema. Data evolution is a
// deliberate, one-shot operator action; startup never mutates legacy data.
func (db *DB) initSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSchemaTimeout)
	defer cancel()
	if _, err := db.ExecContext(ctx, schema); err != nil {
		if db.log != nil {
			db.log.WithError(err).Error("Failed to execute schema initialization")
		}
		return &domain.DatabaseError{Operation: "schema_init", EntityType: "Schema", Cause: err}
	}
	if db.log != nil {
		db.log.Info("Database schema initialized successfully")
	}
	return nil
}
