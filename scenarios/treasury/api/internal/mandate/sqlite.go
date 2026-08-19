package mandate

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"treasury/internal/mandate/flow"
)

type DB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLiteRepository struct{ db DB }

func NewSQLiteRepository(db DB) *SQLiteRepository { return &SQLiteRepository{db: db} }

func (r *SQLiteRepository) Create(ctx context.Context, value Mandate) (Mandate, error) {
	allowed, err := json.Marshal(value.AllowedCounterparties)
	if err != nil {
		return Mandate{}, err
	}
	denied, err := json.Marshal(value.DeniedCounterparties)
	if err != nil {
		return Mandate{}, err
	}
	evidence, err := json.Marshal(value.RequiredEvidence)
	if err != nil {
		return Mandate{}, err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO mandates(id, idempotency_key, book_id, budget_id, authorizer, cap_minor, currency, allowed_counterparties_json, denied_counterparties_json, required_evidence_json, expires_at, issued_at, signature, status) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, value.ID, value.IdempotencyKey, value.BookID, value.BudgetID, value.Authorizer, value.CapMinor, value.Currency, string(allowed), string(denied), string(evidence), value.ExpiresAt.Format(time.RFC3339Nano), value.IssuedAt.Format(time.RFC3339Nano), value.Signature, string(value.Status))
	if err != nil {
		return Mandate{}, fmt.Errorf("create mandate: %w", err)
	}
	return value, nil
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (Mandate, error) {
	return r.get(ctx, `SELECT id, idempotency_key, book_id, budget_id, authorizer, cap_minor, currency, allowed_counterparties_json, denied_counterparties_json, required_evidence_json, expires_at, issued_at, signature, status FROM mandates WHERE id = ?`, id)
}

func (r *SQLiteRepository) GetByIdempotencyKey(ctx context.Context, key string) (Mandate, error) {
	return r.get(ctx, `SELECT id, idempotency_key, book_id, budget_id, authorizer, cap_minor, currency, allowed_counterparties_json, denied_counterparties_json, required_evidence_json, expires_at, issued_at, signature, status FROM mandates WHERE idempotency_key = ?`, key)
}

func (r *SQLiteRepository) get(ctx context.Context, query, arg string) (Mandate, error) {
	var value Mandate
	var allowed, denied, evidence, expiresAt, issuedAt, status string
	err := r.db.QueryRowContext(ctx, query, arg).Scan(&value.ID, &value.IdempotencyKey, &value.BookID, &value.BudgetID, &value.Authorizer, &value.CapMinor, &value.Currency, &allowed, &denied, &evidence, &expiresAt, &issuedAt, &value.Signature, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return Mandate{}, ErrNotFound
	}
	if err != nil {
		return Mandate{}, err
	}
	if err = json.Unmarshal([]byte(allowed), &value.AllowedCounterparties); err != nil {
		return Mandate{}, err
	}
	if err = json.Unmarshal([]byte(denied), &value.DeniedCounterparties); err != nil {
		return Mandate{}, err
	}
	if err = json.Unmarshal([]byte(evidence), &value.RequiredEvidence); err != nil {
		return Mandate{}, err
	}
	value.ExpiresAt, err = time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil {
		return Mandate{}, err
	}
	value.IssuedAt, err = time.Parse(time.RFC3339Nano, issuedAt)
	if err != nil {
		return Mandate{}, err
	}
	value.Status = flow.MandateStatus(status)
	return value, nil
}

var _ Repository = (*SQLiteRepository)(nil)
