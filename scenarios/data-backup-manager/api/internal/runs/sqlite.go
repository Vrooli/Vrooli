package runs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"data-backup-manager/internal/clock"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface the repository depends on.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

const runTimeFormat = time.RFC3339Nano

func (s *sqliteRepository) CreateRun(ctx context.Context, r Run) (Run, error) {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.StartedAt.IsZero() {
		r.StartedAt = s.clock.Now().UTC()
	}
	if r.Status == "" {
		r.Status = RunPending
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO runs (id, plan_id, trigger, status, started_at, finished_at) VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, r.PlanID, string(r.Trigger), string(r.Status), r.StartedAt.Format(runTimeFormat), formatTime(r.FinishedAt),
	)
	if err != nil {
		return Run{}, fmt.Errorf("insert run: %w", err)
	}
	return r, nil
}

func (s *sqliteRepository) SaveRun(ctx context.Context, r Run) (Run, error) {
	_, err := s.db.ExecContext(ctx,
		`UPDATE runs SET status = ?, finished_at = ? WHERE id = ?`,
		string(r.Status), formatTime(r.FinishedAt), r.ID,
	)
	if err != nil {
		return Run{}, fmt.Errorf("update run %q: %w", r.ID, err)
	}
	// Replace outcomes.
	if _, err := s.db.ExecContext(ctx, `DELETE FROM run_outcomes WHERE run_id = ?`, r.ID); err != nil {
		return Run{}, fmt.Errorf("clear outcomes %q: %w", r.ID, err)
	}
	for _, o := range r.Outcomes {
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO run_outcomes (run_id, target_id, destination_id, status, snapshot_id, bytes, error, started_at, finished_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			r.ID, o.TargetID, o.DestinationID, string(o.Status), o.SnapshotID, o.Bytes, o.Error,
			formatTime(o.StartedAt), formatTime(o.FinishedAt),
		); err != nil {
			return Run{}, fmt.Errorf("insert outcome %s/%s: %w", o.TargetID, o.DestinationID, err)
		}
	}
	return r, nil
}

func (s *sqliteRepository) GetRun(ctx context.Context, id string) (Run, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, plan_id, trigger, status, started_at, finished_at FROM runs WHERE id = ?`, id)
	r, err := scanRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Run{}, ErrRunNotFound{ID: id}
	}
	if err != nil {
		return Run{}, fmt.Errorf("get run %q: %w", id, err)
	}
	outcomes, err := s.outcomesFor(ctx, id)
	if err != nil {
		return Run{}, err
	}
	r.Outcomes = outcomes
	return r, nil
}

func (s *sqliteRepository) ListRuns(ctx context.Context, planID string, limit int) ([]Run, error) {
	if limit <= 0 {
		return nil, nil
	}
	var (
		rows *sql.Rows
		err  error
	)
	if planID != "" {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, plan_id, trigger, status, started_at, finished_at FROM runs WHERE plan_id = ? ORDER BY started_at DESC, id DESC LIMIT ?`,
			planID, limit)
	} else {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, plan_id, trigger, status, started_at, finished_at FROM runs ORDER BY started_at DESC, id DESC LIMIT ?`,
			limit)
	}
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}
	defer rows.Close()
	var runs []Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan run: %w", err)
		}
		runs = append(runs, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate runs: %w", err)
	}
	// Attach outcomes per run (small N for catalog views).
	for i := range runs {
		outcomes, err := s.outcomesFor(ctx, runs[i].ID)
		if err != nil {
			return nil, err
		}
		runs[i].Outcomes = outcomes
	}
	return runs, nil
}

func (s *sqliteRepository) TargetStatuses(ctx context.Context, targetIDs []string) ([]TargetStatus, error) {
	// Most recent run per target (status + run start), ordered so the first
	// row seen for a target is its latest run.
	query := `
SELECT ro.target_id, r.status, r.started_at
FROM run_outcomes ro JOIN runs r ON ro.run_id = r.id
%s
ORDER BY r.started_at DESC, r.id DESC`
	where := ""
	args := []any{}
	if len(targetIDs) > 0 {
		where = "WHERE ro.target_id IN (" + placeholders(len(targetIDs)) + ")"
		for _, id := range targetIDs {
			args = append(args, id)
		}
	}
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(query, where), args...)
	if err != nil {
		return nil, fmt.Errorf("target statuses: %w", err)
	}
	defer rows.Close()

	byTarget := map[string]*TargetStatus{}
	order := []string{}
	for rows.Next() {
		var targetID, status, startedRaw string
		if err := rows.Scan(&targetID, &status, &startedRaw); err != nil {
			return nil, fmt.Errorf("scan target status: %w", err)
		}
		if _, seen := byTarget[targetID]; seen {
			continue // first row per target is the latest run
		}
		ts := &TargetStatus{TargetID: targetID, LastRunStatus: RunStatus(status), LastRunAt: parseTime(startedRaw)}
		byTarget[targetID] = ts
		order = append(order, targetID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate target statuses: %w", err)
	}

	// Last successful backup per target (max finished_at over succeeded outcomes).
	succWhere := ""
	succArgs := []any{string(OutcomeSucceeded)}
	if len(targetIDs) > 0 {
		succWhere = "AND target_id IN (" + placeholders(len(targetIDs)) + ")"
		for _, id := range targetIDs {
			succArgs = append(succArgs, id)
		}
	}
	srows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT target_id, MAX(finished_at) FROM run_outcomes WHERE status = ? %s GROUP BY target_id`, succWhere),
		succArgs...)
	if err != nil {
		return nil, fmt.Errorf("last success: %w", err)
	}
	defer srows.Close()
	for srows.Next() {
		var targetID, maxFinished string
		if err := srows.Scan(&targetID, &maxFinished); err != nil {
			return nil, fmt.Errorf("scan last success: %w", err)
		}
		ts, ok := byTarget[targetID]
		if !ok {
			ts = &TargetStatus{TargetID: targetID}
			byTarget[targetID] = ts
			order = append(order, targetID)
		}
		ts.LastSuccessAt = parseTime(maxFinished)
	}
	if err := srows.Err(); err != nil {
		return nil, fmt.Errorf("iterate last success: %w", err)
	}

	out := make([]TargetStatus, 0, len(order))
	for _, id := range order {
		out = append(out, *byTarget[id])
	}
	return out, nil
}

func (s *sqliteRepository) outcomesFor(ctx context.Context, runID string) ([]TargetOutcome, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT target_id, destination_id, status, snapshot_id, bytes, error, started_at, finished_at
		 FROM run_outcomes WHERE run_id = ? ORDER BY target_id, destination_id`, runID)
	if err != nil {
		return nil, fmt.Errorf("outcomes for %q: %w", runID, err)
	}
	defer rows.Close()
	var outcomes []TargetOutcome
	for rows.Next() {
		var (
			o                     TargetOutcome
			status                string
			startedRaw, finishRaw string
		)
		if err := rows.Scan(&o.TargetID, &o.DestinationID, &status, &o.SnapshotID, &o.Bytes, &o.Error, &startedRaw, &finishRaw); err != nil {
			return nil, fmt.Errorf("scan outcome: %w", err)
		}
		o.Status = OutcomeStatus(status)
		o.StartedAt = parseTime(startedRaw)
		o.FinishedAt = parseTime(finishRaw)
		outcomes = append(outcomes, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outcomes: %w", err)
	}
	return outcomes, nil
}

type rowScanner interface{ Scan(dest ...any) error }

func scanRun(sc rowScanner) (Run, error) {
	var (
		r                     Run
		trigger, status       string
		startedRaw, finishRaw string
	)
	if err := sc.Scan(&r.ID, &r.PlanID, &trigger, &status, &startedRaw, &finishRaw); err != nil {
		return Run{}, err
	}
	r.Trigger = TriggerSource(trigger)
	r.Status = RunStatus(status)
	r.StartedAt = parseTime(startedRaw)
	r.FinishedAt = parseTime(finishRaw)
	return r, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(runTimeFormat)
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(runTimeFormat, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}
