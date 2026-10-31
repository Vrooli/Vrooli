package instrument

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type DB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLiteRepository struct{ db DB }

func NewSQLiteRepository(db DB) *SQLiteRepository { return &SQLiteRepository{db: db} }

func (r *SQLiteRepository) Create(ctx context.Context, value Instrument) (Instrument, error) {
	_, err := r.db.ExecContext(ctx, `INSERT INTO instruments(id,book_id,mandate_id,rail,credential_reference,cap_minor,currency,counterparty,expires_at,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, value.ID, value.BookID, value.MandateID, value.Rail, value.CredentialReference, value.CapMinor, value.Currency, value.Counterparty, value.ExpiresAt.Format(time.RFC3339Nano), value.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Instrument{}, fmt.Errorf("create instrument: %w", err)
	}
	return value, nil
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (Instrument, error) {
	var value Instrument
	var expiresAt, createdAt string
	err := r.db.QueryRowContext(ctx, `SELECT id,book_id,mandate_id,rail,credential_reference,cap_minor,currency,counterparty,expires_at,created_at FROM instruments WHERE id=?`, id).Scan(&value.ID, &value.BookID, &value.MandateID, &value.Rail, &value.CredentialReference, &value.CapMinor, &value.Currency, &value.Counterparty, &expiresAt, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Instrument{}, ErrNotFound
	}
	if err != nil {
		return Instrument{}, err
	}
	if value.ExpiresAt, err = time.Parse(time.RFC3339Nano, expiresAt); err != nil {
		return Instrument{}, err
	}
	if value.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
		return Instrument{}, err
	}
	return value, nil
}

var _ Repository = (*SQLiteRepository)(nil)
