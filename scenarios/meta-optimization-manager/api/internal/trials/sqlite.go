package trials

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"meta-optimization-manager/internal/clock"
)

const runTimeFormat = time.RFC3339Nano

// SQLExecutor is the narrow database surface the repository depends on (declared
// at the consumer per seam-discovery). Both *sql.DB (tests) and
// *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepo struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteRepository constructs the production trials Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepo{db: db, clock: clk}
}

var _ Repository = (*sqliteRepo)(nil)

const (
	insertRunSQL = `INSERT INTO trials_runs
(id, task_id, suite, model, guide_task_id, verdict, tokens, duration_ms, sandbox_diff_ref, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	bumpGateSQL = `INSERT INTO trial_gates (task_key, gate_count, updated_at)
VALUES (?, 1, ?)
ON CONFLICT(task_key) DO UPDATE SET gate_count = gate_count + 1, updated_at = excluded.updated_at`
	getRunSQL = `SELECT id, task_id, suite, model, guide_task_id, verdict, tokens, duration_ms, sandbox_diff_ref, created_at
FROM trials_runs WHERE id = ?`
	gatedCountSQL = `SELECT COUNT(*) FROM trial_gates WHERE gate_count > 0`
)

func (r *sqliteRepo) RecordRun(ctx context.Context, run TrialRun) error {
	at := run.At
	if at.IsZero() {
		at = r.clock.Now().UTC()
	}
	if _, err := r.db.ExecContext(ctx, insertRunSQL,
		run.ID, run.TaskID, run.Suite, run.Model, run.GuideTaskID, int(run.Verdict),
		run.Tokens, run.DurationMs, run.SandboxDiffRef, at.Format(runTimeFormat),
	); err != nil {
		return fmt.Errorf("insert run %q: %w", run.ID, err)
	}
	if run.GuideTaskID != "" {
		if _, err := r.db.ExecContext(ctx, bumpGateSQL, run.GuideTaskID, at.Format(runTimeFormat)); err != nil {
			return fmt.Errorf("bump gate %q: %w", run.GuideTaskID, err)
		}
	}
	return nil
}

func (r *sqliteRepo) GetRun(ctx context.Context, id string) (TrialRun, bool, error) {
	run, err := scanRun(r.db.QueryRowContext(ctx, getRunSQL, id))
	if errors.Is(err, sql.ErrNoRows) {
		return TrialRun{}, false, nil
	}
	if err != nil {
		return TrialRun{}, false, fmt.Errorf("get run %q: %w", id, err)
	}
	return run, true, nil
}

func (r *sqliteRepo) Runs(ctx context.Context, filter RunFilter, limit int, desc bool) ([]TrialRun, error) {
	q := `SELECT id, task_id, suite, model, guide_task_id, verdict, tokens, duration_ms, sandbox_diff_ref, created_at
FROM trials_runs`
	var (
		where []string
		args  []any
	)
	if filter.TaskID != "" {
		where = append(where, "task_id = ?")
		args = append(args, filter.TaskID)
	}
	if filter.Suite != "" {
		where = append(where, "suite = ?")
		args = append(args, filter.Suite)
	}
	if len(where) > 0 {
		q += " WHERE " + joinAnd(where)
	}
	if desc {
		q += " ORDER BY created_at DESC, id DESC"
	} else {
		q += " ORDER BY created_at ASC, id ASC"
	}
	if limit > 0 {
		q += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query runs: %w", err)
	}
	defer rows.Close()
	var out []TrialRun
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate runs: %w", err)
	}
	return out, nil
}

func (r *sqliteRepo) GatedGuideTaskCount(ctx context.Context) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, gatedCountSQL).Scan(&n); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("gated count: %w", err)
	}
	return n, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRun(s rowScanner) (TrialRun, error) {
	var (
		run     TrialRun
		verdict int
		atRaw   string
	)
	if err := s.Scan(&run.ID, &run.TaskID, &run.Suite, &run.Model, &run.GuideTaskID,
		&verdict, &run.Tokens, &run.DurationMs, &run.SandboxDiffRef, &atRaw); err != nil {
		return TrialRun{}, err
	}
	run.Verdict = Verdict(verdict)
	if t, err := time.Parse(runTimeFormat, atRaw); err == nil {
		run.At = t
	}
	return run, nil
}

func joinAnd(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " AND "
		}
		out += p
	}
	return out
}
