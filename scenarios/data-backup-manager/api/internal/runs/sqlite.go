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

func (s *sqliteRepository) UpdateRunStatus(ctx context.Context, runID string, event RunEvent) error {
	return s.transitionRun(ctx, runID, event, false, func(next, current RunStatus) error {
		result, err := s.db.ExecContext(ctx,
			`UPDATE runs SET status = ?, updated_at = ? WHERE id = ? AND status = ?`,
			string(next), formatTime(s.clock.Now().UTC()), runID, string(current),
		)
		return requireRunTransitionWrite(result, err, runID, current, event)
	})
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

func (s *sqliteRepository) FinishRun(ctx context.Context, runID string, event RunEvent, errMsg string, finishedAt time.Time, physicalBytes int64) error {
	run, err := s.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	current := run.Status
	if err := run.TransitionStatus(event); err != nil {
		return err
	}
	if !terminal(run.Status) {
		return fmt.Errorf("finish run requires a terminal state, got %q", run.Status)
	}
	run.Error = errMsg
	if err := CheckInvariants(run); err != nil {
		return err
	}
	now := formatTime(s.clock.Now().UTC())
	if physicalBytes < 0 {
		physicalBytes = 0
	}
	result, err := s.db.ExecContext(ctx,
		`UPDATE runs SET status = ?, error = ?, finished_at = ?, updated_at = ?, physical_bytes = ? WHERE id = ? AND status = ?`,
		string(run.Status), errMsg, formatTime(finishedAt), now, physicalBytes, runID, string(current),
	)
	if err := requireRunTransitionWrite(result, err, runID, current, event); err != nil {
		return fmt.Errorf("finish run %q: %w", runID, err)
	}
	return nil
}

func (s *sqliteRepository) transitionRun(ctx context.Context, runID string, event RunEvent, terminalWrite bool, persist func(next, current RunStatus) error) error {
	current, err := s.readRunStatus(ctx, runID)
	if err != nil {
		return err
	}
	run := Run{Status: current}
	if err := run.TransitionStatus(event); err != nil {
		return err
	}
	if terminal(run.Status) != terminalWrite {
		if terminalWrite {
			return fmt.Errorf("terminal event %q cannot use the non-terminal update path", event)
		}
		return fmt.Errorf("non-terminal event %q cannot use the finish path", event)
	}
	return persist(run.Status, current)
}

func (s *sqliteRepository) readRunStatus(ctx context.Context, runID string) (RunStatus, error) {
	var status string
	err := s.db.QueryRowContext(ctx, `SELECT status FROM runs WHERE id = ?`, runID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrRunNotFound{ID: runID}
	}
	if err != nil {
		return "", fmt.Errorf("read run %q before transition: %w", runID, err)
	}
	return RunStatus(status), nil
}

func requireRunTransitionWrite(result sql.Result, err error, runID string, from RunStatus, event RunEvent) error {
	if err != nil {
		return fmt.Errorf("persist transition from %q via %q: %w", from, event, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check transition write: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("run %q changed after reading state %q; transition via %q was not applied", runID, from, event)
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
	return s.listRunsWhere(ctx, planID, "1=1", nil, limit)
}

// StatsWindow returns the runs run stats are computed over: the newest `attempts`
// scheduled or manual runs, plus the automatic retries that started within that
// span. Counting the window in attempts keeps a burst of retries from pushing
// every real attempt out of it.
func (s *sqliteRepository) StatsWindow(ctx context.Context, planID string, attempts int) ([]Run, error) {
	window, err := s.listRunsWhere(ctx, planID, "trigger <> ?", []any{string(TriggerRetry)}, attempts)
	if err != nil {
		return nil, err
	}
	since := ""
	if len(window) == attempts && attempts > 0 {
		since = window[len(window)-1].StartedAt.Format(runTimeFormat)
	}
	retries, err := s.listRunsWhere(ctx, planID, "trigger = ? AND started_at >= ?", []any{string(TriggerRetry), since}, attempts)
	if err != nil {
		return nil, err
	}
	return append(window, retries...), nil
}

func (s *sqliteRepository) listRunsWhere(ctx context.Context, planID, cond string, condArgs []any, limit int) ([]Run, error) {
	if limit <= 0 {
		return nil, nil
	}
	query := `SELECT id, plan_id, trigger, status, started_at, finished_at, error, failure_code, failure_category, next_action, updated_at, physical_bytes FROM runs WHERE ` + cond
	args := append([]any(nil), condArgs...)
	if planID != "" {
		query += ` AND plan_id = ?`
		args = append(args, planID)
	}
	query += ` ORDER BY started_at DESC, id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
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
	// The latest run per target, and what happened to that target in it. The
	// target's own outcome decides its status: a run that failed for another
	// target (or was cut off by a restart after most targets were captured) is
	// not "partial-failed" for the targets it backed up. One row per target comes
	// back from the database, not every outcome ever recorded.
	query := `
SELECT target_id, outcome_status, run_status, started_at, failure_code, failure_category FROM (
  SELECT ro.target_id AS target_id, ro.status AS outcome_status, r.status AS run_status, r.started_at AS started_at,
         ro.failure_code AS failure_code, ro.failure_category AS failure_category,
         ROW_NUMBER() OVER (PARTITION BY ro.target_id ORDER BY r.started_at DESC, r.id DESC) AS rn
  FROM run_outcomes ro JOIN runs r ON ro.run_id = r.id
  %s
) WHERE rn = 1`
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
		var targetID, outcomeStatus, runStatus, startedRaw, failureCode, failureCategory string
		if err := rows.Scan(&targetID, &outcomeStatus, &runStatus, &startedRaw, &failureCode, &failureCategory); err != nil {
			return nil, fmt.Errorf("scan target status: %w", err)
		}
		ts := &TargetStatus{
			TargetID: targetID, LastRunStatus: targetRunStatus(OutcomeStatus(outcomeStatus), RunStatus(runStatus)), LastRunAt: parseTime(startedRaw),
			LastFailureCode: failures.Code(failureCode), LastFailureCategory: failures.Category(failureCategory),
		}
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

func (s *sqliteRepository) TargetDestinationStatuses(ctx context.Context, targetIDs []string) ([]TargetDestinationStatus, error) {
	where := ""
	args := []any{string(OutcomeSucceeded)}
	if len(targetIDs) > 0 {
		where = "AND target_id IN (" + placeholders(len(targetIDs)) + ")"
		for _, id := range targetIDs {
			args = append(args, id)
		}
	}
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT target_id, destination_id, MAX(finished_at) FROM run_outcomes WHERE status = ? %s GROUP BY target_id, destination_id ORDER BY target_id, destination_id`, where), args...)
	if err != nil {
		return nil, fmt.Errorf("target destination statuses: %w", err)
	}
	defer rows.Close()
	var out []TargetDestinationStatus
	for rows.Next() {
		var status TargetDestinationStatus
		var finished string
		if err := rows.Scan(&status.TargetID, &status.DestinationID, &finished); err != nil {
			return nil, fmt.Errorf("scan target destination status: %w", err)
		}
		status.LastSuccessAt = parseTime(finished)
		out = append(out, status)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate target destination statuses: %w", err)
	}
	return out, nil
}

// LastSuccessfulPhysicalBytes returns the newest terminal run that both
// succeeded (fully or partially) and measured positive physical growth. It is
// the empirical increment for the capacity model; a zero result with ok=false
// means there is no usable history, not that the next run writes zero bytes.
func (s *sqliteRepository) LastSuccessfulPhysicalBytes(ctx context.Context, planID string) (int64, bool, error) {
	if strings.TrimSpace(planID) == "" {
		return 0, false, nil
	}
	var physical int64
	err := s.db.QueryRowContext(ctx,
		`SELECT physical_bytes FROM runs
		 WHERE plan_id = ? AND status IN (?, ?) AND physical_bytes > 0
		 ORDER BY finished_at DESC, id DESC LIMIT 1`,
		planID, string(RunCompleted), string(RunPartialFailed),
	).Scan(&physical)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("last successful physical bytes for plan %q: %w", planID, err)
	}
	return physical, true, nil
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

var _ PlanSuccessReader = (*sqliteRepository)(nil)

// PlanSuccess returns when each target of planID last had a succeeded outcome
// under that plan. Outcomes recorded for other plans do not count.
func (s *sqliteRepository) PlanSuccess(ctx context.Context, planID string) (PlanSuccess, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT o.target_id, MAX(o.finished_at) FROM run_outcomes o JOIN runs r ON r.id = o.run_id WHERE r.plan_id = ? AND o.status = ? AND o.finished_at != '' GROUP BY o.target_id`,
		planID, string(OutcomeSucceeded))
	if err != nil {
		return PlanSuccess{}, fmt.Errorf("plan success: %w", err)
	}
	defer rows.Close()
	out := PlanSuccess{ByTarget: map[string]time.Time{}}
	for rows.Next() {
		var targetID, finished string
		if err := rows.Scan(&targetID, &finished); err != nil {
			return PlanSuccess{}, fmt.Errorf("scan plan success: %w", err)
		}
		t := parseTime(finished)
		out.ByTarget[targetID] = t
		if t.After(out.At) {
			out.At = t
		}
	}
	if err := rows.Err(); err != nil {
		return PlanSuccess{}, fmt.Errorf("iterate plan success: %w", err)
	}
	return out, nil
}

// targetRunStatus is a target's status in its latest run: its own outcome when
// the run recorded one, the run's status otherwise (a run still in flight).
func targetRunStatus(outcome OutcomeStatus, run RunStatus) RunStatus {
	switch outcome {
	case OutcomeSucceeded:
		return RunCompleted
	case OutcomeFailed, OutcomeBlocked:
		return RunFailed
	}
	return run
}
