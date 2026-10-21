package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// sqliteRepository is deliberately an implementation detail. Its transaction
// boundary uses SQLite BEGIN IMMEDIATE so a cancellation and worker terminal
// receipt have one serial winner across repository instances/processes.
type SQLDatabase interface {
	Conn(context.Context) (*sql.Conn, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}
type sqliteRepository struct{ db SQLDatabase }

func NewSQLiteRepository(db SQLDatabase) Repository { return &sqliteRepository{db: db} }

var _ Repository = (*sqliteRepository)(nil)

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func encode(v any) (string, error) { b, e := json.Marshal(v); return string(b), e }
func (r *sqliteRepository) withImmediate(ctx context.Context, fn func(*sql.Conn) error) error {
	c, err := r.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer c.Close()
	if _, err = c.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("begin serialized job transition: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = c.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	if err = fn(c); err != nil {
		return err
	}
	if _, err = c.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("commit job transition: %w", err)
	}
	committed = true
	return nil
}

func (r *sqliteRepository) Submit(ctx context.Context, j Job) (out Job, created bool, err error) {
	err = r.withImmediate(ctx, func(c *sql.Conn) error {
		var id, record string
		e := c.QueryRowContext(ctx, "SELECT id,record_json FROM render_jobs WHERE idempotency_key=?", j.IdempotencyKey).Scan(&id, &record)
		if e == nil {
			existing, e := decodeJob(record)
			if e != nil {
				return e
			}
			if existing.InputFingerprint != j.InputFingerprint {
				return ErrIdempotencyConflict
			}
			existing, e = loadJob(ctx, c, id)
			out = existing
			return e
		}
		if !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		if e = insertJob(ctx, c, j); e != nil {
			return fmt.Errorf("insert render job: %w", e)
		}
		out = j
		created = true
		return nil
	})
	return
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Job, error) {
	return loadJob(ctx, r.db, id)
}

func (r *sqliteRepository) FindByIdempotencyKey(ctx context.Context, key string) (Job, error) {
	var id string
	err := r.db.QueryRowContext(ctx, "SELECT id FROM render_jobs WHERE idempotency_key=?", key).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	if err != nil {
		return Job{}, err
	}
	return loadJob(ctx, r.db, id)
}

func (r *sqliteRepository) ListNonterminal(ctx context.Context) ([]Job, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id FROM render_jobs WHERE state IN ('queued','running','checkpointed','cancel_requested') ORDER BY created_at,id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	out := make([]Job, 0, len(ids))
	for _, id := range ids {
		j, e := loadJob(ctx, r.db, id)
		if e != nil {
			return nil, e
		}
		out = append(out, j)
	}
	return out, nil
}

func (r *sqliteRepository) ListProject(ctx context.Context, projectID string, cursor Cursor, limit int) ([]Job, error) {
	query := `SELECT id FROM render_jobs WHERE json_extract(record_json,'$.input.project_id')=?`
	args := []any{projectID}
	if !cursor.CreatedAt.IsZero() {
		query += ` AND (created_at>? OR (created_at=? AND id>?))`
		args = append(args, cursor.CreatedAt.UTC().UnixNano(), cursor.CreatedAt.UTC().UnixNano(), cursor.ID)
	}
	query += ` ORDER BY created_at,id LIMIT ?`
	args = append(args, limit)
	return r.loadIDs(ctx, query, args...)
}

func (r *sqliteRepository) ListProgress(ctx context.Context, cursor Cursor, limit int) ([]Job, error) {
	query := `SELECT j.id FROM render_jobs j WHERE (j.state IN ('queued','running','checkpointed','cancel_requested') OR EXISTS (SELECT 1 FROM render_job_release_pending p WHERE p.job_id=j.id) OR EXISTS (SELECT 1 FROM render_job_subject_finalization_pending f WHERE f.job_id=j.id))`
	args := []any{}
	if !cursor.CreatedAt.IsZero() {
		query += ` AND (j.created_at>? OR (j.created_at=? AND j.id>?))`
		args = append(args, cursor.CreatedAt.UTC().UnixNano(), cursor.CreatedAt.UTC().UnixNano(), cursor.ID)
	}
	query += ` ORDER BY j.created_at,j.id LIMIT ?`
	args = append(args, limit)
	return r.loadIDs(ctx, query, args...)
}

func (r *sqliteRepository) loadIDs(ctx context.Context, query string, args ...any) ([]Job, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	out := make([]Job, 0, len(ids))
	for _, id := range ids {
		j, e := loadJob(ctx, r.db, id)
		if e != nil {
			return nil, e
		}
		out = append(out, j)
	}
	return out, nil
}

func (r *sqliteRepository) RecordObserved(ctx context.Context, id, attempt string, expected int64) (Job, error) {
	return r.mutate(ctx, id, time.Now().UTC().UnixNano(), expected, func(j *Job) error {
		if j.State != Running && j.State != CancelRequested {
			return ErrInvalidTransition
		}
		a, err := attemptFor(j, attempt)
		if err != nil {
			return err
		}
		if a.Execution == nil {
			return ErrInvalidTransition
		}
		a.ExecutionObserved = true
		return nil
	})
}

func (r *sqliteRepository) RecordReleased(ctx context.Context, id, attempt string, expected int64, at time.Time) (Job, error) {
	return r.mutate(ctx, id, at.UTC().UnixNano(), expected, func(j *Job) error {
		if !isTerminal(j.State) && j.State != Checkpointed {
			return ErrInvalidTransition
		}
		a, err := attemptFor(j, attempt)
		if err != nil {
			return err
		}
		if a.Execution == nil {
			return ErrInvalidTransition
		}
		t := at.UTC()
		a.ExecutionReleasedAt = &t
		return nil
	})
}

func (r *sqliteRepository) Start(ctx context.Context, claim ExecutionClaim, now, expected int64) (Job, error) {
	return r.mutate(ctx, claim.JobID, now, expected, func(j *Job) error {
		if j.State != Queued && j.State != Checkpointed {
			return transitionErr(j, "start")
		}
		t := time.Unix(0, now).UTC()
		if j.State == Queued {
			if len(j.Attempts) == 0 {
				return ErrInvalidTransition
			}
			current := len(j.Attempts) - 1
			for i := 0; i < current; i++ {
				previous := j.Attempts[i]
				if (previous.State != AttemptSucceeded && previous.State != AttemptFailed && previous.State != AttemptCancelled) || (previous.Execution != nil && previous.ExecutionReleasedAt == nil) {
					return ErrInvalidTransition
				}
			}
			a := &j.Attempts[current]
			if a.State != AttemptQueued || a.Execution != nil || a.Number != current+1 {
				return ErrInvalidTransition
			}
			if e := validateExecutionClaim(*j, *a, claim); e != nil {
				return e
			}
			a.Execution = &claim
			a.State = AttemptRunning
			x := t
			a.StartedAt = &x
			a.HeartbeatAt = &x
		} else {
			if len(j.Attempts) == 0 {
				return ErrInvalidTransition
			}
			prev := &j.Attempts[len(j.Attempts)-1]
			if prev.Execution != nil && prev.ExecutionReleasedAt == nil {
				return ErrReleasePending
			}
			if prev.State != AttemptCheckpointed || prev.Checkpoint == nil || prev.CheckpointQuiescence == nil || validateCheckpoint(*j, *prev, *prev.Checkpoint) != nil || validateQuiescence(*j, *prev, *prev.CheckpointQuiescence) != nil || prev.CheckpointQuiescence.ObservedAt.Before(prev.Checkpoint.CreatedAt) {
				return ErrCheckpointMismatch
			}
			if attemptIDExists(*j, claim.AttemptID) || executionScopeExists(*j, claim.ScopeID) || claim.AttemptNumber != prev.Number+1 || prev.Execution == nil {
				return ErrInvalidTransition
			}
			a := Attempt{ID: claim.AttemptID, Number: claim.AttemptNumber, State: AttemptRunning, CreatedAt: t, StartedAt: &t, HeartbeatAt: &t, ResumeFrom: prev.Checkpoint}
			if e := validateExecutionClaim(*j, a, claim); e != nil {
				return e
			}
			a.Execution = &claim
			j.Attempts = append(j.Attempts, a)
		}
		j.State = Running
		j.Failure = ""
		return nil
	})
}

func (r *sqliteRepository) Heartbeat(ctx context.Context, id, attempt string, now, expected int64) (Job, error) {
	return r.mutate(ctx, id, now, expected, func(j *Job) error {
		if j.State != Running {
			return transitionErr(j, "heartbeat")
		}
		a, e := attemptFor(j, attempt)
		if e != nil {
			return e
		}
		if a.State != AttemptRunning {
			return ErrInvalidTransition
		}
		t := time.Unix(0, now).UTC()
		a.HeartbeatAt = &t
		return nil
	})
}

func (r *sqliteRepository) Checkpoint(ctx context.Context, id, attempt string, cp Checkpoint, proof QuiescenceProof, expected int64) (Job, error) {
	return r.mutate(ctx, id, cp.CreatedAt.UTC().UnixNano(), expected, func(j *Job) error {
		if j.State != Running {
			return transitionErr(j, "checkpoint")
		}
		a, e := attemptFor(j, attempt)
		if e != nil {
			return e
		}
		if e = validateCheckpoint(*j, *a, cp); e != nil {
			return e
		}
		if e = validateQuiescence(*j, *a, proof); e != nil {
			return e
		}
		if proof.ObservedAt.Before(cp.CreatedAt) {
			return fmt.Errorf("%w: checkpoint proof predates checkpoint", ErrInvalid)
		}
		a.Checkpoint = &cp
		a.CheckpointQuiescence = &proof
		a.State = AttemptCheckpointed
		t := cp.CreatedAt.UTC()
		a.HeartbeatAt = &t
		j.State = Checkpointed
		return nil
	})
}

func (r *sqliteRepository) Complete(ctx context.Context, id, attempt string, c CompletionEvidence, expected int64) (Job, error) {
	return r.mutate(ctx, id, c.CompletedAt.UTC().UnixNano(), expected, func(j *Job) error {
		if j.State != Running {
			return transitionErr(j, "succeed")
		}
		a, e := attemptFor(j, attempt)
		if e != nil {
			return e
		}
		if e = validateCompletion(*j, *a, c); e != nil {
			return e
		}
		a.Completion = &c
		a.State = AttemptSucceeded
		t := c.CompletedAt.UTC()
		a.FinishedAt = &t
		j.State = Succeeded
		return nil
	})
}

func (r *sqliteRepository) Fail(ctx context.Context, id, attempt, reason string, now, expected int64) (Job, error) {
	return r.mutate(ctx, id, now, expected, func(j *Job) error {
		if j.State != Running {
			return transitionErr(j, "fail")
		}
		a, e := attemptFor(j, attempt)
		if e != nil {
			return e
		}
		a.State = AttemptFailed
		a.Failure = reason
		t := time.Unix(0, now).UTC()
		a.FinishedAt = &t
		j.State = Failed
		j.Failure = reason
		return nil
	})
}

func (r *sqliteRepository) FailBeforeDispatch(ctx context.Context, id, attempt, reason string, now, expected int64) (Job, error) {
	return r.mutate(ctx, id, now, expected, func(j *Job) error {
		if j.State != Queued {
			return transitionErr(j, "fail before dispatch")
		}
		a, err := attemptFor(j, attempt)
		if err != nil {
			return err
		}
		if a.State != AttemptQueued || a.Execution != nil {
			return ErrInvalidTransition
		}
		a.State = AttemptFailed
		a.Failure = reason
		finished := time.Unix(0, now).UTC()
		a.FinishedAt = &finished
		j.State = Failed
		j.Failure = reason
		return nil
	})
}

func (r *sqliteRepository) RequestCancel(ctx context.Context, id string, now, expected int64) (Job, error) {
	return r.mutate(ctx, id, now, expected, func(j *Job) error {
		if isTerminal(j.State) {
			return ErrAlreadyTerminal
		}
		t := time.Unix(0, now).UTC()
		j.CancelRequestedAt = &t
		if len(j.Attempts) == 0 {
			return ErrInvalidTransition
		}
		a := &j.Attempts[len(j.Attempts)-1]
		if j.State == Queued {
			if a.State != AttemptQueued {
				return ErrInvalidTransition
			}
			proof := QuiescenceProof{JobID: j.ID, AttemptID: a.ID, InputFingerprint: j.InputFingerprint, OwnerID: "jobs", ScopeID: "pre-dispatch:" + j.ID, ObservedAt: t, NoOwnedExecution: true, PreDispatch: true, Reason: "cancelled before dispatch; no worker execution was admitted"}
			if e := validateQuiescence(*j, *a, proof); e != nil {
				return e
			}
			a.State = AttemptCancelled
			a.FinishedAt = &t
			j.State = Cancelled
			j.Quiescence = &proof
			return nil
		}
		if j.State == Checkpointed {
			if a.State != AttemptCheckpointed || a.CheckpointQuiescence == nil || validateQuiescence(*j, *a, *a.CheckpointQuiescence) != nil || a.CheckpointQuiescence.ObservedAt.Before(a.Checkpoint.CreatedAt) {
				return ErrCheckpointMismatch
			}
			j.State = Cancelled
			j.Quiescence = a.CheckpointQuiescence
			a.State = AttemptCancelled
			a.FinishedAt = &t
			return nil
		}
		if j.State != Running {
			return transitionErr(j, "cancel")
		}
		j.State = CancelRequested
		return nil
	})
}

func (r *sqliteRepository) ConfirmCancelled(ctx context.Context, id, attempt string, p QuiescenceProof, expected int64) (Job, error) {
	return r.mutate(ctx, id, p.ObservedAt.UTC().UnixNano(), expected, func(j *Job) error {
		if j.State != CancelRequested {
			return transitionErr(j, "confirm cancellation")
		}
		a, e := attemptFor(j, attempt)
		if e != nil {
			return e
		}
		if a.State != AttemptRunning && a.State != AttemptCheckpointed {
			return ErrInvalidTransition
		}
		if e = validateQuiescence(*j, *a, p); e != nil {
			return e
		}
		j.State = Cancelled
		j.Quiescence = &p
		a.State = AttemptCancelled
		t := p.ObservedAt.UTC()
		a.FinishedAt = &t
		return nil
	})
}

func (r *sqliteRepository) Settle(ctx context.Context, id string, state SettlementState, now, expected int64) (Job, error) {
	return r.mutate(ctx, id, now, expected, func(j *Job) error {
		if j.Settlement != SettlementPending && j.Settlement != SettlementReconciliation {
			return fmt.Errorf("%w: job has no unsettled accounting operation", ErrInvalid)
		}
		if state != SettlementSettled && state != SettlementReconciliation {
			return fmt.Errorf("%w: invalid settlement state", ErrInvalid)
		}
		j.Settlement = state
		return nil
	})
}

func (r *sqliteRepository) RetrySourceInspection(ctx context.Context, id string, expected int64, attemptID string, now int64) (out Job, created bool, err error) {
	if attemptID == "" || expected <= 0 {
		return Job{}, false, ErrInvalid
	}
	err = r.withImmediate(ctx, func(c *sql.Conn) error {
		j, e := loadJob(ctx, c, id)
		if e != nil {
			return e
		}
		if j.Input.SubjectKind != SubjectSourceInspection {
			return ErrInvalidTransition
		}
		for _, attempt := range j.Attempts {
			if attempt.RetryFromVersion == expected {
				out = j
				return nil
			}
		}
		if j.Version != expected {
			return ErrConcurrentTransition
		}
		if j.State != Failed && j.State != Cancelled || len(j.Attempts) == 0 {
			return ErrInvalidTransition
		}
		current := &j.Attempts[len(j.Attempts)-1]
		if j.State == Failed && current.State != AttemptFailed || j.State == Cancelled && current.State != AttemptCancelled {
			return ErrInvalidTransition
		}
		if j.Settlement == SettlementPending || j.Settlement == SettlementReconciliation || j.Input.AccountingRequired && j.Settlement != SettlementSettled {
			return ErrInvalidTransition
		}
		for _, previous := range j.Attempts {
			if previous.Execution == nil {
				continue
			}
			if previous.ExecutionReleasedAt == nil {
				return ErrReleasePending
			}
			var releasedAt int64
			e = c.QueryRowContext(ctx, `SELECT released_at FROM render_job_release_receipts WHERE job_id=? AND attempt_id=?`, j.ID, previous.ID).Scan(&releasedAt)
			if errors.Is(e, sql.ErrNoRows) || e == nil && releasedAt != previous.ExecutionReleasedAt.UTC().UnixNano() {
				return ErrReleasePending
			}
			if e != nil {
				return e
			}
			var pending int
			e = c.QueryRowContext(ctx, `SELECT COUNT(*) FROM render_job_release_pending WHERE job_id=? AND attempt_id=?`, j.ID, previous.ID).Scan(&pending)
			if e != nil {
				return e
			}
			if pending != 0 {
				return ErrReleasePending
			}
		}
		if j.State == Cancelled {
			if j.Quiescence == nil || j.Quiescence.AttemptID != current.ID || validateQuiescence(j, *current, *j.Quiescence) != nil {
				return ErrInvalidTransition
			}
		}
		var pending int
		e = c.QueryRowContext(ctx, `SELECT COUNT(*) FROM render_job_subject_finalization_pending WHERE job_id=?`, j.ID).Scan(&pending)
		if e != nil {
			return e
		}
		if pending != 0 {
			return ErrInvalidTransition
		}
		oldState := j.State
		if attemptIDExists(j, attemptID) {
			return ErrInvalidTransition
		}
		j.Attempts = append(j.Attempts, Attempt{ID: attemptID, Number: current.Number + 1, RetryFromVersion: expected, State: AttemptQueued, CreatedAt: time.Unix(0, now).UTC()})
		j.State = Queued
		j.Failure = ""
		j.CancelRequestedAt = nil
		j.Quiescence = nil
		j.Version++
		j.UpdatedAt = time.Unix(0, now).UTC()
		if e = saveJob(ctx, c, j, oldState, expected); e != nil {
			return e
		}
		if e = persistAttempts(ctx, c, j); e != nil {
			return e
		}
		if e = syncReleasePending(ctx, c, j); e != nil {
			return e
		}
		out, created = j, true
		return nil
	})
	return
}

func (r *sqliteRepository) mutate(ctx context.Context, id string, now, expected int64, fn func(*Job) error) (out Job, err error) {
	err = r.withImmediate(ctx, func(c *sql.Conn) error {
		j, e := loadJob(ctx, c, id)
		if e != nil {
			return e
		}
		if j.Version != expected {
			return ErrConcurrentTransition
		}
		oldState := j.State
		if e = fn(&j); e != nil {
			return e
		}
		j.Version++
		j.UpdatedAt = time.Unix(0, now).UTC()
		if e = saveJob(ctx, c, j, oldState, expected); e != nil {
			return e
		}
		if e = persistAttempts(ctx, c, j); e != nil {
			return e
		}
		if e = syncReleasePending(ctx, c, j); e != nil {
			return e
		}
		if oldState != j.State && isTerminal(j.State) && j.Input.SubjectKind != "" {
			a, ok := currentAttempt(j)
			if !ok {
				return ErrInvalidTransition
			}
			if _, e = c.ExecContext(ctx, `INSERT OR IGNORE INTO render_job_subject_finalization_pending(job_id,attempt_id,subject_kind,created_at) VALUES(?,?,?,?)`, j.ID, a.ID, j.Input.SubjectKind, now); e != nil {
				return e
			}
		}
		out = j
		return nil
	})
	return
}

func (r *sqliteRepository) PendingSubjectFinalization(ctx context.Context, id string) (SubjectFinalization, error) {
	var out SubjectFinalization
	var kind string
	err := r.db.QueryRowContext(ctx, `SELECT f.job_id,f.attempt_id,f.subject_kind FROM render_job_subject_finalization_pending f WHERE f.job_id=? ORDER BY f.created_at LIMIT 1`, id).Scan(&out.JobID, &out.AttemptID, &kind)
	if errors.Is(err, sql.ErrNoRows) {
		return SubjectFinalization{}, ErrNotFound
	}
	if err != nil {
		return SubjectFinalization{}, err
	}
	job, err := loadJob(ctx, r.db, out.JobID)
	if err != nil {
		return SubjectFinalization{}, err
	}
	if job.Input.SubjectKind != SubjectKind(kind) || job.State != Succeeded && job.State != Failed && job.State != Cancelled {
		return SubjectFinalization{}, ErrInvalidTransition
	}
	out.Kind, out.State, out.Failure = SubjectKind(kind), job.State, job.Failure
	return out, nil
}

func (r *sqliteRepository) CompleteSubjectFinalization(ctx context.Context, id, attempt string, kind SubjectKind) error {
	return r.withImmediate(ctx, func(c *sql.Conn) error {
		res, err := c.ExecContext(ctx, `DELETE FROM render_job_subject_finalization_pending WHERE job_id=? AND attempt_id=? AND subject_kind=?`, id, attempt, kind)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrNotFound
		}
		return nil
	})
}

func syncReleasePending(ctx context.Context, c *sql.Conn, j Job) error {
	for _, a := range j.Attempts {
		if a.Execution == nil {
			continue
		}
		if a.ExecutionReleasedAt == nil {
			if _, err := c.ExecContext(ctx, `INSERT OR IGNORE INTO render_job_release_pending(job_id,attempt_id) VALUES(?,?)`, j.ID, a.ID); err != nil {
				return err
			}
			continue
		}
		at := a.ExecutionReleasedAt.UTC().UnixNano()
		if _, err := c.ExecContext(ctx, `INSERT OR IGNORE INTO render_job_release_receipts(job_id,attempt_id,released_at) VALUES(?,?,?)`, j.ID, a.ID, at); err != nil {
			return err
		}
		if _, err := c.ExecContext(ctx, `DELETE FROM render_job_release_pending WHERE job_id=? AND attempt_id=?`, j.ID, a.ID); err != nil {
			return err
		}
	}
	return nil
}

func transitionErr(j *Job, action string) error {
	return fmt.Errorf("%w: cannot %s job in %s", ErrInvalidTransition, action, j.State)
}

func attemptFor(j *Job, id string) (*Attempt, error) {
	if len(j.Attempts) == 0 || j.Attempts[len(j.Attempts)-1].ID != id {
		return nil, ErrInvalidTransition
	}
	return &j.Attempts[len(j.Attempts)-1], nil
}

func decodeJob(record string) (Job, error) {
	var j Job
	err := json.Unmarshal([]byte(record), &j)
	return j, err
}
func jobRecord(j Job) (string, error) { j.Attempts = nil; return encode(j) }
func insertJob(ctx context.Context, c *sql.Conn, j Job) error {
	rec, e := jobRecord(j)
	if e != nil {
		return e
	}
	_, e = c.ExecContext(ctx, "INSERT INTO render_jobs(id,idempotency_key,state,version,created_at,updated_at,record_json) VALUES(?,?,?,?,?,?,?)", j.ID, j.IdempotencyKey, j.State, j.Version, j.CreatedAt.UnixNano(), j.UpdatedAt.UnixNano(), rec)
	if e != nil {
		return e
	}
	return persistAttempts(ctx, c, j)
}

func saveJob(ctx context.Context, c *sql.Conn, j Job, old State, version int64) error {
	rec, e := jobRecord(j)
	if e != nil {
		return e
	}
	res, e := c.ExecContext(ctx, "UPDATE render_jobs SET state=?,version=?,updated_at=?,record_json=? WHERE id=? AND state=? AND version=?", j.State, j.Version, j.UpdatedAt.UnixNano(), rec, j.ID, old, version)
	if e != nil {
		return e
	}
	n, e := res.RowsAffected()
	if e != nil {
		return e
	}
	if n != 1 {
		return ErrConcurrentTransition
	}
	return nil
}

func persistAttempts(ctx context.Context, c *sql.Conn, j Job) error {
	for _, a := range j.Attempts {
		rec, e := encode(a)
		if e != nil {
			return e
		}
		_, e = c.ExecContext(ctx, `INSERT INTO render_job_attempts(id,job_id,attempt_number,state,record_json) VALUES(?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET state=excluded.state,record_json=excluded.record_json`, a.ID, j.ID, a.Number, a.State, rec)
		if e != nil {
			return e
		}
	}
	return nil
}

func loadJob(ctx context.Context, q queryer, id string) (Job, error) {
	var state string
	var version, created, updated int64
	var rec string
	e := q.QueryRowContext(ctx, "SELECT state,version,created_at,updated_at,record_json FROM render_jobs WHERE id=?", id).Scan(&state, &version, &created, &updated, &rec)
	if errors.Is(e, sql.ErrNoRows) {
		return Job{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	if e != nil {
		return Job{}, e
	}
	j, e := decodeJob(rec)
	if e != nil {
		return Job{}, fmt.Errorf("decode job record: %w", e)
	}
	j.State = State(state)
	j.Version = version
	j.CreatedAt = time.Unix(0, created).UTC()
	j.UpdatedAt = time.Unix(0, updated).UTC()
	rows, e := q.QueryContext(ctx, "SELECT record_json FROM render_job_attempts WHERE job_id=? ORDER BY attempt_number", id)
	if e != nil {
		return Job{}, e
	}
	defer rows.Close()
	j.Attempts = []Attempt{}
	for rows.Next() {
		var ar string
		var a Attempt
		if e = rows.Scan(&ar); e != nil {
			return Job{}, e
		}
		if e = json.Unmarshal([]byte(ar), &a); e != nil {
			return Job{}, e
		}
		j.Attempts = append(j.Attempts, a)
	}
	if e = rows.Err(); e != nil {
		return Job{}, e
	}
	return j, nil
}
