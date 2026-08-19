package authorization

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"treasury/internal/budget"
)

type DB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLiteRepository struct{ db DB }

func NewSQLiteRepository(db DB) *SQLiteRepository { return &SQLiteRepository{db: db} }

func (r *SQLiteRepository) Create(ctx context.Context, value Record) (Record, error) {
	_, err := r.db.ExecContext(ctx, `INSERT INTO authorizations(id, idempotency_key, mandate_id, budget_id, requesting_agent, amount_minor, currency, counterparty, verdict, violated_constraint, remediation, hold_minor, created_at, expires_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, value.ID, value.IdempotencyKey, value.MandateID, value.BudgetID, value.RequestingAgent, value.AmountMinor, value.Currency, value.Counterparty, string(value.Verdict), value.ViolatedConstraint, value.Remediation, value.HoldMinor, value.CreatedAt.Format(time.RFC3339Nano), value.ExpiresAt.Format(time.RFC3339Nano))
	if err != nil {
		return Record{}, fmt.Errorf("create authorization: %w", err)
	}
	return value, nil
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (Record, error) {
	return r.get(ctx, `SELECT id, idempotency_key, mandate_id, budget_id, requesting_agent, amount_minor, currency, counterparty, verdict, violated_constraint, remediation, hold_minor, created_at, expires_at FROM authorizations WHERE id = ?`, id)
}

func (r *SQLiteRepository) GetByIdempotencyKey(ctx context.Context, key string) (Record, error) {
	return r.get(ctx, `SELECT id, idempotency_key, mandate_id, budget_id, requesting_agent, amount_minor, currency, counterparty, verdict, violated_constraint, remediation, hold_minor, created_at, expires_at FROM authorizations WHERE idempotency_key = ?`, key)
}

func (r *SQLiteRepository) get(ctx context.Context, query, arg string) (Record, error) {
	var value Record
	var verdict, createdAt, expiresAt string
	err := r.db.QueryRowContext(ctx, query, arg).Scan(&value.ID, &value.IdempotencyKey, &value.MandateID, &value.BudgetID, &value.RequestingAgent, &value.AmountMinor, &value.Currency, &value.Counterparty, &verdict, &value.ViolatedConstraint, &value.Remediation, &value.HoldMinor, &createdAt, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Record{}, ErrNotFound
	}
	if err != nil {
		return Record{}, err
	}
	value.Verdict = Verdict(verdict)
	if value.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return Record{}, err
	}
	if value.ExpiresAt, err = time.Parse(time.RFC3339Nano, expiresAt); err != nil {
		return Record{}, err
	}
	return value, nil
}

func (r *SQLiteRepository) Usage(ctx context.Context, budgetID, mandateID string, periodStart, now time.Time) (Usage, error) {
	var usage Usage
	err := r.db.QueryRowContext(ctx, `SELECT
COALESCE(SUM(CASE WHEN verdict = 'settled' OR (verdict IN ('pending','approved') AND expires_at > ?) THEN amount_minor ELSE 0 END), 0),
COALESCE(SUM(CASE WHEN created_at >= ? AND (verdict = 'settled' OR (verdict IN ('pending','approved') AND expires_at > ?)) THEN amount_minor ELSE 0 END), 0),
COALESCE(SUM(CASE WHEN mandate_id = ? AND (verdict = 'settled' OR (verdict IN ('pending','approved') AND expires_at > ?)) THEN amount_minor ELSE 0 END), 0)
FROM authorizations WHERE budget_id = ?`, now.Format(time.RFC3339Nano), periodStart.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), mandateID, now.Format(time.RFC3339Nano), budgetID).Scan(&usage.BudgetTotalMinor, &usage.BudgetPeriodMinor, &usage.MandateTotalMinor)
	return usage, err
}

func (r *SQLiteRepository) BudgetUsage(ctx context.Context, budgetID string, periodStart, now time.Time) (budget.Usage, error) {
	var usage budget.Usage
	err := r.db.QueryRowContext(ctx, `SELECT
COALESCE(SUM(CASE WHEN verdict = 'settled' OR (verdict IN ('pending','approved') AND expires_at > ?) THEN amount_minor ELSE 0 END), 0),
COALESCE(SUM(CASE WHEN created_at >= ? AND (verdict = 'settled' OR (verdict IN ('pending','approved') AND expires_at > ?)) THEN amount_minor ELSE 0 END), 0)
FROM authorizations WHERE budget_id = ?`, now.Format(time.RFC3339Nano), periodStart.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), budgetID).Scan(&usage.TotalMinor, &usage.PeriodMinor)
	return usage, err
}

func (r *SQLiteRepository) Release(ctx context.Context, id string) (Record, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE authorizations SET verdict = 'released', hold_minor = 0 WHERE id = ? AND verdict IN ('pending','approved')`, id)
	if err != nil {
		return Record{}, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Record{}, err
	}
	if rows == 0 {
		current, getErr := r.Get(ctx, id)
		if getErr == nil && current.Verdict == VerdictReleased {
			return current, nil
		}
		return Record{}, ErrNotFound
	}
	return r.Get(ctx, id)
}

func (r *SQLiteRepository) Approve(ctx context.Context, id string) (Record, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE authorizations SET verdict = 'approved' WHERE id = ? AND verdict = 'pending'`, id)
	if err != nil {
		return Record{}, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Record{}, err
	}
	if rows == 0 {
		current, getErr := r.Get(ctx, id)
		if getErr == nil && current.Verdict == VerdictApproved {
			return current, nil
		}
		return Record{}, ErrNotFound
	}
	return r.Get(ctx, id)
}

func (r *SQLiteRepository) Settle(ctx context.Context, id string) (Record, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE authorizations SET verdict = 'settled', hold_minor = 0 WHERE id = ? AND verdict = 'approved'`, id)
	if err != nil {
		return Record{}, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Record{}, err
	}
	if rows == 0 {
		current, getErr := r.Get(ctx, id)
		if getErr == nil && current.Verdict == VerdictSettled {
			return current, nil
		}
		return Record{}, ErrNotFound
	}
	return r.Get(ctx, id)
}

var _ Repository = (*SQLiteRepository)(nil)
