package validation_record

import (
	"context"
	"fmt"
)

// addedColumns lists the columns introduced after the initial
// validation_records table shipped. SQLite has no "ADD COLUMN IF NOT
// EXISTS", so EnsureColumns introspects PRAGMA table_info and adds only
// the missing ones — keeping boot idempotent on both fresh and existing
// databases. New columns must also be present in schema.sql so a fresh
// CREATE TABLE includes them.
var addedColumns = []struct {
	name string
	ddl  string
}{
	{name: "tool_detail", ddl: "ALTER TABLE validation_records ADD COLUMN tool_detail TEXT NOT NULL DEFAULT ''"},
	{name: "tool_raw_output", ddl: "ALTER TABLE validation_records ADD COLUMN tool_raw_output TEXT NOT NULL DEFAULT ''"},
}

// EnsureColumns applies additive column migrations to an existing
// validation_records table. It is safe to call on every boot and on a
// freshly created table (the columns already exist, so nothing is added).
// Call it after EnsureSchemas has created the base table.
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
			return fmt.Errorf("add validation_records column %q: %w", col.name, err)
		}
	}
	return nil
}

func tableColumns(ctx context.Context, db SQLExecutor) (map[string]struct{}, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(validation_records)")
	if err != nil {
		return nil, fmt.Errorf("introspect validation_records columns: %w", err)
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
