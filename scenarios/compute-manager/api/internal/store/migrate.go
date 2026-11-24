package store

import (
	"context"
	"database/sql"
	"encoding/json"
)

// MigrateCompatibility repairs recoverable fields when an additive schema
// change meets an existing database. Unknown sources remain empty; migration
// never infers or fabricates a billing source.
func MigrateCompatibility(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Existing instances predate the common idempotency binding. Probe through
	// SQL rather than a SQLite-specific PRAGMA so the additive migration works
	// with each supported database engine.
	if rows, probeErr := tx.QueryContext(ctx, `SELECT spend_idempotency_key FROM instances LIMIT 0`); probeErr != nil {
		if _, err := tx.ExecContext(ctx, `ALTER TABLE instances ADD COLUMN spend_idempotency_key TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	} else if err := rows.Close(); err != nil {
		return err
	}

	rows, err := tx.QueryContext(ctx, `SELECT id,quote_json FROM spend_quotes WHERE source_url=''`)
	if err != nil {
		return err
	}
	type sourceRow struct{ id, source string }
	var backfills []sourceRow
	for rows.Next() {
		var id, raw string
		if err := rows.Scan(&id, &raw); err != nil {
			rows.Close()
			return err
		}
		var payload struct {
			SourceURL string `json:"source_url"`
		}
		if json.Unmarshal([]byte(raw), &payload) == nil && payload.SourceURL != "" {
			backfills = append(backfills, sourceRow{id: id, source: payload.SourceURL})
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, item := range backfills {
		if _, err := tx.ExecContext(ctx, `UPDATE spend_quotes SET source_url=? WHERE id=? AND source_url=''`, item.source, item.id); err != nil {
			return err
		}
	}
	return tx.Commit()
}
