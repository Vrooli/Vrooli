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
	root, _ := os.LookupEnv("AM_SQLITE_PATH")
	root = strings.TrimSpace(root)
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
	root, _ := os.LookupEnv("AM_SQLITE_PATH")
	root = strings.TrimSpace(root)
	if root == "" {
		custom, _ := os.LookupEnv("DATABASE_URL")
		if custom = strings.TrimSpace(custom); strings.HasPrefix(custom, "file:") {
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
// database keeps its data and only gains the missing columns. The one exception
// is task_id nullability: SQLite cannot alter a NOT NULL constraint in place,
// so that migration uses a transactionally verified table replacement that
// preserves every column, index, and trigger before schema validation.
func (db *DB) initSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSchemaTimeout)
	defer cancel()
	// EnsureSchemas validates declared columns before it returns. SQLite's
	// CREATE TABLE IF NOT EXISTS cannot add columns to a table that already
	// exists, so an additive migration for an existing projection table must
	// happen before that validation step. A fresh database has no columns yet;
	// the declarative schema creates it and the normal post-schema migration is
	// then a harmless no-op.
	// Runs gained import-provenance columns after the declarative table shipped.
	// They must land before EnsureSchemas validates the existing table; otherwise
	// SQLite's CREATE TABLE IF NOT EXISTS would make startup fail before the
	// normal post-schema migration gets a chance to repair it.
	if columns, err := db.tableColumns(ctx, "runs"); err != nil {
		return err
	} else if len(columns) > 0 {
		if err := db.migrateRunTaskIDNullability(ctx); err != nil {
			return err
		}
		if err := db.migrateRunColumns(ctx); err != nil {
			return err
		}
	}
	if columns, err := db.tableColumns(ctx, "invocation_read_model_runs"); err != nil {
		return err
	} else if len(columns) > 0 {
		if err := db.migrateInvocationReadModelRunColumns(ctx); err != nil {
			return err
		}
	}
	if columns, err := db.tableColumns(ctx, "model_pricing"); err != nil {
		return err
	} else if len(columns) > 0 {
		if err := db.migrateColumns(ctx, "model_pricing", []columnMigration{{column: "price_book_revision_id", ddl: "ALTER TABLE model_pricing ADD COLUMN price_book_revision_id TEXT"}}); err != nil {
			return err
		}
	}
	if columns, err := db.tableColumns(ctx, "invocation_read_model_facts"); err != nil {
		return err
	} else if len(columns) > 0 {
		if err := db.migrateInvocationReadModelFactColumns(ctx); err != nil {
			return err
		}
		if err := db.migrateInvocationReadModelTimeAccountingColumns(ctx); err != nil {
			return err
		}
	}
	if columns, err := db.tableColumns(ctx, "invocation_read_model_watermarks"); err != nil {
		return err
	} else if len(columns) > 0 {
		if err := db.migrateInvocationReadModelWatermarkColumns(ctx); err != nil {
			return err
		}
	}
	if columns, err := db.tableColumns(ctx, "invocation_read_model_episodes"); err != nil {
		return err
	} else if len(columns) > 0 {
		if err := db.migrateInvocationReadModelEpisodeColumns(ctx); err != nil {
			return err
		}
		if err := db.migrateInvocationReadModelSelfReportColumns(ctx); err != nil {
			return err
		}
	}
	// investigation_cross_scenario_calls gained receipt timing after the
	// original projection shipped. Apply this before schema validation for
	// existing databases, just as we do for the durable invocation projection.
	if columns, err := db.tableColumns(ctx, "investigation_cross_scenario_calls"); err != nil {
		return err
	} else if len(columns) > 0 {
		if err := db.migrateInvestigationProjectionColumns(ctx); err != nil {
			return err
		}
	}
	if columns, err := db.tableColumns(ctx, "investigation_friction_episodes"); err != nil {
		return err
	} else if len(columns) > 0 {
		if err := db.migrateFrictionEpisodeColumns(ctx); err != nil {
			return err
		}
	}
	// Findings gained effectiveness-loop columns after the initial table
	// declaration shipped. As with runs, existing SQLite tables must receive
	// these additive columns before EnsureSchemas performs its drift check.
	if columns, err := db.tableColumns(ctx, "run_findings"); err != nil {
		return err
	} else if len(columns) > 0 {
		if err := db.migrateRunFindingColumns(ctx); err != nil {
			return err
		}
	}
	if err := coredb.EnsureSchemas(ctx, db, modules.AllSchemas()...); err != nil {
		if db.log != nil {
			db.log.WithError(err).Error("Failed to apply domain schemas")
		}
		return &domain.DatabaseError{Operation: "schema_init", EntityType: "Schema", Cause: err}
	}
	if err := db.consolidateLegacyInvestigationProjections(ctx); err != nil {
		return err
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
	if err := db.migrateColumns(ctx, "agent_profiles", []columnMigration{{column: "tool_restriction_policy", ddl: "ALTER TABLE agent_profiles ADD COLUMN tool_restriction_policy TEXT NOT NULL DEFAULT 'enforced'"}, {column: "effort", ddl: "ALTER TABLE agent_profiles ADD COLUMN effort TEXT NOT NULL DEFAULT ''"}, {column: "declared_scopes", ddl: "ALTER TABLE agent_profiles ADD COLUMN declared_scopes TEXT DEFAULT '[]'"}}); err != nil {
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
	if err := db.removeRetiredInvocationReadModelColumns(ctx); err != nil {
		return err
	}
	if err := db.migrateRunFindingColumns(ctx); err != nil {
		return err
	}
	if err := db.migrateInvocationReadModelFactColumns(ctx); err != nil {
		return err
	}
	if err := db.migrateInvocationReadModelTimeAccountingColumns(ctx); err != nil {
		return err
	}
	if err := db.migrateInvocationReadModelEpisodeColumns(ctx); err != nil {
		return err
	}
	if err := db.migrateInvocationReadModelSelfReportColumns(ctx); err != nil {
		return err
	}
	if err := db.migrateInvestigationProjectionColumns(ctx); err != nil {
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
	{column: "goal_id", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN goal_id TEXT NOT NULL DEFAULT ''"},
	{column: "goal_status", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN goal_status TEXT NOT NULL DEFAULT ''"},
	{column: "input_cost_usd", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN input_cost_usd REAL NOT NULL DEFAULT 0"},
	{column: "output_cost_usd", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN output_cost_usd REAL NOT NULL DEFAULT 0"},
	{column: "cache_read_cost_usd", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN cache_read_cost_usd REAL NOT NULL DEFAULT 0"},
	{column: "cache_creation_cost_usd", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN cache_creation_cost_usd REAL NOT NULL DEFAULT 0"},
	{column: "input_tokens", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN input_tokens INTEGER NOT NULL DEFAULT 0"},
	{column: "output_tokens", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN output_tokens INTEGER NOT NULL DEFAULT 0"},
	{column: "cache_read_tokens", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN cache_read_tokens INTEGER NOT NULL DEFAULT 0"},
	{column: "cache_creation_tokens", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN cache_creation_tokens INTEGER NOT NULL DEFAULT 0"},
	{column: "turns", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN turns INTEGER NOT NULL DEFAULT 0"},
	{column: "tool_calls", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN tool_calls INTEGER NOT NULL DEFAULT 0"},
	{column: "total_charge_micro_usd", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN total_charge_micro_usd INTEGER NOT NULL DEFAULT 0"},
	{column: "metered_charge_micro_usd", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN metered_charge_micro_usd INTEGER NOT NULL DEFAULT 0"},
	{column: "unpriced_token_count", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN unpriced_token_count INTEGER NOT NULL DEFAULT 0"},
	{column: "workload_kind", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN workload_kind TEXT NOT NULL DEFAULT 'adhoc'"},
	{column: "workload_key", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN workload_key TEXT NOT NULL DEFAULT ''"},
	{column: "workload_instance", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN workload_instance TEXT NOT NULL DEFAULT ''"},
	{column: "time_basis", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN time_basis TEXT NOT NULL DEFAULT 'ingestion'"},
	{column: "preamble_injected_tokens", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN preamble_injected_tokens INTEGER NOT NULL DEFAULT 0"},
	{column: "preamble_fixed_tokens", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN preamble_fixed_tokens INTEGER NOT NULL DEFAULT 0"},
	{column: "preamble_token_basis", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN preamble_token_basis TEXT NOT NULL DEFAULT 'unknown'"},
	{column: "unattributed_tokens", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN unattributed_tokens INTEGER NOT NULL DEFAULT 0"},
	{column: "unattributed_reason", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN unattributed_reason TEXT NOT NULL DEFAULT ''"},
}

var runFindingColumnMigrations = []columnMigration{
	{column: "target_measure", ddl: "ALTER TABLE run_findings ADD COLUMN target_measure TEXT NOT NULL DEFAULT 'finding-recurrence-rate'"},
	{column: "before_value", ddl: "ALTER TABLE run_findings ADD COLUMN before_value REAL"},
	{column: "after_value", ddl: "ALTER TABLE run_findings ADD COLUMN after_value REAL"},
	{column: "effectiveness", ddl: "ALTER TABLE run_findings ADD COLUMN effectiveness TEXT NOT NULL DEFAULT 'not_yet_measurable'"},
	{column: "friction_topic", ddl: "ALTER TABLE run_findings ADD COLUMN friction_topic TEXT NOT NULL DEFAULT ''"},
	{column: "cites_resolved_commands", ddl: "ALTER TABLE run_findings ADD COLUMN cites_resolved_commands INTEGER NOT NULL DEFAULT 0"},
	{column: "cites_real_outcome", ddl: "ALTER TABLE run_findings ADD COLUMN cites_real_outcome INTEGER NOT NULL DEFAULT 0"},
	{column: "cites_attributed_owner", ddl: "ALTER TABLE run_findings ADD COLUMN cites_attributed_owner INTEGER NOT NULL DEFAULT 0"},
	{column: "quality_signal", ddl: "ALTER TABLE run_findings ADD COLUMN quality_signal TEXT NOT NULL DEFAULT 'unavailable'"},
}

func (db *DB) migrateRunFindingColumns(ctx context.Context) error {
	return db.migrateColumns(ctx, "run_findings", runFindingColumnMigrations)
}

func (db *DB) migrateInvocationReadModelRunColumns(ctx context.Context) error {
	return db.migrateColumns(ctx, "invocation_read_model_runs", invocationReadModelRunColumnMigrations)
}

// removeRetiredInvocationReadModelColumns performs the one destructive part
// of the usage/charge cutover. SQLite cannot drop columns directly on all
// supported versions, so rebuild the narrow read-model table while copying
// every retained analytical fact. The operation is idempotent and only runs
// when one of the retired columns is still present.
func (db *DB) removeRetiredInvocationReadModelColumns(ctx context.Context) error {
	columns, err := db.tableColumns(ctx, "invocation_read_model_runs")
	if err != nil {
		return err
	}
	retired := []string{"authoritative_cost_usd", "estimated_cost_usd", "unknown_cost_usd", "goal_token_budget", "goal_tokens_used", "goal_time_used_seconds"}
	needsRebuild := false
	for _, column := range retired {
		if _, ok := columns[column]; ok {
			needsRebuild = true
			break
		}
	}
	if !needsRebuild {
		return nil
	}
	statements := []string{
		"DROP INDEX IF EXISTS idx_invocation_read_model_runs_occurred_at",
		"DROP INDEX IF EXISTS idx_invocation_read_model_runs_status",
		"DROP INDEX IF EXISTS idx_invocation_read_model_runs_profile",
		"DROP INDEX IF EXISTS idx_invocation_read_model_runs_runner",
		"DROP INDEX IF EXISTS idx_invocation_read_model_runs_model",
		"ALTER TABLE invocation_read_model_runs RENAME TO invocation_read_model_runs_retired",
		`CREATE TABLE invocation_read_model_runs (
            run_id TEXT PRIMARY KEY,
            goal_id TEXT NOT NULL DEFAULT '',
            goal_status TEXT NOT NULL DEFAULT '',
            occurred_at TEXT NOT NULL,
            created_at TEXT NOT NULL,
            started_at TEXT,
            ended_at TEXT,
            duration_ms INTEGER NOT NULL DEFAULT 0,
            status TEXT NOT NULL,
            profile_id TEXT NOT NULL DEFAULT 'unknown',
            runner_type TEXT NOT NULL DEFAULT 'unknown',
            model TEXT NOT NULL DEFAULT 'unknown',
            tag TEXT NOT NULL DEFAULT 'unknown',
            workload_kind TEXT NOT NULL DEFAULT 'adhoc',
            workload_key TEXT NOT NULL DEFAULT '',
            workload_instance TEXT NOT NULL DEFAULT '',
            total_cost_usd REAL NOT NULL DEFAULT 0,
            input_cost_usd REAL NOT NULL DEFAULT 0,
            output_cost_usd REAL NOT NULL DEFAULT 0,
            cache_read_cost_usd REAL NOT NULL DEFAULT 0,
            cache_creation_cost_usd REAL NOT NULL DEFAULT 0,
            total_tokens INTEGER NOT NULL DEFAULT 0,
            input_tokens INTEGER NOT NULL DEFAULT 0,
            output_tokens INTEGER NOT NULL DEFAULT 0,
            cache_read_tokens INTEGER NOT NULL DEFAULT 0,
            cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
            turns INTEGER NOT NULL DEFAULT 0,
            tool_calls INTEGER NOT NULL DEFAULT 0,
            total_charge_micro_usd INTEGER NOT NULL DEFAULT 0,
            metered_charge_micro_usd INTEGER NOT NULL DEFAULT 0,
            unpriced_token_count INTEGER NOT NULL DEFAULT 0,
            preamble_injected_tokens INTEGER NOT NULL DEFAULT 0,
            preamble_fixed_tokens INTEGER NOT NULL DEFAULT 0,
            preamble_token_basis TEXT NOT NULL DEFAULT 'unknown',
            cost_time_basis TEXT NOT NULL DEFAULT 'terminal_projection',
            time_basis TEXT NOT NULL DEFAULT 'ingestion',
            projected_at TEXT NOT NULL DEFAULT (datetime('now'))
        )`,
		`INSERT INTO invocation_read_model_runs (run_id,goal_id,goal_status,occurred_at,created_at,started_at,ended_at,duration_ms,status,profile_id,runner_type,model,tag,workload_kind,workload_key,workload_instance,total_cost_usd,input_cost_usd,output_cost_usd,cache_read_cost_usd,cache_creation_cost_usd,total_tokens,input_tokens,output_tokens,cache_read_tokens,cache_creation_tokens,turns,tool_calls,total_charge_micro_usd,metered_charge_micro_usd,unpriced_token_count,cost_time_basis,time_basis,projected_at)
         SELECT run_id,goal_id,goal_status,occurred_at,created_at,started_at,ended_at,duration_ms,status,profile_id,runner_type,model,tag,workload_kind,workload_key,workload_instance,total_cost_usd,input_cost_usd,output_cost_usd,cache_read_cost_usd,cache_creation_cost_usd,total_tokens,input_tokens,output_tokens,cache_read_tokens,cache_creation_tokens,turns,tool_calls,total_charge_micro_usd,metered_charge_micro_usd,unpriced_token_count,cost_time_basis,time_basis,projected_at
         FROM invocation_read_model_runs_retired`,
		"DROP TABLE invocation_read_model_runs_retired",
		"CREATE INDEX IF NOT EXISTS idx_invocation_read_model_runs_occurred_at ON invocation_read_model_runs(occurred_at)",
		"CREATE INDEX IF NOT EXISTS idx_invocation_read_model_runs_status ON invocation_read_model_runs(status, occurred_at)",
		"CREATE INDEX IF NOT EXISTS idx_invocation_read_model_runs_profile ON invocation_read_model_runs(profile_id, occurred_at)",
		"CREATE INDEX IF NOT EXISTS idx_invocation_read_model_runs_runner ON invocation_read_model_runs(runner_type, occurred_at)",
		"CREATE INDEX IF NOT EXISTS idx_invocation_read_model_runs_model ON invocation_read_model_runs(model, occurred_at)",
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return &domain.DatabaseError{Operation: "schema_drop_retired_columns", EntityType: "Schema", Cause: err}
		}
	}
	if db.log != nil {
		db.log.Info("Removed retired invocation read-model cost and goal columns")
	}
	return nil
}

var invocationReadModelFactColumnMigrations = []columnMigration{
	{column: "ownership_reason", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN ownership_reason TEXT NOT NULL DEFAULT ''"},
	{column: "segment_index", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN segment_index INTEGER NOT NULL DEFAULT 0"},
	{column: "segment_count", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN segment_count INTEGER NOT NULL DEFAULT 1"},
	{column: "capability", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN capability TEXT NOT NULL DEFAULT 'other'"},
	{column: "intent_class", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN intent_class TEXT NOT NULL DEFAULT ''"},
	{column: "pairing_basis", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN pairing_basis TEXT NOT NULL DEFAULT 'unpaired'"},
	{column: "failure_signature", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN failure_signature TEXT NOT NULL DEFAULT ''"},
	{column: "signature_truncated", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN signature_truncated INTEGER NOT NULL DEFAULT 0"},
	{column: "semantics_kind", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN semantics_kind TEXT NOT NULL DEFAULT ''"},
	{column: "semantics_verdict", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN semantics_verdict TEXT NOT NULL DEFAULT ''"},
	{column: "wrapper", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN wrapper TEXT NOT NULL DEFAULT ''"},
	{column: "exit_code", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN exit_code INTEGER"},
	{column: "duration_ms", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN duration_ms INTEGER"},
	{column: "arg_tokens", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN arg_tokens INTEGER NOT NULL DEFAULT 0"},
	{column: "result_tokens", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN result_tokens INTEGER NOT NULL DEFAULT 0"},
	{column: "token_basis", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN token_basis TEXT NOT NULL DEFAULT 'unknown'"},
	{column: "residency_turns", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN residency_turns INTEGER NOT NULL DEFAULT 0"},
	{column: "residency_segment", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN residency_segment INTEGER NOT NULL DEFAULT 0"},
	{column: "incurred_input_tokens", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN incurred_input_tokens INTEGER NOT NULL DEFAULT 0"},
	{column: "incurred_output_tokens", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN incurred_output_tokens INTEGER NOT NULL DEFAULT 0"},
	{column: "incurred_cache_read_tokens", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN incurred_cache_read_tokens INTEGER NOT NULL DEFAULT 0"},
	{column: "incurred_cache_creation_tokens", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN incurred_cache_creation_tokens INTEGER NOT NULL DEFAULT 0"},
}

var invocationReadModelTimeAccountingColumnMigrations = []columnMigration{
	{column: "unattributable_reason", ddl: "ALTER TABLE invocation_read_model_time_accounting ADD COLUMN unattributable_reason TEXT NOT NULL DEFAULT ''"},
}

func (db *DB) migrateInvocationReadModelFactColumns(ctx context.Context) error {
	columns, err := db.tableColumns(ctx, "invocation_read_model_facts")
	if err != nil {
		return err
	}
	// Compound calls require a composite identity. An additive column cannot
	// alter SQLite's old (run_id, call_event_id) primary key, so perform one
	// narrow, replay-safe rebuild when upgrading an older projection.
	if len(columns) > 0 {
		if _, ok := columns["segment_index"]; !ok {
			statements := []string{
				"DROP INDEX IF EXISTS idx_invocation_read_model_occurred_at",
				"DROP INDEX IF EXISTS idx_invocation_read_model_ownership",
				"DROP INDEX IF EXISTS idx_invocation_read_model_outcome",
				"DROP INDEX IF EXISTS idx_invocation_read_model_executable",
				"DROP INDEX IF EXISTS idx_invocation_read_model_fingerprint",
				"ALTER TABLE invocation_read_model_facts RENAME TO invocation_read_model_facts_retired",
				`CREATE TABLE invocation_read_model_facts (
                    run_id TEXT NOT NULL, call_event_id TEXT NOT NULL, result_event_id TEXT,
                    tool_call_id TEXT, occurred_at TEXT NOT NULL, time_basis TEXT NOT NULL DEFAULT 'event',
                    tool_name TEXT NOT NULL, capability TEXT NOT NULL DEFAULT 'other', intent_class TEXT NOT NULL DEFAULT '', executable TEXT, command_path TEXT, ownership TEXT NOT NULL,
                    ownership_reason TEXT NOT NULL DEFAULT '', segment_index INTEGER NOT NULL DEFAULT 0,
                    segment_count INTEGER NOT NULL DEFAULT 1, catalog_snapshot TEXT, outcome TEXT NOT NULL,
                    pairing_basis TEXT NOT NULL DEFAULT 'unpaired', failure_signature TEXT NOT NULL DEFAULT '',
                    signature_truncated INTEGER NOT NULL DEFAULT 0, retry_of_call_event_id TEXT,
                    help_recovery INTEGER NOT NULL DEFAULT 0, fingerprint TEXT NOT NULL, availability TEXT NOT NULL,
                    classifier_version TEXT NOT NULL, profile_id TEXT, runner_type TEXT NOT NULL DEFAULT 'unknown',
                    model TEXT NOT NULL DEFAULT '', tag TEXT NOT NULL DEFAULT '', run_status TEXT NOT NULL,
                    wrapper TEXT NOT NULL DEFAULT '', exit_code INTEGER, duration_ms INTEGER,
                    arg_tokens INTEGER NOT NULL DEFAULT 0, result_tokens INTEGER NOT NULL DEFAULT 0,
                    token_basis TEXT NOT NULL DEFAULT 'unknown', residency_turns INTEGER NOT NULL DEFAULT 0,
                    residency_segment INTEGER NOT NULL DEFAULT 0,
                    PRIMARY KEY (run_id, call_event_id, segment_index)
                )`,
				`INSERT INTO invocation_read_model_facts (run_id,call_event_id,result_event_id,tool_call_id,occurred_at,time_basis,tool_name,capability,intent_class,executable,command_path,ownership,ownership_reason,segment_index,segment_count,catalog_snapshot,outcome,pairing_basis,failure_signature,signature_truncated,retry_of_call_event_id,help_recovery,fingerprint,availability,classifier_version,profile_id,runner_type,model,tag,run_status)
                 SELECT run_id,call_event_id,result_event_id,tool_call_id,occurred_at,time_basis,tool_name,'other','',executable,command_path,
                    CASE ownership WHEN 'project' THEN 'resolved' WHEN 'unknown' THEN 'unparseable' ELSE ownership END,
                    CASE ownership WHEN 'unknown' THEN 'legacy ownership was not classified' ELSE '' END,
                    0,1,catalog_snapshot,outcome,COALESCE(pairing_basis,'unpaired'),COALESCE(failure_signature,''),COALESCE(signature_truncated,0),
                    COALESCE(retry_of_call_event_id,''),COALESCE(help_recovery,0),fingerprint,availability,classifier_version,profile_id,runner_type,model,tag,run_status,'',NULL,NULL
                 FROM invocation_read_model_facts_retired`,
				"DROP TABLE invocation_read_model_facts_retired",
				"CREATE INDEX IF NOT EXISTS idx_invocation_read_model_occurred_at ON invocation_read_model_facts(occurred_at)",
				"CREATE INDEX IF NOT EXISTS idx_invocation_read_model_ownership ON invocation_read_model_facts(ownership, occurred_at)",
				"CREATE INDEX IF NOT EXISTS idx_invocation_read_model_outcome ON invocation_read_model_facts(outcome, occurred_at)",
				"CREATE INDEX IF NOT EXISTS idx_invocation_read_model_executable ON invocation_read_model_facts(executable, occurred_at)",
				"CREATE INDEX IF NOT EXISTS idx_invocation_read_model_fingerprint ON invocation_read_model_facts(fingerprint, occurred_at)",
			}
			for _, statement := range statements {
				if _, err := db.ExecContext(ctx, statement); err != nil {
					return &domain.DatabaseError{Operation: "schema_invocation_fact_rebuild", EntityType: "Schema", Cause: err}
				}
			}
			return nil
		}
	}
	return db.migrateColumns(ctx, "invocation_read_model_facts", invocationReadModelFactColumnMigrations)
}

func (db *DB) migrateInvocationReadModelTimeAccountingColumns(ctx context.Context) error {
	return db.migrateColumns(ctx, "invocation_read_model_time_accounting", invocationReadModelTimeAccountingColumnMigrations)
}

var invocationReadModelWatermarkColumnMigrations = []columnMigration{
	{column: "episode_classifier_version", ddl: "ALTER TABLE invocation_read_model_watermarks ADD COLUMN episode_classifier_version TEXT NOT NULL DEFAULT ''"},
	{column: "self_report_classifier_version", ddl: "ALTER TABLE invocation_read_model_watermarks ADD COLUMN self_report_classifier_version TEXT NOT NULL DEFAULT ''"},
	// Existing watermarks were committed atomically with their derived facts
	// before this marker existed, so they are safe to retain as complete.
	{column: "projection_complete", ddl: "ALTER TABLE invocation_read_model_watermarks ADD COLUMN projection_complete INTEGER NOT NULL DEFAULT 1"},
}

func (db *DB) migrateInvocationReadModelWatermarkColumns(ctx context.Context) error {
	return db.migrateColumns(ctx, "invocation_read_model_watermarks", invocationReadModelWatermarkColumnMigrations)
}

var invocationReadModelEpisodeColumnMigrations = []columnMigration{
	{column: "cycle_count", ddl: "ALTER TABLE invocation_read_model_episodes ADD COLUMN cycle_count INTEGER NOT NULL DEFAULT 0"},
	{column: "repeated_element", ddl: "ALTER TABLE invocation_read_model_episodes ADD COLUMN repeated_element TEXT NOT NULL DEFAULT ''"},
}

func (db *DB) migrateInvocationReadModelEpisodeColumns(ctx context.Context) error {
	return db.migrateColumns(ctx, "invocation_read_model_episodes", invocationReadModelEpisodeColumnMigrations)
}

var invocationReadModelSelfReportColumnMigrations = []columnMigration{
	{column: "role", ddl: "ALTER TABLE invocation_read_model_self_report_spans ADD COLUMN role TEXT NOT NULL DEFAULT 'assistant'"},
	{column: "operator_correction", ddl: "ALTER TABLE invocation_read_model_self_report_spans ADD COLUMN operator_correction INTEGER NOT NULL DEFAULT 0"},
	{column: "span_capped", ddl: "ALTER TABLE invocation_read_model_self_report_spans ADD COLUMN span_capped INTEGER NOT NULL DEFAULT 0"},
}

func (db *DB) migrateInvocationReadModelSelfReportColumns(ctx context.Context) error {
	return db.migrateColumns(ctx, "invocation_read_model_self_report_spans", invocationReadModelSelfReportColumnMigrations)
}

var investigationProjectionColumnMigrations = []columnMigration{
	{column: "occurred_at", ddl: "ALTER TABLE investigation_cross_scenario_calls ADD COLUMN occurred_at TEXT"},
}

func (db *DB) migrateInvestigationProjectionColumns(ctx context.Context) error {
	if err := db.migrateColumns(ctx, "investigation_cross_scenario_calls", investigationProjectionColumnMigrations); err != nil {
		return err
	}
	// Older report builds used an empty sentinel row to preserve availability.
	// Availability now lives in the report state, while the ledger is strictly
	// receipt evidence; remove those non-evidence rows during normal startup.
	_, err := db.ExecContext(ctx, `DELETE FROM investigation_cross_scenario_calls WHERE target_scenario = '' OR operation = ''`)
	return err
}

var frictionEpisodeColumnMigrations = []columnMigration{
	{column: "failed_joined_calls", ddl: "ALTER TABLE investigation_friction_episodes ADD COLUMN failed_joined_calls INTEGER NOT NULL DEFAULT 0"},
}

func (db *DB) migrateFrictionEpisodeColumns(ctx context.Context) error {
	return db.migrateColumns(ctx, "investigation_friction_episodes", frictionEpisodeColumnMigrations)
}

// consolidateLegacyInvestigationProjections performs the one-way cutover from
// the superseded investigation cache. It copies every bounded row into the
// durable read model before dropping the old tables; SQLite never recreates
// them because they are absent from the declarative schema.
func (db *DB) consolidateLegacyInvestigationProjections(ctx context.Context) error {
	legacyFacts, err := db.tableColumns(ctx, "investigation_invocation_facts")
	if err != nil {
		return err
	}
	legacyEpisodes, err := db.tableColumns(ctx, "investigation_friction_episodes")
	if err != nil {
		return err
	}
	legacySpans, err := db.tableColumns(ctx, "investigation_self_report_spans")
	if err != nil {
		return err
	}
	if len(legacyFacts) == 0 && len(legacyEpisodes) == 0 && len(legacySpans) == 0 {
		return nil
	}
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if len(legacyFacts) > 0 {
		if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO invocation_read_model_facts (run_id,call_event_id,result_event_id,tool_call_id,occurred_at,time_basis,tool_name,executable,command_path,ownership,catalog_snapshot,outcome,pairing_basis,failure_signature,signature_truncated,retry_of_call_event_id,help_recovery,fingerprint,availability,classifier_version,profile_id,runner_type,model,tag,run_status)
			SELECT legacy.run_id, legacy.call_event_id, legacy.result_event_id, legacy.tool_call_id, COALESCE((SELECT e.timestamp FROM run_events e WHERE e.run_id=legacy.run_id AND e.id=legacy.call_event_id LIMIT 1), runs.ended_at, runs.created_at), CASE WHEN EXISTS (SELECT 1 FROM run_events e WHERE e.run_id=legacy.run_id AND e.id=legacy.call_event_id) THEN 'call_event' ELSE 'legacy_run_time' END, legacy.tool_name, legacy.executable, legacy.command_path, legacy.ownership, legacy.catalog_snapshot, legacy.outcome, 'legacy', '', 0, legacy.retry_of_call_event_id, legacy.help_recovery, legacy.fingerprint, legacy.availability, legacy.classifier_version, COALESCE(runs.agent_profile_id,'unknown'), COALESCE(json_extract(runs.resolved_config,'$.runnerType'),'unknown'), COALESCE(NULLIF(runs.actual_model,''),NULLIF(runs.requested_model,''),json_extract(runs.resolved_config,'$.model'),'unknown'), COALESCE(NULLIF(runs.tag,''),'unknown'), COALESCE(runs.status,'unknown') FROM investigation_invocation_facts legacy JOIN runs ON runs.id=legacy.run_id`); err != nil {
			return fmt.Errorf("migrate legacy invocation facts: %w", err)
		}
	}
	if len(legacyEpisodes) > 0 {
		if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO invocation_read_model_episodes (run_id,episode_id,occurred_at,time_basis,classifier_version,pattern,cause_scope,severity,honesty_flags_json,start_event_id,end_event_id,evidence_event_ids_json,turns,tokens,wall_clock_ms,suspected_owner_scenario,suspected_owner_command,owner_confidence,failed_joined_calls,fingerprint,profile_id,runner_type,model,tag,run_status)
			SELECT legacy.run_id, legacy.episode_id, COALESCE((SELECT e.timestamp FROM run_events e WHERE e.run_id=legacy.run_id AND e.id=legacy.start_event_id LIMIT 1), runs.ended_at, runs.created_at), 'legacy_run_time', legacy.classifier_version, legacy.pattern, legacy.cause_scope, legacy.severity, legacy.honesty_flags, legacy.start_event_id, legacy.end_event_id, legacy.evidence_event_ids, legacy.turns, legacy.tokens, legacy.wall_clock_ms, legacy.suspected_owner_scenario, legacy.suspected_owner_command, legacy.owner_confidence, legacy.failed_joined_calls, legacy.fingerprint, COALESCE(runs.agent_profile_id,'unknown'), COALESCE(json_extract(runs.resolved_config,'$.runnerType'),'unknown'), COALESCE(NULLIF(runs.actual_model,''),NULLIF(runs.requested_model,''),json_extract(runs.resolved_config,'$.model'),'unknown'), COALESCE(NULLIF(runs.tag,''),'unknown'), COALESCE(runs.status,'unknown') FROM investigation_friction_episodes legacy JOIN runs ON runs.id=legacy.run_id`); err != nil {
			return fmt.Errorf("migrate legacy episodes: %w", err)
		}
	}
	if len(legacySpans) > 0 {
		if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO invocation_read_model_self_report_spans (run_id,event_id,rule_id,start_offset,end_offset,occurred_at,time_basis,classifier_version,cause_scope,text,profile_id,runner_type,model,tag,run_status)
			SELECT legacy.run_id, legacy.event_id, legacy.rule_id, legacy.start_offset, legacy.end_offset, COALESCE((SELECT e.timestamp FROM run_events e WHERE e.run_id=legacy.run_id AND e.id=legacy.event_id LIMIT 1), runs.ended_at, runs.created_at), 'legacy_run_time', legacy.classifier_version, legacy.cause_scope, legacy.text, COALESCE(runs.agent_profile_id,'unknown'), COALESCE(json_extract(runs.resolved_config,'$.runnerType'),'unknown'), COALESCE(NULLIF(runs.actual_model,''),NULLIF(runs.requested_model,''),json_extract(runs.resolved_config,'$.model'),'unknown'), COALESCE(NULLIF(runs.tag,''),'unknown'), COALESCE(runs.status,'unknown') FROM investigation_self_report_spans legacy JOIN runs ON runs.id=legacy.run_id`); err != nil {
			return fmt.Errorf("migrate legacy self-report spans: %w", err)
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO invocation_read_model_watermarks (run_id,last_event_id,last_event_at,classifier_version,episode_classifier_version,self_report_classifier_version,projection_complete,projected_at)
		SELECT legacy.run_id,
		COALESCE((SELECT f.call_event_id FROM invocation_read_model_facts f WHERE f.run_id=legacy.run_id ORDER BY f.occurred_at DESC, f.call_event_id DESC LIMIT 1), ''),
		COALESCE((SELECT MAX(f.occurred_at) FROM invocation_read_model_facts f WHERE f.run_id=legacy.run_id), runs.ended_at, runs.created_at, datetime('now')),
		COALESCE((SELECT f.classifier_version FROM invocation_read_model_facts f WHERE f.run_id=legacy.run_id ORDER BY f.occurred_at DESC, f.call_event_id DESC LIMIT 1), 'legacy'),
		COALESCE((SELECT e.classifier_version FROM invocation_read_model_episodes e WHERE e.run_id=legacy.run_id ORDER BY e.occurred_at DESC, e.episode_id DESC LIMIT 1), 'legacy'),
		COALESCE((SELECT s.classifier_version FROM invocation_read_model_self_report_spans s WHERE s.run_id=legacy.run_id ORDER BY s.occurred_at DESC, s.event_id DESC LIMIT 1), 'legacy'), 1,
		COALESCE((SELECT MAX(f.occurred_at) FROM invocation_read_model_facts f WHERE f.run_id=legacy.run_id), runs.ended_at, runs.created_at, datetime('now'))
		FROM (SELECT run_id FROM investigation_invocation_facts UNION SELECT run_id FROM investigation_friction_episodes UNION SELECT run_id FROM investigation_self_report_spans) legacy
		JOIN runs ON runs.id=legacy.run_id`); err != nil {
		return fmt.Errorf("migrate legacy projection watermarks: %w", err)
	}
	for _, table := range []string{"investigation_invocation_facts", "investigation_friction_episodes", "investigation_self_report_spans"} {
		if _, err = tx.ExecContext(ctx, "DROP TABLE IF EXISTS "+table); err != nil {
			return fmt.Errorf("drop legacy projection %s: %w", table, err)
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// runColumnMigrations lists columns added to the runs table after its original
// CREATE TABLE shipped. Each is applied only when missing, so re-running is
// safe and existing rows are untouched (the new columns take their DEFAULT).
var runColumnMigrations = []columnMigration{
	{column: "label", ddl: "ALTER TABLE runs ADD COLUMN label TEXT NOT NULL DEFAULT ''"},
	{column: "label_source", ddl: "ALTER TABLE runs ADD COLUMN label_source TEXT NOT NULL DEFAULT ''"},
	{column: "subject", ddl: "ALTER TABLE runs ADD COLUMN subject TEXT NOT NULL DEFAULT '[]'"},
	{column: "owner_subject", ddl: "ALTER TABLE runs ADD COLUMN owner_subject TEXT NOT NULL DEFAULT ''"},
	{column: "owner_scopes", ddl: "ALTER TABLE runs ADD COLUMN owner_scopes TEXT NOT NULL DEFAULT '[]'"},
	{column: "requested_scopes", ddl: "ALTER TABLE runs ADD COLUMN requested_scopes TEXT NOT NULL DEFAULT '[]'"},
	{column: "execution_mode", ddl: "ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe'"},
	{column: "harness_kind", ddl: "ALTER TABLE runs ADD COLUMN harness_kind TEXT DEFAULT ''"},
	{column: "harness_session_id", ddl: "ALTER TABLE runs ADD COLUMN harness_session_id TEXT DEFAULT ''"},
	{column: "web_console_session_id", ddl: "ALTER TABLE runs ADD COLUMN web_console_session_id TEXT DEFAULT ''"},
	{column: "run_result", ddl: "ALTER TABLE runs ADD COLUMN run_result TEXT"},
	{column: "commit_hash", ddl: "ALTER TABLE runs ADD COLUMN commit_hash TEXT DEFAULT ''"},
	{column: "import_source_harness", ddl: "ALTER TABLE runs ADD COLUMN import_source_harness TEXT DEFAULT ''"},
	{column: "import_source_session_id", ddl: "ALTER TABLE runs ADD COLUMN import_source_session_id TEXT DEFAULT ''"},
	{column: "imported_at", ddl: "ALTER TABLE runs ADD COLUMN imported_at TEXT"},
	{column: "workload_key", ddl: "ALTER TABLE runs ADD COLUMN workload_key TEXT NOT NULL DEFAULT ''"},
	{column: "workload_kind", ddl: "ALTER TABLE runs ADD COLUMN workload_kind TEXT NOT NULL DEFAULT 'adhoc'"},
	{column: "workload_instance", ddl: "ALTER TABLE runs ADD COLUMN workload_instance TEXT NOT NULL DEFAULT ''"},
	{column: "billing_snapshot", ddl: "ALTER TABLE runs ADD COLUMN billing_snapshot TEXT NOT NULL DEFAULT '{}'"},
	{column: "goal_id", ddl: "ALTER TABLE runs ADD COLUMN goal_id TEXT NOT NULL DEFAULT ''"},
	{column: "goal_status", ddl: "ALTER TABLE runs ADD COLUMN goal_status TEXT NOT NULL DEFAULT ''"},
	{column: "canary_arm", ddl: "ALTER TABLE runs ADD COLUMN canary_arm TEXT NOT NULL DEFAULT ''"},
}

const nullableTaskRunsTable = "runs__task_nullable"

type runSchemaObject struct {
	kind string
	name string
	sql  string
}

// migrateRunTaskIDNullability upgrades databases created before attached runs
// existed. SQLite has no ALTER COLUMN operation, so the only safe way to relax
// this constraint is a single-connection transaction that recreates the table
// from sqlite_master's exact schema, copies every column, then restores the
// table's indexes and triggers. Any failure rolls back the replacement.
func (db *DB) migrateRunTaskIDNullability(ctx context.Context) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return &domain.DatabaseError{Operation: "schema_migrate_task_id", EntityType: "Schema", Cause: err}
	}
	defer conn.Close()

	var (
		taskIDFound   bool
		taskIDNotNull int
	)
	rows, err := conn.QueryContext(ctx, "PRAGMA table_info(runs)")
	if err != nil {
		return &domain.DatabaseError{Operation: "schema_introspect_task_id", EntityType: "Schema", Cause: err}
	}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, dataType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			return &domain.DatabaseError{Operation: "schema_introspect_task_id", EntityType: "Schema", Cause: err}
		}
		if name == "task_id" {
			taskIDFound = true
			taskIDNotNull = notNull
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return &domain.DatabaseError{Operation: "schema_introspect_task_id", EntityType: "Schema", Cause: err}
	}
	rows.Close()
	if !taskIDFound || taskIDNotNull == 0 {
		return nil
	}

	var createSQL string
	if err := conn.QueryRowContext(ctx, "SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'runs'").Scan(&createSQL); err != nil {
		return &domain.DatabaseError{Operation: "schema_read_runs", EntityType: "Schema", Cause: err}
	}
	var existingTemporaryTable string
	if err := conn.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE name = ?", nullableTaskRunsTable).Scan(&existingTemporaryTable); err == nil {
		return &domain.DatabaseError{Operation: "schema_migrate_task_id", EntityType: "Schema", Cause: fmt.Errorf("temporary table %q already exists", nullableTaskRunsTable)}
	} else if err != sql.ErrNoRows {
		return &domain.DatabaseError{Operation: "schema_migrate_task_id", EntityType: "Schema", Cause: err}
	}
	temporaryCreateSQL, ok := replaceSQLFragment(createSQL, "CREATE TABLE IF NOT EXISTS runs", "CREATE TABLE IF NOT EXISTS "+nullableTaskRunsTable)
	if !ok {
		temporaryCreateSQL, ok = replaceSQLFragment(createSQL, "CREATE TABLE runs", "CREATE TABLE "+nullableTaskRunsTable)
	}
	if !ok {
		return &domain.DatabaseError{Operation: "schema_migrate_task_id", EntityType: "Schema", Cause: fmt.Errorf("unrecognized runs CREATE TABLE statement")}
	}
	temporaryCreateSQL, ok = replaceSQLFragment(temporaryCreateSQL, "task_id TEXT NOT NULL", "task_id TEXT")
	if !ok {
		return &domain.DatabaseError{Operation: "schema_migrate_task_id", EntityType: "Schema", Cause: fmt.Errorf("runs.task_id NOT NULL declaration not found")}
	}

	var columns []string
	rows, err = conn.QueryContext(ctx, "PRAGMA table_info(runs)")
	if err != nil {
		return &domain.DatabaseError{Operation: "schema_introspect_runs", EntityType: "Schema", Cause: err}
	}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, dataType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			return &domain.DatabaseError{Operation: "schema_introspect_runs", EntityType: "Schema", Cause: err}
		}
		columns = append(columns, quoteSQLiteIdentifier(name))
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return &domain.DatabaseError{Operation: "schema_introspect_runs", EntityType: "Schema", Cause: err}
	}
	rows.Close()
	if len(columns) == 0 {
		return &domain.DatabaseError{Operation: "schema_migrate_task_id", EntityType: "Schema", Cause: fmt.Errorf("runs has no columns")}
	}

	objects := make([]runSchemaObject, 0, 16)
	rows, err = conn.QueryContext(ctx, `SELECT type, name, sql FROM sqlite_master
		WHERE tbl_name = 'runs' AND type IN ('index', 'trigger') AND sql IS NOT NULL ORDER BY type, name`)
	if err != nil {
		return &domain.DatabaseError{Operation: "schema_read_runs_objects", EntityType: "Schema", Cause: err}
	}
	for rows.Next() {
		var object runSchemaObject
		if err := rows.Scan(&object.kind, &object.name, &object.sql); err != nil {
			rows.Close()
			return &domain.DatabaseError{Operation: "schema_read_runs_objects", EntityType: "Schema", Cause: err}
		}
		objects = append(objects, object)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return &domain.DatabaseError{Operation: "schema_read_runs_objects", EntityType: "Schema", Cause: err}
	}
	rows.Close()

	var beforeCount int64
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM runs").Scan(&beforeCount); err != nil {
		return &domain.DatabaseError{Operation: "schema_count_runs", EntityType: "Schema", Cause: err}
	}
	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return &domain.DatabaseError{Operation: "schema_migrate_task_id", EntityType: "Schema", Cause: err}
	}
	foreignKeysRestored := false
	defer func() {
		if !foreignKeysRestored {
			_, _ = conn.ExecContext(context.Background(), "PRAGMA foreign_keys = ON")
		}
	}()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return &domain.DatabaseError{Operation: "schema_migrate_task_id", EntityType: "Schema", Cause: err}
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	execMigration := func(statement string) error {
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			return err
		}
		return nil
	}
	if err := execMigration(temporaryCreateSQL); err != nil {
		return &domain.DatabaseError{Operation: "schema_create_nullable_runs", EntityType: "Schema", Cause: err}
	}
	columnList := strings.Join(columns, ", ")
	if err := execMigration("INSERT INTO " + quoteSQLiteIdentifier(nullableTaskRunsTable) + " (" + columnList + ") SELECT " + columnList + " FROM runs"); err != nil {
		return &domain.DatabaseError{Operation: "schema_copy_runs", EntityType: "Schema", Cause: err}
	}
	if err := execMigration("DROP TABLE runs"); err != nil {
		return &domain.DatabaseError{Operation: "schema_drop_legacy_runs", EntityType: "Schema", Cause: err}
	}
	if err := execMigration("ALTER TABLE " + quoteSQLiteIdentifier(nullableTaskRunsTable) + " RENAME TO runs"); err != nil {
		return &domain.DatabaseError{Operation: "schema_rename_nullable_runs", EntityType: "Schema", Cause: err}
	}
	for _, object := range objects {
		if err := execMigration(object.sql); err != nil {
			return &domain.DatabaseError{Operation: "schema_restore_runs_" + object.kind, EntityType: "Schema", Cause: fmt.Errorf("%s: %w", object.name, err)}
		}
	}
	var afterCount int64
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM runs").Scan(&afterCount); err != nil {
		return &domain.DatabaseError{Operation: "schema_verify_runs_count", EntityType: "Schema", Cause: err}
	}
	if afterCount != beforeCount {
		return &domain.DatabaseError{Operation: "schema_verify_runs_count", EntityType: "Schema", Cause: fmt.Errorf("row count changed from %d to %d", beforeCount, afterCount)}
	}
	var taskIDNotNullAfter int
	rows, err = conn.QueryContext(ctx, "PRAGMA table_info(runs)")
	if err != nil {
		return &domain.DatabaseError{Operation: "schema_verify_task_id", EntityType: "Schema", Cause: err}
	}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, dataType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			return &domain.DatabaseError{Operation: "schema_verify_task_id", EntityType: "Schema", Cause: err}
		}
		if name == "task_id" {
			taskIDNotNullAfter = notNull
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return &domain.DatabaseError{Operation: "schema_verify_task_id", EntityType: "Schema", Cause: err}
	}
	rows.Close()
	if taskIDNotNullAfter != 0 {
		return &domain.DatabaseError{Operation: "schema_verify_task_id", EntityType: "Schema", Cause: fmt.Errorf("runs.task_id remains NOT NULL")}
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return &domain.DatabaseError{Operation: "schema_commit_task_id", EntityType: "Schema", Cause: err}
	}
	committed = true
	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		return &domain.DatabaseError{Operation: "schema_restore_foreign_keys", EntityType: "Schema", Cause: err}
	}
	foreignKeysRestored = true
	return nil
}

func replaceSQLFragment(source, old, replacement string) (string, bool) {
	index := strings.Index(strings.ToLower(source), strings.ToLower(old))
	if index < 0 {
		return source, false
	}
	return source[:index] + replacement + source[index+len(old):], true
}

func quoteSQLiteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

// migrateRunColumns adds any missing additive columns to the runs table.
func (db *DB) migrateRunColumns(ctx context.Context) error {
	if err := db.migrateColumns(ctx, "runs", runColumnMigrations); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_runs_import_provenance ON runs(import_source_harness, import_source_session_id) WHERE import_source_harness <> '' AND import_source_session_id <> ''`)
	return err
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
