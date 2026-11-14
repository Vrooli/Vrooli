package codingjobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Create(ctx context.Context, item Job) (Job, error) {
	result, err := r.db.ExecContext(ctx, `INSERT INTO coding_jobs (id,replay_key,case_id,project,title,task,state,source_seal_ref,solver_run_ref,evaluator_receipt_ref,evaluator_receipt_sha256,score,revision,events_json,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT DO NOTHING`, item.ID, item.ReplayKey, item.CaseID, item.Project, item.Title, item.Task, item.State, item.SourceSealRef, item.SolverRunRef, item.EvaluatorReceiptRef, item.EvaluatorReceiptSHA256, item.Score, item.Revision, encodeEvents(item.Events), item.CreatedAt.Format(time.RFC3339Nano), item.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Job{}, fmt.Errorf("insert coding job %q: %w", item.ID, err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return Job{}, fmt.Errorf("read coding job insert result: %w", err)
	}
	if inserted == 1 {
		return item, nil
	}
	existing, err := r.GetByReplayKey(ctx, item.ReplayKey)
	if err != nil {
		return Job{}, ErrConflict{Reason: "coding job insert conflicted without a matching replay key"}
	}
	if !sameCreateRequest(existing, item) {
		return Job{}, ErrConflict{Reason: "replay_key already identifies a different coding job request"}
	}
	return existing, nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Job, error) {
	item, err := scan(r.db.QueryRowContext(ctx, selectJob+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, ErrNotFound{ID: id}
	}
	if err != nil {
		return Job{}, fmt.Errorf("get coding job %q: %w", id, err)
	}
	return item, nil
}

func (r *sqliteRepository) GetByReplayKey(ctx context.Context, replayKey string) (Job, error) {
	item, err := scan(r.db.QueryRowContext(ctx, selectJob+` WHERE replay_key = ?`, replayKey))
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, ErrNotFound{ID: "replay_key:" + replayKey}
	}
	if err != nil {
		return Job{}, fmt.Errorf("get coding job by replay key: %w", err)
	}
	return item, nil
}

func (r *sqliteRepository) List(ctx context.Context, limit int) ([]Job, error) {
	if limit <= 0 {
		return []Job{}, nil
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, selectJob+` ORDER BY created_at DESC,id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list coding jobs: %w", err)
	}
	defer rows.Close()
	items := make([]Job, 0, limit)
	for rows.Next() {
		item, e := scan(rows)
		if e != nil {
			return nil, e
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate coding jobs: %w", err)
	}
	return items, nil
}

func (r *sqliteRepository) Update(ctx context.Context, item Job, expectedRevision int64) (Job, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE coding_jobs SET state=?,solver_run_ref=?,evaluator_receipt_ref=?,evaluator_receipt_sha256=?,score=?,revision=?,events_json=?,updated_at=? WHERE id=? AND revision=?`, item.State, item.SolverRunRef, item.EvaluatorReceiptRef, item.EvaluatorReceiptSHA256, item.Score, item.Revision, encodeEvents(item.Events), item.UpdatedAt.Format(time.RFC3339Nano), item.ID, expectedRevision)
	if err != nil {
		return Job{}, fmt.Errorf("update coding job %q: %w", item.ID, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return Job{}, fmt.Errorf("read coding job update result: %w", err)
	}
	if n == 0 {
		if _, getErr := r.Get(ctx, item.ID); getErr != nil {
			return Job{}, getErr
		}
		return Job{}, ErrConflict{Reason: "coding job changed concurrently; reload its current revision"}
	}
	return item, nil
}

func sameCreateRequest(existing, requested Job) bool {
	return existing.ReplayKey == requested.ReplayKey && existing.CaseID == requested.CaseID && existing.Project == requested.Project && existing.Title == requested.Title && existing.Task == requested.Task && existing.SourceSealRef == requested.SourceSealRef
}

func encodeEvents(events []Event) string {
	b, _ := json.Marshal(events)
	return string(b)
}

const selectJob = `SELECT id,replay_key,case_id,project,title,task,state,source_seal_ref,solver_run_ref,evaluator_receipt_ref,evaluator_receipt_sha256,score,revision,events_json,created_at,updated_at FROM coding_jobs`

type scanner interface{ Scan(...any) error }

func scan(row scanner) (Job, error) {
	var item Job
	var events, created, updated string
	err := row.Scan(&item.ID, &item.ReplayKey, &item.CaseID, &item.Project, &item.Title, &item.Task, &item.State, &item.SourceSealRef, &item.SolverRunRef, &item.EvaluatorReceiptRef, &item.EvaluatorReceiptSHA256, &item.Score, &item.Revision, &events, &created, &updated)
	if err != nil {
		return Job{}, err
	}
	if err = json.Unmarshal([]byte(events), &item.Events); err != nil {
		return Job{}, fmt.Errorf("decode coding job events: %w", err)
	}
	if item.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
		return Job{}, fmt.Errorf("parse coding job created_at: %w", err)
	}
	if item.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated); err != nil {
		return Job{}, fmt.Errorf("parse coding job updated_at: %w", err)
	}
	return item, nil
}
