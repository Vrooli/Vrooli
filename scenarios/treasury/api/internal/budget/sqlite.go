package budget

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type DB interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLiteRepository struct{ db DB }

func NewSQLiteRepository(db DB) *SQLiteRepository { return &SQLiteRepository{db: db} }

func (r *SQLiteRepository) Create(ctx context.Context, value Budget) (Budget, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Budget{}, err
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `INSERT INTO budgets(id, book_id, currency, total_cap_minor, periodic_cap_minor, per_transaction_cap_minor, period_seconds, requires_approval, frozen, created_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, value.ID, value.BookID, value.Currency, value.TotalCapMinor, value.PeriodicCapMinor, value.PerTransactionCapMinor, int64(value.Period/time.Second), value.RequiresApproval, value.Frozen, value.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Budget{}, fmt.Errorf("create budget: %w", err)
	}
	for _, counterparty := range value.AllowedCounterparties {
		if _, err = tx.ExecContext(ctx, `INSERT INTO budget_scope_entries(budget_id, counterparty, effect) VALUES(?, ?, 'allow')`, value.ID, counterparty); err != nil {
			return Budget{}, err
		}
	}
	for _, counterparty := range value.DeniedCounterparties {
		if _, err = tx.ExecContext(ctx, `INSERT INTO budget_scope_entries(budget_id, counterparty, effect) VALUES(?, ?, 'deny')`, value.ID, counterparty); err != nil {
			return Budget{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return Budget{}, err
	}
	return value, nil
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (Budget, error) {
	var value Budget
	var periodSeconds int64
	var createdAt string
	err := r.db.QueryRowContext(ctx, `SELECT id, book_id, currency, total_cap_minor, periodic_cap_minor, per_transaction_cap_minor, period_seconds, requires_approval, frozen, created_at FROM budgets WHERE id = ?`, id).Scan(&value.ID, &value.BookID, &value.Currency, &value.TotalCapMinor, &value.PeriodicCapMinor, &value.PerTransactionCapMinor, &periodSeconds, &value.RequiresApproval, &value.Frozen, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Budget{}, ErrNotFound
	}
	if err != nil {
		return Budget{}, err
	}
	value.Period = time.Duration(periodSeconds) * time.Second
	value.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Budget{}, fmt.Errorf("parse created_at: %w", err)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT counterparty, effect FROM budget_scope_entries WHERE budget_id = ? ORDER BY counterparty, effect`, id)
	if err != nil {
		return Budget{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var counterparty, effect string
		if err := rows.Scan(&counterparty, &effect); err != nil {
			return Budget{}, err
		}
		if effect == "deny" {
			value.DeniedCounterparties = append(value.DeniedCounterparties, counterparty)
		} else {
			value.AllowedCounterparties = append(value.AllowedCounterparties, counterparty)
		}
	}
	return value, rows.Err()
}

func (r *SQLiteRepository) SetFreezeControl(ctx context.Context, value FreezeControl) (FreezeControl, error) {
	// Use a short transaction so the control is visible atomically before the
	// next authorization or settlement begins.
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return FreezeControl{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `INSERT INTO freeze_controls(scope,scope_id,frozen,updated_at) VALUES(?,?,?,?) ON CONFLICT(scope,scope_id) DO UPDATE SET frozen=excluded.frozen,updated_at=excluded.updated_at`, string(value.Scope), value.ScopeID, value.Frozen, value.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
		return FreezeControl{}, err
	}
	if err := tx.Commit(); err != nil {
		return FreezeControl{}, err
	}
	return value, nil
}

func (r *SQLiteRepository) GetFreezeControl(ctx context.Context, scope FreezeScope, id string) (FreezeControl, error) {
	var value FreezeControl
	var storedScope, updated string
	err := r.db.QueryRowContext(ctx, `SELECT scope,scope_id,frozen,updated_at FROM freeze_controls WHERE scope=? AND scope_id=?`, string(scope), id).Scan(&storedScope, &value.ScopeID, &value.Frozen, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return FreezeControl{}, ErrNotFound
	}
	if err != nil {
		return FreezeControl{}, err
	}
	value.Scope = FreezeScope(storedScope)
	value.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	return value, err
}

func (r *SQLiteRepository) Update(ctx context.Context, value Budget) (Budget, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Budget{}, err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `UPDATE budgets SET total_cap_minor = ?, periodic_cap_minor = ?, per_transaction_cap_minor = ?, period_seconds = ?, requires_approval = ?, frozen = ? WHERE id = ?`, value.TotalCapMinor, value.PeriodicCapMinor, value.PerTransactionCapMinor, int64(value.Period/time.Second), value.RequiresApproval, value.Frozen, value.ID)
	if err != nil {
		return Budget{}, fmt.Errorf("update budget: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Budget{}, err
	}
	if rows != 1 {
		return Budget{}, ErrNotFound
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM budget_scope_entries WHERE budget_id = ?`, value.ID); err != nil {
		return Budget{}, err
	}
	for _, counterparty := range value.AllowedCounterparties {
		if _, err = tx.ExecContext(ctx, `INSERT INTO budget_scope_entries(budget_id, counterparty, effect) VALUES(?, ?, 'allow')`, value.ID, counterparty); err != nil {
			return Budget{}, err
		}
	}
	for _, counterparty := range value.DeniedCounterparties {
		if _, err = tx.ExecContext(ctx, `INSERT INTO budget_scope_entries(budget_id, counterparty, effect) VALUES(?, ?, 'deny')`, value.ID, counterparty); err != nil {
			return Budget{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return Budget{}, err
	}
	return value, nil
}

var _ Repository = (*SQLiteRepository)(nil)
