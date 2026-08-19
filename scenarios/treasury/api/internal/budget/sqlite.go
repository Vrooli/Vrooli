package budget

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type SQLiteRepository struct{ db *sql.DB }

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository { return &SQLiteRepository{db: db} }

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

var _ Repository = (*SQLiteRepository)(nil)
