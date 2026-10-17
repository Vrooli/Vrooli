package runs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

const runTimeFormat = time.RFC3339Nano

type sqliteRepository struct {
	db    *sql.DB
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production Repository. db is the
// connection pool opened in main.go; clk supplies StartedAt/FinishedAt
// timestamps when callers omit them so tests can advance time
// deterministically. The constructor runs an idempotent column
// migration so existing dev databases pick up failure_reason /
// missing_artifacts without a separate migration tool — additive only,
// no data movement.
func NewSQLiteRepository(db *sql.DB, clk schedule.Clock) Repository {
	_ = ensureRunColumns(context.Background(), db)
	return &sqliteRepository{db: db, clock: clk}
}

// ensureRunColumns inspects verification_runs and ALTERs the table to
// add any columns the current build expects but the on-disk schema
// hasn't declared yet. Safe to call repeatedly. Errors are intentionally
// best-effort: a brand-new database created from schema.sql already has
// every column, so the migration is a no-op there; if the migration
// fails on an exotic schema the next Insert call surfaces the error
// directly.
func ensureRunColumns(ctx context.Context, db *sql.DB) error {
	want := map[string]string{
		"failure_reason":    "TEXT NOT NULL DEFAULT ''",
		"missing_artifacts": "TEXT NOT NULL DEFAULT ''",
	}
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(verification_runs)")
	if err != nil {
		return err
	}
	defer rows.Close()
	have := map[string]struct{}{}
	for rows.Next() {
		var (
			cid     int
			name    string
			ctype   string
			notnull int
			dflt    sql.NullString
			pk      int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		have[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for col, def := range want {
		if _, ok := have[col]; ok {
			continue
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE verification_runs ADD COLUMN %s %s", col, def)); err != nil {
			return err
		}
	}
	return nil
}

var _ Repository = (*sqliteRepository)(nil)

const insertRunSQL = `
INSERT INTO verification_runs (
  id, flow_id, flow_path, root,
  source_sha256, model_sha256, gen_sha256,
  mode, status, counterexample, error_message,
  failure_reason, missing_artifacts, output,
  started_at, finished_at, duration_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

const selectRunByIDSQL = `
SELECT id, flow_id, flow_path, root,
       source_sha256, model_sha256, gen_sha256,
       mode, status, counterexample, error_message,
       failure_reason, missing_artifacts, output,
       started_at, finished_at, duration_ms
FROM verification_runs
WHERE id = ?
`

const listRunsBaseSQL = `
SELECT id, flow_id, flow_path, root,
       source_sha256, model_sha256, gen_sha256,
       mode, status, counterexample, error_message,
       failure_reason, missing_artifacts, output,
       started_at, finished_at, duration_ms
FROM verification_runs
`

func (s *sqliteRepository) Insert(ctx context.Context, run Run) (Run, error) {
	if run.ID == "" {
		run.ID = uuid.NewString()
	}
	if run.StartedAt.IsZero() {
		run.StartedAt = s.clock.Now().UTC()
	}
	if run.FinishedAt.IsZero() {
		run.FinishedAt = s.clock.Now().UTC()
	}
	if run.DurationMs == 0 {
		run.DurationMs = run.FinishedAt.Sub(run.StartedAt).Milliseconds()
	}
	if run.Mode == "" {
		run.Mode = ModeCheck
	}
	var ce any
	if run.Counterexample != "" {
		ce = run.Counterexample
	}
	missingArtifacts := ""
	if len(run.MissingArtifacts) > 0 {
		raw, err := json.Marshal(run.MissingArtifacts)
		if err != nil {
			return Run{}, fmt.Errorf("encode missing_artifacts: %w", err)
		}
		missingArtifacts = string(raw)
	}
	if _, err := s.db.ExecContext(ctx, insertRunSQL,
		run.ID,
		run.FlowID,
		run.FlowPath,
		run.Root,
		run.SourceSHA256,
		run.ModelSHA256,
		run.GenSHA256,
		string(run.Mode),
		string(run.Status),
		ce,
		run.ErrorMessage,
		run.FailureReason,
		missingArtifacts,
		run.Output,
		run.StartedAt.Format(runTimeFormat),
		run.FinishedAt.Format(runTimeFormat),
		run.DurationMs,
	); err != nil {
		return Run{}, fmt.Errorf("insert verification_run %q: %w", run.ID, err)
	}
	return run, nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Run, error) {
	row := s.db.QueryRowContext(ctx, selectRunByIDSQL, id)
	r, err := scanRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Run{}, ErrNotFound{ID: id}
	}
	if err != nil {
		return Run{}, fmt.Errorf("get verification_run %q: %w", id, err)
	}
	return r, nil
}

func (s *sqliteRepository) List(ctx context.Context, q ListQuery) ([]Run, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}
	query := listRunsBaseSQL
	args := []any{}
	if q.FlowID != "" {
		query += "WHERE flow_id = ?\n"
		args = append(args, q.FlowID)
	}
	query += "ORDER BY finished_at DESC, id DESC\nLIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list verification_runs: %w", err)
	}
	defer rows.Close()

	var out []Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan verification_run: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate verification_runs: %w", err)
	}
	return out, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRun(s rowScanner) (Run, error) {
	var (
		r           Run
		mode        string
		status      string
		ce          sql.NullString
		missingRaw  string
		startedRaw  string
		finishedRaw string
	)
	if err := s.Scan(
		&r.ID, &r.FlowID, &r.FlowPath, &r.Root,
		&r.SourceSHA256, &r.ModelSHA256, &r.GenSHA256,
		&mode, &status, &ce, &r.ErrorMessage,
		&r.FailureReason, &missingRaw, &r.Output,
		&startedRaw, &finishedRaw, &r.DurationMs,
	); err != nil {
		return Run{}, err
	}
	r.Mode = Mode(mode)
	r.Status = Status(status)
	if ce.Valid {
		r.Counterexample = ce.String
	}
	if strings.TrimSpace(missingRaw) != "" {
		if err := json.Unmarshal([]byte(missingRaw), &r.MissingArtifacts); err != nil {
			return Run{}, fmt.Errorf("decode missing_artifacts: %w", err)
		}
	}
	started, err := time.Parse(runTimeFormat, startedRaw)
	if err != nil {
		return Run{}, fmt.Errorf("parse started_at %q: %w", startedRaw, err)
	}
	finished, err := time.Parse(runTimeFormat, finishedRaw)
	if err != nil {
		return Run{}, fmt.Errorf("parse finished_at %q: %w", finishedRaw, err)
	}
	r.StartedAt = started
	r.FinishedAt = finished
	return r, nil
}
