package administration

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
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

type AdminSessionState struct {
	ExpiresAt         time.Time
	LastActivity      time.Time
	Assurance         string
	ReauthenticatedAt sql.NullTime
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
	return s.CreateSessionWithAssurance(ctx, id, email, expiresAt, clientIP, userAgent, "full")
}

func (s *AdminAuthService) CreateSessionWithAssurance(ctx context.Context, id, email string, expiresAt time.Time, clientIP, userAgent, assurance string) error {
	_, err := s.store.ExecContext(ctx, `
		INSERT INTO admin_sessions (id, admin_email, expires_at, ip_address, user_agent, assurance)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, email, expiresAt, clientIP, userAgent, assurance)
	return err
}

func (s *AdminAuthService) SessionState(ctx context.Context, id, email string) (AdminSessionState, error) {
	var state AdminSessionState
	err := s.store.QueryRowContext(ctx, `
		SELECT expires_at, last_activity, assurance, reauthenticated_at
		FROM admin_sessions WHERE id = $1 AND admin_email = $2
	`, id, email).Scan(&state.ExpiresAt, &state.LastActivity, &state.Assurance, &state.ReauthenticatedAt)
	return state, err
}

// MarkReauthenticated records a successful recent-authentication proof for a
// single server-side session. The bearer identifier remains unchanged; the
// timestamp is intentionally session-scoped.
func (s *AdminAuthService) MarkReauthenticated(ctx context.Context, id, email string) error {
	_, err := s.store.ExecContext(ctx, `UPDATE admin_sessions SET reauthenticated_at = NOW() WHERE id = $1 AND admin_email = $2`, id, email)
	return err
}

// RotateSession preserves assurance state while replacing the bearer session
// identifier after a credential or privilege change.
func (s *AdminAuthService) RotateSession(ctx context.Context, oldID, oldEmail, email string) (string, error) {
	return s.rotateSession(ctx, oldID, oldEmail, email, "")
}

// PromoteSession rotates a session and upgrades it after successful MFA
// enrollment. The upgrade is persisted on the new bearer session.
func (s *AdminAuthService) PromoteSession(ctx context.Context, oldID, email string) (string, error) {
	return s.rotateSession(ctx, oldID, email, email, "full")
}

func (s *AdminAuthService) rotateSession(ctx context.Context, oldID, oldEmail, email, assurance string) (string, error) {
	state, err := s.SessionState(ctx, oldID, oldEmail)
	if err != nil {
		return "", err
	}
	newID := uuid.NewString()
	if assurance == "" {
		assurance = state.Assurance
	}
	if _, err := s.store.ExecContext(ctx, `INSERT INTO admin_sessions (id, admin_email, expires_at, assurance, reauthenticated_at) VALUES ($1, $2, $3, $4, $5)`, newID, email, state.ExpiresAt, assurance, state.ReauthenticatedAt); err != nil {
		return "", err
	}
	if _, err := s.store.ExecContext(ctx, `DELETE FROM admin_sessions WHERE id = $1`, oldID); err != nil {
		return "", err
	}
	return newID, nil
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
	_, err := s.store.ExecContext(ctx, `UPDATE admin_sessions SET last_activity = NOW() WHERE id = $1 AND last_activity < NOW() - INTERVAL '60 seconds'`, id)
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

// PurgeExpiredSessions removes administrator sessions that have been expired
// long enough to no longer be useful for operational investigation.
func (s *AdminAuthService) PurgeExpiredSessions(ctx context.Context) (int64, error) {
	result, err := s.store.ExecContext(ctx, `
		DELETE FROM admin_sessions
		WHERE expires_at < NOW() - INTERVAL '1 day'
	`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
