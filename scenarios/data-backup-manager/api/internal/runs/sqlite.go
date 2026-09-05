package runs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"data-backup-manager/internal/failures"

	"github.com/vrooli/api-core/schedule"

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
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk schedule.Clock) Repository {
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
	if r.UpdatedAt.IsZero() {
		r.UpdatedAt = r.StartedAt
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO runs (id, plan_id, trigger, status, started_at, finished_at, error, failure_code, failure_category, next_action, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.PlanID, string(r.Trigger), string(r.Status), r.StartedAt.Format(runTimeFormat), formatTime(r.FinishedAt), r.Error, string(r.FailureCode), string(r.FailureCategory), r.NextAction, formatTime(r.UpdatedAt),
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "runs_one_active_plan") || strings.Contains(strings.ToLower(err.Error()), "unique constraint failed: runs.plan_id") {
			return Run{}, ErrRunAlreadyActive{PlanID: r.PlanID}
		}
		return Run{}, fmt.Errorf("insert run: %w", err)
	}
	return r, nil
}

func (s *sqliteRepository) UpdateRunStatus(ctx context.Context, runID string, status RunStatus) error {
	now := formatTime(s.clock.Now().UTC())
	if _, err := s.db.ExecContext(ctx,
		`UPDATE runs SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), now, runID,
	); err != nil {
		return fmt.Errorf("update run status %q: %w", runID, err)
	}
	return nil
}

func (s *sqliteRepository) SaveOutcome(ctx context.Context, runID string, o TargetOutcome) error {
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO run_outcomes (run_id, target_id, destination_id, status, snapshot_id, bytes, error, failure_code, failure_category, warning, started_at, finished_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(run_id, target_id, destination_id) DO UPDATE SET
		   status = excluded.status, snapshot_id = excluded.snapshot_id, bytes = excluded.bytes,
		   error = excluded.error, failure_code = excluded.failure_code, failure_category = excluded.failure_category,
		   warning = excluded.warning, started_at = excluded.started_at, finished_at = excluded.finished_at`,
		runID, o.TargetID, o.DestinationID, string(o.Status), o.SnapshotID, o.Bytes, o.Error, string(o.FailureCode), string(o.FailureCategory), o.Warning,
		formatTime(o.StartedAt), formatTime(o.FinishedAt),
	); err != nil {
		return fmt.Errorf("save outcome %s/%s: %w", o.TargetID, o.DestinationID, err)
	}
	// Heartbeat: a run actively writing outcomes is alive.
	if _, err := s.db.ExecContext(ctx,
		`UPDATE runs SET updated_at = ? WHERE id = ?`, formatTime(s.clock.Now().UTC()), runID,
	); err != nil {
		return fmt.Errorf("heartbeat run %q: %w", runID, err)
	}
	return nil
}

func (s *sqliteRepository) FinishRun(ctx context.Context, runID string, status RunStatus, errMsg string, finishedAt time.Time, physicalBytes int64) error {
	now := formatTime(s.clock.Now().UTC())
	if physicalBytes < 0 {
		physicalBytes = 0
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE runs SET status = ?, error = ?, finished_at = ?, updated_at = ?, physical_bytes = ? WHERE id = ?`,
		string(status), errMsg, formatTime(finishedAt), now, physicalBytes, runID,
	); err != nil {
		return fmt.Errorf("finish run %q: %w", runID, err)
	}
	return nil
}

func (s *sqliteRepository) UpdateRunFailure(ctx context.Context, runID string, code failures.Code, category failures.Category, nextAction string) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE runs SET failure_code = ?, failure_category = ?, next_action = ?, updated_at = ? WHERE id = ?`,
		string(code), string(category), nextAction, formatTime(s.clock.Now().UTC()), runID); err != nil {
		return fmt.Errorf("update run failure %q: %w", runID, err)
	}
	return nil
}

func (s *sqliteRepository) SavePreflightIncidents(ctx context.Context, runID string, incidents []failures.Cause) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM run_incidents WHERE run_id = ?`, runID); err != nil {
		return fmt.Errorf("clear run incidents %q: %w", runID, err)
	}
	for _, c := range incidents {
		ids, err := json.Marshal(c.TargetIDs)
		if err != nil {
			return fmt.Errorf("encode run incident targets: %w", err)
		}
		if _, err := s.db.ExecContext(ctx, `INSERT INTO run_incidents (run_id, code, category, scope, message, next_action, destination_id, target_ids, first_observed, last_observed, last_known_good, retryable, retry_after_seconds) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			runID, string(c.Code), string(c.Category), string(c.Scope), c.Message, c.NextAction, c.DestinationID, string(ids), formatTime(c.FirstObserved), formatTime(c.LastObserved), formatTime(c.LastKnownGood), c.Retryable, int64(c.RetryAfter.Seconds())); err != nil {
			return fmt.Errorf("save run incident %q: %w", runID, err)
		}
	}
	return nil
}

func (s *sqliteRepository) IncidentsForRun(ctx context.Context, runID string) ([]failures.Cause, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT code, category, scope, message, next_action, destination_id, target_ids, first_observed, last_observed, last_known_good, retryable, retry_after_seconds FROM run_incidents WHERE run_id = ? ORDER BY code, scope, destination_id`, runID)
	if err != nil {
		return nil, fmt.Errorf("list run incidents %q: %w", runID, err)
	}
	defer rows.Close()
	var out []failures.Cause
	for rows.Next() {
		var c failures.Cause
		var code, category, scope, targetJSON, first, last, good string
		var retryable int
		var retryAfter int64
		if err := rows.Scan(&code, &category, &scope, &c.Message, &c.NextAction, &c.DestinationID, &targetJSON, &first, &last, &good, &retryable, &retryAfter); err != nil {
			return nil, fmt.Errorf("scan run incident: %w", err)
		}
		c.Code, c.Category, c.Scope = failures.Code(code), failures.Category(category), failures.Scope(scope)
		c.FirstObserved, c.LastObserved, c.LastKnownGood = parseTime(first), parseTime(last), parseTime(good)
		c.Retryable, c.RetryAfter = retryable != 0, time.Duration(retryAfter)*time.Second
		if err := json.Unmarshal([]byte(targetJSON), &c.TargetIDs); err != nil {
			return nil, fmt.Errorf("decode run incident targets: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate run incidents: %w", err)
	}
	return out, nil
}

func (s *sqliteRepository) ListNonTerminalRuns(ctx context.Context) ([]Run, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, plan_id, trigger, status, started_at, finished_at, error, failure_code, failure_category, next_action, updated_at, physical_bytes
		 FROM runs WHERE status IN (?, ?, ?) ORDER BY started_at ASC, id ASC`,
		string(RunPending), string(RunCapturing), string(RunSnapshotting))
	if err != nil {
		return nil, fmt.Errorf("list non-terminal runs: %w", err)
	}
	defer rows.Close()
	var out []Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan non-terminal run: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate non-terminal runs: %w", err)
	}
	return out, nil
}

func (s *sqliteRepository) GetRun(ctx context.Context, id string) (Run, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, plan_id, trigger, status, started_at, finished_at, error, failure_code, failure_category, next_action, updated_at, physical_bytes FROM runs WHERE id = ?`, id)
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
	incidents, err := s.IncidentsForRun(ctx, id)
	if err != nil {
		return Run{}, err
	}
	r.Preflight = incidents
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
			`SELECT id, plan_id, trigger, status, started_at, finished_at, error, failure_code, failure_category, next_action, updated_at, physical_bytes FROM runs WHERE plan_id = ? ORDER BY started_at DESC, id DESC LIMIT ?`,
			planID, limit)
	} else {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, plan_id, trigger, status, started_at, finished_at, error, failure_code, failure_category, next_action, updated_at, physical_bytes FROM runs ORDER BY started_at DESC, id DESC LIMIT ?`,
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
		incidents, err := s.IncidentsForRun(ctx, runs[i].ID)
		if err != nil {
			return nil, err
		}
		runs[i].Preflight = incidents
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
		`SELECT target_id, destination_id, status, snapshot_id, bytes, error, failure_code, failure_category, warning, started_at, finished_at
		 FROM run_outcomes WHERE run_id = ? ORDER BY target_id, destination_id`, runID)
	if err != nil {
		return nil, fmt.Errorf("outcomes for %q: %w", runID, err)
	}
	defer rows.Close()
	var outcomes []TargetOutcome
	for rows.Next() {
		var (
			o                                    TargetOutcome
			status, failureCode, failureCategory string
			warning                              string
			startedRaw, finishRaw                string
		)
		if err := rows.Scan(&o.TargetID, &o.DestinationID, &status, &o.SnapshotID, &o.Bytes, &o.Error, &failureCode, &failureCategory, &warning, &startedRaw, &finishRaw); err != nil {
			return nil, fmt.Errorf("scan outcome: %w", err)
		}
		o.Status = OutcomeStatus(status)
		o.FailureCode = failures.Code(failureCode)
		o.FailureCategory = failures.Category(failureCategory)
		o.Warning = warning
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
		r                                             Run
		trigger, status, failureCode, failureCategory string
		nextAction                                    string
		startedRaw, finishRaw, updedRaw               string
	)
	if err := sc.Scan(&r.ID, &r.PlanID, &trigger, &status, &startedRaw, &finishRaw, &r.Error, &failureCode, &failureCategory, &nextAction, &updedRaw, &r.PhysicalBytes); err != nil {
		return Run{}, err
	}
	r.Trigger = TriggerSource(trigger)
	r.Status = RunStatus(status)
	r.FailureCode = failures.Code(failureCode)
	r.FailureCategory = failures.Category(failureCategory)
	r.NextAction = nextAction
	r.StartedAt = parseTime(startedRaw)
	r.FinishedAt = parseTime(finishRaw)
	r.UpdatedAt = parseTime(updedRaw)
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
