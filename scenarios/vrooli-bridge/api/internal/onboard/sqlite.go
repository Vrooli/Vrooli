package onboard

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface sqliteRepository depends on. Both
// *sql.DB (repository unit tests) and *database.RoutedDB (production) satisfy
// it, so production participates in per-request routing without the test
// fixture wrapping its handle.
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

// Compile-time guarantee.
var _ Repository = (*sqliteRepository)(nil)

// opTimeFormat sorts lexicographically in time order for a fixed zone, so a
// string range/order comparison is a correct filter — matching the wire format
// and the provision domain convention.
const opTimeFormat = time.RFC3339Nano

const (
	insertOpSQL = `
	INSERT INTO onboarding_ops (id, host, port, user_name, node_name, target_revision, source_mode, repo_url, state, node_id, correlation_id, failure_reason, failure_detail, control_plane_url, reachability_mode, exit_code, created_at, started_at, finished_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	selectOpColumns = `
	SELECT id, host, port, user_name, node_name, target_revision, source_mode, repo_url, state, node_id, correlation_id, failure_reason, failure_detail, control_plane_url, reachability_mode, exit_code, created_at, started_at, finished_at
FROM onboarding_ops
`

	selectOpByIDSQL = selectOpColumns + `WHERE id = ?`

	updateOpSQL = `
UPDATE onboarding_ops
SET state = ?, node_id = ?, failure_reason = ?, failure_detail = ?, exit_code = ?, started_at = ?, finished_at = ?
WHERE id = ?
`

	insertEventSQL = `
INSERT OR IGNORE INTO onboarding_step_events (op_id, sequence, step_id, status, detail, emitted_at)
VALUES (?, ?, ?, ?, ?, ?)
`

	listEventsSQL = `
SELECT op_id, sequence, step_id, status, detail, emitted_at
FROM onboarding_step_events
WHERE op_id = ?
ORDER BY sequence ASC
`

	deleteEventsSQL   = `DELETE FROM onboarding_step_events WHERE op_id = ?`
	deleteFailedOpSQL = `DELETE FROM onboarding_ops WHERE id = ? AND state = ?`
)

func (s *sqliteRepository) Create(ctx context.Context, op Op) (Op, error) {
	if op.ID == "" {
		op.ID = uuid.NewString()
	}
	if op.CreatedAt.IsZero() {
		op.CreatedAt = s.clock.Now().UTC()
	}
	if op.State == StateUnspecified {
		op.State = StatePending
	}
	if _, err := s.db.ExecContext(ctx, insertOpSQL,
		op.ID, op.Host, op.Port, op.User, op.NodeName, op.TargetRevision, op.SourceMode.String(), op.RepoURL,
		int(op.State), op.NodeID, op.CorrelationID, string(op.FailureReason), op.FailureDetail, op.ControlPlaneURL, op.ReachabilityMode, op.ExitCode,
		op.CreatedAt.Format(opTimeFormat), formatNullableTime(op.StartedAt), formatNullableTime(op.FinishedAt),
	); err != nil {
		return Op{}, fmt.Errorf("insert onboarding op %q: %w", op.ID, err)
	}
	return op, nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Op, error) {
	row := s.db.QueryRowContext(ctx, selectOpByIDSQL, id)
	op, err := scanOp(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Op{}, ErrOpNotFound{ID: id}
	}
	if err != nil {
		return Op{}, fmt.Errorf("get onboarding op %q: %w", id, err)
	}
	return op, nil
}

func (s *sqliteRepository) List(ctx context.Context, filter ListFilter) ([]Op, error) {
	query := selectOpColumns
	args := make([]any, 0, 2)
	if filter.Host != "" {
		query += `WHERE host = ? `
		args = append(args, filter.Host)
	}
	query += `ORDER BY created_at DESC, id DESC`
	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list onboarding ops: %w", err)
	}
	defer rows.Close()

	ops, err := scanOps(rows)
	if err != nil {
		return nil, err
	}
	return ops, nil
}

func (s *sqliteRepository) ListNonTerminal(ctx context.Context) ([]Op, error) {
	query := selectOpColumns + `WHERE state IN (?, ?, ?, ?, ?) ORDER BY created_at ASC`
	rows, err := s.db.QueryContext(ctx, query,
		int(StatePending), int(StateSSHSetup), int(StatePushingScript), int(StateBootstrapping), int(StateVerifying),
	)
	if err != nil {
		return nil, fmt.Errorf("list non-terminal onboarding ops: %w", err)
	}
	defer rows.Close()
	return scanOps(rows)
}

func (s *sqliteRepository) Update(ctx context.Context, op Op) (Op, error) {
	existing, err := s.Get(ctx, op.ID)
	if err != nil {
		return Op{}, err
	}
	existing.State = op.State
	existing.NodeID = op.NodeID
	existing.FailureReason = op.FailureReason
	existing.FailureDetail = op.FailureDetail
	existing.ExitCode = op.ExitCode
	existing.StartedAt = op.StartedAt
	existing.FinishedAt = op.FinishedAt

	if _, err := s.db.ExecContext(ctx, updateOpSQL,
		int(existing.State), existing.NodeID, string(existing.FailureReason), existing.FailureDetail, existing.ExitCode,
		formatNullableTime(existing.StartedAt), formatNullableTime(existing.FinishedAt), existing.ID,
	); err != nil {
		return Op{}, fmt.Errorf("update onboarding op %q: %w", op.ID, err)
	}
	return existing, nil
}

func (s *sqliteRepository) AppendEvent(ctx context.Context, ev StepEvent) error {
	if _, err := s.db.ExecContext(ctx, insertEventSQL,
		ev.OpID, ev.Sequence, ev.StepID, int(ev.Status), ev.Detail, formatNullableTime(ev.EmittedAt),
	); err != nil {
		return fmt.Errorf("append step event for op %q: %w", ev.OpID, err)
	}
	return nil
}

func (s *sqliteRepository) ListEvents(ctx context.Context, opID string) ([]StepEvent, error) {
	rows, err := s.db.QueryContext(ctx, listEventsSQL, opID)
	if err != nil {
		return nil, fmt.Errorf("list step events for op %q: %w", opID, err)
	}
	defer rows.Close()

	var events []StepEvent
	for rows.Next() {
		var (
			ev         StepEvent
			status     int
			emittedRaw string
		)
		if err := rows.Scan(&ev.OpID, &ev.Sequence, &ev.StepID, &status, &ev.Detail, &emittedRaw); err != nil {
			return nil, fmt.Errorf("scan step event: %w", err)
		}
		ev.Status = StepStatus(status)
		if ev.EmittedAt, err = parseNullableTime(emittedRaw); err != nil {
			return nil, fmt.Errorf("parse emitted_at %q: %w", emittedRaw, err)
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate step events: %w", err)
	}
	return events, nil
}

func (s *sqliteRepository) DeleteFailed(ctx context.Context, id string) error {
	op, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if op.State != StateFailed {
		return ErrInvalid{Field: "id", Reason: "only failed onboarding attempts can be removed"}
	}
	if _, err := s.db.ExecContext(ctx, deleteEventsSQL, id); err != nil {
		return fmt.Errorf("delete onboarding events %q: %w", id, err)
	}
	result, err := s.db.ExecContext(ctx, deleteFailedOpSQL, id, int(StateFailed))
	if err != nil {
		return fmt.Errorf("delete failed onboarding op %q: %w", id, err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return ErrOpNotFound{ID: id}
	}
	return nil
}

// rowScanner unifies *sql.Row and *sql.Rows under their common Scan surface.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanOps(rows *sql.Rows) ([]Op, error) {
	var ops []Op
	for rows.Next() {
		op, err := scanOp(rows)
		if err != nil {
			return nil, fmt.Errorf("scan onboarding op: %w", err)
		}
		ops = append(ops, op)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate onboarding ops: %w", err)
	}
	return ops, nil
}

func scanOp(sc rowScanner) (Op, error) {
	var (
		op          Op
		state       int
		failure     string
		createdRaw  string
		startedRaw  string
		finishedRaw string
		sourceMode  string
	)
	if err := sc.Scan(&op.ID, &op.Host, &op.Port, &op.User, &op.NodeName, &op.TargetRevision, &sourceMode, &op.RepoURL,
		&state, &op.NodeID, &op.CorrelationID, &failure, &op.FailureDetail, &op.ControlPlaneURL, &op.ReachabilityMode, &op.ExitCode, &createdRaw, &startedRaw, &finishedRaw); err != nil {
		return Op{}, err
	}
	if sourceMode == "working-tree" {
		op.SourceMode = SourceModeWorkingTree
	} else {
		op.SourceMode = SourceModePinned
	}
	op.State = State(state)
	op.FailureReason = FailureReason(failure)

	var err error
	if op.CreatedAt, err = time.Parse(opTimeFormat, createdRaw); err != nil {
		return Op{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	if op.StartedAt, err = parseNullableTime(startedRaw); err != nil {
		return Op{}, fmt.Errorf("parse started_at %q: %w", startedRaw, err)
	}
	if op.FinishedAt, err = parseNullableTime(finishedRaw); err != nil {
		return Op{}, fmt.Errorf("parse finished_at %q: %w", finishedRaw, err)
	}
	return op, nil
}

// formatNullableTime renders a zero time as "" (the column default) so absence
// is distinguishable from a real timestamp.
func formatNullableTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(opTimeFormat)
}

func parseNullableTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}
	return time.Parse(opTimeFormat, raw)
}
