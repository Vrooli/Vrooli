package provision

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"vrooli-bridge/internal/clock"

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
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

// Compile-time guarantee.
var _ Repository = (*sqliteRepository)(nil)

// opTimeFormat sorts lexicographically in time order for a fixed zone, so a
// string range/order comparison is a correct filter — matching the wire format
// and the runs domain convention.
const opTimeFormat = time.RFC3339Nano

const (
	insertOpSQL = `
INSERT INTO provisioning_ops (id, node_id, target_revision, rollback_revision, status, resulting_revision, exit_code, timeout_seconds, created_at, started_at, finished_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	selectOpColumns = `
SELECT id, node_id, target_revision, rollback_revision, status, resulting_revision, exit_code, timeout_seconds, created_at, started_at, finished_at
FROM provisioning_ops
`

	selectOpByIDSQL = selectOpColumns + `WHERE id = ?`

	updateOpSQL = `
UPDATE provisioning_ops
SET status = ?, resulting_revision = ?, exit_code = ?, started_at = ?, finished_at = ?
WHERE id = ?
`

	insertEventSQL = `
INSERT OR IGNORE INTO provision_events (op_id, sequence, kind, log_chunk, status, revision, exit_code, emitted_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`

	listEventsSQL = `
SELECT op_id, sequence, kind, log_chunk, status, revision, exit_code, emitted_at
FROM provision_events
WHERE op_id = ?
ORDER BY sequence ASC
`

	upsertNodeVersionSQL = `
INSERT INTO node_versions (node_id, revision, op_id, reported_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(node_id) DO UPDATE SET revision = excluded.revision, op_id = excluded.op_id, reported_at = excluded.reported_at
`

	selectNodeVersionSQL = `
SELECT node_id, revision, op_id, reported_at FROM node_versions WHERE node_id = ?
`
)

func (s *sqliteRepository) Create(ctx context.Context, op ProvisioningOp) (ProvisioningOp, error) {
	if op.ID == "" {
		op.ID = uuid.NewString()
	}
	if op.CreatedAt.IsZero() {
		op.CreatedAt = s.clock.Now().UTC()
	}
	if op.Status == StatusUnspecified {
		op.Status = StatusQueued
	}
	if _, err := s.db.ExecContext(ctx, insertOpSQL,
		op.ID, op.NodeID, op.TargetRevision, op.RollbackRevision, int(op.Status), op.ResultingRevision,
		op.ExitCode, op.TimeoutSeconds, op.CreatedAt.Format(opTimeFormat),
		formatNullableTime(op.StartedAt), formatNullableTime(op.FinishedAt),
	); err != nil {
		return ProvisioningOp{}, fmt.Errorf("insert provisioning op %q: %w", op.ID, err)
	}
	return op, nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (ProvisioningOp, error) {
	row := s.db.QueryRowContext(ctx, selectOpByIDSQL, id)
	op, err := scanOp(row)
	if errors.Is(err, sql.ErrNoRows) {
		return ProvisioningOp{}, ErrOpNotFound{ID: id}
	}
	if err != nil {
		return ProvisioningOp{}, fmt.Errorf("get provisioning op %q: %w", id, err)
	}
	return op, nil
}

func (s *sqliteRepository) List(ctx context.Context, filter ListFilter) ([]ProvisioningOp, error) {
	query := selectOpColumns
	args := make([]any, 0, 2)
	if filter.NodeID != "" {
		query += `WHERE node_id = ? `
		args = append(args, filter.NodeID)
	}
	query += `ORDER BY created_at DESC, id DESC`
	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list provisioning ops: %w", err)
	}
	defer rows.Close()

	var ops []ProvisioningOp
	for rows.Next() {
		op, err := scanOp(rows)
		if err != nil {
			return nil, fmt.Errorf("scan provisioning op: %w", err)
		}
		ops = append(ops, op)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provisioning ops: %w", err)
	}
	return ops, nil
}

func (s *sqliteRepository) Update(ctx context.Context, op ProvisioningOp) (ProvisioningOp, error) {
	existing, err := s.Get(ctx, op.ID)
	if err != nil {
		return ProvisioningOp{}, err
	}
	existing.Status = op.Status
	existing.ResultingRevision = op.ResultingRevision
	existing.ExitCode = op.ExitCode
	existing.StartedAt = op.StartedAt
	existing.FinishedAt = op.FinishedAt

	if _, err := s.db.ExecContext(ctx, updateOpSQL,
		int(existing.Status), existing.ResultingRevision, existing.ExitCode,
		formatNullableTime(existing.StartedAt), formatNullableTime(existing.FinishedAt), existing.ID,
	); err != nil {
		return ProvisioningOp{}, fmt.Errorf("update provisioning op %q: %w", op.ID, err)
	}
	return existing, nil
}

func (s *sqliteRepository) AppendEvent(ctx context.Context, ev ProvisionEvent) error {
	if _, err := s.db.ExecContext(ctx, insertEventSQL,
		ev.OpID, ev.Sequence, int(ev.Kind), ev.LogChunk, ev.Status, ev.Revision, ev.ExitCode, formatNullableTime(ev.EmittedAt),
	); err != nil {
		return fmt.Errorf("append event for op %q: %w", ev.OpID, err)
	}
	return nil
}

func (s *sqliteRepository) ListEvents(ctx context.Context, opID string) ([]ProvisionEvent, error) {
	rows, err := s.db.QueryContext(ctx, listEventsSQL, opID)
	if err != nil {
		return nil, fmt.Errorf("list events for op %q: %w", opID, err)
	}
	defer rows.Close()

	var events []ProvisionEvent
	for rows.Next() {
		var (
			ev         ProvisionEvent
			kind       int
			emittedRaw string
		)
		if err := rows.Scan(&ev.OpID, &ev.Sequence, &kind, &ev.LogChunk, &ev.Status, &ev.Revision, &ev.ExitCode, &emittedRaw); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		ev.Kind = EventKind(kind)
		if ev.EmittedAt, err = parseNullableTime(emittedRaw); err != nil {
			return nil, fmt.Errorf("parse emitted_at %q: %w", emittedRaw, err)
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}
	return events, nil
}

func (s *sqliteRepository) UpsertNodeVersion(ctx context.Context, v NodeVersion) error {
	if v.ReportedAt.IsZero() {
		v.ReportedAt = s.clock.Now().UTC()
	}
	if _, err := s.db.ExecContext(ctx, upsertNodeVersionSQL,
		v.NodeID, v.Revision, v.OpID, v.ReportedAt.Format(opTimeFormat),
	); err != nil {
		return fmt.Errorf("upsert node version for %q: %w", v.NodeID, err)
	}
	return nil
}

func (s *sqliteRepository) GetNodeVersion(ctx context.Context, nodeID string) (NodeVersion, error) {
	row := s.db.QueryRowContext(ctx, selectNodeVersionSQL, nodeID)
	var (
		v          NodeVersion
		reportedAt string
	)
	err := row.Scan(&v.NodeID, &v.Revision, &v.OpID, &reportedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return NodeVersion{}, ErrNoNodeVersion{NodeID: nodeID}
	}
	if err != nil {
		return NodeVersion{}, fmt.Errorf("get node version for %q: %w", nodeID, err)
	}
	if v.ReportedAt, err = parseNullableTime(reportedAt); err != nil {
		return NodeVersion{}, fmt.Errorf("parse reported_at %q: %w", reportedAt, err)
	}
	return v, nil
}

// rowScanner unifies *sql.Row and *sql.Rows under their common Scan surface.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanOp(sc rowScanner) (ProvisioningOp, error) {
	var (
		op          ProvisioningOp
		status      int
		createdRaw  string
		startedRaw  string
		finishedRaw string
	)
	if err := sc.Scan(&op.ID, &op.NodeID, &op.TargetRevision, &op.RollbackRevision, &status,
		&op.ResultingRevision, &op.ExitCode, &op.TimeoutSeconds, &createdRaw, &startedRaw, &finishedRaw); err != nil {
		return ProvisioningOp{}, err
	}
	op.Status = ProvisioningStatus(status)

	var err error
	if op.CreatedAt, err = time.Parse(opTimeFormat, createdRaw); err != nil {
		return ProvisioningOp{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	if op.StartedAt, err = parseNullableTime(startedRaw); err != nil {
		return ProvisioningOp{}, fmt.Errorf("parse started_at %q: %w", startedRaw, err)
	}
	if op.FinishedAt, err = parseNullableTime(finishedRaw); err != nil {
		return ProvisioningOp{}, fmt.Errorf("parse finished_at %q: %w", finishedRaw, err)
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
