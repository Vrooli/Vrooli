package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// heal_state.go: durable backing store for the auto-heal failure
// counter (Round 3 Phase 6). Adds Get/Upsert/Clear/List against the
// `heal_state` table introduced in schema.sql. The healTracker in
// internal/sandbox is a write-through cache over this; this module
// is the source of truth across restarts.

// GetHealState returns the heal-state row for the given sandbox, or
// (nil, nil) when no row exists. A typed error surfaces on real DB
// trouble.
func (r *SandboxRepository) GetHealState(ctx context.Context, id uuid.UUID) (*HealStateRow, error) {
	return getHealState(ctx, r.db, id)
}

func (r *TxSandboxRepository) GetHealState(ctx context.Context, id uuid.UUID) (*HealStateRow, error) {
	return getHealState(ctx, r.tx, id)
}

// UpsertHealState writes the row, replacing any existing entry.
func (r *SandboxRepository) UpsertHealState(ctx context.Context, row HealStateRow) error {
	return upsertHealState(ctx, r.db, row)
}

func (r *TxSandboxRepository) UpsertHealState(ctx context.Context, row HealStateRow) error {
	return upsertHealState(ctx, r.tx, row)
}

// ClearHealState removes the row for a sandbox. Idempotent: a missing
// row is not an error. Called when a heal succeeds, when the sandbox
// is deleted, or when the auto-heal-exhausted escalation happens.
func (r *SandboxRepository) ClearHealState(ctx context.Context, id uuid.UUID) error {
	return clearHealState(ctx, r.db, id)
}

func (r *TxSandboxRepository) ClearHealState(ctx context.Context, id uuid.UUID) error {
	return clearHealState(ctx, r.tx, id)
}

// ListHealState returns every row in the table. Used at boot to warm
// the in-memory cache of the heal tracker so a freshly-started runner
// already knows the durable state.
func (r *SandboxRepository) ListHealState(ctx context.Context) ([]HealStateRow, error) {
	return listHealState(ctx, r.db)
}

func (r *TxSandboxRepository) ListHealState(ctx context.Context) ([]HealStateRow, error) {
	return listHealState(ctx, r.tx)
}

func getHealState(ctx context.Context, exec dbExec, id uuid.UUID) (*HealStateRow, error) {
	const q = `SELECT sandbox_id, consecutive_failures, last_attempt, last_error
	             FROM heal_state WHERE sandbox_id = ?`
	row := exec.QueryRowContext(ctx, q, id.String())

	var (
		idStr       string
		failures    int
		lastAttempt string
		lastError   string
	)
	if err := row.Scan(&idStr, &failures, &lastAttempt, &lastError); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan heal_state: %w", err)
	}
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse heal_state.sandbox_id: %w", err)
	}
	parsedAt, err := parseTime(lastAttempt)
	if err != nil {
		return nil, fmt.Errorf("parse heal_state.last_attempt: %w", err)
	}
	return &HealStateRow{
		SandboxID:           parsedID,
		ConsecutiveFailures: failures,
		LastAttempt:         parsedAt,
		LastError:           lastError,
	}, nil
}

func upsertHealState(ctx context.Context, exec dbExec, row HealStateRow) error {
	const q = `INSERT INTO heal_state (sandbox_id, consecutive_failures, last_attempt, last_error)
	             VALUES (?, ?, ?, ?)
	             ON CONFLICT(sandbox_id) DO UPDATE SET
	                 consecutive_failures = excluded.consecutive_failures,
	                 last_attempt         = excluded.last_attempt,
	                 last_error           = excluded.last_error`
	_, err := exec.ExecContext(ctx, q,
		row.SandboxID.String(),
		row.ConsecutiveFailures,
		formatTime(row.LastAttempt),
		row.LastError,
	)
	if err != nil {
		return fmt.Errorf("upsert heal_state: %w", err)
	}
	return nil
}

func clearHealState(ctx context.Context, exec dbExec, id uuid.UUID) error {
	const q = `DELETE FROM heal_state WHERE sandbox_id = ?`
	if _, err := exec.ExecContext(ctx, q, id.String()); err != nil {
		return fmt.Errorf("clear heal_state: %w", err)
	}
	return nil
}

func listHealState(ctx context.Context, exec dbExec) ([]HealStateRow, error) {
	const q = `SELECT sandbox_id, consecutive_failures, last_attempt, last_error FROM heal_state`
	rows, err := exec.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query heal_state: %w", err)
	}
	defer rows.Close()

	var out []HealStateRow
	for rows.Next() {
		var (
			idStr       string
			failures    int
			lastAttempt string
			lastError   string
		)
		if err := rows.Scan(&idStr, &failures, &lastAttempt, &lastError); err != nil {
			return nil, fmt.Errorf("scan heal_state row: %w", err)
		}
		parsedID, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("parse heal_state.sandbox_id: %w", err)
		}
		parsedAt, err := parseTime(lastAttempt)
		if err != nil {
			return nil, fmt.Errorf("parse heal_state.last_attempt: %w", err)
		}
		out = append(out, HealStateRow{
			SandboxID:           parsedID,
			ConsecutiveFailures: failures,
			LastAttempt:         parsedAt,
			LastError:           lastError,
		})
	}
	return out, rows.Err()
}
