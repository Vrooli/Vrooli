package reviewjobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	pathpkg "path"
	"strings"
	"time"
	"unicode/utf8"

	"git-control-tower/internal/dbschema"
)

var (
	ErrQueueConflict     = errors.New("worktree suggestion queue conflict")
	ErrQueuePathConflict = errors.New("a path already belongs to another active queue entry")
)

type WorktreeQueueEntry struct {
	ID, RepositoryID, JobID, SnapshotDigest string
	SelectionID                             string
	SelectionSource                         int32
	SelectionPayload                        string
	Paths                                   []string
	Mode                                    int32
	RequestKey                              string
	Sequence                                uint64
	State                                   WorktreeQueueState
	FailureReason                           string
	CreatedAt, UpdatedAt                    time.Time
}

// WorktreeQueueState owns the entry lifecycle. Values through commit-pending
// are the protobuf contract; committed and removed are internal persisted
// terminal values that replace the legacy membership flags.
type WorktreeQueueState int32

const (
	WorktreeQueueQueued WorktreeQueueState = iota + 1
	WorktreeQueueStaged
	WorktreeQueueFailed
	WorktreeQueueStale
	WorktreeQueueCommitPending
	WorktreeQueueCommitted
	WorktreeQueueRemoved
	WorktreeQueueAwaitingUnstage
)

func (state WorktreeQueueState) Valid() bool {
	return state >= WorktreeQueueQueued && state <= WorktreeQueueAwaitingUnstage
}

func (state WorktreeQueueState) Active() bool {
	return state.Valid() && state != WorktreeQueueCommitted && state != WorktreeQueueRemoved
}

type WorktreeQueueDisposition int32

const (
	WorktreeQueueDispositionCommitted WorktreeQueueDisposition = iota + 1
	WorktreeQueueDispositionRemoved
)

func (state WorktreeQueueState) CanTransitionTo(next WorktreeQueueState) bool {
	switch state {
	case WorktreeQueueQueued:
		return next == WorktreeQueueStaged || next == WorktreeQueueFailed || next == WorktreeQueueStale || next == WorktreeQueueCommitPending || next == WorktreeQueueRemoved || next == WorktreeQueueAwaitingUnstage
	case WorktreeQueueAwaitingUnstage:
		return next == WorktreeQueueQueued || next == WorktreeQueueFailed || next == WorktreeQueueStale || next == WorktreeQueueRemoved
	case WorktreeQueueStaged:
		return next == WorktreeQueueQueued || next == WorktreeQueueCommitPending || next == WorktreeQueueRemoved
	case WorktreeQueueFailed, WorktreeQueueStale:
		return next == WorktreeQueueQueued || next == WorktreeQueueRemoved
	case WorktreeQueueCommitPending:
		return next == WorktreeQueueQueued || next == WorktreeQueueStaged || next == WorktreeQueueCommitted
	default:
		return false
	}
}

type WorktreeQueue struct {
	RepositoryID string
	Revision     uint64
	Entries      []WorktreeQueueEntry
}

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
	if len(paths) == 0 {
		return fmt.Errorf("at least one worktree path is required")
	}
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if path == "" || !utf8.ValidString(path) || strings.ContainsRune(path, '\x00') || path == "." || path == ".." || strings.HasPrefix(path, "../") || strings.HasPrefix(path, "/") || pathpkg.Clean(path) != path {
			return fmt.Errorf("worktree paths must be canonical repository-relative Git paths")
		}
		if _, exists := seen[path]; exists {
			return fmt.Errorf("worktree path %q occurs more than once", path)
		}
		seen[path] = struct{}{}
	}
	return nil
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

func (s *Store) ListWorktreeQueue(ctx context.Context, repo string) (WorktreeQueue, error) {
	if strings.TrimSpace(repo) == "" {
		return WorktreeQueue{}, fmt.Errorf("repository id is required")
	}
	if err := s.ensureWorktreeQueue(ctx); err != nil {
		return WorktreeQueue{}, err
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO gct_worktree_suggestion_queues(repository_id) VALUES(?) ON CONFLICT(repository_id) DO NOTHING`, repo); err != nil {
		return WorktreeQueue{}, err
	}
	var out WorktreeQueue
	out.RepositoryID = repo
	if err := s.db.QueryRowContext(ctx, `SELECT revision FROM gct_worktree_suggestion_queues WHERE repository_id=?`, repo).Scan(&out.Revision); err != nil {
		return WorktreeQueue{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT q.id,q.job_id,q.snapshot_digest,q.paths_json,q.mode,q.request_key,q.sequence,q.state,q.failure_reason,q.created_at,q.updated_at,COALESCE(p.selection_id,''),COALESCE(p.source,0),COALESCE(p.payload_json,'') FROM gct_worktree_suggestion_queue_entries q LEFT JOIN gct_worktree_selection_payloads p ON p.entry_id=q.id WHERE q.repository_id=? AND q.state NOT IN (?,?) ORDER BY q.sequence ASC`, repo, WorktreeQueueCommitted, WorktreeQueueRemoved)
	if err != nil {
		return WorktreeQueue{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var e WorktreeQueueEntry
		var paths, created, updated string
		e.RepositoryID = repo
		if err = rows.Scan(&e.ID, &e.JobID, &e.SnapshotDigest, &paths, &e.Mode, &e.RequestKey, &e.Sequence, &e.State, &e.FailureReason, &created, &updated, &e.SelectionID, &e.SelectionSource, &e.SelectionPayload); err != nil {
			return WorktreeQueue{}, err
		}
		if !e.State.Active() {
			return WorktreeQueue{}, fmt.Errorf("queue entry %q has invalid active state %d", e.ID, e.State)
		}
		if err = json.Unmarshal([]byte(paths), &e.Paths); err != nil {
			return WorktreeQueue{}, err
		}
		if err = validateWorktreeQueuePaths(e.Paths); err != nil {
			return WorktreeQueue{}, fmt.Errorf("validate queue entry %q paths: %w", e.ID, err)
		}
		e.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return WorktreeQueue{}, err
		}
		e.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
		if err != nil {
			return WorktreeQueue{}, err
		}
		out.Entries = append(out.Entries, e)
	}
	return out, rows.Err()
}

func (s *Store) EnqueueWorktreeQueue(ctx context.Context, entry WorktreeQueueEntry) (WorktreeQueueEntry, uint64, error) {
	if entry.RepositoryID == "" || entry.JobID == "" || entry.SnapshotDigest == "" || entry.RequestKey == "" || entry.ID == "" || len(entry.Paths) == 0 {
		return WorktreeQueueEntry{}, 0, fmt.Errorf("complete worktree queue entry is required")
	}
	if err := validateWorktreeQueuePaths(entry.Paths); err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	if err := s.ensureWorktreeQueue(ctx); err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	defer tx.Rollback()
	queued, revision, err := enqueueWorktreeQueueTx(ctx, tx, entry)
	if err != nil {
		return WorktreeQueueEntry{}, revision, err
	}
	if err = tx.Commit(); err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	return queued, revision, nil
}

// EnqueueCapturedSelection records immutable capture bytes in the authoritative
// ordered queue without creating or dispatching a review job.
func (s *Store) EnqueueCapturedSelection(ctx context.Context, entry WorktreeQueueEntry) (WorktreeQueueEntry, uint64, error) {
	if entry.JobID != "" || entry.SelectionID == "" || entry.SelectionPayload == "" || (entry.State != WorktreeQueueQueued && entry.State != WorktreeQueueAwaitingUnstage) {
		return WorktreeQueueEntry{}, 0, fmt.Errorf("captured queue selection identity and payload are required")
	}
	if err := s.ensureWorktreeQueue(ctx); err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	defer tx.Rollback()
	queued, rev, err := enqueueWorktreeQueueTx(ctx, tx, entry)
	if err != nil {
		return WorktreeQueueEntry{}, rev, err
	}
	if err = tx.Commit(); err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	return queued, rev, nil
}

// FinalizeCapturedSelection is idempotent after unstage succeeds. It changes
// only the lifecycle state and payload, preserving the entry's queue position.
func (s *Store) FinalizeCapturedSelection(ctx context.Context, repo, id, digest, payload string) (WorktreeQueue, error) {
	if err := s.ensureWorktreeQueue(ctx); err != nil {
		return WorktreeQueue{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return WorktreeQueue{}, err
	}
	defer tx.Rollback()
	var got string
	var state WorktreeQueueState
	if err = tx.QueryRowContext(ctx, `SELECT snapshot_digest,state FROM gct_worktree_suggestion_queue_entries WHERE repository_id=? AND id=?`, repo, id).Scan(&got, &state); err != nil {
		return WorktreeQueue{}, err
	}
	if got != digest {
		return WorktreeQueue{}, ErrQueueConflict
	}
	if state != WorktreeQueueAwaitingUnstage && state != WorktreeQueueQueued {
		return WorktreeQueue{}, ErrQueueConflict
	}
	if state == WorktreeQueueAwaitingUnstage {
		if _, err = tx.ExecContext(ctx, `UPDATE gct_worktree_suggestion_queue_entries SET state=?,updated_at=? WHERE repository_id=? AND id=?`, WorktreeQueueQueued, time.Now().UTC().Format(time.RFC3339Nano), repo, id); err != nil {
			return WorktreeQueue{}, err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE gct_worktree_selection_payloads SET payload_json=? WHERE entry_id=?`, payload, id); err != nil {
			return WorktreeQueue{}, err
		}
		if err = incrementQueueRevision(ctx, tx, repo); err != nil {
			return WorktreeQueue{}, err
		}
	}
	q, err := queueFromTx(ctx, tx, repo)
	if err != nil {
		return WorktreeQueue{}, err
	}
	if err = tx.Commit(); err != nil {
		return WorktreeQueue{}, err
	}
	return q, nil
}

func incrementQueueRevision(ctx context.Context, tx *sql.Tx, repo string) error {
	_, err := tx.ExecContext(ctx, `UPDATE gct_worktree_suggestion_queues SET revision=revision+1 WHERE repository_id=?`, repo)
	return err
}

func (s *Store) UpdateWorktreeQueueEntry(ctx context.Context, repo, id, head string, expected uint64, state WorktreeQueueState, reason string) (WorktreeQueue, *WorktreeQueueEntry, error) {
	if !state.Valid() {
		return WorktreeQueue{}, nil, fmt.Errorf("queue entry state must be queued, staged, failed, stale, or commit-pending")
	}
	if state != WorktreeQueueFailed && state != WorktreeQueueStale && reason != "" {
		return WorktreeQueue{}, nil, fmt.Errorf("only failed or stale entries can carry a failure reason")
	}
	if (state == WorktreeQueueFailed || state == WorktreeQueueStale) && strings.TrimSpace(reason) == "" {
		return WorktreeQueue{}, nil, fmt.Errorf("failed and stale entries require a reason")
	}
	tx, err := s.beginQueueMutation(ctx, repo, expected, head)
	if err != nil {
		return WorktreeQueue{}, nil, err
	}
	defer tx.Rollback()
	var old WorktreeQueueState
	var oldReason string
	var entry WorktreeQueueEntry
	err = tx.QueryRowContext(ctx, `SELECT id,job_id,snapshot_digest,mode,request_key,state,failure_reason FROM gct_worktree_suggestion_queue_entries WHERE repository_id=? AND id=?`, repo, id).Scan(&entry.ID, &entry.JobID, &entry.SnapshotDigest, &entry.Mode, &entry.RequestKey, &old, &oldReason)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return WorktreeQueue{}, nil, ErrQueueConflict
		}
		return WorktreeQueue{}, nil, err
	}
	if id != head {
		return WorktreeQueue{}, nil, ErrQueueConflict
	}
	if old == state && oldReason == reason {
		q, e := queueFromTx(ctx, tx, repo)
		if e != nil {
			return WorktreeQueue{}, nil, e
		}
		entry.RepositoryID = repo
		entry.State = old
		entry.FailureReason = oldReason
		return q, &entry, nil
	}
	if old == state {
		now := time.Now().UTC()
		if _, err = tx.ExecContext(ctx, `UPDATE gct_worktree_suggestion_queue_entries SET failure_reason=?,updated_at=? WHERE repository_id=? AND id=? AND state=?`, reason, now.Format(time.RFC3339Nano), repo, id, old); err != nil {
			return WorktreeQueue{}, nil, err
		}
		if err = incQueueRevision(ctx, tx, repo, expected); err != nil {
			return WorktreeQueue{}, nil, err
		}
		q, err := queueFromTx(ctx, tx, repo)
		if err != nil {
			return WorktreeQueue{}, nil, err
		}
		if err = tx.Commit(); err != nil {
			return WorktreeQueue{}, nil, err
		}
		entry.RepositoryID = repo
		entry.State = state
		entry.FailureReason = reason
		entry.UpdatedAt = now
		q.Revision = expected + 1
		return q, &entry, nil
	}
	if !old.CanTransitionTo(state) {
		return WorktreeQueue{}, nil, ErrQueueConflict
	}
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx, `UPDATE gct_worktree_suggestion_queue_entries SET state=?,failure_reason=?,updated_at=? WHERE repository_id=? AND id=? AND state=?`, state, reason, now.Format(time.RFC3339Nano), repo, id, old)
	if err != nil {
		return WorktreeQueue{}, nil, err
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return WorktreeQueue{}, nil, ErrQueueConflict
	}
	if err = incQueueRevision(ctx, tx, repo, expected); err != nil {
		return WorktreeQueue{}, nil, err
	}
	q, err := queueFromTx(ctx, tx, repo)
	if err != nil {
		return WorktreeQueue{}, nil, err
	}
	if err = tx.Commit(); err != nil {
		return WorktreeQueue{}, nil, err
	}
	entry.RepositoryID = repo
	entry.State = state
	entry.FailureReason = reason
	entry.UpdatedAt = now
	q.Revision = expected + 1
	return q, &entry, nil
}

func (s *Store) ReplaceWorktreeQueueEntry(ctx context.Context, repo, id string, expected uint64, replacement WorktreeQueueEntry) (WorktreeQueue, *WorktreeQueueEntry, error) {
	if repo == "" || id == "" || replacement.JobID == "" || replacement.SnapshotDigest == "" || replacement.RequestKey == "" || len(replacement.Paths) == 0 {
		return WorktreeQueue{}, nil, fmt.Errorf("complete worktree replacement is required")
	}
	if err := validateWorktreeQueuePaths(replacement.Paths); err != nil {
		return WorktreeQueue{}, nil, err
	}
	if err := s.ensureWorktreeQueue(ctx); err != nil {
		return WorktreeQueue{}, nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return WorktreeQueue{}, nil, err
	}
	defer tx.Rollback()
	var revision uint64
	if err = tx.QueryRowContext(ctx, `SELECT revision FROM gct_worktree_suggestion_queues WHERE repository_id=?`, repo).Scan(&revision); err != nil {
		return WorktreeQueue{}, nil, err
	}
	var current WorktreeQueueEntry
	var paths, created, updated string
	err = tx.QueryRowContext(ctx, `SELECT id,job_id,snapshot_digest,paths_json,mode,request_key,sequence,state,failure_reason,created_at,updated_at FROM gct_worktree_suggestion_queue_entries WHERE repository_id=? AND id=?`, repo, id).Scan(&current.ID, &current.JobID, &current.SnapshotDigest, &paths, &current.Mode, &current.RequestKey, &current.Sequence, &current.State, &current.FailureReason, &created, &updated)
	if err != nil {
		return WorktreeQueue{}, nil, err
	}
	if err = json.Unmarshal([]byte(paths), &current.Paths); err != nil {
		return WorktreeQueue{}, nil, err
	}
	if err = validateWorktreeQueuePaths(current.Paths); err != nil {
		return WorktreeQueue{}, nil, fmt.Errorf("validate queue entry %q paths: %w", id, err)
	}
	current.RepositoryID = repo
	current.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return WorktreeQueue{}, nil, fmt.Errorf("parse queue entry created_at: %w", err)
	}
	current.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	if err != nil {
		return WorktreeQueue{}, nil, fmt.Errorf("parse queue entry updated_at: %w", err)
	}
	if current.JobID == replacement.JobID && current.SnapshotDigest == replacement.SnapshotDigest && current.Mode == replacement.Mode && current.RequestKey == replacement.RequestKey && sameWorktreeQueuePaths(current.Paths, replacement.Paths) {
		q, e := queueFromTx(ctx, tx, repo)
		if e != nil {
			return WorktreeQueue{}, nil, e
		}
		if e = tx.Commit(); e != nil {
			return WorktreeQueue{}, nil, e
		}
		return q, &current, nil
	}
	if revision != expected {
		return WorktreeQueue{}, nil, ErrQueueConflict
	}
	if current.State == WorktreeQueueCommitPending {
		return WorktreeQueue{}, nil, fmt.Errorf("commit-pending queue head cannot be replaced: %w", ErrQueueConflict)
	}
	if err = ensureWorktreePathsAvailable(ctx, tx, repo, id, replacement.Paths); err != nil {
		return WorktreeQueue{}, nil, err
	}
	raw, err := json.Marshal(replacement.Paths)
	if err != nil {
		return WorktreeQueue{}, nil, err
	}
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx, `UPDATE gct_worktree_suggestion_queue_entries SET job_id=?,snapshot_digest=?,paths_json=?,mode=?,request_key=?,state=?,failure_reason='',updated_at=? WHERE repository_id=? AND id=?`, replacement.JobID, replacement.SnapshotDigest, string(raw), replacement.Mode, replacement.RequestKey, WorktreeQueueQueued, now.Format(time.RFC3339Nano), repo, id)
	if err != nil {
		return WorktreeQueue{}, nil, err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return WorktreeQueue{}, nil, ErrQueueConflict
	}
	if err = incQueueRevision(ctx, tx, repo, expected); err != nil {
		return WorktreeQueue{}, nil, err
	}
	q, err := queueFromTx(ctx, tx, repo)
	if err != nil {
		return WorktreeQueue{}, nil, err
	}
	if err = tx.Commit(); err != nil {
		return WorktreeQueue{}, nil, err
	}
	replacement.ID = id
	replacement.RepositoryID = repo
	replacement.Sequence = current.Sequence
	replacement.State = WorktreeQueueQueued
	replacement.CreatedAt = current.CreatedAt
	replacement.UpdatedAt = now
	q.Revision = expected + 1
	return q, &replacement, nil
}

func (s *Store) RetireWorktreeQueueHead(ctx context.Context, repo, head string, expected uint64, disposition WorktreeQueueDisposition) (WorktreeQueue, *WorktreeQueueEntry, error) {
	if disposition != WorktreeQueueDispositionCommitted && disposition != WorktreeQueueDispositionRemoved {
		return WorktreeQueue{}, nil, fmt.Errorf("queue disposition must be committed or removed")
	}
	tx, err := s.beginQueueMutation(ctx, repo, expected, head)
	if err != nil {
		return WorktreeQueue{}, nil, err
	}
	defer tx.Rollback()
	var entry WorktreeQueueEntry
	err = tx.QueryRowContext(ctx, `SELECT id,job_id,snapshot_digest,mode,request_key,state,failure_reason FROM gct_worktree_suggestion_queue_entries WHERE repository_id=? AND id=? AND state NOT IN (?,?)`, repo, head, WorktreeQueueCommitted, WorktreeQueueRemoved).Scan(&entry.ID, &entry.JobID, &entry.SnapshotDigest, &entry.Mode, &entry.RequestKey, &entry.State, &entry.FailureReason)
	if err != nil {
		return WorktreeQueue{}, nil, err
	}
	next := WorktreeQueueRemoved
	if disposition == WorktreeQueueDispositionCommitted {
		next = WorktreeQueueCommitted
	}
	if next == WorktreeQueueCommitted && entry.State != WorktreeQueueCommitPending {
		return WorktreeQueue{}, nil, fmt.Errorf("committed queue head must be commit-pending: %w", ErrQueueConflict)
	}
	if !entry.State.CanTransitionTo(next) {
		return WorktreeQueue{}, nil, ErrQueueConflict
	}
	result, err := tx.ExecContext(ctx, `UPDATE gct_worktree_suggestion_queue_entries SET state=?,updated_at=? WHERE repository_id=? AND id=? AND state=?`, next, time.Now().UTC().Format(time.RFC3339Nano), repo, head, entry.State)
	if err != nil {
		return WorktreeQueue{}, nil, err
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		return WorktreeQueue{}, nil, ErrQueueConflict
	}
	if err = incQueueRevision(ctx, tx, repo, expected); err != nil {
		return WorktreeQueue{}, nil, err
	}
	q, err := queueFromTx(ctx, tx, repo)
	if err != nil {
		return WorktreeQueue{}, nil, err
	}
	if err = tx.Commit(); err != nil {
		return WorktreeQueue{}, nil, err
	}
	entry.RepositoryID = repo
	entry.State = next
	q.Revision = expected + 1
	return q, &entry, nil
}

func (s *Store) RemoveWorktreeQueueEntry(ctx context.Context, repo, id string, expected uint64) (WorktreeQueue, *WorktreeQueueEntry, error) {
	tx, err := s.beginQueueRevision(ctx, repo, expected)
	if err != nil {
		return WorktreeQueue{}, nil, err
	}
	defer tx.Rollback()
	var entry WorktreeQueueEntry
	err = tx.QueryRowContext(ctx, `SELECT id,job_id,snapshot_digest,mode,request_key,state,failure_reason FROM gct_worktree_suggestion_queue_entries WHERE repository_id=? AND id=? AND state NOT IN (?,?)`, repo, id, WorktreeQueueCommitted, WorktreeQueueRemoved).Scan(&entry.ID, &entry.JobID, &entry.SnapshotDigest, &entry.Mode, &entry.RequestKey, &entry.State, &entry.FailureReason)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return WorktreeQueue{}, nil, ErrQueueConflict
		}
		return WorktreeQueue{}, nil, err
	}
	if entry.State == WorktreeQueueCommitPending {
		return WorktreeQueue{}, nil, fmt.Errorf("commit-pending queue head cannot be removed: %w", ErrQueueConflict)
	}
	if !entry.State.CanTransitionTo(WorktreeQueueRemoved) {
		return WorktreeQueue{}, nil, ErrQueueConflict
	}
	result, err := tx.ExecContext(ctx, `UPDATE gct_worktree_suggestion_queue_entries SET state=?,updated_at=? WHERE repository_id=? AND id=? AND state=?`, WorktreeQueueRemoved, time.Now().UTC().Format(time.RFC3339Nano), repo, id, entry.State)
	if err != nil {
		return WorktreeQueue{}, nil, err
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		return WorktreeQueue{}, nil, ErrQueueConflict
	}
	if err = incQueueRevision(ctx, tx, repo, expected); err != nil {
		return WorktreeQueue{}, nil, err
	}
	q, err := queueFromTx(ctx, tx, repo)
	if err != nil {
		return WorktreeQueue{}, nil, err
	}
	if err = tx.Commit(); err != nil {
		return WorktreeQueue{}, nil, err
	}
	entry.RepositoryID = repo
	entry.State = WorktreeQueueRemoved
	q.Revision = expected + 1
	return q, &entry, nil
}

func (s *Store) beginQueueRevision(ctx context.Context, repo string, expected uint64) (*sql.Tx, error) {
	if repo == "" {
		return nil, fmt.Errorf("repository is required")
	}
	if err := s.ensureWorktreeQueue(ctx); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO gct_worktree_suggestion_queues(repository_id) VALUES(?) ON CONFLICT(repository_id) DO NOTHING`, repo); err != nil {
		tx.Rollback()
		return nil, err
	}
	var revision uint64
	if err = tx.QueryRowContext(ctx, `SELECT revision FROM gct_worktree_suggestion_queues WHERE repository_id=?`, repo).Scan(&revision); err != nil {
		tx.Rollback()
		return nil, err
	}
	if revision != expected {
		tx.Rollback()
		return nil, ErrQueueConflict
	}
	return tx, nil
}

func (s *Store) beginQueueMutation(ctx context.Context, repo string, expected uint64, expectedHead string) (*sql.Tx, error) {
	if repo == "" || expectedHead == "" {
		return nil, fmt.Errorf("repository and expected queue head are required")
	}
	if err := s.ensureWorktreeQueue(ctx); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO gct_worktree_suggestion_queues(repository_id) VALUES(?) ON CONFLICT(repository_id) DO NOTHING`, repo); err != nil {
		tx.Rollback()
		return nil, err
	}
	var revision uint64
	if err = tx.QueryRowContext(ctx, `SELECT revision FROM gct_worktree_suggestion_queues WHERE repository_id=?`, repo).Scan(&revision); err != nil {
		tx.Rollback()
		return nil, err
	}
	if revision != expected {
		tx.Rollback()
		return nil, ErrQueueConflict
	}
	var head string
	err = tx.QueryRowContext(ctx, `SELECT id FROM gct_worktree_suggestion_queue_entries WHERE repository_id=? AND state NOT IN (?,?) ORDER BY sequence LIMIT 1`, repo, WorktreeQueueCommitted, WorktreeQueueRemoved).Scan(&head)
	if err != nil || head != expectedHead {
		tx.Rollback()
		return nil, ErrQueueConflict
	}
	return tx, nil
}

func incQueueRevision(ctx context.Context, tx *sql.Tx, repo string, expected uint64) error {
	result, err := tx.ExecContext(ctx, `UPDATE gct_worktree_suggestion_queues SET revision=revision+1 WHERE repository_id=? AND revision=?`, repo, expected)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return ErrQueueConflict
	}
	return nil
}

func queueFromTx(ctx context.Context, tx *sql.Tx, repo string) (WorktreeQueue, error) {
	var q WorktreeQueue
	q.RepositoryID = repo
	err := tx.QueryRowContext(ctx, `SELECT revision FROM gct_worktree_suggestion_queues WHERE repository_id=?`, repo).Scan(&q.Revision)
	if err != nil {
		return q, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT q.id,q.job_id,q.snapshot_digest,q.paths_json,q.mode,q.request_key,q.sequence,q.state,q.failure_reason,q.created_at,q.updated_at,COALESCE(p.selection_id,''),COALESCE(p.source,0),COALESCE(p.payload_json,'') FROM gct_worktree_suggestion_queue_entries q LEFT JOIN gct_worktree_selection_payloads p ON p.entry_id=q.id WHERE q.repository_id=? AND q.state NOT IN (?,?) ORDER BY q.sequence`, repo, WorktreeQueueCommitted, WorktreeQueueRemoved)
	if err != nil {
		return q, err
	}
	defer rows.Close()
	for rows.Next() {
		var e WorktreeQueueEntry
		var paths, created, updated string
		e.RepositoryID = repo
		if err = rows.Scan(&e.ID, &e.JobID, &e.SnapshotDigest, &paths, &e.Mode, &e.RequestKey, &e.Sequence, &e.State, &e.FailureReason, &created, &updated, &e.SelectionID, &e.SelectionSource, &e.SelectionPayload); err != nil {
			return q, err
		}
		if !e.State.Active() {
			return q, fmt.Errorf("queue entry %q has invalid active state %d", e.ID, e.State)
		}
		if err = json.Unmarshal([]byte(paths), &e.Paths); err != nil {
			return q, err
		}
		if err = validateWorktreeQueuePaths(e.Paths); err != nil {
			return q, fmt.Errorf("validate queue entry %q paths: %w", e.ID, err)
		}
		e.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return q, err
		}
		e.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
		if err != nil {
			return q, err
		}
		q.Entries = append(q.Entries, e)
	}
	return q, rows.Err()
}

// CreateWorktreeJobAndEnqueue makes initial job admission and queue membership one transaction.
func (s *Store) CreateWorktreeJobAndEnqueue(ctx context.Context, job Job, entry WorktreeQueueEntry) (Job, WorktreeQueueEntry, uint64, error) {
	if err := s.ensureWorktreeQueue(ctx); err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
	defer tx.Rollback()
	if job.ID == "" || job.IdempotencyKey == "" || job.InputDigest == "" || entry.RepositoryID != job.RepositoryID || entry.JobID != job.ID {
		return Job{}, WorktreeQueueEntry{}, 0, fmt.Errorf("job and queue entry identities must agree")
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	job.UpdatedAt = job.CreatedAt
	if job.State == "" {
		job.State = Queued
	}
	checks, err := json.Marshal(job.Checks)
	if err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO gct_review_jobs (id,idempotency_key,input_digest,policy_version,state,checks_json,result_ref,result_json,error,created_at,updated_at,scenario_name,detail_count,thresholds_json,operation,repository_id,input_json,execution_id,cancel_requested) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(idempotency_key) DO NOTHING`, job.ID, job.IdempotencyKey, job.InputDigest, job.PolicyVersion, job.State, string(checks), job.ResultRef, job.ResultJSON, job.Error, job.CreatedAt.Format(time.RFC3339Nano), job.UpdatedAt.Format(time.RFC3339Nano), job.ScenarioName, job.DetailCount, job.ThresholdsJSON, job.Operation, job.RepositoryID, job.InputJSON, job.ExecutionID, job.CancelRequested)
	if err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
	var effectiveID, effectiveDigest string
	if err = tx.QueryRowContext(ctx, `SELECT id,input_digest FROM gct_review_jobs WHERE idempotency_key=?`, job.IdempotencyKey).Scan(&effectiveID, &effectiveDigest); err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
	if effectiveID != job.ID || effectiveDigest != job.InputDigest {
		return Job{}, WorktreeQueueEntry{}, 0, ErrQueueConflict
	}
	entry.JobID = effectiveID
	queued, rev, err := enqueueWorktreeQueueTx(ctx, tx, entry)
	if err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
	if err = tx.Commit(); err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
	job.ID = effectiveID
	return job, queued, rev, nil
}

// CreateWorktreeJobAndReplaceQueueEntry atomically admits a refreshed job and
// swaps its queue row without changing that row's sequence.
func (s *Store) CreateWorktreeJobAndReplaceQueueEntry(ctx context.Context, job Job, entryID string, expected uint64, replacement WorktreeQueueEntry) (Job, WorktreeQueueEntry, uint64, error) {
	if err := s.ensureWorktreeQueue(ctx); err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
	defer tx.Rollback()
	if entryID == "" || job.ID == "" || job.IdempotencyKey == "" || job.InputDigest == "" || replacement.RepositoryID != job.RepositoryID || replacement.JobID != job.ID {
		return Job{}, WorktreeQueueEntry{}, 0, fmt.Errorf("job and replacement identities must agree")
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	job.UpdatedAt = job.CreatedAt
	if job.State == "" {
		job.State = Queued
	}
	checks, err := json.Marshal(job.Checks)
	if err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO gct_review_jobs (id,idempotency_key,input_digest,policy_version,state,checks_json,result_ref,result_json,error,created_at,updated_at,scenario_name,detail_count,thresholds_json,operation,repository_id,input_json,execution_id,cancel_requested) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(idempotency_key) DO NOTHING`, job.ID, job.IdempotencyKey, job.InputDigest, job.PolicyVersion, job.State, string(checks), job.ResultRef, job.ResultJSON, job.Error, job.CreatedAt.Format(time.RFC3339Nano), job.UpdatedAt.Format(time.RFC3339Nano), job.ScenarioName, job.DetailCount, job.ThresholdsJSON, job.Operation, job.RepositoryID, job.InputJSON, job.ExecutionID, job.CancelRequested)
	if err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
	var effectiveID, effectiveDigest string
	if err = tx.QueryRowContext(ctx, `SELECT id,input_digest FROM gct_review_jobs WHERE idempotency_key=?`, job.IdempotencyKey).Scan(&effectiveID, &effectiveDigest); err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
	if effectiveID != job.ID || effectiveDigest != job.InputDigest {
		return Job{}, WorktreeQueueEntry{}, 0, ErrQueueConflict
	}
	queued, revision, err := replaceWorktreeQueueEntryTx(ctx, tx, job.RepositoryID, entryID, expected, replacement)
	if err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
	if err = tx.Commit(); err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
	job.ID = effectiveID
	return job, queued, revision, nil
}

func replaceWorktreeQueueEntryTx(ctx context.Context, tx *sql.Tx, repo, entryID string, expected uint64, replacement WorktreeQueueEntry) (WorktreeQueueEntry, uint64, error) {
	var revision uint64
	if err := tx.QueryRowContext(ctx, `SELECT revision FROM gct_worktree_suggestion_queues WHERE repository_id=?`, repo).Scan(&revision); err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	var current WorktreeQueueEntry
	var raw, created, updated string
	err := tx.QueryRowContext(ctx, `SELECT id,job_id,snapshot_digest,paths_json,mode,request_key,sequence,state,failure_reason,created_at,updated_at FROM gct_worktree_suggestion_queue_entries WHERE repository_id=? AND id=? AND state NOT IN (?,?)`, repo, entryID, WorktreeQueueCommitted, WorktreeQueueRemoved).Scan(&current.ID, &current.JobID, &current.SnapshotDigest, &raw, &current.Mode, &current.RequestKey, &current.Sequence, &current.State, &current.FailureReason, &created, &updated)
	if err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	if err = json.Unmarshal([]byte(raw), &current.Paths); err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	if err = validateWorktreeQueuePaths(current.Paths); err != nil {
		return WorktreeQueueEntry{}, 0, fmt.Errorf("validate queue entry %q paths: %w", entryID, err)
	}
	if current.JobID == replacement.JobID && current.SnapshotDigest == replacement.SnapshotDigest && current.Mode == replacement.Mode && current.RequestKey == replacement.RequestKey && sameWorktreeQueuePaths(current.Paths, replacement.Paths) {
		return current, revision, nil
	}
	if revision != expected {
		return WorktreeQueueEntry{}, 0, ErrQueueConflict
	}
	if current.State == WorktreeQueueCommitPending {
		return WorktreeQueueEntry{}, 0, fmt.Errorf("commit-pending queue head cannot be replaced: %w", ErrQueueConflict)
	}
	if err = ensureWorktreePathsAvailable(ctx, tx, repo, entryID, replacement.Paths); err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	newPathsRaw, err := json.Marshal(replacement.Paths)
	if err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx, `UPDATE gct_worktree_suggestion_queue_entries SET job_id=?,snapshot_digest=?,paths_json=?,mode=?,request_key=?,state=?,failure_reason='',updated_at=? WHERE repository_id=? AND id=? AND state NOT IN (?,?)`, replacement.JobID, replacement.SnapshotDigest, string(newPathsRaw), replacement.Mode, replacement.RequestKey, WorktreeQueueQueued, now.Format(time.RFC3339Nano), repo, entryID, WorktreeQueueCommitted, WorktreeQueueRemoved)
	if err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return WorktreeQueueEntry{}, 0, ErrQueueConflict
	}
	if err = incQueueRevision(ctx, tx, repo, expected); err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	replacement.ID = entryID
	replacement.RepositoryID = repo
	replacement.Sequence = current.Sequence
	replacement.State = WorktreeQueueQueued
	replacement.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return WorktreeQueueEntry{}, 0, fmt.Errorf("parse replacement queue created_at: %w", err)
	}
	replacement.UpdatedAt = now
	return replacement, expected + 1, nil
}

func enqueueWorktreeQueueTx(ctx context.Context, tx *sql.Tx, entry WorktreeQueueEntry) (WorktreeQueueEntry, uint64, error) {
	if entry.ID == "" || entry.RepositoryID == "" || (entry.JobID == "" && (entry.SelectionID == "" || entry.SelectionPayload == "")) || entry.SnapshotDigest == "" || entry.RequestKey == "" || len(entry.Paths) == 0 {
		return WorktreeQueueEntry{}, 0, fmt.Errorf("complete worktree queue entry is required")
	}
	if err := validateWorktreeQueuePaths(entry.Paths); err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO gct_worktree_suggestion_queues(repository_id) VALUES(?) ON CONFLICT(repository_id) DO NOTHING`, entry.RepositoryID); err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	var oldID, oldJob, oldDigest, oldPaths, oldKey, oldCreated, oldUpdated, oldFailure string
	var oldMode int32
	var oldSeq, rev uint64
	var oldState WorktreeQueueState
	err := tx.QueryRowContext(ctx, `SELECT id,job_id,snapshot_digest,paths_json,mode,request_key,sequence,state,failure_reason,created_at,updated_at FROM gct_worktree_suggestion_queue_entries WHERE repository_id=? AND request_key=?`, entry.RepositoryID, entry.RequestKey).Scan(&oldID, &oldJob, &oldDigest, &oldPaths, &oldMode, &oldKey, &oldSeq, &oldState, &oldFailure, &oldCreated, &oldUpdated)
	if err == nil {
		if !oldState.Valid() {
			return WorktreeQueueEntry{}, 0, fmt.Errorf("queue entry %q has invalid stored state %d", oldID, oldState)
		}
		var paths []string
		if err = json.Unmarshal([]byte(oldPaths), &paths); err != nil {
			return WorktreeQueueEntry{}, 0, fmt.Errorf("decode replayed queue paths: %w", err)
		}
		if err = validateWorktreeQueuePaths(paths); err != nil {
			return WorktreeQueueEntry{}, 0, fmt.Errorf("validate replayed queue paths: %w", err)
		}
		if oldJob != entry.JobID || oldDigest != entry.SnapshotDigest || oldMode != entry.Mode || !sameWorktreeQueuePaths(paths, entry.Paths) {
			return WorktreeQueueEntry{}, 0, ErrQueueConflict
		}
		if entry.SelectionID != "" {
			var oldSelection string
			var oldSource int32
			var oldPayload string
			if err = tx.QueryRowContext(ctx, `SELECT selection_id,source,payload_json FROM gct_worktree_selection_payloads WHERE entry_id=?`, oldID).Scan(&oldSelection, &oldSource, &oldPayload); err != nil {
				return WorktreeQueueEntry{}, 0, ErrQueueConflict
			}
			if oldSelection != entry.SelectionID || oldSource != entry.SelectionSource || oldPayload != entry.SelectionPayload {
				return WorktreeQueueEntry{}, 0, ErrQueueConflict
			}
		}
		if err = tx.QueryRowContext(ctx, `SELECT revision FROM gct_worktree_suggestion_queues WHERE repository_id=?`, entry.RepositoryID).Scan(&rev); err != nil {
			return WorktreeQueueEntry{}, 0, err
		}
		entry.ID = oldID
		entry.Sequence = oldSeq
		entry.State = oldState
		entry.FailureReason = oldFailure
		entry.CreatedAt, err = time.Parse(time.RFC3339Nano, oldCreated)
		if err != nil {
			return WorktreeQueueEntry{}, 0, fmt.Errorf("parse replayed queue created_at: %w", err)
		}
		entry.UpdatedAt, err = time.Parse(time.RFC3339Nano, oldUpdated)
		if err != nil {
			return WorktreeQueueEntry{}, 0, fmt.Errorf("parse replayed queue updated_at: %w", err)
		}
		return entry, rev, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return WorktreeQueueEntry{}, 0, err
	}
	var seq uint64
	if err = tx.QueryRowContext(ctx, `SELECT revision,next_sequence FROM gct_worktree_suggestion_queues WHERE repository_id=?`, entry.RepositoryID).Scan(&rev, &seq); err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	if err = ensureWorktreePathsAvailable(ctx, tx, entry.RepositoryID, "", entry.Paths); err != nil {
		return WorktreeQueueEntry{}, rev, err
	}
	raw, err := json.Marshal(entry.Paths)
	if err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	now := time.Now().UTC()
	entry.Sequence = seq
	if entry.State == 0 {
		entry.State = WorktreeQueueQueued
	}
	entry.CreatedAt = now
	entry.UpdatedAt = now
	if _, err = tx.ExecContext(ctx, `INSERT INTO gct_worktree_suggestion_queue_entries(id,repository_id,job_id,snapshot_digest,paths_json,mode,request_key,sequence,state,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, entry.ID, entry.RepositoryID, entry.JobID, entry.SnapshotDigest, string(raw), entry.Mode, entry.RequestKey, seq, entry.State, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	if entry.SelectionID != "" {
		if _, err = tx.ExecContext(ctx, `INSERT INTO gct_worktree_selection_payloads(entry_id,selection_id,source,payload_json) VALUES(?,?,?,?)`, entry.ID, entry.SelectionID, entry.SelectionSource, entry.SelectionPayload); err != nil {
			return WorktreeQueueEntry{}, 0, err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE gct_worktree_suggestion_queues SET revision=revision+1,next_sequence=next_sequence+1 WHERE repository_id=?`, entry.RepositoryID); err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	return entry, rev + 1, nil
}
