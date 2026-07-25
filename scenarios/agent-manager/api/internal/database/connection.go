package database

import (
	"context"
	"database/sql"
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

// initSchema applies the current declarative schema, then runs the additive
// column migrations for tables that gained columns after their CREATE TABLE
// statement was first shipped. The declarative schema uses CREATE TABLE IF NOT
// EXISTS, so it does not add columns to a table that already exists; the
// migration step covers that case. Both steps are idempotent: a fresh database
// gets every column from the schema and the migration is a no-op; an existing
// database keeps its data and only gains the missing columns. Data values are
// never rewritten — this honours the SQLite migrate-never-recreate rule.
func (db *DB) initSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSchemaTimeout)
	defer cancel()
	if _, err := db.ExecContext(ctx, schema); err != nil {
		if db.log != nil {
			db.log.WithError(err).Error("Failed to execute schema initialization")
		}
		return &domain.DatabaseError{Operation: "schema_init", EntityType: "Schema", Cause: err}
	}
	if err := db.migrateRunColumns(ctx); err != nil {
		if db.log != nil {
			db.log.WithError(err).Error("Failed to apply additive runs-table migrations")
		}
		return err
	}
	if err := db.migrateRunEventPayloads(ctx); err != nil {
		if db.log != nil {
			db.log.WithError(err).Error("Failed to migrate run event payloads")
		}
		return err
	}
	if err := db.migrateColumns(ctx, "agent_profiles", []columnMigration{{column: "tool_restriction_policy", ddl: "ALTER TABLE agent_profiles ADD COLUMN tool_restriction_policy TEXT NOT NULL DEFAULT 'enforced'"}, {column: "effort", ddl: "ALTER TABLE agent_profiles ADD COLUMN effort TEXT NOT NULL DEFAULT ''"}}); err != nil {
		return err
	}
	if err := db.migrateColumns(ctx, "workflow_executions", []columnMigration{{column: "parent_attempt_id", ddl: "ALTER TABLE workflow_executions ADD COLUMN parent_attempt_id TEXT"}, {column: "depth", ddl: "ALTER TABLE workflow_executions ADD COLUMN depth INTEGER NOT NULL DEFAULT 0"}}); err != nil {
		return err
	}
	if err := db.migrateColumns(ctx, "workflow_node_attempts", []columnMigration{{column: "child_execution_id", ddl: "ALTER TABLE workflow_node_attempts ADD COLUMN child_execution_id TEXT"}, {column: "raw_output", ddl: "ALTER TABLE workflow_node_attempts ADD COLUMN raw_output TEXT"}, {column: "validation_error", ddl: "ALTER TABLE workflow_node_attempts ADD COLUMN validation_error TEXT"}, {column: "experiment_id", ddl: "ALTER TABLE workflow_node_attempts ADD COLUMN experiment_id TEXT"}, {column: "variant_id", ddl: "ALTER TABLE workflow_node_attempts ADD COLUMN variant_id TEXT"}, {column: "prompt_hash", ddl: "ALTER TABLE workflow_node_attempts ADD COLUMN prompt_hash TEXT"}}); err != nil {
		return err
	}
	if db.log != nil {
		db.log.Info("Database schema initialized successfully")
	}
	return nil
}

// columnMigration is one additive column: apply ddl only when column is absent.
type columnMigration struct {
	column string
	ddl    string
}

// runColumnMigrations lists columns added to the runs table after its original
// CREATE TABLE shipped. Each is applied only when missing, so re-running is
// safe and existing rows are untouched (the new columns take their DEFAULT).
var runColumnMigrations = []columnMigration{
	{column: "execution_mode", ddl: "ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe'"},
	{column: "web_console_session_id", ddl: "ALTER TABLE runs ADD COLUMN web_console_session_id TEXT DEFAULT ''"},
	{column: "run_result", ddl: "ALTER TABLE runs ADD COLUMN run_result TEXT"},
}

// migrateRunColumns adds any missing additive columns to the runs table.
func (db *DB) migrateRunColumns(ctx context.Context) error {
	return db.migrateColumns(ctx, "runs", runColumnMigrations)
}

// migrateRunEventPayloads advances rows written by the retired tagged-union
// event model to the typed payload field names. It updates only changed JSON,
// preserves all unrelated fields, and never recreates the SQLite table.
func (db *DB) migrateRunEventPayloads(ctx context.Context) error {
	type row struct {
		id        string
		eventType domain.RunEventType
		data      []byte
	}
	rows, err := db.QueryxContext(ctx, `SELECT id, event_type, data FROM run_events`)
	if err != nil {
		return &domain.DatabaseError{Operation: "event_payload_migration_query", EntityType: "RunEvent", Cause: err}
	}
	defer rows.Close()

	var updates []row
	for rows.Next() {
		var current row
		if err := rows.Scan(&current.id, &current.eventType, &current.data); err != nil {
			return &domain.DatabaseError{Operation: "event_payload_migration_scan", EntityType: "RunEvent", Cause: err}
		}
		normalized, err := domain.NormalizeEventPayloadJSON(current.eventType, current.data)
		if err != nil {
			return &domain.DatabaseError{Operation: "event_payload_migration_decode", EntityType: "RunEvent", Cause: err}
		}
		if string(normalized) != string(current.data) {
			current.data = normalized
			updates = append(updates, current)
		}
	}
	if err := rows.Err(); err != nil {
		return &domain.DatabaseError{Operation: "event_payload_migration_iterate", EntityType: "RunEvent", Cause: err}
	}
	for _, update := range updates {
		if _, err := db.ExecContext(ctx, `UPDATE run_events SET data = ? WHERE id = ?`, update.data, update.id); err != nil {
			return &domain.DatabaseError{Operation: "event_payload_migration_update", EntityType: "RunEvent", Cause: err}
		}
	}
	if db.log != nil && len(updates) > 0 {
		db.log.WithField("rows", len(updates)).Info("Migrated legacy run event payloads")
	}
	return nil
}

func (db *DB) migrateColumns(ctx context.Context, table string, migrations []columnMigration) error {
	existing, err := db.tableColumns(ctx, table)
	if err != nil {
		return err
	}
	for _, m := range migrations {
		if _, ok := existing[m.column]; ok {
			continue
		}
		if _, err := db.ExecContext(ctx, m.ddl); err != nil {
			return &domain.DatabaseError{Operation: "schema_migrate", EntityType: "Schema", Cause: err}
		}
		if db.log != nil {
			db.log.WithFields(logrus.Fields{"table": table, "column": m.column}).Info("Applied additive table migration")
		}
	}
	return nil
}

// tableColumns returns the set of column names on a table via PRAGMA
// table_info. The table name is a trusted internal constant, never user input.
func (db *DB) tableColumns(ctx context.Context, table string) (map[string]struct{}, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return nil, &domain.DatabaseError{Operation: "schema_introspect", EntityType: "Schema", Cause: err}
	}
	defer rows.Close()

	cols := make(map[string]struct{})
	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notNull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dfltValue, &pk); err != nil {
			return nil, &domain.DatabaseError{Operation: "schema_introspect", EntityType: "Schema", Cause: err}
		}
		cols[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, &domain.DatabaseError{Operation: "schema_introspect", EntityType: "Schema", Cause: err}
	}
	return cols, nil
}
