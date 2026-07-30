package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/modules"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/sirupsen/logrus"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite" // SQLite driver
)

const (
	defaultQueryTimeout  = 30 * time.Second
	defaultSchemaTimeout = 60 * time.Second
	defaultPingTimeout   = 5 * time.Second
)

// DB provides Agent Manager's existing sqlx-shaped repository operations over
// api-core's RoutedDB. Every operation picks its SQLite pool from the request
// context, so Test Genie traffic cannot reach the production database.
type DB struct {
	Routed *coredb.RoutedDB
	// DB exists only for legacy unit tests that construct a local sqlx handle
	// directly. Production construction leaves it nil; request paths always use
	// Routed above.
	DB     *sqlx.DB
	log    *logrus.Logger
	mapper *reflectx.Mapper
}

// NewDB wraps an existing sqlx test handle in the same routed seam used by
// production. It is retained for focused unit tests that own their database.
func NewDB(db *sqlx.DB, log *logrus.Logger) *DB {
	if db == nil {
		return &DB{log: log, mapper: reflectx.NewMapper("db")}
	}
	return &DB{Routed: coredb.NewFromPrimary(db.DB), DB: db, log: log, mapper: reflectx.NewMapper("db")}
}

// NewConnection is retained for repository tests that exercise the historical
// standalone constructor. The server composition root uses api-core Open.
func NewConnection(log *logrus.Logger) (*DB, error) {
	dsn, err := SQLiteDSN(log)
	if err != nil {
		return nil, err
	}
	raw, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	raw.SetMaxOpenConns(1)
	db := NewDB(raw, log)
	if err := db.InitializeSchema(); err != nil {
		_ = raw.Close()
		return nil, err
	}
	return db, nil
}

// NewRoutedDB adds Agent Manager's repository helpers to a RoutedDB.
func NewRoutedDB(routed *coredb.RoutedDB, log *logrus.Logger) *DB {
	return &DB{Routed: routed, log: log, mapper: reflectx.NewMapper("db")}
}

// DataDir returns the directory where the SQLite database file is stored.
// This follows the same resolution priority as SQLiteDSN.
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

// SQLiteDSN resolves Agent Manager's canonical SQLite DSN for the API-core
// connection seam. It is intentionally exported to keep main's composition
// root explicit without exposing path-resolution internals.
func SQLiteDSN(log *logrus.Logger) (string, error) {
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

// sqliteDSN remains a package-private compatibility alias for tests that
// verify the canonical path policy directly.
func sqliteDSN(log *logrus.Logger) (string, error) { return SQLiteDSN(log) }

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
func (db *DB) Close() error {
	if db == nil {
		return nil
	}
	if db.Routed == nil && db.DB != nil {
		return db.DB.Close()
	}
	if db.Routed == nil {
		return nil
	}
	return db.Routed.Close()
}

// HealthCheck performs a health check on the database.
func (db *DB) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultPingTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return &domain.DatabaseError{Operation: "health_check", EntityType: "Database", Cause: err, IsTransient: true}
	}
	return nil
}

func (db *DB) PingContext(ctx context.Context) error {
	if db == nil {
		return fmt.Errorf("database is not configured")
	}
	if db.Routed == nil && db.DB != nil {
		return db.DB.PingContext(ctx)
	}
	if db.Routed == nil {
		return fmt.Errorf("database is not configured")
	}
	return db.Routed.PingContext(ctx)
}

func (db *DB) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	if db == nil {
		return nil, fmt.Errorf("database is not configured")
	}
	if db.Routed == nil && db.DB != nil {
		tx, err := db.DB.BeginTxx(ctx, opts)
		if err != nil {
			return nil, err
		}
		return &Tx{Tx: tx.Tx}, nil
	}
	if db.Routed == nil {
		return nil, fmt.Errorf("database is not configured")
	}
	tx, err := db.Routed.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx}, nil
}

// WithTransaction executes a function within a database transaction.
func (db *DB) WithTransaction(ctx context.Context, fn func(*Tx) error) error {
	if db == nil {
		return &domain.DatabaseError{Operation: "transaction_begin", EntityType: "Database", Cause: fmt.Errorf("database is not configured")}
	}
	tx, err := db.BeginTxx(ctx, nil)
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

// Tx is the small transaction surface Agent Manager needs. Keeping it local
// avoids leaking a raw sqlx transaction that could bypass the routed seam.
type Tx struct{ *sql.Tx }

func (tx *Tx) NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error) {
	bound, args, err := sqlx.Named(query, arg)
	if err != nil {
		return nil, err
	}
	return tx.ExecContext(ctx, sqlx.Rebind(sqlx.BindType("sqlite"), bound), args...)
}

// SQLXCompat is the compatibility seam accepted by database read-side
// components. Both *sqlx.DB (unit tests) and *DB (the routed production graph)
// satisfy it. It is owned here because it describes this adapter's SQL surface,
// not a separate product capability.
type SQLXCompat interface {
	Exec(string, ...any) (sql.Result, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryxContext(context.Context, string, ...any) (*sqlx.Rows, error)
	GetContext(context.Context, any, string, ...any) error
	SelectContext(context.Context, any, string, ...any) error
	Conn(context.Context) (*sql.Conn, error)
}

func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if db != nil && db.Routed == nil && db.DB != nil {
		return db.DB.QueryContext(ctx, query, args...)
	}
	return db.Routed.QueryContext(ctx, query, args...)
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	if db != nil && db.Routed == nil && db.DB != nil {
		return db.DB.QueryRowContext(ctx, query, args...)
	}
	return db.Routed.QueryRowContext(ctx, query, args...)
}

func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if db != nil && db.Routed == nil && db.DB != nil {
		return db.DB.ExecContext(ctx, query, args...)
	}
	return db.Routed.ExecContext(ctx, query, args...)
}

func (db *DB) Conn(ctx context.Context) (*sql.Conn, error) {
	if db != nil && db.Routed == nil && db.DB != nil {
		return db.DB.Conn(ctx)
	}
	return db.Routed.Conn(ctx)
}

func (db *DB) Exec(query string, args ...any) (sql.Result, error) {
	if db != nil && db.DB != nil {
		return db.DB.Exec(query, args...)
	}
	return nil, fmt.Errorf("non-context database access is unavailable in production")
}

func (db *DB) Get(dest any, query string, args ...any) error {
	if db != nil && db.DB != nil {
		return db.DB.Get(dest, query, args...)
	}
	return fmt.Errorf("non-context database access is unavailable in production")
}

func (db *DB) QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error) {
	rows, err := db.routedRows(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	mapper := db.mapper
	if mapper == nil {
		mapper = reflectx.NewMapper("db")
	}
	return &sqlx.Rows{Rows: rows, Mapper: mapper}, nil
}

// routedRows makes ownership explicit for the sqlx compatibility adapter: the
// caller receives the cursor and is required to close it, exactly as sqlx.DB's
// QueryxContext contract requires.
func (db *DB) routedRows(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.QueryContext(ctx, query, args...)
}

func (db *DB) NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error) {
	bound, args, err := sqlx.Named(query, arg)
	if err != nil {
		return nil, err
	}
	return db.ExecContext(ctx, sqlx.Rebind(sqlx.BindType("sqlite"), bound), args...)
}

func (db *DB) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	rows, err := db.QueryxContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	if isScannable(dest) {
		return rows.Scan(dest)
	}
	return rows.StructScan(dest)
}

func (db *DB) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	value := reflect.ValueOf(dest)
	if value.Kind() != reflect.Ptr || value.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("select destination must be a pointer to a slice, got %T", dest)
	}
	rows, err := db.QueryxContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	slice := value.Elem()
	elementType := slice.Type().Elem()
	for rows.Next() {
		isPointer := elementType.Kind() == reflect.Ptr
		valueType := elementType
		if isPointer {
			valueType = valueType.Elem()
		}
		element := reflect.New(valueType)
		if isScannable(element.Interface()) {
			err = rows.Scan(element.Interface())
		} else {
			err = rows.StructScan(element.Interface())
		}
		if err != nil {
			return err
		}
		if isPointer {
			slice = reflect.Append(slice, element)
		} else {
			slice = reflect.Append(slice, element.Elem())
		}
	}
	value.Elem().Set(slice)
	return rows.Err()
}

var sqlScannerType = reflect.TypeOf((*sql.Scanner)(nil)).Elem()

func isScannable(dest any) bool {
	typeOf := reflect.TypeOf(dest)
	if typeOf == nil {
		return true
	}
	for typeOf.Kind() == reflect.Ptr {
		typeOf = typeOf.Elem()
	}
	return typeOf.Kind() != reflect.Struct || reflect.PointerTo(typeOf).Implements(sqlScannerType)
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
	// EnsureSchemas validates declared columns before it returns. SQLite's
	// CREATE TABLE IF NOT EXISTS cannot add columns to a table that already
	// exists, so an additive migration for an existing projection table must
	// happen before that validation step. A fresh database has no columns yet;
	// the declarative schema creates it and the normal post-schema migration is
	// then a harmless no-op.
	if columns, err := db.tableColumns(ctx, "invocation_read_model_runs"); err != nil {
		return err
	} else if len(columns) > 0 {
		if err := db.migrateInvocationReadModelRunColumns(ctx); err != nil {
			return err
		}
	}
	if err := coredb.EnsureSchemas(ctx, db, modules.AllSchemas()...); err != nil {
		if db.log != nil {
			db.log.WithError(err).Error("Failed to apply domain schemas")
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
	if err := db.migrateInvocationReadModelRunColumns(ctx); err != nil {
		return err
	}
	if db.log != nil {
		db.log.Info("Database schema initialized successfully")
	}
	return nil
}

// InitializeSchema applies Agent Manager's additive SQLite schema contract to
// the primary pool. Startup calls it once; devrouting calls the same contract
// for each newly leased test pool before requests can reach it.
func (db *DB) InitializeSchema() error { return db.initSchema() }

// columnMigration is one additive column: apply ddl only when column is absent.
type columnMigration struct {
	column string
	ddl    string
}

var invocationReadModelRunColumnMigrations = []columnMigration{
	{column: "authoritative_cost_usd", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN authoritative_cost_usd REAL NOT NULL DEFAULT 0"},
	{column: "estimated_cost_usd", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN estimated_cost_usd REAL NOT NULL DEFAULT 0"},
	{column: "unknown_cost_usd", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN unknown_cost_usd REAL NOT NULL DEFAULT 0"},
	{column: "input_cost_usd", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN input_cost_usd REAL NOT NULL DEFAULT 0"},
	{column: "output_cost_usd", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN output_cost_usd REAL NOT NULL DEFAULT 0"},
	{column: "cache_read_cost_usd", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN cache_read_cost_usd REAL NOT NULL DEFAULT 0"},
	{column: "cache_creation_cost_usd", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN cache_creation_cost_usd REAL NOT NULL DEFAULT 0"},
	{column: "input_tokens", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN input_tokens INTEGER NOT NULL DEFAULT 0"},
	{column: "output_tokens", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN output_tokens INTEGER NOT NULL DEFAULT 0"},
	{column: "cache_read_tokens", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN cache_read_tokens INTEGER NOT NULL DEFAULT 0"},
	{column: "cache_creation_tokens", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN cache_creation_tokens INTEGER NOT NULL DEFAULT 0"},
}

func (db *DB) migrateInvocationReadModelRunColumns(ctx context.Context) error {
	return db.migrateColumns(ctx, "invocation_read_model_runs", invocationReadModelRunColumnMigrations)
}

// runColumnMigrations lists columns added to the runs table after its original
// CREATE TABLE shipped. Each is applied only when missing, so re-running is
// safe and existing rows are untouched (the new columns take their DEFAULT).
var runColumnMigrations = []columnMigration{
	{column: "execution_mode", ddl: "ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe'"},
	{column: "web_console_session_id", ddl: "ALTER TABLE runs ADD COLUMN web_console_session_id TEXT DEFAULT ''"},
	{column: "run_result", ddl: "ALTER TABLE runs ADD COLUMN run_result TEXT"},
	{column: "commit_hash", ddl: "ALTER TABLE runs ADD COLUMN commit_hash TEXT DEFAULT ''"},
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
