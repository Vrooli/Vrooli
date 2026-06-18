package accounts

import (
	"context"
	"errors"
	"strings"

	"scenario-authenticator/internal/authcrypto"
	"scenario-authenticator/internal/realm"
	"scenario-authenticator/internal/sessions"
)

// ErrRefreshRejected — the presented refresh token was invalid or a detected
// replay. Both surface as UNAUTHENTICATED; the replay case additionally revoked
// the token family and wrote an audit event.
var ErrRefreshRejected = errors.New("refresh token rejected")

// Refresh rotates a refresh token single-use and mints a new access token.
// Replaying an already-rotated token revokes the whole family (reuse detection)
// and audits it.
func (s *Service) Refresh(ctx context.Context, refreshToken string, meta RequestMeta) (AuthResult, error) {
	newRefresh, userID, err := s.sessions.RotateRefresh(ctx, strings.TrimSpace(refreshToken))
	if err != nil {
		if errors.Is(err, sessions.ErrRefreshReuse) {
			s.logEvent(ctx, "", "", "token.refresh.reuse", meta, false,
				map[string]any{"reason": "refresh_token_reuse_family_revoked"})
		}
		return AuthResult{}, ErrRefreshRejected
	}

	acc, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return AuthResult{}, ErrRefreshRejected
	}
	aud, err := s.repo.RealmAudience(ctx, acc.RealmID)
	if err != nil {
		return AuthResult{}, err
	}
	access, err := s.signer.Sign(authcrypto.TokenInput{
		UserID: acc.ID, Email: acc.Email, Roles: acc.Roles, Audience: aud,
	})
	if err != nil {
		return AuthResult{}, err
	}
	if _, err := s.sessions.StoreSession(ctx, acc.ID, meta.IP, meta.UserAgent); err != nil {
		return AuthResult{}, err
	}
	s.logEvent(ctx, acc.ID, acc.RealmID, "token.refreshed", meta, true, nil)
	return AuthResult{
		Account: acc, AccessToken: access, RefreshToken: newRefresh,
		AccessExpiresAt: s.clock.Now().Add(s.signer.Expiry()),
	}, nil
}

// Logout blacklists the access token until its own expiry and revokes the
// caller's sessions. Idempotent: an invalid/expired token is a no-op success
// (nothing left to revoke).
func (s *Service) Logout(ctx context.Context, accessToken string, meta RequestMeta) error {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil
	}
	aud, err := s.repo.RealmAudience(ctx, realm.DefaultID)
	if err != nil {
		return err
	}
	claims, err := s.signer.Validate(accessToken, aud)
	if err != nil {
		return nil // already invalid — idempotent no-op
	}
	if claims.ExpiresAt != nil {
		if err := s.sessions.BlacklistAccess(ctx, accessToken, claims.ExpiresAt.Time); err != nil {
			return err
		}
	}
	if _, err := s.sessions.RevokeAllSessions(ctx, claims.UserID); err != nil {
		return err
	}
	s.logEvent(ctx, claims.UserID, realm.DefaultID, "user.logged_out", meta, true, nil)
	return nil
}

// ListSessions returns the active sessions for the access token's owner.
func (s *Service) ListSessions(ctx context.Context, accessToken string) ([]sessions.Session, error) {
	vt, ok, err := s.Validate(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInvalidCredentials
	}
	return s.sessions.ListSessions(ctx, vt.UserID)
}

// RevokeSession drops a single session by id. Idempotent and unauthenticated by
// id (the device-sync-hub un-pair contract): a caller holding a session id may
// revoke it.
func (s *Service) RevokeSession(ctx context.Context, sessionID string) error {
	return s.sessions.RevokeSession(ctx, sessionID)
}

// RevokeAllSessions revokes every session for the access token's owner.
func (s *Service) RevokeAllSessions(ctx context.Context, accessToken string) (int, error) {
	vt, ok, err := s.Validate(ctx, accessToken)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, ErrInvalidCredentials
	}
	return s.sessions.RevokeAllSessions(ctx, vt.UserID)
}
