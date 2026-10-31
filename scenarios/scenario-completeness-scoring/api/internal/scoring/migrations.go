package scoring

import (
	"context"
	"database/sql"
	"fmt"
)

// migrateDB is the narrow handle scoring.Migrate needs: introspection +
// statement execution. *sql.DB and the routed primary handle both satisfy it.
type migrateDB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// recencyColumns are the scenario-level test-recency columns added after the
// original score_snapshots shape. Fresh databases get them from schema.sql
// (CREATE TABLE); this migration only matters for databases created before they
// existed. Order is irrelevant — each is guarded independently.
var recencyColumns = []string{"last_run_at", "last_status"}

// Migrate evolves an existing score_snapshots table to the current shape
// without recreating it — the accumulated score history is the trend and fleet
// denominator and must never be dropped. It is guarded and idempotent: every
// step introspects current state before acting, so it is safe to run on every
// boot. It must run BEFORE the EnsureSchemas drift check, which would otherwise
// fail on a pre-existing table missing the declared columns (CREATE TABLE IF
// NOT EXISTS cannot add a column to a data-bearing table).
//
// When the table does not exist yet (a fresh database) this is a no-op: the
// subsequent CREATE TABLE creates it complete with the recency columns.
func Migrate(ctx context.Context, db migrateDB) error {
	exists, err := tableExists(ctx, db, "score_snapshots")
	if err != nil {
		return fmt.Errorf("introspect score_snapshots: %w", err)
	}
	if !exists {
		return nil
	}
	for _, col := range recencyColumns {
		has, err := columnExists(ctx, db, "score_snapshots", col)
		if err != nil {
			return fmt.Errorf("introspect score_snapshots.%s: %w", col, err)
		}
		if has {
			continue
		}
		// SQLite has no ADD COLUMN IF NOT EXISTS; the PRAGMA guard above is the
		// idempotency check. NOT NULL requires a constant default, which '' is.
		stmt := fmt.Sprintf("ALTER TABLE score_snapshots ADD COLUMN %s TEXT NOT NULL DEFAULT ''", col)
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("add score_snapshots.%s: %w", col, err)
		}
	}
	return nil
}

// tableExists reports whether a table is present, via sqlite_master lookup
// (SQLite-portable). On non-SQLite engines this query errors; callers treat
// the migration as SQLite-scoped (the only dialect this scenario runs on).
func tableExists(ctx context.Context, db migrateDB, table string) (bool, error) {
	rows, err := db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name=?", table)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	if rows.Next() {
		return true, rows.Err()
	}
	return false, rows.Err()
}

// columnExists reports whether a table already has the named column, via
// PRAGMA table_info introspection (SQLite-portable, no information_schema).
func columnExists(ctx context.Context, db migrateDB, table, column string) (bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid       int
			name      string
			colType   string
			notNull   int
			dfltValue any
			pk        int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
