package components

import (
	"context"
	"database/sql"
	"fmt"
)

type schemaMigrator interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// EnsureSchemaMigrations applies additive SQLite migrations that cannot
// be expressed idempotently in schema.sql. Run before the declarative
// Schema() provider so api-core's drift check validates the final shape.
func EnsureSchemaMigrations(ctx context.Context, db schemaMigrator) error {
	migrations := []struct {
		table  string
		column string
		sql    string
	}{
		{
			table:  "components",
			column: "category",
			sql:    `ALTER TABLE components ADD COLUMN category TEXT NOT NULL DEFAULT '';`,
		},
		{
			table:  "component_design_affinities",
			column: "reason",
			sql:    `ALTER TABLE component_design_affinities ADD COLUMN reason TEXT NOT NULL DEFAULT '';`,
		},
		{
			table:  "component_version_files",
			column: "slot",
			sql:    `ALTER TABLE component_version_files ADD COLUMN slot TEXT NOT NULL DEFAULT '';`,
		},
	}
	for _, m := range migrations {
		has, err := tableHasColumn(ctx, db, m.table, m.column)
		if err != nil {
			return err
		}
		if has {
			continue
		}
		exists, err := tableExists(ctx, db, m.table)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if _, err := db.ExecContext(ctx, m.sql); err != nil {
			return fmt.Errorf("migrate %s.%s: %w", m.table, m.column, err)
		}
	}
	return nil
}

func tableExists(ctx context.Context, db schemaMigrator, table string) (bool, error) {
	cols, err := tableColumns(ctx, db, table)
	if err != nil {
		return false, err
	}
	return len(cols) > 0, nil
}

func tableHasColumn(ctx context.Context, db schemaMigrator, table, column string) (bool, error) {
	cols, err := tableColumns(ctx, db, table)
	if err != nil {
		return false, err
	}
	return cols[column], nil
}

func tableColumns(ctx context.Context, db schemaMigrator, table string) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%q)", table))
	if err != nil {
		return nil, fmt.Errorf("inspect %s columns: %w", table, err)
	}
	defer rows.Close()

	cols := map[string]bool{}
	for rows.Next() {
		var (
			cid     int
			name    string
			ctype   string
			notnull int
			dflt    sql.NullString
			pk      int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, fmt.Errorf("scan %s columns: %w", table, err)
		}
		cols[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read %s columns: %w", table, err)
	}
	return cols, nil
}
