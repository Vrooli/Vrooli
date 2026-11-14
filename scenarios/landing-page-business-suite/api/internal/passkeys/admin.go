package passkeys

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

type AdminUser struct {
	ID          int64
	Email       string
	Handle      []byte
	Credentials []webauthn.Credential
}

func (u AdminUser) WebAuthnID() []byte                         { return u.Handle }
func (u AdminUser) WebAuthnName() string                       { return u.Email }
func (u AdminUser) WebAuthnDisplayName() string                { return u.Email }
func (u AdminUser) WebAuthnCredentials() []webauthn.Credential { return u.Credentials }

type AdminCredentialRecord struct {
	ID, Nickname string
	CreatedAt    time.Time
	LastUsedAt   sql.NullTime
	BackupState  sql.NullString
	Credential   webauthn.Credential
}
type AdminCredentials struct{ DB CredentialStore }

func (r *AdminCredentials) HasTOTP(ctx context.Context, adminID int64) (bool, error) {
	var enabled bool
	err := r.DB.QueryRowContext(ctx, `SELECT totp_enabled_at IS NOT NULL FROM admin_users WHERE id = $1`, adminID).Scan(&enabled)
	return enabled, err
}

func (r *AdminCredentials) User(ctx context.Context, email string) (AdminUser, error) {
	var u AdminUser
	if err := r.DB.QueryRowContext(ctx, `SELECT id, email, webauthn_user_handle FROM admin_users WHERE LOWER(email) = LOWER($1)`, email).Scan(&u.ID, &u.Email, &u.Handle); err != nil {
		return AdminUser{}, err
	}
	if len(u.Handle) == 0 {
		u.Handle = make([]byte, 64)
		if _, err := rand.Read(u.Handle); err != nil {
			return AdminUser{}, err
		}
		if _, err := r.DB.ExecContext(ctx, `UPDATE admin_users SET webauthn_user_handle = $1 WHERE id = $2 AND webauthn_user_handle IS NULL`, u.Handle, u.ID); err != nil {
			return AdminUser{}, err
		}
	}
	records, err := r.List(ctx, u.ID)
	if err != nil {
		return AdminUser{}, err
	}
	for _, record := range records {
		u.Credentials = append(u.Credentials, record.Credential)
	}
	return u, nil
}
func (r *AdminCredentials) List(ctx context.Context, adminID int64) ([]AdminCredentialRecord, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, nickname, created_at, last_used_at, backup_state, credential_id, public_key, sign_count FROM admin_passkeys WHERE admin_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC`, adminID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AdminCredentialRecord
	for rows.Next() {
		var v AdminCredentialRecord
		var id, key []byte
		if err := rows.Scan(&v.ID, &v.Nickname, &v.CreatedAt, &v.LastUsedAt, &v.BackupState, &id, &key, &v.Credential.Authenticator.SignCount); err != nil {
			return nil, err
		}
		v.Credential.ID, v.Credential.PublicKey = id, key
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *AdminCredentials) Save(ctx context.Context, adminID int64, rpID, nickname string, credential webauthn.Credential) (string, error) {
	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		nickname = "Passkey"
	}
	if len([]rune(nickname)) > 64 {
		return "", fmt.Errorf("passkey nickname must be 1-64 characters")
	}
	id := uuid.NewString()
	_, err := r.DB.ExecContext(ctx, `INSERT INTO admin_passkeys (id, admin_id, credential_id, public_key, sign_count, rp_id, nickname) VALUES ($1,$2,$3,$4,$5,$6,$7)`, id, adminID, credential.ID, credential.PublicKey, credential.Authenticator.SignCount, rpID, nickname)
	return id, err
}
func (r *AdminCredentials) Rename(ctx context.Context, adminID int64, id, nickname string) error {
	if n := len([]rune(strings.TrimSpace(nickname))); n < 1 || n > 64 {
		return fmt.Errorf("passkey nickname must be 1-64 characters")
	}
	result, err := r.DB.ExecContext(ctx, `UPDATE admin_passkeys SET nickname = $1 WHERE id = $2 AND admin_id = $3 AND revoked_at IS NULL`, strings.TrimSpace(nickname), id, adminID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return sql.ErrNoRows
	}
	return nil
}
func (r *AdminCredentials) Revoke(ctx context.Context, adminID int64, id string) error {
	result, err := r.DB.ExecContext(ctx, `UPDATE admin_passkeys SET revoked_at = NOW() WHERE id = $1 AND admin_id = $2 AND revoked_at IS NULL`, id, adminID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return sql.ErrNoRows
	}
	return nil
}
func (r *AdminCredentials) RecordUse(ctx context.Context, credential webauthn.Credential) error {
	result, err := r.DB.ExecContext(ctx, `UPDATE admin_passkeys SET sign_count = $1, last_used_at = NOW() WHERE credential_id = $2 AND revoked_at IS NULL`, credential.Authenticator.SignCount, credential.ID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return sql.ErrNoRows
	}
	return nil
}
