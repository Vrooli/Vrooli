package passkeys

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

type CredentialStore interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type CredentialRecord struct {
	ID          string
	Nickname    string
	CreatedAt   time.Time
	LastUsedAt  sql.NullTime
	BackupState sql.NullString
	Credential  webauthn.Credential
}

type Credentials struct{ DB CredentialStore }

func (r *Credentials) UserForCredential(ctx context.Context, credentialID []byte) (string, error) {
	var userID string
	err := r.DB.QueryRowContext(ctx, `SELECT user_id FROM user_passkeys WHERE credential_id = $1 AND revoked_at IS NULL`, credentialID).Scan(&userID)
	return userID, err
}

func (r *Credentials) List(ctx context.Context, userID string) ([]CredentialRecord, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, nickname, created_at, last_used_at, backup_state, credential_id, public_key, sign_count FROM user_passkeys WHERE user_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []CredentialRecord{}
	for rows.Next() {
		var record CredentialRecord
		var credentialID, publicKey []byte
		if err := rows.Scan(&record.ID, &record.Nickname, &record.CreatedAt, &record.LastUsedAt, &record.BackupState, &credentialID, &publicKey, &record.Credential.Authenticator.SignCount); err != nil {
			return nil, err
		}
		record.Credential.ID, record.Credential.PublicKey = credentialID, publicKey
		result = append(result, record)
	}
	return result, rows.Err()
}

func (r *Credentials) Save(ctx context.Context, userID, rpID, nickname string, credential webauthn.Credential) (string, error) {
	if strings.TrimSpace(nickname) == "" {
		nickname = "Passkey"
	}
	if len([]rune(nickname)) > 64 {
		return "", fmt.Errorf("passkey nickname must be 1-64 characters")
	}
	id := uuid.NewString()
	_, err := r.DB.ExecContext(ctx, `INSERT INTO user_passkeys (id, user_id, credential_id, public_key, sign_count, rp_id, nickname) VALUES ($1, $2, $3, $4, $5, $6, $7)`, id, userID, credential.ID, credential.PublicKey, credential.Authenticator.SignCount, rpID, nickname)
	return id, err
}

func (r *Credentials) RecordUse(ctx context.Context, credential webauthn.Credential) error {
	result, err := r.DB.ExecContext(ctx, `UPDATE user_passkeys SET sign_count = $1, last_used_at = NOW() WHERE credential_id = $2 AND revoked_at IS NULL`, credential.Authenticator.SignCount, credential.ID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Credentials) Rename(ctx context.Context, userID, id, nickname string) error {
	if n := len([]rune(strings.TrimSpace(nickname))); n < 1 || n > 64 {
		return fmt.Errorf("passkey nickname must be 1-64 characters")
	}
	result, err := r.DB.ExecContext(ctx, `UPDATE user_passkeys SET nickname = $1 WHERE id = $2 AND user_id = $3 AND revoked_at IS NULL`, strings.TrimSpace(nickname), id, userID)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Credentials) Revoke(ctx context.Context, userID, id string) error {
	result, err := r.DB.ExecContext(ctx, `UPDATE user_passkeys SET revoked_at = NOW() WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`, id, userID)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return sql.ErrNoRows
	}
	return nil
}
