package reviewjobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"git-control-tower/internal/dbschema"
	"git-control-tower/internal/gitpaths"
)

func (s *Store) ensureWorktreeQueue(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS gct_worktree_suggestion_queues (repository_id TEXT PRIMARY KEY, revision INTEGER NOT NULL DEFAULT 0, next_sequence INTEGER NOT NULL DEFAULT 1)`); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS gct_worktree_suggestion_queue_entries (
		id TEXT PRIMARY KEY, repository_id TEXT NOT NULL, job_id TEXT, snapshot_digest TEXT NOT NULL,
		paths_json TEXT NOT NULL, mode INTEGER NOT NULL, request_key TEXT NOT NULL, sequence INTEGER NOT NULL,
		state INTEGER NOT NULL, failure_reason TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
		UNIQUE(repository_id, request_key), UNIQUE(repository_id, sequence))`)
	if err != nil {
		return err
	}
	if err := migrateWorktreeQueueMembership(ctx, s.db); err != nil {
		return err
	}
	if err := migrateWorktreeQueueJobOptional(ctx, s.db); err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS gct_worktree_selection_payloads (entry_id TEXT PRIMARY KEY, selection_id TEXT NOT NULL UNIQUE, source INTEGER NOT NULL, payload_json TEXT NOT NULL, FOREIGN KEY(entry_id) REFERENCES gct_worktree_suggestion_queue_entries(id) ON DELETE CASCADE)`)
	return err
}

// migrateWorktreeQueueJobOptional lets a selection own an ordered queue row
// before an AI job exists. Existing rows retain their identity and sequence.
func migrateWorktreeQueueJobOptional(ctx context.Context, db dbschema.DB) error {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(gct_worktree_suggestion_queue_entries)`)
	if err != nil {
		return err
	}
	jobNotNull := false
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var def any
		if err = rows.Scan(&cid, &name, &typ, &notNull, &def, &pk); err != nil {
			rows.Close()
			return err
		}
		if name == "job_id" {
			jobNotNull = notNull != 0
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err = rows.Close(); err != nil {
		return err
	}
	if !jobNotNull {
		_, err = db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_gct_worktree_queue_job ON gct_worktree_suggestion_queue_entries(repository_id,job_id) WHERE job_id IS NOT NULL AND job_id<>''`)
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `ALTER TABLE gct_worktree_suggestion_queue_entries RENAME TO gct_worktree_suggestion_queue_entries_old`); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `CREATE TABLE gct_worktree_suggestion_queue_entries (
		id TEXT PRIMARY KEY, repository_id TEXT NOT NULL, job_id TEXT, snapshot_digest TEXT NOT NULL,
		paths_json TEXT NOT NULL, mode INTEGER NOT NULL, request_key TEXT NOT NULL, sequence INTEGER NOT NULL,
		state INTEGER NOT NULL, failure_reason TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
		UNIQUE(repository_id, request_key), UNIQUE(repository_id, sequence))`)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO gct_worktree_suggestion_queue_entries SELECT * FROM gct_worktree_suggestion_queue_entries_old`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DROP TABLE gct_worktree_suggestion_queue_entries_old`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_gct_worktree_queue_job ON gct_worktree_suggestion_queue_entries(repository_id,job_id) WHERE job_id IS NOT NULL AND job_id<>''`); err != nil {
		return err
	}
	return tx.Commit()
}

// migrateWorktreeQueueMembership folds the legacy active/disposition pair into
// the queue lifecycle state. All rows and their prior meaning are preserved.
func migrateWorktreeQueueMembership(ctx context.Context, db dbschema.DB) error {
	columns := map[string]bool{}
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(gct_worktree_suggestion_queue_entries)`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err = rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			return err
		}
		columns[name] = true
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err = rows.Close(); err != nil {
		return err
	}
	activeExists, dispositionExists := columns["active"], columns["disposition"]
	if !activeExists && !dispositionExists {
		return nil
	}
	if !activeExists || !dispositionExists {
		return fmt.Errorf("incomplete legacy worktree queue membership columns")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var invalidRows int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM gct_worktree_suggestion_queue_entries WHERE state NOT BETWEEN ? AND ? OR active NOT IN (0,1) OR disposition NOT IN (0,1,2) OR (active=1 AND disposition<>0) OR (active=0 AND disposition NOT IN (1,2))`, WorktreeQueueQueued, WorktreeQueueCommitPending).Scan(&invalidRows); err != nil {
		return err
	}
	if invalidRows != 0 {
		return fmt.Errorf("cannot migrate %d worktree queue rows with inconsistent legacy membership", invalidRows)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE gct_worktree_suggestion_queue_entries SET state=CASE disposition WHEN 1 THEN ? WHEN 2 THEN ? ELSE state END WHERE active=0`, WorktreeQueueCommitted, WorktreeQueueRemoved); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `ALTER TABLE gct_worktree_suggestion_queue_entries DROP COLUMN disposition`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `ALTER TABLE gct_worktree_suggestion_queue_entries DROP COLUMN active`); err != nil {
		return err
	}
	return tx.Commit()
}

func ensureWorktreePathsAvailable(ctx context.Context, tx *sql.Tx, repo, replacingID string, requested []string) error {
	if err := validateWorktreeQueuePaths(requested); err != nil {
		return err
	}
	wanted := make(map[string]struct{}, len(requested))
	for _, path := range requested {
		wanted[path] = struct{}{}
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,paths_json FROM gct_worktree_suggestion_queue_entries WHERE repository_id=? AND state NOT IN (?,?)`, repo, WorktreeQueueCommitted, WorktreeQueueRemoved)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, raw string
		if err = rows.Scan(&id, &raw); err != nil {
			return err
		}
		if id == replacingID {
			continue
		}
		var paths []string
		if err = json.Unmarshal([]byte(raw), &paths); err != nil {
			return fmt.Errorf("decode active queue paths: %w", err)
		}
		if err = validateWorktreeQueuePaths(paths); err != nil {
			return fmt.Errorf("validate active queue paths: %w", err)
		}
		for _, path := range paths {
			if _, exists := wanted[path]; exists {
				return ErrQueuePathConflict
			}
		}
	}
	return rows.Err()
}

func validateWorktreeQueuePaths(paths []string) error {
	return gitpaths.ValidateRepositoryRelativePaths(paths)
}

func sameWorktreeQueuePaths(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
