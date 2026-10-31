package database

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"reflect"
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

// DataDir returns the directory holding this scenario's SQLite database.
// It resolves through the same seam as SQLiteDSN, so the two can never
// disagree about where the database lives.
func DataDir() string {
	path, err := storage.SQLitePath(storage.SQLiteConfig{Scenario: "agent-manager"})
	if err != nil {
		return "."
	}
	return filepath.Dir(path)
}

// SQLiteDSN resolves Agent Manager's canonical SQLite DSN for the API-core
// connection seam. It is intentionally exported to keep main's composition
// root explicit without exposing path-resolution internals.
//
// It reads no database-path variable. It previously accepted AM_SQLITE_PATH and
// a "file:" DATABASE_URL ahead of the scenario's own identity, so any process
// that exported either and then started this scenario redirected its database.
func SQLiteDSN(log *logrus.Logger) (string, error) {
	dsn, err := storage.SQLiteDSN(storage.SQLiteConfig{
		Scenario: "agent-manager",
		// A read-heavy store: agent runs are written once and queried often.
		Tuning: storage.SQLiteTuning{PageSizeBytes: 4096},
	})
	if err != nil {
		return "", domain.NewConfigInvalidError("storage", "resolve canonical sqlite path", err)
	}
	dsn = withWALLimit(dsn)
	if log != nil {
		log.WithField("dsn", dsn).Info("Using SQLite database")
	}
	return dsn, nil
}

// walJournalSizeLimit caps the WAL file a connection leaves behind when it
// resets the log after a checkpoint. Without it the WAL keeps its high-water
// size: 1 GB stayed on disk on 2026-09-14 long after the burst that grew it.
const walJournalSizeLimit = 64 << 20

func withWALLimit(dsn string) string {
	return fmt.Sprintf("%s&_pragma=journal_size_limit(%d)", dsn, walJournalSizeLimit)
}

// SQLiteFilePath is the database file SQLiteDSN opens, for owners that measure
// the file itself (storage health) rather than connect to it.
func SQLiteFilePath() (string, error) {
	return storage.SQLitePath(storage.SQLiteConfig{Scenario: "agent-manager"})
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

// initSchema adds the columns that tables gained after their CREATE TABLE
// shipped, then applies the declarative schema. SQLite's CREATE TABLE IF NOT
// EXISTS cannot add columns to an existing table and EnsureSchemas drift-checks
// declared columns, so the additive migrations run first and only for tables
// that already exist; a fresh database gets every column from the schema.
//
// Besides the additive column migrations, only the one-shots from 2026-10-29
// remain (runOnceMigrations), each gated by a marker. The older data and
// table-shape migrations (task_id nullability, the investigation projection
// cutover, watch action rebuild, retired read model columns, fact segment key)
// are at least two weeks old and have run on every install that follows the
// branch, so they are not carried: an install older than those cutovers must be
// upgraded through a build that still has them.
func (db *DB) initSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSchemaTimeout)
	defer cancel()
	for _, table := range additiveColumnMigrations {
		columns, err := db.tableColumns(ctx, table.name)
		if err != nil {
			return err
		}
		if len(columns) == 0 {
			continue
		}
		if table.name == "run_findings" {
			if err := db.retireRunFindingReviewState(ctx); err != nil {
				return err
			}
		}
		if err := db.migrateColumns(ctx, table.name, table.columns); err != nil {
			return err
		}
	}
	if err := coredb.EnsureSchemas(ctx, db, modules.AllSchemas()...); err != nil {
		if db.log != nil {
			db.log.WithError(err).Error("Failed to apply domain schemas")
		}
		return &domain.DatabaseError{Operation: "schema_init", EntityType: "Schema", Cause: err}
	}
	if err := db.runOnceMigrations(ctx); err != nil {
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

// additiveColumnMigrations lists, per table, the columns added after the
// table's CREATE TABLE first shipped. Each is applied only when missing, so
// re-running is safe and existing rows keep their data (new columns take their
// DEFAULT).
var additiveColumnMigrations = []struct {
	name    string
	columns []columnMigration
}{
	{"tasks", []columnMigration{{column: "current_run_id", ddl: "ALTER TABLE tasks ADD COLUMN current_run_id TEXT"}}},
	{"runs", runColumnMigrations},
	{"model_pricing", []columnMigration{{column: "price_book_revision_id", ddl: "ALTER TABLE model_pricing ADD COLUMN price_book_revision_id TEXT"}}},
	{"quota_observations", []columnMigration{{column: "account_scope", ddl: "ALTER TABLE quota_observations ADD COLUMN account_scope TEXT NOT NULL DEFAULT ''"}}},
	{"invocation_read_model_runs", invocationReadModelRunColumnMigrations},
	{"invocation_read_model_facts", invocationReadModelFactColumnMigrations},
	{"invocation_read_model_time_accounting", invocationReadModelTimeAccountingColumnMigrations},
	{"invocation_read_model_watermarks", invocationReadModelWatermarkColumnMigrations},
	{"invocation_read_model_self_report_spans", invocationReadModelSelfReportColumnMigrations},
	{"investigation_cross_scenario_calls", investigationProjectionColumnMigrations},
	{"run_findings", runFindingColumnMigrations},
	{"agent_profiles", agentProfileColumnMigrations},
	{"workflow_executions", workflowExecutionColumnMigrations},
	{"workflow_node_attempts", workflowNodeAttemptColumnMigrations},
	{"transcript_files", transcriptFileColumnMigrations},
	{"run_ledger_facts", runLedgerArmColumnMigrations},
	{"usage_mode_periods", usageModePeriodLearnColumnMigrations},
}

// runLedgerArmColumnMigrations record a run's place in a Learn comparison.
var runLedgerArmColumnMigrations = []columnMigration{
	{column: "canary_arm", ddl: "ALTER TABLE run_ledger_facts ADD COLUMN canary_arm TEXT NOT NULL DEFAULT ''"},
	{column: "arm", ddl: "ALTER TABLE run_ledger_facts ADD COLUMN arm TEXT NOT NULL DEFAULT ''"},
	{column: "arm_mode", ddl: "ALTER TABLE run_ledger_facts ADD COLUMN arm_mode TEXT NOT NULL DEFAULT ''"},
	{column: "arm_basis", ddl: "ALTER TABLE run_ledger_facts ADD COLUMN arm_basis TEXT NOT NULL DEFAULT ''"},
	{column: "arm_exception", ddl: "ALTER TABLE run_ledger_facts ADD COLUMN arm_exception TEXT NOT NULL DEFAULT ''"},
}

// usageModePeriodLearnColumnMigrations record how a learn period randomizes.
var usageModePeriodLearnColumnMigrations = []columnMigration{
	{column: "kind", ddl: "ALTER TABLE usage_mode_periods ADD COLUMN kind TEXT NOT NULL DEFAULT ''"},
	{column: "label", ddl: "ALTER TABLE usage_mode_periods ADD COLUMN label TEXT NOT NULL DEFAULT ''"},
	{column: "split", ddl: "ALTER TABLE usage_mode_periods ADD COLUMN split REAL NOT NULL DEFAULT 0"},
	{column: "session_opt_in", ddl: "ALTER TABLE usage_mode_periods ADD COLUMN session_opt_in INTEGER NOT NULL DEFAULT 0"},
	{column: "block_days", ddl: "ALTER TABLE usage_mode_periods ADD COLUMN block_days INTEGER NOT NULL DEFAULT 0"},
}

var agentProfileColumnMigrations = []columnMigration{
	{column: "tool_restriction_policy", ddl: "ALTER TABLE agent_profiles ADD COLUMN tool_restriction_policy TEXT NOT NULL DEFAULT 'enforced'"},
	{column: "effort", ddl: "ALTER TABLE agent_profiles ADD COLUMN effort TEXT NOT NULL DEFAULT ''"},
	{column: "declared_scopes", ddl: "ALTER TABLE agent_profiles ADD COLUMN declared_scopes TEXT DEFAULT '[]'"},
	{column: "skill_pack", ddl: "ALTER TABLE agent_profiles ADD COLUMN skill_pack TEXT DEFAULT '[]'"},
	{column: "skill_experiment_id", ddl: "ALTER TABLE agent_profiles ADD COLUMN skill_experiment_id TEXT DEFAULT ''"},
	{column: "spawn_policy", ddl: "ALTER TABLE agent_profiles ADD COLUMN spawn_policy TEXT DEFAULT '{}'"},
	{column: "max_weighted_tokens", ddl: "ALTER TABLE agent_profiles ADD COLUMN max_weighted_tokens INTEGER NOT NULL DEFAULT 0"},
}

var workflowExecutionColumnMigrations = []columnMigration{
	{column: "parent_attempt_id", ddl: "ALTER TABLE workflow_executions ADD COLUMN parent_attempt_id TEXT"},
	{column: "depth", ddl: "ALTER TABLE workflow_executions ADD COLUMN depth INTEGER NOT NULL DEFAULT 0"},
	{column: "engagement_grant_json", ddl: "ALTER TABLE workflow_executions ADD COLUMN engagement_grant_json TEXT"},
	{column: "approval_digest", ddl: "ALTER TABLE workflow_executions ADD COLUMN approval_digest TEXT NOT NULL DEFAULT ''"},
	{column: "grant_digest", ddl: "ALTER TABLE workflow_executions ADD COLUMN grant_digest TEXT NOT NULL DEFAULT ''"},
	{column: "execution_preferences_json", ddl: "ALTER TABLE workflow_executions ADD COLUMN execution_preferences_json TEXT NOT NULL DEFAULT ''"},
}

var workflowNodeAttemptColumnMigrations = []columnMigration{
	{column: "child_execution_id", ddl: "ALTER TABLE workflow_node_attempts ADD COLUMN child_execution_id TEXT"},
	{column: "raw_output", ddl: "ALTER TABLE workflow_node_attempts ADD COLUMN raw_output TEXT"},
	{column: "validation_error", ddl: "ALTER TABLE workflow_node_attempts ADD COLUMN validation_error TEXT"},
	{column: "experiment_id", ddl: "ALTER TABLE workflow_node_attempts ADD COLUMN experiment_id TEXT"},
	{column: "variant_id", ddl: "ALTER TABLE workflow_node_attempts ADD COLUMN variant_id TEXT"},
	{column: "prompt_hash", ddl: "ALTER TABLE workflow_node_attempts ADD COLUMN prompt_hash TEXT"},
}

var invocationReadModelRunColumnMigrations = []columnMigration{
	{column: "goal_id", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN goal_id TEXT NOT NULL DEFAULT ''"},
	{column: "goal_status", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN goal_status TEXT NOT NULL DEFAULT ''"},
	{column: "input_cost_usd", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN input_cost_usd REAL NOT NULL DEFAULT 0"},
	{column: "cost_source", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN cost_source TEXT NOT NULL DEFAULT 'unknown'"},
	{column: "charge_reason", ddl: "ALTER TABLE invocation_read_model_runs ADD COLUMN charge_reason TEXT NOT NULL DEFAULT ''"},
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
	{column: "friction_topic", ddl: "ALTER TABLE run_findings ADD COLUMN friction_topic TEXT NOT NULL DEFAULT ''"},
	{column: "cites_resolved_commands", ddl: "ALTER TABLE run_findings ADD COLUMN cites_resolved_commands INTEGER NOT NULL DEFAULT 0"},
	{column: "cites_real_outcome", ddl: "ALTER TABLE run_findings ADD COLUMN cites_real_outcome INTEGER NOT NULL DEFAULT 0"},
	{column: "cites_attributed_owner", ddl: "ALTER TABLE run_findings ADD COLUMN cites_attributed_owner INTEGER NOT NULL DEFAULT 0"},
	{column: "quality_signal", ddl: "ALTER TABLE run_findings ADD COLUMN quality_signal TEXT NOT NULL DEFAULT 'unavailable'"},
}

var invocationReadModelFactColumnMigrations = []columnMigration{
	{column: "program_id", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN program_id TEXT NOT NULL DEFAULT ''"},
	{column: "skill_id", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN skill_id TEXT NOT NULL DEFAULT ''"},
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
	// residency_turns (attenuated turns across compactions) became provider
	// requests until the next compaction; facts re-derive at their new version.
	{column: "residency_requests", ddl: "ALTER TABLE invocation_read_model_facts RENAME COLUMN residency_turns TO residency_requests"},
	{column: "output_event_id", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN output_event_id TEXT NOT NULL DEFAULT ''"},
	{column: "output_bytes", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN output_bytes INTEGER NOT NULL DEFAULT 0"},
	{column: "residency_segment", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN residency_segment INTEGER NOT NULL DEFAULT 0"},
	{column: "incurred_input_tokens", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN incurred_input_tokens INTEGER NOT NULL DEFAULT 0"},
	{column: "incurred_output_tokens", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN incurred_output_tokens INTEGER NOT NULL DEFAULT 0"},
	{column: "incurred_cache_read_tokens", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN incurred_cache_read_tokens INTEGER NOT NULL DEFAULT 0"},
	{column: "incurred_cache_creation_tokens", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN incurred_cache_creation_tokens INTEGER NOT NULL DEFAULT 0"},
	// Tool friction features, read from the call and result text at projection.
	{column: "tool_profiled", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN tool_profiled INTEGER NOT NULL DEFAULT 0"},
	{column: "tool_command", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN tool_command TEXT NOT NULL DEFAULT ''"},
	{column: "tool_key", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN tool_key TEXT NOT NULL DEFAULT ''"},
	{column: "tool_help", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN tool_help INTEGER NOT NULL DEFAULT 0"},
	{column: "tool_filtered", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN tool_filtered INTEGER NOT NULL DEFAULT 0"},
	{column: "tool_wrapped", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN tool_wrapped INTEGER NOT NULL DEFAULT 0"},
	{column: "tool_truncated", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN tool_truncated INTEGER NOT NULL DEFAULT 0"},
	{column: "tool_shape_guess", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN tool_shape_guess INTEGER NOT NULL DEFAULT 0"},
	{column: "tool_json_paths", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN tool_json_paths TEXT NOT NULL DEFAULT ''"},
	{column: "tool_signature", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN tool_signature TEXT NOT NULL DEFAULT ''"},
	{column: "tool_error_line", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN tool_error_line TEXT NOT NULL DEFAULT ''"},
	{column: "tool_notice", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN tool_notice TEXT NOT NULL DEFAULT ''"},
	{column: "tool_notice_line", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN tool_notice_line TEXT NOT NULL DEFAULT ''"},
	{column: "tool_bypass_owner", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN tool_bypass_owner TEXT NOT NULL DEFAULT ''"},
	{column: "tool_bypass_method", ddl: "ALTER TABLE invocation_read_model_facts ADD COLUMN tool_bypass_method TEXT NOT NULL DEFAULT ''"},
}

var invocationReadModelTimeAccountingColumnMigrations = []columnMigration{
	{column: "unattributable_reason", ddl: "ALTER TABLE invocation_read_model_time_accounting ADD COLUMN unattributable_reason TEXT NOT NULL DEFAULT ''"},
}

var invocationReadModelWatermarkColumnMigrations = []columnMigration{
	{column: "self_report_classifier_version", ddl: "ALTER TABLE invocation_read_model_watermarks ADD COLUMN self_report_classifier_version TEXT NOT NULL DEFAULT ''"},
	// Existing watermarks were committed atomically with their derived facts
	// before this marker existed, so they are safe to retain as complete.
	{column: "projection_complete", ddl: "ALTER TABLE invocation_read_model_watermarks ADD COLUMN projection_complete INTEGER NOT NULL DEFAULT 1"},
}

var invocationReadModelSelfReportColumnMigrations = []columnMigration{
	{column: "role", ddl: "ALTER TABLE invocation_read_model_self_report_spans ADD COLUMN role TEXT NOT NULL DEFAULT 'assistant'"},
	{column: "span_capped", ddl: "ALTER TABLE invocation_read_model_self_report_spans ADD COLUMN span_capped INTEGER NOT NULL DEFAULT 0"},
}

var transcriptFileColumnMigrations = []columnMigration{
	{column: "pending_facts_json", ddl: "ALTER TABLE transcript_files ADD COLUMN pending_facts_json TEXT NOT NULL DEFAULT ''"},
	{column: "parser_version", ddl: "ALTER TABLE transcript_files ADD COLUMN parser_version INTEGER NOT NULL DEFAULT 0"},
	{column: "last_observed_event_at", ddl: "ALTER TABLE transcript_files ADD COLUMN last_observed_event_at TEXT"},
}

var investigationProjectionColumnMigrations = []columnMigration{
	{column: "occurred_at", ddl: "ALTER TABLE investigation_cross_scenario_calls ADD COLUMN occurred_at TEXT"},
}

// runColumnMigrations lists columns added to the runs table after its original
// CREATE TABLE shipped. Each is applied only when missing, so re-running is
// safe and existing rows are untouched (the new columns take their DEFAULT).
var runColumnMigrations = []columnMigration{
	{column: "interactive_invocation_started_at", ddl: "ALTER TABLE runs ADD COLUMN interactive_invocation_started_at TEXT"},
	{column: "lifecycle_version", ddl: "ALTER TABLE runs ADD COLUMN lifecycle_version INTEGER NOT NULL DEFAULT 0"},
	{column: "owner_identity", ddl: "ALTER TABLE runs ADD COLUMN owner_identity TEXT NOT NULL DEFAULT ''"},
	{column: "owner_epoch", ddl: "ALTER TABLE runs ADD COLUMN owner_epoch INTEGER NOT NULL DEFAULT 0"},
	{column: "fresh_recovery_request_hash", ddl: "ALTER TABLE runs ADD COLUMN fresh_recovery_request_hash TEXT NOT NULL DEFAULT ''"},
	{column: "label", ddl: "ALTER TABLE runs ADD COLUMN label TEXT NOT NULL DEFAULT ''"},
	{column: "label_source", ddl: "ALTER TABLE runs ADD COLUMN label_source TEXT NOT NULL DEFAULT ''"},
	{column: "subject", ddl: "ALTER TABLE runs ADD COLUMN subject TEXT NOT NULL DEFAULT '[]'"},
	{column: "owner_subject", ddl: "ALTER TABLE runs ADD COLUMN owner_subject TEXT NOT NULL DEFAULT ''"},
	{column: "owner_expires_at", ddl: "ALTER TABLE runs ADD COLUMN owner_expires_at TEXT"},
	{column: "owner_scopes", ddl: "ALTER TABLE runs ADD COLUMN owner_scopes TEXT NOT NULL DEFAULT '[]'"},
	{column: "requested_scopes", ddl: "ALTER TABLE runs ADD COLUMN requested_scopes TEXT NOT NULL DEFAULT '[]'"},
	{column: "work_references", ddl: "ALTER TABLE runs ADD COLUMN work_references TEXT NOT NULL DEFAULT '[]'"},
	{column: "effort_signal_cursors", ddl: "ALTER TABLE runs ADD COLUMN effort_signal_cursors TEXT NOT NULL DEFAULT '{}'"},
	{column: "execution_mode", ddl: "ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe'"},
	{column: "harness_kind", ddl: "ALTER TABLE runs ADD COLUMN harness_kind TEXT DEFAULT ''"},
	{column: "harness_session_id", ddl: "ALTER TABLE runs ADD COLUMN harness_session_id TEXT DEFAULT ''"},
	{column: "cancel_requested_at", ddl: "ALTER TABLE runs ADD COLUMN cancel_requested_at TEXT"},
	{column: "goal_delivery", ddl: "ALTER TABLE runs ADD COLUMN goal_delivery TEXT DEFAULT ''"},
	{column: "terminal_class", ddl: "ALTER TABLE runs ADD COLUMN terminal_class TEXT DEFAULT ''"},
	{column: "stop_reason", ddl: "ALTER TABLE runs ADD COLUMN stop_reason TEXT DEFAULT ''"},
	{column: "last_handoff", ddl: "ALTER TABLE runs ADD COLUMN last_handoff TEXT DEFAULT ''"},
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
	{column: "identity_renewal_anchor_hash", ddl: "ALTER TABLE runs ADD COLUMN identity_renewal_anchor_hash TEXT NOT NULL DEFAULT ''"},
	{column: "identity_token_issued_at", ddl: "ALTER TABLE runs ADD COLUMN identity_token_issued_at TEXT"},
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
	return columnsOf(ctx, db, table)
}

// columnsOf is tableColumns over any query runner, so a migration can read a
// table's shape inside its own transaction.
func columnsOf(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, table string,
) (map[string]struct{}, error) {
	rows, err := q.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
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
