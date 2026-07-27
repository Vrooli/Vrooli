package main

import (
	"context"
	"database/sql"
	"time"
)

// AdminAuthStore is the persistence boundary for administrator authentication.
// Implementations must honor the request context passed to every operation.
type AdminAuthStore interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// AdminAuthService owns administrator credential and server-side session
// persistence. HTTP handlers retain cookie and response responsibilities only.
type AdminAuthService struct {
	store AdminAuthStore
}

type AdminProfile struct {
	ID           int64
	Email        string
	PasswordHash string
}

func NewAdminAuthService(store AdminAuthStore) *AdminAuthService {
	return &AdminAuthService{store: store}
}

func (s *AdminAuthService) PasswordHash(ctx context.Context, email string) (string, error) {
	var passwordHash string
	err := s.store.QueryRowContext(ctx,
		"SELECT password_hash FROM admin_users WHERE email = $1", email,
	).Scan(&passwordHash)
	return passwordHash, err
}

func (s *AdminAuthService) UpdateLastLogin(ctx context.Context, email string) error {
	_, err := s.store.ExecContext(ctx, "UPDATE admin_users SET last_login = NOW() WHERE email = $1", email)
	return err
}

func (s *AdminAuthService) CreateSession(ctx context.Context, id, email string, expiresAt time.Time, clientIP, userAgent string) error {
	_, err := s.store.ExecContext(ctx, `
		INSERT INTO admin_sessions (id, admin_email, expires_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5)
	`, id, email, expiresAt, clientIP, userAgent)
	return err
}

func (s *AdminAuthService) DeleteSession(ctx context.Context, id string) error {
	_, err := s.store.ExecContext(ctx, "DELETE FROM admin_sessions WHERE id = $1", id)
	return err
}

func (s *AdminAuthService) SessionExpiry(ctx context.Context, id, email string) (time.Time, error) {
	var expiresAt time.Time
	err := s.store.QueryRowContext(ctx, `
		SELECT expires_at FROM admin_sessions WHERE id = $1 AND admin_email = $2
	`, id, email).Scan(&expiresAt)
	return expiresAt, err
}

func (s *AdminAuthService) TouchSession(ctx context.Context, id string) error {
	_, err := s.store.ExecContext(ctx, `UPDATE admin_sessions SET last_activity = NOW() WHERE id = $1`, id)
	return err
}

func (s *AdminAuthService) Profile(ctx context.Context, email string) (AdminProfile, error) {
	var profile AdminProfile
	err := s.store.QueryRowContext(ctx,
		`SELECT id, email, password_hash FROM admin_users WHERE email = $1`, email,
	).Scan(&profile.ID, &profile.Email, &profile.PasswordHash)
	return profile, err
}

func (s *AdminAuthService) EmailInUse(ctx context.Context, email string, excludingID int64) (bool, error) {
	var count int
	err := s.store.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM admin_users WHERE LOWER(email) = LOWER($1) AND id <> $2`, email, excludingID,
	).Scan(&count)
	return count > 0, err
}

func (s *AdminAuthService) UpdateProfile(ctx context.Context, id int64, email, passwordHash string) error {
	_, err := s.store.ExecContext(ctx,
		`UPDATE admin_users SET email = $1, password_hash = $2 WHERE id = $3`, email, passwordHash, id,
	)
	return err
}

func (s *AdminAuthService) RevokeOtherSessions(ctx context.Context, email, currentID string) (int64, error) {
	result, err := s.store.ExecContext(ctx,
		`DELETE FROM admin_sessions WHERE admin_email = $1 AND id != $2`, email, currentID,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
