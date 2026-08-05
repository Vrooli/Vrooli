package journal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	vectorcodec "vrooli-memory/internal/vector"
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
	if exists, err := tableExists(ctx, db, "embeddings"); err != nil {
		return err
	} else if exists {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('embeddings') WHERE name='vector_blob'`).Scan(&count); err != nil {
			return fmt.Errorf("inspect embeddings.vector_blob: %w", err)
		}
		if count == 0 {
			if _, err := db.ExecContext(ctx, `ALTER TABLE embeddings ADD COLUMN vector_blob BLOB NOT NULL DEFAULT X''`); err != nil {
				return fmt.Errorf("add embeddings.vector_blob: %w", err)
			}
		}
		rows, err := db.QueryContext(ctx, `SELECT id,vector_json FROM embeddings WHERE length(vector_blob)=0 AND vector_json<>''`)
		if err != nil {
			return err
		}
		defer rows.Close()
		legacy := make([]struct{ id, raw string }, 0)
		for rows.Next() {
			var id, raw string
			if err := rows.Scan(&id, &raw); err != nil {
				rows.Close()
				return err
			}
			legacy = append(legacy, struct{ id, raw string }{id: id, raw: raw})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
		for _, item := range legacy {
			var values []float64
			if err := json.Unmarshal([]byte(item.raw), &values); err != nil {
				return fmt.Errorf("decode legacy embedding %s: %w", item.id, err)
			}
			if _, err := db.ExecContext(ctx, `UPDATE embeddings SET vector_blob=?,vector_json='' WHERE id=?`, vectorcodec.Encode(values), item.id); err != nil {
				return err
			}
		}
	}
	return nil
}

func tableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&n); err != nil {
		return false, fmt.Errorf("inspect %s: %w", table, err)
	}
	return n != 0, nil
}
