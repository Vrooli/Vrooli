package fleet

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/database"
)

//go:embed schema.sql
var fleetSchemaSQL string

// Schema returns the fleet domain's embedded, idempotent per-domain schema.
// Registered in modules.AllSchemas so EnsureSchemas creates it at boot.
func Schema() string { return fleetSchemaSQL }

type schemaExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// MigrateSchema applies one-shot additive migrations required before
// EnsureSchemas runs its drift check. SQLite ignores new columns added only to
// CREATE TABLE IF NOT EXISTS for pre-existing tables, so changed table shapes
// need explicit guarded ALTER TABLE statements.
func MigrateSchema(ctx context.Context, db schemaExecer) error {
	if db == nil {
		return nil
	}
	exists, err := tableExists(ctx, db, "fleet_entries")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	for _, column := range fleetBudgetColumns() {
		exists, err := columnExists(ctx, db, "fleet_entries", column.name)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE fleet_entries ADD COLUMN %s %s`, column.name, column.definition)); err != nil {
			return fmt.Errorf("fleet schema migration: add column %s: %w", column.name, err)
		}
	}
	return nil
}

// SQLStore persists the latest fleet snapshot to storage-manager's own SQLite
// store. It satisfies the Store seam.
//
// It holds *database.RoutedDB (not a raw *sql.DB) so it participates in the
// per-request routing the rest of the scenario uses — dogfooding the very
// handle-capture rule storage-manager enforces. The connection pool is capped at
// one connection (the SQLite-pool=1 convention), so every read drains its rows
// fully before issuing another query — never a nested query inside an open rows
// loop. Save replaces the whole snapshot in a single transaction.
type SQLStore struct {
	db *database.RoutedDB
}

// NewSQLStore wraps a routed database handle. A nil handle yields a nil store,
// so the no-DB path disables persistence cleanly.
func NewSQLStore(db *database.RoutedDB) *SQLStore {
	if db == nil {
		return nil
	}
	return &SQLStore{db: db}
}

var _ Store = (*SQLStore)(nil)

// Save replaces the persisted snapshot with res's entries in one transaction.
func (s *SQLStore) Save(ctx context.Context, res Result) error {
	if s == nil {
		return nil
	}
	if err := s.ensureBudgetColumns(ctx); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("fleet store: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM fleet_entries`); err != nil {
		return fmt.Errorf("fleet store: clear: %w", err)
	}
	stamp := ""
	if !res.ScannedAt.IsZero() {
		stamp = res.ScannedAt.UTC().Format(time.RFC3339)
	}
	const insert = `INSERT INTO fleet_entries
		(scenario, engines, primary_engine, language, storage_stage,
		 isolation_ready, isolation_reason, namespace_adopted, has_backup_target,
		 finding_count, error_count, autofixable_count, data_dir_bytes, data_dir_budget,
		 data_dir_util, data_dir_over_budget, data_dir_severity, data_dir_paths, scanned_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
	for _, e := range res.Entries {
		if _, err := tx.ExecContext(ctx, insert,
			e.Scenario, strings.Join(e.Engines, ","), e.PrimaryEngine, e.Language, e.StorageStage,
			boolToInt(e.IsolationReady), e.IsolationReason, boolToInt(e.NamespaceAdopted), boolToInt(e.HasBackupTarget),
			e.FindingCount, e.ErrorCount, e.AutofixableCount, e.DataDirBytes, e.DataDirBudget,
			e.DataDirUtil, boolToInt(e.DataDirOverBudget), e.DataDirSeverity, strings.Join(e.DataDirPaths, "\n"), stamp,
		); err != nil {
			return fmt.Errorf("fleet store: insert %q: %w", e.Scenario, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("fleet store: commit: %w", err)
	}
	return nil
}

// Load reads the persisted entries fully, then recomputes the aggregates from
// them so the snapshot's distributions never drift from its rows.
func (s *SQLStore) Load(ctx context.Context) (Result, error) {
	if s == nil {
		return Result{}, nil
	}
	if err := s.ensureBudgetColumns(ctx); err != nil {
		return Result{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT
		scenario, engines, primary_engine, language, storage_stage,
		isolation_ready, isolation_reason, namespace_adopted, has_backup_target,
		finding_count, error_count, autofixable_count, data_dir_bytes, data_dir_budget,
		data_dir_util, data_dir_over_budget, data_dir_severity, data_dir_paths, scanned_at
		FROM fleet_entries ORDER BY scenario`)
	if err != nil {
		return Result{}, fmt.Errorf("fleet store: query: %w", err)
	}
	// Drain every row fully here; the recompute below is pure Go over the loaded
	// slice (no nested query), so deferring Close is safe under pool=1.
	defer func() { _ = rows.Close() }()
	var entries []ScenarioEntry
	var scannedAt string
	for rows.Next() {
		var (
			e          ScenarioEntry
			engines    string
			isoReady   int
			nsAdopted  int
			hasBackup  int
			overBudget int
			dataPaths  string
			rowStamp   string
		)
		if err := rows.Scan(
			&e.Scenario, &engines, &e.PrimaryEngine, &e.Language, &e.StorageStage,
			&isoReady, &e.IsolationReason, &nsAdopted, &hasBackup,
			&e.FindingCount, &e.ErrorCount, &e.AutofixableCount, &e.DataDirBytes, &e.DataDirBudget,
			&e.DataDirUtil, &overBudget, &e.DataDirSeverity, &dataPaths, &rowStamp,
		); err != nil {
			return Result{}, fmt.Errorf("fleet store: scan: %w", err)
		}
		if engines != "" {
			e.Engines = strings.Split(engines, ",")
		}
		e.IsolationReady = isoReady != 0
		e.NamespaceAdopted = nsAdopted != 0
		e.HasBackupTarget = hasBackup != 0
		e.DataDirOverBudget = overBudget != 0
		if dataPaths != "" {
			e.DataDirPaths = strings.Split(dataPaths, "\n")
		}
		entries = append(entries, e)
		if rowStamp != "" {
			scannedAt = rowStamp
		}
	}
	if err := rows.Err(); err != nil {
		return Result{}, fmt.Errorf("fleet store: rows: %w", err)
	}

	res := Result{Entries: entries}
	engineCounts := map[string]int{}
	stageCounts := map[string]int{}
	for _, e := range entries {
		res.ScenarioCount++
		res.FindingCount += e.FindingCount
		if e.DataDirOverBudget {
			res.DataDirOverBudgetCount++
		}
		if !e.IsolationReady {
			res.IsolationUnreadyCount++
		}
		if !e.HasBackupTarget && e.dataPersisting() {
			res.NoBackupCount++
		}
		seen := map[string]struct{}{}
		for _, eng := range e.Engines {
			if _, dup := seen[eng]; dup {
				continue
			}
			seen[eng] = struct{}{}
			engineCounts[eng]++
		}
		stage := e.StorageStage
		if stage == "" {
			stage = "greenfield"
		}
		stageCounts[stage]++
	}
	res.EngineDistribution = engineDistribution(engineCounts)
	res.StageDistribution = stageDistribution(stageCounts)
	if scannedAt != "" {
		if t, perr := time.Parse(time.RFC3339, scannedAt); perr == nil {
			res.ScannedAt = t
		}
	}
	return res, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *SQLStore) ensureBudgetColumns(ctx context.Context) error {
	return MigrateSchema(ctx, s.db)
}

type fleetBudgetColumn struct {
	name       string
	definition string
}

func fleetBudgetColumns() []fleetBudgetColumn {
	return []fleetBudgetColumn{
		{name: "data_dir_bytes", definition: "INTEGER NOT NULL DEFAULT 0"},
		{name: "data_dir_budget", definition: "INTEGER NOT NULL DEFAULT 0"},
		{name: "data_dir_util", definition: "REAL NOT NULL DEFAULT 0"},
		{name: "data_dir_over_budget", definition: "INTEGER NOT NULL DEFAULT 0"},
		{name: "data_dir_severity", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "data_dir_paths", definition: "TEXT NOT NULL DEFAULT ''"},
	}
}

func tableExists(ctx context.Context, db schemaExecer, table string) (bool, error) {
	var name string
	err := db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, fmt.Errorf("fleet schema migration: inspect table %s: %w", table, err)
}

func columnExists(ctx context.Context, db schemaExecer, table string, name string) (bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return false, fmt.Errorf("fleet schema migration: inspect columns: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var cid int
		var column, typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &column, &typ, &notNull, &defaultValue, &pk); err != nil {
			return false, fmt.Errorf("fleet schema migration: scan column: %w", err)
		}
		if column == name {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("fleet schema migration: inspect columns: %w", err)
	}
	return false, nil
}
