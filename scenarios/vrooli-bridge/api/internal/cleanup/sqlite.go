package cleanup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
	once  sync.Once
	err   error
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock}
}

func (r *sqliteRepository) migrate(ctx context.Context) error {
	r.once.Do(func() {
		_, r.err = r.db.ExecContext(ctx, `ALTER TABLE cleanup_operations ADD COLUMN sealing_public_key BLOB`)
		if r.err != nil && !strings.Contains(strings.ToLower(r.err.Error()), "duplicate column") {
			return
		}
		r.err = nil
	})
	return r.err
}

const opTimeFormat = time.RFC3339Nano

const opColumns = `id, machine_id, node_id, target, scope, status, transport, transport_reason, reason, plan_hash, plan_json, receipt_json, operator_id, sealed_passphrase, capability, sealing_public_key, created_at, updated_at, finished_at`

func (r *sqliteRepository) Create(ctx context.Context, op Operation) (Operation, error) {
	if err := r.migrate(ctx); err != nil {
		return Operation{}, err
	}
	if op.ID == "" {
		op.ID = uuid.NewString()
	}
	now := r.clock.Now().UTC()
	if op.CreatedAt.IsZero() {
		op.CreatedAt = now
	}
	if op.UpdatedAt.IsZero() {
		op.UpdatedAt = op.CreatedAt
	}
	if op.Status == StatusUnspecified {
		op.Status = StatusQueued
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO cleanup_operations (`+opColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		op.ID, op.MachineID, op.NodeID, op.Target, op.Scope, int(op.Status), op.Transport, op.TransportReason, op.Reason, op.PlanHash,
		op.PlanJSON, op.ReceiptJSON, op.OperatorID, op.SealedPassphrase, op.Capability, op.SealingPublicKey, op.CreatedAt.Format(opTimeFormat), op.UpdatedAt.Format(opTimeFormat), formatTime(op.FinishedAt))
	if err != nil {
		return Operation{}, fmt.Errorf("insert cleanup operation %q: %w", op.ID, err)
	}
	return op, nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Operation, error) {
	if err := r.migrate(ctx); err != nil {
		return Operation{}, err
	}
	row := r.db.QueryRowContext(ctx, `SELECT `+opColumns+` FROM cleanup_operations WHERE id = ?`, id)
	op, err := scanOperation(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Operation{}, ErrNotFound{ID: id}
	}
	if err != nil {
		return Operation{}, fmt.Errorf("get cleanup operation %q: %w", id, err)
	}
	return op, nil
}

// FindActive is used only to turn the partial unique-index conflict into a
// stable operator-facing ErrInFlight. The index remains the concurrency
// authority; this query is explanatory, not the lock.
func (r *sqliteRepository) FindActive(ctx context.Context, machineID string) (Operation, error) {
	if err := r.migrate(ctx); err != nil {
		return Operation{}, err
	}
	row := r.db.QueryRowContext(ctx, `SELECT `+opColumns+` FROM cleanup_operations WHERE machine_id = ? AND status IN (1, 2, 3, 4, 5) ORDER BY created_at ASC LIMIT 1`, machineID)
	op, err := scanOperation(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Operation{}, ErrNotFound{ID: machineID}
	}
	return op, err
}

func (r *sqliteRepository) Update(ctx context.Context, op Operation) (Operation, error) {
	if err := r.migrate(ctx); err != nil {
		return Operation{}, err
	}
	existing, err := r.Get(ctx, op.ID)
	if err != nil {
		return Operation{}, err
	}
	// Updates are intentionally field-complete so a stale caller cannot erase a
	// frozen plan, receipt, or confirmation while resuming an apply.
	op.CreatedAt = existing.CreatedAt
	if op.UpdatedAt.IsZero() {
		op.UpdatedAt = r.clock.Now().UTC()
	}
	_, err = r.db.ExecContext(ctx, `UPDATE cleanup_operations SET machine_id=?, node_id=?, target=?, scope=?, status=?, transport=?, transport_reason=?, reason=?, plan_hash=?, plan_json=?, receipt_json=?, operator_id=?, sealed_passphrase=?, capability=?, sealing_public_key=?, created_at=?, updated_at=?, finished_at=? WHERE id=?`,
		op.MachineID, op.NodeID, op.Target, op.Scope, int(op.Status), op.Transport, op.TransportReason, op.Reason, op.PlanHash, op.PlanJSON, op.ReceiptJSON, op.OperatorID, op.SealedPassphrase, op.Capability,
		op.SealingPublicKey, op.CreatedAt.Format(opTimeFormat), op.UpdatedAt.Format(opTimeFormat), formatTime(op.FinishedAt), op.ID)
	if err != nil {
		return Operation{}, fmt.Errorf("update cleanup operation %q: %w", op.ID, err)
	}
	return op, nil
}

func (r *sqliteRepository) AppendEvent(ctx context.Context, ev Event) (bool, error) {
	if err := r.migrate(ctx); err != nil {
		return false, err
	}
	if ev.EmittedAt.IsZero() {
		ev.EmittedAt = r.clock.Now().UTC()
	}
	res, err := r.db.ExecContext(ctx, `INSERT OR IGNORE INTO cleanup_events (operation_id, sequence, kind, status, log_chunk, plan_json, receipt_json, reason, exit_code, emitted_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ev.OperationID, ev.Sequence, int(ev.Kind), ev.Status, ev.LogChunk, ev.PlanJSON, ev.ReceiptJSON, ev.Reason, ev.ExitCode, ev.EmittedAt.Format(opTimeFormat))
	if err != nil {
		return false, fmt.Errorf("append cleanup event: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (r *sqliteRepository) ListEvents(ctx context.Context, id string) ([]Event, error) {
	if err := r.migrate(ctx); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT operation_id, sequence, kind, status, log_chunk, plan_json, receipt_json, reason, exit_code, emitted_at FROM cleanup_events WHERE operation_id=? ORDER BY sequence ASC`, id)
	if err != nil {
		return nil, fmt.Errorf("list cleanup events: %w", err)
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var ev Event
		var kind int
		var emitted string
		if err := rows.Scan(&ev.OperationID, &ev.Sequence, &kind, &ev.Status, &ev.LogChunk, &ev.PlanJSON, &ev.ReceiptJSON, &ev.Reason, &ev.ExitCode, &emitted); err != nil {
			return nil, err
		}
		ev.Kind = EventKind(kind)
		ev.EmittedAt, err = time.Parse(opTimeFormat, emitted)
		if err != nil {
			return nil, fmt.Errorf("parse cleanup event time: %w", err)
		}
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

type scanner interface{ Scan(...any) error }

func scanOperation(s scanner) (Operation, error) {
	var op Operation
	var status int
	var created, updated, finished string
	err := s.Scan(&op.ID, &op.MachineID, &op.NodeID, &op.Target, &op.Scope, &status, &op.Transport, &op.TransportReason, &op.Reason, &op.PlanHash, &op.PlanJSON, &op.ReceiptJSON, &op.OperatorID, &op.SealedPassphrase, &op.Capability, &op.SealingPublicKey, &created, &updated, &finished)
	if err != nil {
		return Operation{}, err
	}
	op.Status = Status(status)
	if op.CreatedAt, err = time.Parse(opTimeFormat, created); err != nil {
		return Operation{}, err
	}
	if op.UpdatedAt, err = time.Parse(opTimeFormat, updated); err != nil {
		return Operation{}, err
	}
	if strings.TrimSpace(finished) != "" {
		op.FinishedAt, err = time.Parse(opTimeFormat, finished)
		if err != nil {
			return Operation{}, err
		}
	}
	return op, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(opTimeFormat)
}
