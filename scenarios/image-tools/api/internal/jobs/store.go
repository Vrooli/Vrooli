package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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

// migrate adds columns to an install created before they existed. SQLite has no
// ADD COLUMN IF NOT EXISTS, so a duplicate-column error is the steady state
// rather than a failure. Without this a running install keeps the old table —
// CREATE TABLE IF NOT EXISTS is a no-op there — and every insert fails on the
// column count.
func (s *store) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `ALTER TABLE jobs ADD COLUMN meta TEXT NOT NULL DEFAULT ''`)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return fmt.Errorf("jobs: migrate schema: %w", err)
	}
	return nil
}

const insertJobSQL = `
INSERT INTO jobs (id, operation, lane, state, progress, message, error, result_ref, meta, payload, estimated_seconds, created_at, started_at, finished_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

func (s *store) insert(ctx context.Context, j Job) error {
	_, err := s.db.ExecContext(ctx, insertJobSQL,
		j.ID, j.Operation, string(j.Lane), string(j.State), j.Progress, j.Message, j.Error, j.ResultRef, encodeMeta(j.Meta),
		j.Payload, j.EstimatedSeconds, j.CreatedAt.Format(jobTimeFormat), formatTimePtr(j.StartedAt), formatTimePtr(j.FinishedAt),
	)
	if err != nil {
		return fmt.Errorf("jobs: insert: %w", err)
	}
	return nil
}

const updateJobSQL = `
UPDATE jobs SET state=?, progress=?, message=?, error=?, result_ref=?, meta=?, started_at=?, finished_at=?
WHERE id=?`

func (s *store) update(ctx context.Context, j Job) error {
	_, err := s.db.ExecContext(ctx, updateJobSQL,
		string(j.State), j.Progress, j.Message, j.Error, j.ResultRef, encodeMeta(j.Meta),
		formatTimePtr(j.StartedAt), formatTimePtr(j.FinishedAt), j.ID,
	)
	if err != nil {
		return fmt.Errorf("jobs: update: %w", err)
	}
	return nil
}

const selectJobColumns = `id, operation, lane, state, progress, message, error, result_ref, meta, payload, estimated_seconds, created_at, started_at, finished_at`

// encodeMeta stores an absent map as the empty string rather than "{}" so a row
// that carries no backend record is distinguishable from one that carries an
// empty record, which is the same distinction ai.MetaCostUSD is written to
// preserve: "cost nothing" and "nobody reported a cost" are different facts.
func encodeMeta(meta map[string]string) string {
	if len(meta) == 0 {
		return ""
	}
	encoded, err := json.Marshal(meta)
	if err != nil {
		// Unreachable: a map[string]string always marshals.
		return ""
	}
	return string(encoded)
}

func decodeMeta(raw string) map[string]string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var meta map[string]string
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil
	}
	return meta
}

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
		meta                  string
		payload               []byte
	)
	if err := sc.Scan(
		&j.ID, &j.Operation, &lane, &state, &j.Progress, &j.Message, &j.Error, &j.ResultRef, &meta,
		&payload, &j.EstimatedSeconds, &createdAt, &startedAt, &finishedAt,
	); err != nil {
		return Job{}, fmt.Errorf("jobs: scan: %w", err)
	}
	j.Lane = Lane(lane)
	j.State = State(state)
	j.Meta = decodeMeta(meta)
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
