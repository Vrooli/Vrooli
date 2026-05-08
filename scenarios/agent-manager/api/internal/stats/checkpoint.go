// SQLite-backed CheckpointStore.
//
// Persists Engine.watermark across restarts so Rebuild resumes from the
// saved rowid instead of replaying the entire run_events table. Driven
// off a single (name, last_rowid, updated_at) row per engine.

package stats

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// SQLiteCheckpointStore implements CheckpointStore on top of the
// agent-manager primary SQLite database. Schema lives in
// database/schema.sql under the "stats_checkpoint" table.
type SQLiteCheckpointStore struct {
	db *sqlx.DB
}

// NewSQLiteCheckpointStore wraps a sqlx handle.
func NewSQLiteCheckpointStore(db *sqlx.DB) *SQLiteCheckpointStore {
	return &SQLiteCheckpointStore{db: db}
}

// Load returns the saved rowid, or 0 if no checkpoint has been persisted yet.
func (s *SQLiteCheckpointStore) Load(ctx context.Context, name string) (int64, error) {
	var rowid int64
	err := s.db.GetContext(ctx, &rowid, `SELECT last_rowid FROM stats_checkpoint WHERE name = ?`, name)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("stats checkpoint load: %w", err)
	}
	return rowid, nil
}

// Save upserts the watermark for the given engine name. Idempotent.
func (s *SQLiteCheckpointStore) Save(ctx context.Context, name string, rowid int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO stats_checkpoint (name, last_rowid, updated_at)
		VALUES (?, ?, datetime('now'))
		ON CONFLICT(name) DO UPDATE SET
			last_rowid = excluded.last_rowid,
			updated_at = excluded.updated_at
	`, name, rowid)
	if err != nil {
		return fmt.Errorf("stats checkpoint save: %w", err)
	}
	return nil
}

// ensure SQLiteCheckpointStore satisfies CheckpointStore.
var _ CheckpointStore = (*SQLiteCheckpointStore)(nil)
