// Package nativegrants owns durable, one-use native authorization grants.
package nativegrants

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrRejected = errors.New("native authorization grant rejected")

type Store interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type Grant struct {
	UserID          string
	CodeChallenge   string
	RedirectURI     string
	AuthenticatedAt time.Time
	SignInIP        string
	SignInUserAgent string
	SessionID       string
}

type Repository struct{ db Store }

func NewRepository(db Store) *Repository { return &Repository{db: db} }

func hashCode(code string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(code)))
	return hex.EncodeToString(digest[:])
}

func (r *Repository) Issue(ctx context.Context, code, userID, challenge, redirectURI, browserBinding, ipAddress, userAgent string, authenticatedAt time.Time, ttl time.Duration) error {
	if r == nil || r.db == nil || strings.TrimSpace(code) == "" || strings.TrimSpace(userID) == "" || strings.TrimSpace(challenge) == "" || strings.TrimSpace(redirectURI) == "" {
		return ErrRejected
	}
	if ttl <= 0 {
		ttl = time.Minute
	}
	if authenticatedAt.IsZero() {
		authenticatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO native_auth_grants
		(code_hash, user_id, code_challenge, redirect_uri, browser_binding_hash, sign_in_ip, sign_in_user_agent, authenticated_at, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6::inet,$7,$8,$9)`,
		hashCode(code), userID, challenge, redirectURI, nullableHash(browserBinding), nullableValue(ipAddress), nullableValue(userAgent), authenticatedAt, time.Now().UTC().Add(ttl))
	if err != nil {
		return fmt.Errorf("issue native authorization grant: %w", err)
	}
	return nil
}

func (r *Repository) Consume(ctx context.Context, code string) (*Grant, error) {
	if r == nil || r.db == nil || strings.TrimSpace(code) == "" {
		return nil, ErrRejected
	}
	hash := hashCode(code)
	grant := &Grant{}
	err := r.db.QueryRowContext(ctx, `
		UPDATE native_auth_grants
		SET used_at = NOW()
		WHERE code_hash = $1 AND used_at IS NULL AND expires_at > NOW()
		RETURNING user_id, code_challenge, redirect_uri, authenticated_at, COALESCE(sign_in_ip::text, ''), COALESCE(sign_in_user_agent, '')`, hash).
		Scan(&grant.UserID, &grant.CodeChallenge, &grant.RedirectURI, &grant.AuthenticatedAt, &grant.SignInIP, &grant.SignInUserAgent)
	if err == nil {
		return grant, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("consume native authorization grant: %w", err)
	}
	var sessionID sql.NullString
	if lookupErr := r.db.QueryRowContext(ctx, `SELECT session_id FROM native_auth_grants WHERE code_hash = $1`, hash).Scan(&sessionID); lookupErr == nil && sessionID.Valid && sessionID.String != "" {
		_, _ = r.db.ExecContext(ctx, `UPDATE user_sessions SET revoked = TRUE, revoked_at = NOW(), revoked_reason = 'native_code_replay' WHERE id = $1 AND revoked = FALSE`, sessionID.String)
	}
	return nil, ErrRejected
}

func (r *Repository) AttachSession(ctx context.Context, code, sessionID string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE native_auth_grants SET session_id = $1 WHERE code_hash = $2 AND used_at IS NOT NULL AND session_id IS NULL`, sessionID, hashCode(code))
	if err != nil {
		return fmt.Errorf("attach native grant session: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return ErrRejected
	}
	return nil
}

func (r *Repository) PurgeExpired(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM native_auth_grants WHERE expires_at < NOW() - INTERVAL '1 day'`)
	if err != nil {
		return 0, fmt.Errorf("purge native authorization grants: %w", err)
	}
	return result.RowsAffected()
}

func nullableHash(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return hashCode(value)
}

func nullableValue(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
