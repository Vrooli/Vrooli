package accounts

import (
	"context"
	"errors"
	"strings"

	"scenario-authenticator/internal/authcrypto"
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
	newRefresh, userID, aud, err := s.sessions.RotateRefresh(ctx, strings.TrimSpace(refreshToken))
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
	if aud == "" {
		aud, err = s.resolveAudience(ctx, acc.RealmID, "")
		if err != nil {
			return AuthResult{}, err
		}
	}
	scopes, err := s.scopes(ctx, acc)
	if err != nil {
		return AuthResult{}, err
	}
	familyID, _, err := s.sessions.RefreshFamilyForToken(ctx, newRefresh)
	if err != nil {
		return AuthResult{}, ErrRefreshRejected
	}
	session, found, err := s.sessions.SessionForRefreshFamily(ctx, acc.ID, familyID)
	if err != nil {
		return AuthResult{}, err
	}
	if !found {
		session.ID, err = s.sessions.StoreSessionForFamily(ctx, acc.ID, meta.IP, meta.UserAgent, familyID)
		if err != nil {
			return AuthResult{}, err
		}
	}
	authVersion, err := s.sessions.CurrentAuthVersion(ctx, acc.ID)
	if err != nil {
		return AuthResult{}, err
	}
	access, err := s.signer.Sign(authcrypto.TokenInput{
		UserID: acc.ID, Email: acc.Email, Roles: acc.Roles, Scopes: scopes, Audience: aud,
		SessionID: session.ID, AuthVersion: authVersion,
	})
	if err != nil {
		return AuthResult{}, err
	}
	s.logEvent(ctx, acc.ID, acc.RealmID, "token.refreshed", meta, true, nil)
	acc.Scopes = scopes
	return AuthResult{
		Account: acc, AccessToken: access, RefreshToken: newRefresh,
		AccessExpiresAt: s.clock.Now().Add(s.signer.Expiry()), Audience: aud,
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
	validated, ok, err := s.Validate(ctx, accessToken)
	if err != nil {
		return err
	}
	if !ok {
		return nil // already invalid — idempotent no-op
	}
	if !validated.ExpiresAt.IsZero() {
		if err := s.sessions.BlacklistAccess(ctx, accessToken, validated.ExpiresAt); err != nil {
			return err
		}
	}
	if _, err := s.sessions.RevokeAllSessions(ctx, validated.UserID); err != nil {
		return err
	}
	s.logEvent(ctx, validated.UserID, validated.Realm, "user.logged_out", meta, true, nil)
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

// RevokeAuthorizedSession is the user-facing single-session operation. The
// caller may revoke its own session, or any session when it carries admin.
// Missing sessions remain idempotent after the caller is authenticated.
func (s *Service) RevokeAuthorizedSession(ctx context.Context, accessToken, sessionID string, meta RequestMeta) error {
	vt, ok, err := s.Validate(ctx, accessToken)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidCredentials
	}
	target, found, err := s.sessions.FindSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if found && target.UserID != vt.UserID && !hasRole(vt.Roles, "admin") {
		return ErrInvalidCredentials
	}
	if err := s.sessions.RevokeSession(ctx, sessionID); err != nil {
		return err
	}
	s.logEvent(ctx, vt.UserID, vt.Realm, "session.revoked", meta, true, map[string]any{
		"session_id": sessionID,
		"target_user_id": func() string {
			if found {
				return target.UserID
			}
			return vt.UserID
		}(),
	})
	return nil
}

// RevokeAllSessions revokes every session for the access token's owner.
func (s *Service) RevokeAllSessions(ctx context.Context, accessToken string, meta RequestMeta) (int, error) {
	vt, ok, err := s.Validate(ctx, accessToken)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, ErrInvalidCredentials
	}
	count, err := s.sessions.RevokeAllSessions(ctx, vt.UserID)
	if err != nil {
		return 0, err
	}
	s.logEvent(ctx, vt.UserID, vt.Realm, "sessions.revoked_all", meta, true, map[string]any{"revoked_sessions": count})
	return count, nil
}
