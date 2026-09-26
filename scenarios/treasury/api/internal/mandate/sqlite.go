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
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
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
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return Mandate{}, err
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `INSERT INTO mandates(id, idempotency_key, book_id, budget_id, authorizer, cap_minor, currency, allowed_counterparties_json, denied_counterparties_json, required_evidence_json, expires_at, issued_at, signature, status) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, value.ID, value.IdempotencyKey, value.BookID, value.BudgetID, value.Authorizer, value.CapMinor, value.Currency, string(allowed), string(denied), string(evidence), value.ExpiresAt.Format(time.RFC3339Nano), value.IssuedAt.Format(time.RFC3339Nano), value.Signature, string(value.Status))
	if err != nil {
		return Mandate{}, fmt.Errorf("create mandate: %w", err)
	}
	if value.RecurrenceInterval > 0 {
		if _, err := tx.ExecContext(ctx, `INSERT INTO mandate_recurrences(mandate_id,interval_seconds,next_charge_at,cancelled_at,updated_at) VALUES(?,?,?,'',?)`, value.ID, int64(value.RecurrenceInterval/time.Second), value.NextChargeAt.Format(time.RFC3339Nano), value.IssuedAt.Format(time.RFC3339Nano)); err != nil {
			return Mandate{}, fmt.Errorf("create mandate recurrence: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Mandate{}, err
	}
	return value, nil
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (Mandate, error) {
	return r.get(ctx, selectMandate+` WHERE m.id = ?`, id)
}

func (r *SQLiteRepository) GetByIdempotencyKey(ctx context.Context, key string) (Mandate, error) {
	return r.get(ctx, selectMandate+` WHERE m.idempotency_key = ?`, key)
}

func (r *SQLiteRepository) List(ctx context.Context) ([]Mandate, error) {
	rows, err := r.db.QueryContext(ctx, selectMandate+` ORDER BY m.issued_at DESC, m.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]Mandate, 0)
	for rows.Next() {
		value, err := scanMandate(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *SQLiteRepository) CancelStanding(ctx context.Context, id string, at time.Time) (Mandate, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return Mandate{}, err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `UPDATE mandates SET status='revoked' WHERE id=? AND status='live' AND EXISTS (SELECT 1 FROM mandate_recurrences WHERE mandate_id=mandates.id)`, id)
	if err != nil {
		return Mandate{}, err
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		return Mandate{}, fmt.Errorf("cancel standing mandate: %w", ErrNotFound)
	}
	stamp := at.Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `UPDATE mandate_recurrences SET cancelled_at=?,updated_at=? WHERE mandate_id=? AND cancelled_at=''`, stamp, stamp, id); err != nil {
		return Mandate{}, err
	}
	value, err := scanMandate(tx.QueryRowContext(ctx, selectMandate+` WHERE m.id=?`, id))
	if err != nil {
		return Mandate{}, err
	}
	if err := tx.Commit(); err != nil {
		return Mandate{}, err
	}
	return value, nil
}

func (r *SQLiteRepository) ClaimDue(ctx context.Context, id string, now time.Time) (DueOccurrence, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return DueOccurrence{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var interval int64
	var chargeAt string
	err = tx.QueryRowContext(ctx, `SELECT r.interval_seconds,r.next_charge_at FROM mandate_recurrences r JOIN mandates m ON m.id=r.mandate_id WHERE r.mandate_id=? AND r.cancelled_at='' AND r.next_charge_at<=? AND m.status='live' AND m.expires_at>?`, id, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)).Scan(&interval, &chargeAt)
	if errors.Is(err, sql.ErrNoRows) {
		return DueOccurrence{}, ErrNotFound
	}
	if err != nil {
		return DueOccurrence{}, err
	}
	chargeTime, err := time.Parse(time.RFC3339Nano, chargeAt)
	if err != nil {
		return DueOccurrence{}, err
	}
	next := chargeTime.Add(time.Duration(interval) * time.Second)
	result, err := tx.ExecContext(ctx, `UPDATE mandate_recurrences SET next_charge_at=?,updated_at=? WHERE mandate_id=? AND cancelled_at='' AND next_charge_at=?`, next.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), id, chargeAt)
	if err != nil {
		return DueOccurrence{}, err
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		return DueOccurrence{}, ErrNotFound
	}
	if err := tx.Commit(); err != nil {
		return DueOccurrence{}, err
	}
	return DueOccurrence{MandateID: id, IdempotencyKey: id + ":recurrence:" + chargeTime.UTC().Format(time.RFC3339Nano), ChargeAt: chargeTime, NextChargeAt: next}, nil
}

func (r *SQLiteRepository) UpdateStatus(ctx context.Context, id string, from, to flow.MandateStatus) error {
	result, err := r.db.ExecContext(ctx, `UPDATE mandates SET status = ? WHERE id = ? AND status = ?`, string(to), id, string(from))
	if err != nil {
		return fmt.Errorf("update mandate status: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("update mandate status: %w", ErrNotFound)
	}
	return nil
}

func (r *SQLiteRepository) get(ctx context.Context, query, arg string) (Mandate, error) {
	value, err := scanMandate(r.db.QueryRowContext(ctx, query, arg))
	if errors.Is(err, sql.ErrNoRows) {
		return Mandate{}, ErrNotFound
	}
	return value, err
}

type scanner interface {
	Scan(...any) error
}

func scanMandate(row scanner) (Mandate, error) {
	var value Mandate
	var allowed, denied, evidence, expiresAt, issuedAt, status string
	var recurrenceSeconds sql.NullInt64
	var nextChargeAt, cancelledAt sql.NullString
	err := row.Scan(&value.ID, &value.IdempotencyKey, &value.BookID, &value.BudgetID, &value.Authorizer, &value.CapMinor, &value.Currency, &allowed, &denied, &evidence, &expiresAt, &issuedAt, &value.Signature, &status, &recurrenceSeconds, &nextChargeAt, &cancelledAt)
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
	if recurrenceSeconds.Valid {
		value.RecurrenceInterval = time.Duration(recurrenceSeconds.Int64) * time.Second
		if value.NextChargeAt, err = time.Parse(time.RFC3339Nano, nextChargeAt.String); err != nil {
			return Mandate{}, err
		}
		if cancelledAt.String != "" {
			value.CancelledAt, err = time.Parse(time.RFC3339Nano, cancelledAt.String)
		}
	}
	return value, nil
}

const selectMandate = `SELECT m.id,m.idempotency_key,m.book_id,m.budget_id,m.authorizer,m.cap_minor,m.currency,m.allowed_counterparties_json,m.denied_counterparties_json,m.required_evidence_json,m.expires_at,m.issued_at,m.signature,m.status,r.interval_seconds,r.next_charge_at,r.cancelled_at FROM mandates m LEFT JOIN mandate_recurrences r ON r.mandate_id=m.id`

var _ Repository = (*SQLiteRepository)(nil)
