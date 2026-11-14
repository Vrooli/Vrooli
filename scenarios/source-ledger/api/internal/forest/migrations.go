package forest

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	vectorcodec "source-ledger/internal/vector"
)

// EnsureMigrations adds cache columns without recreating the derived forest.
// Existing summaries receive an empty vector and become rankable again on the
// next rebuild or compaction pass.
func EnsureMigrations(ctx context.Context, db *sql.DB) error {
	var exists int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='summaries'`).Scan(&exists); err != nil {
		return fmt.Errorf("inspect summaries table: %w", err)
	}
	if exists == 0 {
		return nil
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('summaries') WHERE name='vector_json'`).Scan(&count); err != nil {
		return fmt.Errorf("inspect summaries.vector_json: %w", err)
	}
	if count == 0 {
		if _, err := db.ExecContext(ctx, `ALTER TABLE summaries ADD COLUMN vector_json TEXT NOT NULL DEFAULT '[]'`); err != nil {
			return fmt.Errorf("add summaries.vector_json: %w", err)
		}
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('summaries') WHERE name='vector_blob'`).Scan(&count); err != nil {
		return fmt.Errorf("inspect summaries.vector_blob: %w", err)
	}
	if count == 0 {
		if _, err := db.ExecContext(ctx, `ALTER TABLE summaries ADD COLUMN vector_blob BLOB NOT NULL DEFAULT X''`); err != nil {
			return fmt.Errorf("add summaries.vector_blob: %w", err)
		}
	}
	rows, err := db.QueryContext(ctx, `SELECT id,vector_json FROM summaries WHERE length(vector_blob)=0 AND vector_json<>''`)
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
			return fmt.Errorf("decode legacy summary %s: %w", item.id, err)
		}
		if _, err := db.ExecContext(ctx, `UPDATE summaries SET vector_blob=?,vector_json='' WHERE id=?`, vectorcodec.Encode(values), item.id); err != nil {
			return err
		}
	}
	return nil
}
