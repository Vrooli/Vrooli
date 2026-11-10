package control

import (
	"context"
	"fmt"
	"time"
)

func (s *Service) claimCommandReceipt(ctx context.Context, deviceID, transport, key, stepKind, runID string) (bool, error) {
	if s.db == nil {
		return true, nil
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := s.db.ExecContext(ctx, `INSERT INTO device_control_command_receipts
 (device_id, transport, idempotency_key, step_kind, run_id, outcome, created_at, updated_at)
 VALUES (?, ?, ?, ?, ?, 'in_flight', ?, ?) ON CONFLICT(device_id, transport, idempotency_key) DO NOTHING`, deviceID, transport, key, stepKind, runID, now, now)
	if err != nil {
		return false, fmt.Errorf("claim idempotency receipt: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read idempotency receipt claim: %w", err)
	}
	if inserted == 1 {
		return true, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT step_kind, outcome FROM device_control_command_receipts WHERE device_id=? AND transport=? AND idempotency_key=?`, deviceID, transport, key)
	if err != nil {
		return false, fmt.Errorf("read idempotency receipt: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return false, fmt.Errorf("read idempotency receipt: %w", err)
		}
		return false, fmt.Errorf("idempotency receipt disappeared for %q", key)
	}
	var existingKind, existingOutcome string
	if err := rows.Scan(&existingKind, &existingOutcome); err != nil {
		return false, fmt.Errorf("read idempotency receipt: %w", err)
	}
	if existingKind != stepKind {
		return false, fmt.Errorf("idempotency key %q was previously bound to step %q", key, existingKind)
	}
	if existingOutcome != "passed" {
		return false, fmt.Errorf("idempotency key %q has prior outcome %q; refusing replay", key, existingOutcome)
	}
	return false, nil
}

func (s *Service) finishCommandReceipt(ctx context.Context, deviceID, transport, key, outcome string) error {
	if s.db == nil {
		return nil
	}
	result, err := s.db.ExecContext(ctx, `UPDATE device_control_command_receipts SET outcome=?, updated_at=? WHERE device_id=? AND transport=? AND idempotency_key=?`, outcome, time.Now().UTC().Format(time.RFC3339Nano), deviceID, transport, key)
	if err != nil {
		return fmt.Errorf("record idempotency receipt: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read idempotency receipt update: %w", err)
	}
	if updated != 1 {
		return fmt.Errorf("idempotency receipt disappeared for %q", key)
	}
	return nil
}
