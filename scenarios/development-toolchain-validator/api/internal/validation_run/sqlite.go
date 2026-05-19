package validation_run

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	vr "development-toolchain-validator/internal/validation_record"
)

const runTimeFormat = time.RFC3339Nano

const insertRunSQL = `
INSERT INTO validation_runs (
  id, tuple_kind, subject_id, golden_slug, status, terminal_verdict,
  agent_manager_run_id, created_at, started_at, ended_at, error_message, force_re_run
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const updateRunSQL = `
UPDATE validation_runs SET
  status               = ?,
  terminal_verdict     = ?,
  agent_manager_run_id = ?,
  started_at           = ?,
  ended_at             = ?,
  error_message        = ?
WHERE id = ?
`

const selectRunByIDSQL = `
SELECT id, tuple_kind, subject_id, golden_slug, status, terminal_verdict,
       agent_manager_run_id, created_at, started_at, ended_at, error_message, force_re_run
FROM validation_runs
WHERE id = ?
`

const listActiveRunsSQL = `
SELECT id, tuple_kind, subject_id, golden_slug, status, terminal_verdict,
       agent_manager_run_id, created_at, started_at, ended_at, error_message, force_re_run
FROM validation_runs
WHERE status != ?
ORDER BY created_at DESC
`

const findRecentMatchingSQL = `
SELECT id, tuple_kind, subject_id, golden_slug, status, terminal_verdict,
       agent_manager_run_id, created_at, started_at, ended_at, error_message, force_re_run
FROM validation_runs
WHERE tuple_kind = ? AND subject_id = ? AND golden_slug = ?
ORDER BY created_at DESC
LIMIT 1
`

const claimQueuedSQL = `
UPDATE validation_runs SET status = ?
WHERE id = (
  SELECT id FROM validation_runs WHERE status = ?
  ORDER BY created_at ASC LIMIT 1
)
RETURNING id, tuple_kind, subject_id, golden_slug, status, terminal_verdict,
          agent_manager_run_id, created_at, started_at, ended_at, error_message, force_re_run
`

type sqliteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db *sql.DB) Repository {
	return &sqliteRepository{db: db}
}

var _ Repository = (*sqliteRepository)(nil)

func (s *sqliteRepository) Create(ctx context.Context, r Run) error {
	_, err := s.db.ExecContext(ctx, insertRunSQL,
		r.ID, int(r.TupleKind), r.SubjectID, r.GoldenSlug,
		int(r.Status), int(r.TerminalVerdict),
		r.AgentManagerRunID,
		r.CreatedAt.UTC().Format(runTimeFormat),
		formatNullable(r.StartedAt), formatNullable(r.EndedAt),
		r.ErrorMessage, boolToInt(r.ForceReRun),
	)
	if err != nil {
		return fmt.Errorf("create validation_run %q: %w", r.ID, err)
	}
	return nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Run, error) {
	row := s.db.QueryRowContext(ctx, selectRunByIDSQL, id)
	r, err := scanRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Run{}, ErrRunNotFound{ID: id}
	}
	if err != nil {
		return Run{}, fmt.Errorf("get validation_run %q: %w", id, err)
	}
	return r, nil
}

func (s *sqliteRepository) UpdateStatus(ctx context.Context, r Run) error {
	res, err := s.db.ExecContext(ctx, updateRunSQL,
		int(r.Status), int(r.TerminalVerdict), r.AgentManagerRunID,
		formatNullable(r.StartedAt), formatNullable(r.EndedAt),
		r.ErrorMessage, r.ID,
	)
	if err != nil {
		return fmt.Errorf("update validation_run %q: %w", r.ID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update rows affected %q: %w", r.ID, err)
	}
	if n == 0 {
		return ErrRunNotFound{ID: r.ID}
	}
	return nil
}

func (s *sqliteRepository) ListActive(ctx context.Context) ([]Run, error) {
	rows, err := s.db.QueryContext(ctx, listActiveRunsSQL, int(StatusTerminal))
	if err != nil {
		return nil, fmt.Errorf("list active runs: %w", err)
	}
	defer rows.Close()
	var out []Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan run: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate runs: %w", err)
	}
	return out, nil
}

func (s *sqliteRepository) FindRecentMatching(ctx context.Context, kind int, subjectID, goldenSlug string) (Run, error) {
	row := s.db.QueryRowContext(ctx, findRecentMatchingSQL, kind, subjectID, goldenSlug)
	r, err := scanRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Run{}, ErrRunNotFound{}
	}
	if err != nil {
		return Run{}, fmt.Errorf("find recent matching: %w", err)
	}
	return r, nil
}

func (s *sqliteRepository) ClaimNextQueued(ctx context.Context) (Run, error) {
	row := s.db.QueryRowContext(ctx, claimQueuedSQL, int(StatusRunning), int(StatusQueued))
	r, err := scanRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Run{}, ErrRunNotFound{}
	}
	if err != nil {
		return Run{}, fmt.Errorf("claim queued: %w", err)
	}
	return r, nil
}

func formatNullable(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(runTimeFormat)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRun(s rowScanner) (Run, error) {
	var (
		r              Run
		tupleKindInt   int
		statusInt      int
		verdictInt     int
		createdRaw     string
		startedRaw     string
		endedRaw       string
		forceInt       int
	)
	if err := s.Scan(
		&r.ID, &tupleKindInt, &r.SubjectID, &r.GoldenSlug,
		&statusInt, &verdictInt,
		&r.AgentManagerRunID,
		&createdRaw, &startedRaw, &endedRaw,
		&r.ErrorMessage, &forceInt,
	); err != nil {
		return Run{}, err
	}
	r.TupleKind = vr.TupleKind(tupleKindInt)
	r.Status = Status(statusInt)
	r.TerminalVerdict = vr.Verdict(verdictInt)
	r.ForceReRun = forceInt != 0
	t, err := time.Parse(runTimeFormat, createdRaw)
	if err != nil {
		return Run{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	r.CreatedAt = t
	if startedRaw != "" {
		t, err := time.Parse(runTimeFormat, startedRaw)
		if err != nil {
			return Run{}, fmt.Errorf("parse started_at %q: %w", startedRaw, err)
		}
		r.StartedAt = t
	}
	if endedRaw != "" {
		t, err := time.Parse(runTimeFormat, endedRaw)
		if err != nil {
			return Run{}, fmt.Errorf("parse ended_at %q: %w", endedRaw, err)
		}
		r.EndedAt = t
	}
	return r, nil
}
