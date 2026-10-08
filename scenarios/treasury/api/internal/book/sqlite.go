package book

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type DB interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLiteRepository struct{ db DB }

func NewSQLiteRepository(db DB) *SQLiteRepository { return &SQLiteRepository{db: db} }

func (r *SQLiteRepository) Create(ctx context.Context, value Book) (Book, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Book{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `INSERT INTO treasury_beneficiary(singleton_key, identity) VALUES(1, ?) ON CONFLICT(singleton_key) DO NOTHING`, value.BeneficiaryIdentity); err != nil {
		return Book{}, fmt.Errorf("establish beneficiary: %w", err)
	}
	var beneficiary string
	if err = tx.QueryRowContext(ctx, `SELECT identity FROM treasury_beneficiary WHERE singleton_key = 1`).Scan(&beneficiary); err != nil {
		return Book{}, fmt.Errorf("read beneficiary: %w", err)
	}
	if beneficiary != value.BeneficiaryIdentity {
		return Book{}, ErrBeneficiaryConflict
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO books(id, name, beneficiary_identity, created_at) VALUES(?, ?, ?, ?)`, value.ID, value.Name, value.BeneficiaryIdentity, value.CreatedAt.Format(time.RFC3339Nano)); err != nil {
		return Book{}, fmt.Errorf("create book: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return Book{}, err
	}
	return value, nil
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (Book, error) {
	var value Book
	var createdAt string
	err := r.db.QueryRowContext(ctx, `SELECT id, name, beneficiary_identity, created_at FROM books WHERE id = ?`, id).Scan(&value.ID, &value.Name, &value.BeneficiaryIdentity, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Book{}, ErrNotFound
	}
	if err != nil {
		return Book{}, err
	}
	value.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Book{}, fmt.Errorf("parse created_at: %w", err)
	}
	return value, nil
}

var _ Repository = (*SQLiteRepository)(nil)
