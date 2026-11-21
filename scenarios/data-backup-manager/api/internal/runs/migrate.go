package runs

import (
	"context"
	"fmt"
)

// addedColumns lists columns introduced on the runs table after it first
// shipped. SQLite has no "ADD COLUMN IF NOT EXISTS", so EnsureColumns
// introspects PRAGMA table_info and adds only the missing ones — keeping boot
// idempotent on both fresh and existing databases. New columns must also be
// present in schema.sql so a fresh CREATE TABLE includes them.
var addedColumns = []struct {
	name string
	ddl  string
}{
	{name: "error", ddl: "ALTER TABLE runs ADD COLUMN error TEXT NOT NULL DEFAULT ''"},
	{name: "updated_at", ddl: "ALTER TABLE runs ADD COLUMN updated_at TEXT NOT NULL DEFAULT ''"},
	{name: "physical_bytes", ddl: "ALTER TABLE runs ADD COLUMN physical_bytes INTEGER NOT NULL DEFAULT 0"},
	{name: "failure_code", ddl: "ALTER TABLE runs ADD COLUMN failure_code TEXT NOT NULL DEFAULT ''"},
	{name: "failure_category", ddl: "ALTER TABLE runs ADD COLUMN failure_category TEXT NOT NULL DEFAULT ''"},
	{name: "next_action", ddl: "ALTER TABLE runs ADD COLUMN next_action TEXT NOT NULL DEFAULT ''"},
}

var addedOutcomeColumns = []struct{ name, ddl string }{
	{name: "failure_code", ddl: "ALTER TABLE run_outcomes ADD COLUMN failure_code TEXT NOT NULL DEFAULT ''"},
	{name: "failure_category", ddl: "ALTER TABLE run_outcomes ADD COLUMN failure_category TEXT NOT NULL DEFAULT ''"},
	{name: "warning", ddl: "ALTER TABLE run_outcomes ADD COLUMN warning TEXT NOT NULL DEFAULT ''"},
}

// EnsureColumns applies additive column migrations to an existing runs table.
// It is safe to call on every boot and on a freshly created table (the columns
// already exist, so nothing is added). Call it after EnsureSchemas has created
// the base table. This is what lets the async-execution columns (error,
// updated_at) land on a database that predates them without losing run history.
func EnsureColumns(ctx context.Context, db SQLExecutor) error {
	existing, err := tableColumns(ctx, db)
	if err != nil {
		return err
	}
	for _, col := range addedColumns {
		if _, ok := existing[col.name]; ok {
			continue
		}
		if _, err := db.ExecContext(ctx, col.ddl); err != nil {
			return fmt.Errorf("add runs column %q: %w", col.name, err)
		}
	}
	outcomeColumns, err := tableColumns(ctx, db, "run_outcomes")
	if err != nil {
		return err
	}
	for _, col := range addedOutcomeColumns {
		if _, ok := outcomeColumns[col.name]; ok {
			continue
		}
		if _, err := db.ExecContext(ctx, col.ddl); err != nil {
			return fmt.Errorf("add run_outcomes column %q: %w", col.name, err)
		}
	}
	if _, err := db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS runs_one_active_plan ON runs(plan_id) WHERE status IN ('pending', 'capturing', 'snapshotting')`); err != nil {
		return fmt.Errorf("active plan run index migration failed: %w", err)
	}
	if err := ensureIncidentTargetKey(ctx, db); err != nil {
		return err
	}
	return nil
}

// ensureIncidentTargetKey widens the run_incidents primary key to include the
// target set. A target-scope cause belongs to exactly one target, so two
// unreadable sources are two incidents even when code, scope, and destination
// are equal. Without target_ids in the key the second one violated the
// constraint and its evidence was dropped. SQLite cannot alter a primary key
// in place, so the table is rebuilt and copied.
func ensureIncidentTargetKey(ctx context.Context, db SQLExecutor) error {
	columns, err := tableColumns(ctx, db, "run_incidents")
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		// A crash between DROP and RENAME leaves only the migrated table.
		migrated, err := tableColumns(ctx, db, "run_incidents_migrated")
		if err != nil {
			return err
		}
		if len(migrated) == 0 {
			return nil // EnsureSchemas creates the fresh table on the next boot.
		}
		_, err = db.ExecContext(ctx, `ALTER TABLE run_incidents_migrated RENAME TO run_incidents`)
		return err
	}
	key, err := primaryKeyColumns(ctx, db, "run_incidents")
	if err != nil {
		return err
	}
	if key["target_ids"] {
		return nil
	}
	statements := []string{
		`DROP TABLE IF EXISTS run_incidents_migrated`,
		`CREATE TABLE run_incidents_migrated (
			run_id TEXT NOT NULL,
			code TEXT NOT NULL,
			category TEXT NOT NULL,
			scope TEXT NOT NULL,
			message TEXT NOT NULL DEFAULT '',
			next_action TEXT NOT NULL DEFAULT '',
			destination_id TEXT NOT NULL DEFAULT '',
			target_ids TEXT NOT NULL DEFAULT '[]',
			first_observed TEXT NOT NULL DEFAULT '',
			last_observed TEXT NOT NULL DEFAULT '',
			last_known_good TEXT NOT NULL DEFAULT '',
			retryable INTEGER NOT NULL DEFAULT 0,
			retry_after_seconds INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (run_id, code, scope, destination_id, target_ids),
			FOREIGN KEY (run_id) REFERENCES runs(id) ON DELETE CASCADE
		)`,
		// Historical runs can be deleted while their incident rows survive in a
		// database that predates the foreign-key enforcement. The rebuilt table
		// has the same FK, so copy only incidents whose run still exists; an
		// orphan cannot become evidence for a run that is gone.
		`INSERT OR IGNORE INTO run_incidents_migrated SELECT run_id, code, category, scope, message, next_action, destination_id, target_ids, first_observed, last_observed, last_known_good, retryable, retry_after_seconds FROM run_incidents WHERE run_id IN (SELECT id FROM runs)`,
		`DROP TABLE run_incidents`,
		`ALTER TABLE run_incidents_migrated RENAME TO run_incidents`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("widen run_incidents key: %w", err)
		}
	}
	return nil
}

func primaryKeyColumns(ctx context.Context, db SQLExecutor, table string) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return nil, fmt.Errorf("introspect %s primary key: %w", table, err)
	}
	defer rows.Close()
	key := map[string]bool{}
	for rows.Next() {
		var (
			cid        int
			name       string
			ctype      string
			notNull    int
			dfltValue  any
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dfltValue, &primaryKey); err != nil {
			return nil, fmt.Errorf("scan %s primary key: %w", table, err)
		}
		if primaryKey > 0 {
			key[name] = true
		}
	}
	return key, rows.Err()
}

func tableColumns(ctx context.Context, db SQLExecutor, table ...string) (map[string]struct{}, error) {
	name := "runs"
	if len(table) > 0 && table[0] != "" {
		name = table[0]
	}
	rows, err := db.QueryContext(ctx, "PRAGMA table_info("+name+")")
	if err != nil {
		return nil, fmt.Errorf("introspect runs columns: %w", err)
	}
	defer rows.Close()

	cols := map[string]struct{}{}
	for rows.Next() {
		var (
			cid        int
			name       string
			ctype      string
			notNull    int
			dfltValue  any
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dfltValue, &primaryKey); err != nil {
			return nil, fmt.Errorf("scan table_info row: %w", err)
		}
		cols[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate table_info rows: %w", err)
	}
	return cols, nil
}
