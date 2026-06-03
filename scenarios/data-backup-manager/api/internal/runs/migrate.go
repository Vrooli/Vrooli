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
	return nil
}

func tableColumns(ctx context.Context, db SQLExecutor) (map[string]struct{}, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(runs)")
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
