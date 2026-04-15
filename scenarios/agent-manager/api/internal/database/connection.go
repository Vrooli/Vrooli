package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite" // SQLite driver
)

// Default configuration values
const (
	defaultQueryTimeout     = 30 * time.Second
	defaultMigrationTimeout = 60 * time.Second
	defaultPingTimeout      = 5 * time.Second
)

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

	// SQLite only supports a single connection
	db.SetMaxOpenConns(1)

	dbWrapper := &DB{
		DB:  db,
		log: log,
	}

	// Initialize database schema
	if err := dbWrapper.initSchema(); err != nil {
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
	dataRoot := strings.TrimSpace(os.Getenv("SQLITE_DATABASE_PATH"))
	if dataRoot == "" {
		dataRoot = strings.TrimSpace(os.Getenv("VROOLI_DATA"))
	}
	if dataRoot == "" {
		if path, err := scenarioDBPath(); err == nil {
			return filepath.Dir(path)
		}
		home, _ := os.UserHomeDir()
		if home == "" {
			home = "."
		}
		dataRoot = filepath.Join(home, ".vrooli", "data", "sqlite", "databases")
	}
	return dataRoot
}

func sqliteDSN(log *logrus.Logger) (string, error) {
	root := strings.TrimSpace(os.Getenv("AM_SQLITE_PATH"))
	if root == "" {
		if custom := strings.TrimSpace(os.Getenv("DATABASE_URL")); strings.HasPrefix(custom, "file:") {
			return custom, nil
		}
		dataRoot := strings.TrimSpace(os.Getenv("SQLITE_DATABASE_PATH"))
		if dataRoot == "" {
			dataRoot = strings.TrimSpace(os.Getenv("VROOLI_DATA"))
		}
		if dataRoot == "" {
			if path, err := scenarioDBPath(); err == nil {
				if migrateErr := migrateLegacySQLite(path); migrateErr != nil {
					return "", domain.NewConfigInvalidError("AM_SQLITE_PATH", "migrate legacy sqlite", migrateErr)
				}
				root = path
				goto ensureDir
			}
			home, _ := os.UserHomeDir()
			if home == "" {
				home = "."
			}
			dataRoot = filepath.Join(home, ".vrooli", "data", "sqlite", "databases")
		}
		root = filepath.Join(dataRoot, "agent-manager.db")
	}

ensureDir:
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
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", err
	}
	return resolver.Path(storage.Options{ScenarioID: "agent-manager"}, storage.ClassData, "agent-manager.db")
}

func migrateLegacySQLite(dst string) error {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	src := filepath.Join(home, ".vrooli", "data", "sqlite", "databases", "agent-manager.db")
	if src == dst {
		return nil
	}
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if _, err := os.Stat(dst); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.DB.Close()
}

// HealthCheck performs a health check on the database.
func (db *DB) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultPingTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return &domain.DatabaseError{
			Operation:   "health_check",
			EntityType:  "Database",
			Cause:       err,
			IsTransient: true,
		}
	}
	return nil
}

// WithTransaction executes a function within a database transaction.
func (db *DB) WithTransaction(fn func(*sqlx.Tx) error) error {
	tx, err := db.Beginx()
	if err != nil {
		return &domain.DatabaseError{
			Operation:   "transaction_begin",
			EntityType:  "Database",
			Cause:       err,
			IsTransient: true,
		}
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
		return &domain.DatabaseError{
			Operation:   "transaction_commit",
			EntityType:  "Database",
			Cause:       err,
			IsTransient: true,
		}
	}
	return nil
}

// initSchema initializes the database schema.
func (db *DB) initSchema() error {
	schemaPath := filepath.Join(filepath.Dir(getCurrentFilePath()), "schema.sql")

	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		db.log.WithError(err).WithField("path", schemaPath).Error("Failed to read schema file")
		return &domain.DatabaseError{
			Operation:  "schema_read",
			EntityType: "Schema",
			Cause:      fmt.Errorf("failed to read schema file at %s: %w", schemaPath, err),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultMigrationTimeout)
	defer cancel()

	if err := db.ensureRunsTableCompatibility(ctx); err != nil {
		db.log.WithError(err).Error("Failed to prepare runs table compatibility")
		return err
	}

	if err := db.dropInvestigationPromptColumns(ctx); err != nil {
		db.log.WithError(err).Error("Failed to drop investigation prompt columns")
		return err
	}

	if err := db.ensureProfileNetworkAccessColumn(ctx); err != nil {
		db.log.WithError(err).Error("Failed to add network_access column to agent_profiles")
		return err
	}

	_, err = db.ExecContext(ctx, string(schemaBytes))
	if err != nil {
		db.log.WithError(err).Error("Failed to execute schema initialization")
		return &domain.DatabaseError{
			Operation:  "schema_init",
			EntityType: "Schema",
			Cause:      err,
		}
	}

	db.log.Info("Database schema initialized successfully")
	return nil
}

func (db *DB) ensureRunsTableCompatibility(ctx context.Context) error {
	var runsTableCount int
	if err := db.GetContext(ctx, &runsTableCount, `
		SELECT COUNT(*)
		FROM sqlite_master
		WHERE type = 'table' AND name = 'runs'
	`); err != nil {
		return &domain.DatabaseError{
			Operation:  "schema_preflight",
			EntityType: "Schema",
			Cause:      err,
		}
	}
	if runsTableCount == 0 {
		return nil
	}

	type tableColumn struct {
		Name string `db:"name"`
	}
	var columns []tableColumn
	if err := db.SelectContext(ctx, &columns, "SELECT name FROM pragma_table_info('runs')"); err != nil {
		return &domain.DatabaseError{
			Operation:  "schema_preflight",
			EntityType: "Schema",
			Cause:      err,
		}
	}

	hasColumn := make(map[string]bool, len(columns))
	for _, col := range columns {
		hasColumn[col.Name] = true
	}

	if !hasColumn["source_run_ids"] {
		if _, err := db.ExecContext(ctx, "ALTER TABLE runs ADD COLUMN source_run_ids TEXT DEFAULT '[]'"); err != nil {
			return &domain.DatabaseError{
				Operation:  "schema_preflight",
				EntityType: "Schema",
				Cause:      err,
			}
		}
	}

	if !hasColumn["source_investigation_run_id"] {
		if _, err := db.ExecContext(ctx, "ALTER TABLE runs ADD COLUMN source_investigation_run_id TEXT"); err != nil {
			return &domain.DatabaseError{
				Operation:  "schema_preflight",
				EntityType: "Schema",
				Cause:      err,
			}
		}
	}

	if !hasColumn["identity_token_hash"] {
		if _, err := db.ExecContext(ctx, "ALTER TABLE runs ADD COLUMN identity_token_hash TEXT"); err != nil {
			return &domain.DatabaseError{
				Operation:  "schema_preflight",
				EntityType: "Schema",
				Cause:      err,
			}
		}
	}

	if !hasColumn["identity_token_revoked_at"] {
		if _, err := db.ExecContext(ctx, "ALTER TABLE runs ADD COLUMN identity_token_revoked_at TEXT"); err != nil {
			return &domain.DatabaseError{
				Operation:  "schema_preflight",
				EntityType: "Schema",
				Cause:      err,
			}
		}
	}

	return nil
}

// dropInvestigationPromptColumns removes prompt_template and apply_prompt_template
// from investigation_settings if they exist (prompts now live in prompt-manager skills).
func (db *DB) dropInvestigationPromptColumns(ctx context.Context) error {
	var tableCount int
	if err := db.GetContext(ctx, &tableCount, `
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'table' AND name = 'investigation_settings'
	`); err != nil {
		return &domain.DatabaseError{
			Operation:  "migration",
			EntityType: "Schema",
			Cause:      err,
		}
	}
	if tableCount == 0 {
		return nil
	}

	type tableColumn struct {
		Name string `db:"name"`
	}
	var columns []tableColumn
	if err := db.SelectContext(ctx, &columns, "SELECT name FROM pragma_table_info('investigation_settings')"); err != nil {
		return &domain.DatabaseError{
			Operation:  "migration",
			EntityType: "Schema",
			Cause:      err,
		}
	}

	hasColumn := make(map[string]bool, len(columns))
	for _, col := range columns {
		hasColumn[col.Name] = true
	}

	for _, col := range []string{"prompt_template", "apply_prompt_template"} {
		if hasColumn[col] {
			if _, err := db.ExecContext(ctx, "ALTER TABLE investigation_settings DROP COLUMN "+col); err != nil {
				return &domain.DatabaseError{
					Operation:  "migration",
					EntityType: "Schema",
					Cause:      fmt.Errorf("drop column %s: %w", col, err),
				}
			}
		}
	}

	return nil
}

// ensureProfileNetworkAccessColumn adds the network_access column to agent_profiles
// if it doesn't already exist (migration for existing databases).
func (db *DB) ensureProfileNetworkAccessColumn(ctx context.Context) error {
	var tableCount int
	if err := db.GetContext(ctx, &tableCount, `
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'table' AND name = 'agent_profiles'
	`); err != nil {
		return &domain.DatabaseError{
			Operation:  "migration",
			EntityType: "Schema",
			Cause:      err,
		}
	}
	if tableCount == 0 {
		return nil
	}

	type tableColumn struct {
		Name string `db:"name"`
	}
	var columns []tableColumn
	if err := db.SelectContext(ctx, &columns, "SELECT name FROM pragma_table_info('agent_profiles')"); err != nil {
		return &domain.DatabaseError{
			Operation:  "migration",
			EntityType: "Schema",
			Cause:      err,
		}
	}

	hasColumn := false
	for _, col := range columns {
		if col.Name == "network_access" {
			hasColumn = true
			break
		}
	}

	if !hasColumn {
		if _, err := db.ExecContext(ctx, "ALTER TABLE agent_profiles ADD COLUMN network_access TEXT NOT NULL DEFAULT 'localhost'"); err != nil {
			return &domain.DatabaseError{
				Operation:  "migration",
				EntityType: "Schema",
				Cause:      fmt.Errorf("add column network_access: %w", err),
			}
		}
	}

	return nil
}

func getCurrentFilePath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filename
}
