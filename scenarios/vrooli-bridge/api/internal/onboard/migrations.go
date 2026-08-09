package onboard

import (
	"context"
	"fmt"
)

// Migrate brings a pre-existing onboarding_ops table up to the current column
// shape without recreating it — the durable op history is the source of truth a
// re-attaching client reads and must never be dropped. It is guarded and
// idempotent, so it is safe to run on every boot:
//
//   - A fresh DB has no table yet (Migrate runs BEFORE EnsureSchemas): it skips,
//     and EnsureSchemas' CREATE TABLE brings in the column.
//   - A DB created before failure_detail existed: Migrate adds the column, so
//     EnsureSchemas' declared-column drift check (which runs inside EnsureSchemas)
//     then passes.
//   - A DB already at shape: every step no-ops.
//
// This is the onboard domain's column-evolution hook — the minimal,
// data-preserving substrate (PRAGMA introspect → ALTER ADD COLUMN), matching the
// execution domain's Migrate in test-genie.
func Migrate(ctx context.Context, db SQLExecutor) error {
	exists, err := tableExists(ctx, db, "onboarding_ops")
	if err != nil {
		return fmt.Errorf("introspect onboarding_ops: %w", err)
	}
	if !exists {
		// Fresh DB — EnsureSchemas' CREATE TABLE will include the column.
		return nil
	}
	for _, column := range []string{"failure_detail", "control_plane_url", "reachability_mode", "correlation_id", "source_mode"} {
		has, err := columnExists(ctx, db, "onboarding_ops", column)
		if err != nil {
			return fmt.Errorf("introspect onboarding_ops.%s: %w", column, err)
		}
		if has {
			continue
		}
		defaultValue := "''"
		if column == "source_mode" {
			defaultValue = "'pinned'"
		}
		if _, err := db.ExecContext(ctx,
			fmt.Sprintf("ALTER TABLE onboarding_ops ADD COLUMN %s TEXT NOT NULL DEFAULT %s", column, defaultValue)); err != nil {
			return fmt.Errorf("add onboarding_ops.%s: %w", column, err)
		}
	}
	return nil
}

// tableExists reports whether a table is present, via sqlite_master (SQLite-
// portable). PRAGMA table_info on a missing table returns zero rows with no
// error, so it cannot distinguish "absent" from "empty" — sqlite_master can.
func tableExists(ctx context.Context, db SQLExecutor, table string) (bool, error) {
	row := db.QueryRowContext(ctx,
		"SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?", table)
	var one int
	switch err := row.Scan(&one); err {
	case nil:
		return true, nil
	default:
		// sql.ErrNoRows (absent) reads as false; any other error would surface on
		// the next query, but we treat a scan miss as "absent" to stay guarded.
		return false, nil
	}
}

// columnExists reports whether a table already has the named column, via PRAGMA
// table_info introspection (SQLite-portable, no information_schema).
func columnExists(ctx context.Context, db SQLExecutor, table, column string) (bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%q)", table))
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
