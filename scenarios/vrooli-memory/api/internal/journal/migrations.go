package journal

import (
	"context"
	"database/sql"
	"fmt"
)

// EnsureMigrations applies additive, idempotent upgrades that CREATE TABLE IF
// NOT EXISTS cannot apply to an already-created SQLite table.
func EnsureMigrations(ctx context.Context, db *sql.DB) error {
	var exists int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='entries'`).Scan(&exists); err != nil {
		return fmt.Errorf("inspect entries table: %w", err)
	}
	if exists == 0 {
		return nil
	}
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(entries)`)
	if err != nil {
		return fmt.Errorf("inspect entries schema: %w", err)
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for name, statement := range map[string]string{
		"source_harness": `ALTER TABLE entries ADD COLUMN source_harness TEXT NOT NULL DEFAULT ''`,
		"source_path":    `ALTER TABLE entries ADD COLUMN source_path TEXT NOT NULL DEFAULT ''`,
		"imported_at":    `ALTER TABLE entries ADD COLUMN imported_at TEXT NOT NULL DEFAULT ''`,
	} {
		if !columns[name] {
			if _, err := db.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("add entries.%s: %w", name, err)
			}
		}
	}
	return nil
}
