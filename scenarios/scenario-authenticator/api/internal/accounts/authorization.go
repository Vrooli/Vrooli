package accounts

import (
	"context"
	"errors"

	"scenario-authenticator/internal/authorization"
)

// GrantScope assigns a scope to the authenticated principal. Until a separate
// administrative policy exists, a principal may manage only its own opaque
// assignments; this prevents a valid ordinary account from granting another
// account capabilities.
func (s *Service) GrantScope(ctx context.Context, accessToken, principalID, scope string, meta RequestMeta) ([]string, error) {
	vt, err := s.authorizedPrincipal(ctx, accessToken, principalID)
	if err != nil {
		return nil, err
	}
	if s.authorization == nil {
		return nil, errors.New("authorization service unavailable")
	}
	return s.authorization.Grant(ctx, vt.UserID, scope, authorization.Meta{
		RealmID: vt.Realm, IPAddress: meta.IP, UserAgent: meta.UserAgent,
	})
}

// PrincipalID resolves the target used by the scope RPCs, applying the
// authenticated-self default without exposing token claims to transport code.
func (s *Service) PrincipalID(ctx context.Context, accessToken, principalID string) (string, error) {
	vt, err := s.authorizedPrincipal(ctx, accessToken, principalID)
	if err != nil {
		return "", err
	}
	return vt.UserID, nil
}

func (s *Service) RevokeScope(ctx context.Context, accessToken, principalID, scope string, meta RequestMeta) ([]string, error) {
	vt, err := s.authorizedPrincipal(ctx, accessToken, principalID)
	if err != nil {
		return nil, err
	}
	if s.authorization == nil {
		return nil, errors.New("authorization service unavailable")
	}
	return s.authorization.Revoke(ctx, vt.UserID, scope, authorization.Meta{
		RealmID: vt.Realm, IPAddress: meta.IP, UserAgent: meta.UserAgent,
	})
}

func (s *Service) ListScopes(ctx context.Context, accessToken, principalID string) ([]string, error) {
	vt, err := s.authorizedPrincipal(ctx, accessToken, principalID)
	if err != nil {
		return nil, err
	}
	if s.authorization == nil {
		return nil, errors.New("authorization service unavailable")
	}
	return s.authorization.List(ctx, vt.UserID)
}

func (s *Service) authorizedPrincipal(ctx context.Context, accessToken, principalID string) (ValidatedToken, error) {
	vt, ok, err := s.Validate(ctx, accessToken)
	if err != nil {
		return ValidatedToken{}, err
	}
	if !ok {
		return ValidatedToken{}, ErrInvalidCredentials
	}
	if principalID == "" {
		principalID = vt.UserID
	}
	if principalID != vt.UserID {
		return ValidatedToken{}, ErrInvalidCredentials
	}
	return vt, nil
}
