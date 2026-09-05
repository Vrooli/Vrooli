package golden

import (
	"context"
	"fmt"
)

var addedColumns = []struct {
	name string
	ddl  string
}{
	{name: "generation_options_json", ddl: "ALTER TABLE goldens ADD COLUMN generation_options_json TEXT NOT NULL DEFAULT ''"},
	{name: "materialization_mode", ddl: "ALTER TABLE goldens ADD COLUMN materialization_mode TEXT NOT NULL DEFAULT 'ephemeral'"},
	{name: "logical_root", ddl: "ALTER TABLE goldens ADD COLUMN logical_root TEXT NOT NULL DEFAULT ''"},
	{name: "last_materialized_path", ddl: "ALTER TABLE goldens ADD COLUMN last_materialized_path TEXT NOT NULL DEFAULT ''"},
	{name: "last_materialized_status", ddl: "ALTER TABLE goldens ADD COLUMN last_materialized_status TEXT NOT NULL DEFAULT 'never'"},
}

// EnsureColumns applies additive golden table migrations. Call after the base
// schema has been created; safe for fresh and existing databases.
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
			return fmt.Errorf("add goldens column %q: %w", col.name, err)
		}
	}
	return nil
}

func tableColumns(ctx context.Context, db SQLExecutor) (map[string]struct{}, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(goldens)")
	if err != nil {
		return nil, fmt.Errorf("introspect goldens columns: %w", err)
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
			return nil, fmt.Errorf("scan goldens table_info row: %w", err)
		}
		cols[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate goldens table_info rows: %w", err)
	}
	return cols, nil
}
