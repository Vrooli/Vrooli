package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const jobTimeFormat = time.RFC3339Nano

// SQLExecutor is the narrow database surface the store depends on. Both *sql.DB
// (unit tests) and *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// store persists Job records.
type store struct {
	db SQLExecutor
}

func newStore(db SQLExecutor) *store { return &store{db: db} }

const insertJobSQL = `
INSERT INTO jobs (id, operation, lane, state, progress, message, error, result_ref, payload, estimated_seconds, created_at, started_at, finished_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

func (s *store) insert(ctx context.Context, j Job) error {
	_, err := s.db.ExecContext(ctx, insertJobSQL,
		j.ID, j.Operation, string(j.Lane), string(j.State), j.Progress, j.Message, j.Error, j.ResultRef,
		j.Payload, j.EstimatedSeconds, j.CreatedAt.Format(jobTimeFormat), formatTimePtr(j.StartedAt), formatTimePtr(j.FinishedAt),
	)
	if err != nil {
		return fmt.Errorf("jobs: insert: %w", err)
	}
	return nil
}

const updateJobSQL = `
UPDATE jobs SET state=?, progress=?, message=?, error=?, result_ref=?, started_at=?, finished_at=?
WHERE id=?`

func (s *store) update(ctx context.Context, j Job) error {
	_, err := s.db.ExecContext(ctx, updateJobSQL,
		string(j.State), j.Progress, j.Message, j.Error, j.ResultRef,
		formatTimePtr(j.StartedAt), formatTimePtr(j.FinishedAt), j.ID,
	)
	if err != nil {
		return fmt.Errorf("jobs: update: %w", err)
	}
	return nil
}

const selectJobColumns = `id, operation, lane, state, progress, message, error, result_ref, payload, estimated_seconds, created_at, started_at, finished_at`

func (s *store) get(ctx context.Context, id string) (Job, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+selectJobColumns+" FROM jobs WHERE id=?", id)
	return scanJob(row)
}

func (s *store) list(ctx context.Context, limit int) ([]Job, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, "SELECT "+selectJobColumns+" FROM jobs ORDER BY created_at DESC LIMIT ?", limit)
	if err != nil {
		return nil, fmt.Errorf("jobs: list: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("jobs: list rows: %w", err)
	}
	return out, nil
}

// listNonTerminal returns jobs still queued/running (used at boot for recovery).
func (s *store) listNonTerminal(ctx context.Context) ([]Job, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+selectJobColumns+" FROM jobs WHERE state IN (?, ?)", string(StateQueued), string(StateRunning))
	if err != nil {
		return nil, fmt.Errorf("jobs: list non-terminal: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanJob(sc scanner) (Job, error) {
	var (
		j                     Job
		lane, state           string
		createdAt             string
		startedAt, finishedAt string
		payload               []byte
	)
	if err := sc.Scan(
		&j.ID, &j.Operation, &lane, &state, &j.Progress, &j.Message, &j.Error, &j.ResultRef,
		&payload, &j.EstimatedSeconds, &createdAt, &startedAt, &finishedAt,
	); err != nil {
		return Job{}, fmt.Errorf("jobs: scan: %w", err)
	}
	j.Lane = Lane(lane)
	j.State = State(state)
	j.Payload = payload
	t, err := time.Parse(jobTimeFormat, createdAt)
	if err != nil {
		return Job{}, fmt.Errorf("jobs: parse created_at: %w", err)
	}
	j.CreatedAt = t
	j.StartedAt = parseTimePtr(startedAt)
	j.FinishedAt = parseTimePtr(finishedAt)
	return j, nil
}

func formatTimePtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format(jobTimeFormat)
}

func parseTimePtr(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(jobTimeFormat, s)
	if err != nil {
		return nil
	}
	return &t
}
