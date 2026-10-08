package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock}
}

func (r *sqliteRepository) Create(ctx context.Context, job Job) (Job, error) {
	if job.ID == "" {
		job.ID = uuid.NewString()
	}
	if job.WorkspaceID == "" || job.Type == "" || job.DedupKey == "" {
		return Job{}, errors.New("job requires workspace, type, and dedup key")
	}
	if job.State == "" {
		job.State = Queued
	}
	revisions, err := json.Marshal(job.InputRevisions)
	if err != nil {
		return Job{}, err
	}
	var existingID, existingHash string
	if err := r.db.QueryRowContext(ctx, `SELECT id,request_hash FROM jobs WHERE workspace_id=? AND type=? AND dedup_key=?`, job.WorkspaceID, job.Type, job.DedupKey).Scan(&existingID, &existingHash); err == nil {
		if existingHash != job.RequestHash {
			return Job{}, ErrIdempotencyConflict{job.DedupKey}
		}
		return r.Get(ctx, existingID, job.WorkspaceID)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Job{}, err
	}
	now := r.clock.Now().UTC().Format(time.RFC3339Nano)
	_, err = r.db.ExecContext(ctx, `INSERT INTO jobs(id,workspace_id,type,dedup_key,request_hash,state,input_revisions_json,attempts,provider,budget_units,result_reference,error_code,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, job.ID, job.WorkspaceID, job.Type, job.DedupKey, job.RequestHash, string(job.State), string(revisions), job.Attempts, job.Provider, job.BudgetUnits, job.ResultReference, job.ErrorCode, now, now)
	if err != nil {
		return Job{}, err
	}
	return job, nil
}

func (r *sqliteRepository) List(ctx context.Context, workspaceID string) ([]Job, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,workspace_id,type,dedup_key,request_hash,state,input_revisions_json,attempts,provider,budget_units,result_reference,error_code FROM jobs WHERE workspace_id=? ORDER BY updated_at DESC,id DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Job
	for rows.Next() {
		job, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) Get(ctx context.Context, id, workspaceID string) (Job, error) {
	job, err := scan(r.db.QueryRowContext(ctx, `SELECT id,workspace_id,type,dedup_key,request_hash,state,input_revisions_json,attempts,provider,budget_units,result_reference,error_code FROM jobs WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, ErrNotFound{id}
	}
	if err != nil {
		return Job{}, err
	}
	if job.WorkspaceID != workspaceID {
		return Job{}, ErrForbidden{id}
	}
	return job, nil
}

func (r *sqliteRepository) Transition(ctx context.Context, in TransitionInput) (Job, error) {
	current, err := r.Get(ctx, in.ID, in.WorkspaceID)
	if err != nil {
		return Job{}, err
	}
	next, err := current.Transition(in.NextState)
	if err != nil {
		return Job{}, err
	}
	if in.Attempts > 0 {
		next.Attempts = in.Attempts
	}
	next.Provider = in.Provider
	next.ResultReference = in.ResultReference
	next.ErrorCode = in.ErrorCode
	now := r.clock.Now().UTC().Format(time.RFC3339Nano)
	result, err := r.db.ExecContext(ctx, `UPDATE jobs SET state=?,attempts=?,provider=?,result_reference=?,error_code=?,updated_at=? WHERE id=? AND workspace_id=? AND state=?`, string(next.State), next.Attempts, next.Provider, next.ResultReference, next.ErrorCode, now, in.ID, in.WorkspaceID, string(current.State))
	if err != nil {
		return Job{}, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Job{}, err
	}
	if rows != 1 {
		return Job{}, fmt.Errorf("job %q changed while transitioning", in.ID)
	}
	return next, nil
}

func scan(s interface{ Scan(...any) error }) (Job, error) {
	var job Job
	var state, revisions string
	if err := s.Scan(&job.ID, &job.WorkspaceID, &job.Type, &job.DedupKey, &job.RequestHash, &state, &revisions, &job.Attempts, &job.Provider, &job.BudgetUnits, &job.ResultReference, &job.ErrorCode); err != nil {
		return Job{}, err
	}
	job.State = State(state)
	if err := json.Unmarshal([]byte(revisions), &job.InputRevisions); err != nil {
		return Job{}, err
	}
	return job, nil
}
