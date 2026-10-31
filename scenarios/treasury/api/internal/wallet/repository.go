package wallet

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Repository owns custody metadata and serialization. Its lock spans the
// credential operation, witness and public binding, independent of SQL engine.
type Repository interface {
	Prepare(context.Context, Wallet) (Wallet, error)
	Get(context.Context, string) (Wallet, error)
	WithCustodyLock(context.Context, string, func(Repository) error) error
	Minted(context.Context, string, bool) (bool, error)
	RecordMint(context.Context, string, bool) error
	SetPublic(context.Context, string, string, string) error
	SetRecovery(context.Context, string, string) error
}
type DB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type SQLiteRepository struct{ db DB }

func NewSQLiteRepository(db DB) *SQLiteRepository { return &SQLiteRepository{db: db} }
func (r *SQLiteRepository) Prepare(ctx context.Context, w Wallet) (Wallet, error) {
	_, err := r.db.ExecContext(ctx, `INSERT INTO research_wallets(id,book_id,operator_identity,network,custody_reference,created_at) VALUES(?,?,?,?,?,?) ON CONFLICT(id) DO NOTHING`, w.ID, w.BookID, w.OperatorIdentity, w.Network, w.CustodyReference, w.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return Wallet{}, err
	}
	return r.Get(ctx, w.ID)
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (Wallet, error) {
	var w Wallet
	var created string
	err := r.db.QueryRowContext(ctx, `SELECT id,book_id,operator_identity,network,address,public_key,custody_reference,created_at,recovery_status,recovery_sha256 FROM research_wallets WHERE id=?`, id).Scan(&w.ID, &w.BookID, &w.OperatorIdentity, &w.Network, &w.Address, &w.PublicKey, &w.CustodyReference, &created, &w.RecoveryStatus, &w.RecoverySHA256)
	if errors.Is(err, sql.ErrNoRows) {
		return Wallet{}, ErrNotFound
	}
	if err != nil {
		return Wallet{}, err
	}
	w.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	return w, err
}

func (r *SQLiteRepository) WithCustodyLock(ctx context.Context, id string, fn func(Repository) error) error {
	starter, ok := r.db.(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return ErrUnavailable
	}
	tx, err := starter.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	// First statement is a write, before any transactional read, so SQLite
	// serializes independent processes without a deferred-read upgrade race.
	result, err := tx.ExecContext(ctx, `UPDATE research_wallets SET key_minted=key_minted WHERE id=?`, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrNotFound
	}
	if err = fn(NewSQLiteRepository(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLiteRepository) Minted(ctx context.Context, id string, recovery bool) (bool, error) {
	column := "key_minted"
	if recovery {
		column = "recovery_minted"
	}
	var minted int
	err := r.db.QueryRowContext(ctx, `SELECT `+column+` FROM research_wallets WHERE id=?`, id).Scan(&minted)
	return minted == 1, err
}

func (r *SQLiteRepository) RecordMint(ctx context.Context, id string, recovery bool) error {
	column := "key_minted"
	if recovery {
		column = "recovery_minted"
	}
	result, err := r.db.ExecContext(ctx, `UPDATE research_wallets SET `+column+`=1 WHERE id=?`, id)
	return changedOne(result, err)
}

func (r *SQLiteRepository) SetPublic(ctx context.Context, id, address, public string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE research_wallets SET address=?,public_key=? WHERE id=? AND address=''`, address, public, id)
	return changedOne(result, err)
}

func (r *SQLiteRepository) SetRecovery(ctx context.Context, id, digest string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE research_wallets SET recovery_status='same_host_encrypted_verified',recovery_sha256=? WHERE id=? AND (recovery_sha256='' OR recovery_sha256=?)`, digest, id, digest)
	return changedOne(result, err)
}

func changedOne(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrMismatch
	}
	return nil
}
