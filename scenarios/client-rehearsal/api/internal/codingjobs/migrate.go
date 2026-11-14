package codingjobs

import (
	"context"
	"fmt"
)

// EnsureReplayKeySchema upgrades an existing coding_jobs table created before
// replay-key recovery was introduced. The client-rehearsal owner is SQLite
// backed; this runs once during module construction before the API is served.
func EnsureReplayKeySchema(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(coding_jobs)`)
	if err != nil {
		return fmt.Errorf("inspect coding job schema: %w", err)
	}
	hasReplayKey := false
	for rows.Next() {
		var cid, notNull, primary int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primary); err != nil {
			_ = rows.Close()
			return fmt.Errorf("read coding job schema: %w", err)
		}
		if name == "replay_key" {
			hasReplayKey = true
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterate coding job schema: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close coding job schema rows: %w", err)
	}
	if !hasReplayKey {
		if _, err := db.ExecContext(ctx, `ALTER TABLE coding_jobs ADD COLUMN replay_key TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("add coding job replay key: %w", err)
		}
	}
	if _, err := db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_coding_jobs_replay_key ON coding_jobs(replay_key) WHERE replay_key <> ''`); err != nil {
		return fmt.Errorf("index coding job replay keys: %w", err)
	}
	return nil
}
