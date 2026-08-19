package approval

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type DB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLiteRepository struct{ db DB }

func NewSQLiteRepository(db DB) *SQLiteRepository { return &SQLiteRepository{db: db} }

func (r *SQLiteRepository) Create(ctx context.Context, value Request) (Request, error) {
	_, err := r.db.ExecContext(ctx, `INSERT INTO approval_requests(id,authorization_id,mandate_id,requesting_agent,amount_minor,currency,counterparty,status,resolver_identity,created_at,expires_at,resolved_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO NOTHING`, value.ID, value.AuthorizationID, value.MandateID, value.RequestingAgent, value.AmountMinor, value.Currency, value.Counterparty, string(value.Status), value.ResolverIdentity, value.CreatedAt.Format(time.RFC3339Nano), value.ExpiresAt.Format(time.RFC3339Nano), "")
	if err != nil {
		return Request{}, fmt.Errorf("create approval: %w", err)
	}
	return r.Get(ctx, value.ID)
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (Request, error) {
	var value Request
	var status, createdAt, expiresAt, resolvedAt string
	err := r.db.QueryRowContext(ctx, `SELECT id,authorization_id,mandate_id,requesting_agent,amount_minor,currency,counterparty,status,resolver_identity,created_at,expires_at,resolved_at FROM approval_requests WHERE id=?`, id).Scan(&value.ID, &value.AuthorizationID, &value.MandateID, &value.RequestingAgent, &value.AmountMinor, &value.Currency, &value.Counterparty, &status, &value.ResolverIdentity, &createdAt, &expiresAt, &resolvedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Request{}, ErrNotFound
	}
	if err != nil {
		return Request{}, err
	}
	value.Status = Status(status)
	if value.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return Request{}, err
	}
	if value.ExpiresAt, err = time.Parse(time.RFC3339Nano, expiresAt); err != nil {
		return Request{}, err
	}
	if resolvedAt != "" {
		if value.ResolvedAt, err = time.Parse(time.RFC3339Nano, resolvedAt); err != nil {
			return Request{}, err
		}
	}
	return value, nil
}

func (r *SQLiteRepository) Resolve(ctx context.Context, id string, status Status, resolver, resolvedAt string) (Request, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE approval_requests SET status=?,resolver_identity=?,resolved_at=? WHERE id=? AND status='queued'`, string(status), resolver, resolvedAt, id)
	if err != nil {
		return Request{}, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Request{}, err
	}
	if rows == 0 {
		return Request{}, ErrNotFound
	}
	return r.Get(ctx, id)
}

func (r *SQLiteRepository) RecordRelay(ctx context.Context, value RelayAttempt) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO approval_relay_attempts(id,approval_id,outcome,error,attempted_at) VALUES(?,?,?,?,?)`, value.ID, value.ApprovalID, value.Outcome, value.Error, value.AttemptedAt.Format(time.RFC3339Nano))
	return err
}

func (r *SQLiteRepository) ListRelayAttempts(ctx context.Context, approvalID string) ([]RelayAttempt, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,approval_id,outcome,error,attempted_at FROM approval_relay_attempts WHERE approval_id=? ORDER BY attempted_at,id`, approvalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []RelayAttempt
	for rows.Next() {
		var value RelayAttempt
		var attemptedAt string
		if err := rows.Scan(&value.ID, &value.ApprovalID, &value.Outcome, &value.Error, &attemptedAt); err != nil {
			return nil, err
		}
		value.AttemptedAt, err = time.Parse(time.RFC3339Nano, attemptedAt)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

var _ Repository = (*SQLiteRepository)(nil)
