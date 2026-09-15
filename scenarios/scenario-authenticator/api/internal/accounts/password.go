package accounts

import (
	"context"
	"errors"
	"strings"
)

// ChangePassword verifies the authenticated account's current credential,
// stores a fresh Argon2id hash, revokes all live sessions, and records the
// security event. Password failures intentionally use the same credential
// error as login so the handler does not disclose account state.
func (s *Service) ChangePassword(ctx context.Context, accessToken, currentPassword, newPassword string, meta RequestMeta) (int, error) {
	vt, ok, err := s.Validate(ctx, accessToken)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, ErrInvalidCredentials
	}
	_, hash, err := s.repo.FindByEmail(ctx, vt.Realm, strings.TrimSpace(vt.Email))
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			return 0, ErrInvalidCredentials
		}
		return 0, err
	}
	valid, verifyErr := VerifyPassword(currentPassword, hash)
	if verifyErr != nil || !valid {
		s.logEvent(ctx, vt.UserID, vt.Realm, "user.password.change.failed", meta, false, map[string]any{"reason": "invalid_current_password"})
		return 0, ErrInvalidCredentials
	}
	if ok, msg := ValidatePassword(newPassword); !ok {
		return 0, InvalidInputError{Msg: msg}
	}
	newHash, err := HashPassword(newPassword)
	if err != nil {
		return 0, err
	}
	if err := s.repo.UpdatePasswordHash(ctx, vt.UserID, newHash); err != nil {
		return 0, err
	}
	revoked, err := s.sessions.RevokeAllSessions(ctx, vt.UserID)
	if err != nil {
		return 0, err
	}
	s.logEvent(ctx, vt.UserID, vt.Realm, "user.password.changed", meta, true, map[string]any{"revoked_sessions": revoked})
	return revoked, nil
}
