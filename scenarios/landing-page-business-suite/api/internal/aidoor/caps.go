package aidoor

import (
	"context"
	"database/sql"
	"errors"
)

var ErrKeyOperationTerminal = errors.New("released key operation is terminal; use a new operation id")

// ReserveCap joins the wallet-owned account transaction. Device and session
// scope claims cannot commit before the wallet's admission decision.
func (s KeyStore) ReserveCap(ctx context.Context, tx *sql.Tx, key Key, operation string, credits int64) error {
	scopes := []string{key.ID}
	if key.ParentID != "" {
		scopes = []string{key.ParentID, key.ID}
	}
	for _, id := range scopes {
		// Updating a row acquires its lock on both supported engines. The no-op
		// avoids SELECT FOR UPDATE dialect branches while preserving serialization.
		if _, err := tx.ExecContext(ctx, `UPDATE ai_access_keys SET max_credits=max_credits WHERE id=$1`, id); err != nil {
			return err
		}
		var cap, used int64
		if err := tx.QueryRowContext(ctx, `SELECT max_credits FROM ai_access_keys WHERE id=$1 AND revoked=FALSE`, id).Scan(&cap); err != nil {
			return err
		}
		var reserved int64
		var actual sql.NullInt64
		err := tx.QueryRowContext(ctx, `SELECT reserved_credits,actual_credits FROM ai_key_charges WHERE key_id=$1 AND operation_id=$2`, id, operation).Scan(&reserved, &actual)
		if err == nil {
			if actual.Valid && actual.Int64 == 0 {
				return ErrKeyOperationTerminal
			}
			if reserved != credits {
				return errors.New("operation cap conflict")
			}
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(COALESCE(actual_credits,reserved_credits)),0) FROM ai_key_charges WHERE key_id=$1`, id).Scan(&used); err != nil {
			return err
		}
		if credits <= 0 || used > cap-credits {
			return errors.New("key credit cap exceeded")
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO ai_key_charges(operation_id,key_id,reserved_credits) VALUES($1,$2,$3)`, operation, id, credits); err != nil {
			return err
		}
	}
	return nil
}

func (s KeyStore) SettleCap(ctx context.Context, operation string, actual int64) error {
	return settleCap(ctx, s.DB, operation, actual)
}

func (s KeyStore) RejectCap(ctx context.Context, tx *sql.Tx, operation string) error {
	return settleCap(ctx, tx, operation, 0)
}

func settleCap(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, operation string, actual int64,
) error {
	_, err := db.ExecContext(ctx, `UPDATE ai_key_charges SET actual_credits=$1 WHERE operation_id=$2 AND (actual_credits IS NULL OR actual_credits=$1)`, actual, operation)
	return err
}
