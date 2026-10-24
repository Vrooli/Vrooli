package destinations

import (
	"context"
	"fmt"
)

// addedColumns lists columns introduced after the initial destinations table
// shipped. SQLite has no "ADD COLUMN IF NOT EXISTS", so EnsureColumns
// introspects PRAGMA table_info and adds only the missing ones — keeping boot
// idempotent on both fresh and existing databases. New columns must also be
// present in schema.sql so a fresh CREATE TABLE includes them.
var addedColumns = []struct {
	name string
	ddl  string
}{
	{name: "relative_path", ddl: "ALTER TABLE destinations ADD COLUMN relative_path TEXT NOT NULL DEFAULT ''"},
	{name: "repository_location", ddl: "ALTER TABLE destinations ADD COLUMN repository_location TEXT NOT NULL DEFAULT ''"},
	{name: "device_path", ddl: "ALTER TABLE destinations ADD COLUMN device_path TEXT NOT NULL DEFAULT ''"},
	{name: "device_mountpoint", ddl: "ALTER TABLE destinations ADD COLUMN device_mountpoint TEXT NOT NULL DEFAULT ''"},
	{name: "device_label", ddl: "ALTER TABLE destinations ADD COLUMN device_label TEXT NOT NULL DEFAULT ''"},
	{name: "device_filesystem", ddl: "ALTER TABLE destinations ADD COLUMN device_filesystem TEXT NOT NULL DEFAULT ''"},
	{name: "device_total_bytes", ddl: "ALTER TABLE destinations ADD COLUMN device_total_bytes INTEGER NOT NULL DEFAULT 0"},
	{name: "device_model", ddl: "ALTER TABLE destinations ADD COLUMN device_model TEXT NOT NULL DEFAULT ''"},
	{name: "device_serial", ddl: "ALTER TABLE destinations ADD COLUMN device_serial TEXT NOT NULL DEFAULT ''"},
	{name: "device_uuid", ddl: "ALTER TABLE destinations ADD COLUMN device_uuid TEXT NOT NULL DEFAULT ''"},
	{name: "device_observed_at", ddl: "ALTER TABLE destinations ADD COLUMN device_observed_at TEXT NOT NULL DEFAULT ''"},
}

// EnsureColumns applies additive column migrations to an existing destinations
// table. It is safe to call on every boot and on a freshly created table (the
// columns already exist, so nothing is added). Call it after EnsureSchemas has
// created the base table.
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
			return fmt.Errorf("add destinations column %q: %w", col.name, err)
		}
	}
	return nil
}

func tableColumns(ctx context.Context, db SQLExecutor) (map[string]struct{}, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(destinations)")
	if err != nil {
		return nil, fmt.Errorf("introspect destinations columns: %w", err)
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
