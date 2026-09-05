package administration

import (
	"context"
	"database/sql"
	"errors"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// CredentialWitness records only that a generated credential has existed on
// this deployment. It deliberately never stores credential material.
type CredentialWitness struct {
	db interface {
		QueryRowContext(context.Context, string, ...any) *sql.Row
		ExecContext(context.Context, string, ...any) (sql.Result, error)
	}
}

func NewCredentialWitness(db interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
},
) *CredentialWitness {
	return &CredentialWitness{db: db}
}

func (w *CredentialWitness) Minted(identity credentialauthority.Identity, field string) (bool, error) {
	if w == nil || w.db == nil {
		return false, errors.New("credential witness database is unavailable")
	}
	var exists bool
	err := w.db.QueryRowContext(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM credential_mint_witness WHERE logical_id = $1 AND field = $2
		)
	`, string(identity), field).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (w *CredentialWitness) RecordMint(identity credentialauthority.Identity, field string) error {
	if w == nil || w.db == nil {
		return errors.New("credential witness database is unavailable")
	}
	_, err := w.db.ExecContext(context.Background(), `
		INSERT INTO credential_mint_witness (logical_id, field) VALUES ($1, $2)
		ON CONFLICT (logical_id, field) DO NOTHING
	`, string(identity), field)
	return err
}
