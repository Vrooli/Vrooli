package execution

import (
	"context"
	"fmt"

	"test-genie/internal/dbexec"
)

// Migrate evolves an existing suite_executions table to the current shape
// without recreating it — the accumulated execution history is the reliability
// ledger's denominator and must never be dropped. It is guarded and idempotent:
// every step introspects current state before acting, so it is safe to run on
// every boot (fresh DBs already get the column from Schema(); this only matters
// for databases created before terminal_outcome existed).
//
// This is the execution domain's column-evolution hook — the minimal,
// data-preserving substrate aligned with storage-steer's deferred
// MigrationProvider direction (PRAGMA introspect → ALTER ADD COLUMN → backfill).
func Migrate(ctx context.Context, db dbexec.Executor) error {
	for _, column := range []string{"terminal_outcome", "run_id"} {
		hasColumn, err := columnExists(ctx, db, "suite_executions", column)
		if err != nil {
			return fmt.Errorf("introspect suite_executions: %w", err)
		}
		if !hasColumn {
			if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE suite_executions ADD COLUMN %s TEXT", column)); err != nil {
				return fmt.Errorf("add suite_executions.%s: %w", column, err)
			}
		}
	}
	// Backfill rows that predate the column (or were written before it was
	// populated): derive the run-level outcome from the success flag. The
	// IS NULL guard makes this a no-op once every row carries a value.
	if _, err := db.ExecContext(ctx, `
UPDATE suite_executions
SET terminal_outcome = CASE WHEN success = 1 THEN ? ELSE ? END
WHERE terminal_outcome IS NULL OR terminal_outcome = ''`,
		TerminalOutcomePassed.String(), TerminalOutcomeFailed.String()); err != nil {
		return fmt.Errorf("backfill suite_executions.terminal_outcome: %w", err)
	}
	return nil
}

// columnExists reports whether a table already has the named column, via
// PRAGMA table_info introspection (SQLite-portable, no information_schema).
func columnExists(ctx context.Context, db dbexec.Executor, table, column string) (bool, error) {
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
