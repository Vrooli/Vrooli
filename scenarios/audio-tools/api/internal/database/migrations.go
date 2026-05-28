package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// migrationExecer is the minimal database surface ApplyMigrations needs.
// *sql.DB satisfies it directly; tests supply a fake.
type migrationExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

var _ migrationExecer = (*sql.DB)(nil)

// forwardOnlyAlters are additive ADD COLUMN statements applied AFTER the
// declarative schema (EnsureSchemas). The declarative schema already lists
// these columns in CREATE TABLE for fresh databases; these ALTERs bring an
// already-created table up to the current shape.
//
// SQLite has no `ADD COLUMN IF NOT EXISTS`, so re-running an ALTER on a column
// that already exists raises "duplicate column name". That specific error is
// the success signal for an already-migrated database and is swallowed; any
// other error is fatal. This is a flat forward-only list, not a versioned
// migration framework — add a line when a column is introduced.
var forwardOnlyAlters = []string{
	`ALTER TABLE speaker_profiles ADD COLUMN clip_count INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE speaker_profiles ADD COLUMN total_voiced_seconds REAL NOT NULL DEFAULT 0`,
	`ALTER TABLE speaker_profiles ADD COLUMN sample_rate INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE speaker_profiles ADD COLUMN embedding_dim INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE speaker_profiles ADD COLUMN model_name TEXT NOT NULL DEFAULT ''`,
}

// ApplyMigrations runs the forward-only ALTERs against db. It is idempotent:
// the "duplicate column name" error from an already-applied ALTER is treated
// as success. Call it once at boot, after EnsureSchemas.
func ApplyMigrations(ctx context.Context, db migrationExecer) error {
	for _, stmt := range forwardOnlyAlters {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
				continue
			}
			return fmt.Errorf("apply migration %q: %w", stmt, err)
		}
	}
	return nil
}
