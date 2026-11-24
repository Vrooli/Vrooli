package reviewjobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
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

// worktreeQueueTransitions is the canonical persisted lifecycle graph. The
// table is indexed by the stable state values above; committed and removed
// are terminal states with no outgoing transitions.
var worktreeQueueTransitions = [...]uint16{
	0,
	1<<WorktreeQueueStaged | 1<<WorktreeQueueFailed | 1<<WorktreeQueueStale | 1<<WorktreeQueueCommitPending | 1<<WorktreeQueueRemoved | 1<<WorktreeQueueAwaitingUnstage,
	1<<WorktreeQueueQueued | 1<<WorktreeQueueFailed | 1<<WorktreeQueueStale | 1<<WorktreeQueueCommitPending | 1<<WorktreeQueueRemoved,
	1<<WorktreeQueueQueued | 1<<WorktreeQueueRemoved,
	1<<WorktreeQueueQueued | 1<<WorktreeQueueRemoved,
	1<<WorktreeQueueQueued | 1<<WorktreeQueueStaged | 1<<WorktreeQueueStale | 1<<WorktreeQueueCommitted,
	0,
	0,
	1<<WorktreeQueueQueued | 1<<WorktreeQueueFailed | 1<<WorktreeQueueStale | 1<<WorktreeQueueRemoved,
}

func (state WorktreeQueueState) CanTransitionTo(next WorktreeQueueState) bool {
	if state < WorktreeQueueQueued || state > WorktreeQueueAwaitingUnstage || next < WorktreeQueueQueued || next > WorktreeQueueAwaitingUnstage {
		return false
	}
	return worktreeQueueTransitions[state]&(1<<next) != 0
}

type WorktreeQueue struct {
	RepositoryID string
	Revision     uint64
	Entries      []WorktreeQueueEntry
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
	if old != state && !old.CanTransitionTo(state) {
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
	entry, revision, err := replaceWorktreeQueueEntryTx(ctx, tx, repo, id, expected, replacement)
	if err != nil {
		return WorktreeQueue{}, nil, err
	}
	q, err := queueFromTx(ctx, tx, repo)
	if err != nil {
		return WorktreeQueue{}, nil, err
	}
	if err = tx.Commit(); err != nil {
		return WorktreeQueue{}, nil, err
	}
	q.Revision = revision
	return q, &entry, nil
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

// insertWorktreeJobTx admits an idempotent job inside the caller's queue
// transaction and returns the persisted identity. Both queue admission flows
// use this so job replay and digest-conflict behavior cannot drift.
func insertWorktreeJobTx(ctx context.Context, tx *sql.Tx, job Job) (Job, error) {
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	job.UpdatedAt = job.CreatedAt
	if job.State == "" {
		job.State = Queued
	}
	checks, err := json.Marshal(job.Checks)
	if err != nil {
		return Job{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO gct_review_jobs (id,idempotency_key,input_digest,policy_version,state,checks_json,result_ref,result_json,error,created_at,updated_at,scenario_name,detail_count,thresholds_json,operation,repository_id,input_json,execution_id,cancel_requested) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(idempotency_key) DO NOTHING`, job.ID, job.IdempotencyKey, job.InputDigest, job.PolicyVersion, job.State, string(checks), job.ResultRef, job.ResultJSON, job.Error, job.CreatedAt.Format(time.RFC3339Nano), job.UpdatedAt.Format(time.RFC3339Nano), job.ScenarioName, job.DetailCount, job.ThresholdsJSON, job.Operation, job.RepositoryID, job.InputJSON, job.ExecutionID, job.CancelRequested)
	if err != nil {
		return Job{}, err
	}
	var effectiveID, effectiveDigest string
	if err = tx.QueryRowContext(ctx, `SELECT id,input_digest FROM gct_review_jobs WHERE idempotency_key=?`, job.IdempotencyKey).Scan(&effectiveID, &effectiveDigest); err != nil {
		return Job{}, err
	}
	if effectiveID != job.ID || effectiveDigest != job.InputDigest {
		return Job{}, ErrQueueConflict
	}
	job.ID = effectiveID
	return job, nil
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
	job, err = insertWorktreeJobTx(ctx, tx, job)
	if err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
	entry.JobID = job.ID
	queued, rev, err := enqueueWorktreeQueueTx(ctx, tx, entry)
	if err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
	if err = tx.Commit(); err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
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
	job, err = insertWorktreeJobTx(ctx, tx, job)
	if err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
	queued, revision, err := replaceWorktreeQueueEntryTx(ctx, tx, job.RepositoryID, entryID, expected, replacement)
	if err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
	if err = tx.Commit(); err != nil {
		return Job{}, WorktreeQueueEntry{}, 0, err
	}
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
	current.RepositoryID = repo
	current.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return WorktreeQueueEntry{}, 0, fmt.Errorf("parse queue entry created_at: %w", err)
	}
	current.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	if err != nil {
		return WorktreeQueueEntry{}, 0, fmt.Errorf("parse queue entry updated_at: %w", err)
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
	replacement.CreatedAt = current.CreatedAt
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
	replayed, found, err := replayWorktreeQueueRequestTx(ctx, tx, entry)
	if err != nil || found {
		return replayed.entry, replayed.revision, err
	}
	return insertWorktreeQueueEntryTx(ctx, tx, entry)
}

type worktreeQueueReplay struct {
	entry    WorktreeQueueEntry
	revision uint64
}

func replayWorktreeQueueRequestTx(ctx context.Context, tx *sql.Tx, entry WorktreeQueueEntry) (worktreeQueueReplay, bool, error) {
	var replay worktreeQueueReplay
	var id, jobID, digest, rawPaths, created, updated, failure string
	var mode int32
	var sequence, revision uint64
	var state WorktreeQueueState
	err := tx.QueryRowContext(ctx, `SELECT id,job_id,snapshot_digest,paths_json,mode,sequence,state,failure_reason,created_at,updated_at FROM gct_worktree_suggestion_queue_entries WHERE repository_id=? AND request_key=?`, entry.RepositoryID, entry.RequestKey).Scan(&id, &jobID, &digest, &rawPaths, &mode, &sequence, &state, &failure, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return replay, false, nil
	}
	if err != nil {
		return replay, false, err
	}
	if !state.Valid() {
		return replay, true, fmt.Errorf("queue entry %q has invalid stored state %d", id, state)
	}
	var paths []string
	if err = json.Unmarshal([]byte(rawPaths), &paths); err != nil {
		return replay, true, fmt.Errorf("decode replayed queue paths: %w", err)
	}
	if err = validateWorktreeQueuePaths(paths); err != nil {
		return replay, true, fmt.Errorf("validate replayed queue paths: %w", err)
	}
	if jobID != entry.JobID || digest != entry.SnapshotDigest || mode != entry.Mode || !sameWorktreeQueuePaths(paths, entry.Paths) {
		return replay, true, ErrQueueConflict
	}
	if entry.SelectionID != "" {
		var selectionID, payload string
		var source int32
		if err = tx.QueryRowContext(ctx, `SELECT selection_id,source,payload_json FROM gct_worktree_selection_payloads WHERE entry_id=?`, id).Scan(&selectionID, &source, &payload); err != nil {
			return replay, true, ErrQueueConflict
		}
		if selectionID != entry.SelectionID || source != entry.SelectionSource || payload != entry.SelectionPayload {
			return replay, true, ErrQueueConflict
		}
	}
	if err = tx.QueryRowContext(ctx, `SELECT revision FROM gct_worktree_suggestion_queues WHERE repository_id=?`, entry.RepositoryID).Scan(&revision); err != nil {
		return replay, true, err
	}
	entry.ID = id
	entry.Sequence = sequence
	entry.State = state
	entry.FailureReason = failure
	entry.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return replay, true, fmt.Errorf("parse replayed queue created_at: %w", err)
	}
	entry.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	if err != nil {
		return replay, true, fmt.Errorf("parse replayed queue updated_at: %w", err)
	}
	return worktreeQueueReplay{entry: entry, revision: revision}, true, nil
}

func insertWorktreeQueueEntryTx(ctx context.Context, tx *sql.Tx, entry WorktreeQueueEntry) (WorktreeQueueEntry, uint64, error) {
	var revision, sequence uint64
	err := tx.QueryRowContext(ctx, `SELECT revision,next_sequence FROM gct_worktree_suggestion_queues WHERE repository_id=?`, entry.RepositoryID).Scan(&revision, &sequence)
	if err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	if err = ensureWorktreePathsAvailable(ctx, tx, entry.RepositoryID, "", entry.Paths); err != nil {
		return WorktreeQueueEntry{}, revision, err
	}
	raw, err := json.Marshal(entry.Paths)
	if err != nil {
		return WorktreeQueueEntry{}, 0, err
	}
	now := time.Now().UTC()
	entry.Sequence = sequence
	if entry.State == 0 {
		entry.State = WorktreeQueueQueued
	}
	entry.CreatedAt = now
	entry.UpdatedAt = now
	if _, err = tx.ExecContext(ctx, `INSERT INTO gct_worktree_suggestion_queue_entries(id,repository_id,job_id,snapshot_digest,paths_json,mode,request_key,sequence,state,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, entry.ID, entry.RepositoryID, entry.JobID, entry.SnapshotDigest, string(raw), entry.Mode, entry.RequestKey, sequence, entry.State, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
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
	return entry, revision + 1, nil
}
