package accounts

import (
	"context"
	"errors"
	"sort"
	"strings"

	"scenario-authenticator/internal/authorization"
)

// GrantScope assigns a scope to the authenticated principal, or to another
// principal when the caller carries the realm's explicit admin role.
func (s *Service) GrantScope(ctx context.Context, accessToken, principalID, scope string, meta RequestMeta) ([]string, error) {
	vt, err := s.authorizedPrincipal(ctx, accessToken, principalID)
	if err != nil {
		return nil, err
	}
	if s.authorization == nil {
		return nil, errors.New("authorization service unavailable")
	}
	return s.authorization.Grant(ctx, vt.PrincipalID, scope, authorization.Meta{
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
	return vt.PrincipalID, nil
}

func (s *Service) RevokeScope(ctx context.Context, accessToken, principalID, scope string, meta RequestMeta) ([]string, error) {
	vt, err := s.authorizedPrincipal(ctx, accessToken, principalID)
	if err != nil {
		return nil, err
	}
	if s.authorization == nil {
		return nil, errors.New("authorization service unavailable")
	}
	return s.authorization.Revoke(ctx, vt.PrincipalID, scope, authorization.Meta{
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
	return s.authorization.List(ctx, vt.PrincipalID)
}

// SetRoles replaces a principal's coarse realm roles. Only an administrator
// may target another principal, and the last administrator cannot remove its
// own administrator role.
func (s *Service) SetRoles(ctx context.Context, accessToken, principalID string, roles []string, meta RequestMeta) (Account, error) {
	vt, err := s.authorizedPrincipal(ctx, accessToken, principalID)
	if err != nil {
		return Account{}, err
	}
	if !hasRole(vt.Roles, "admin") {
		return Account{}, ErrInvalidCredentials
	}
	store, ok := s.repo.(RoleStore)
	if !ok {
		return Account{}, errors.New("role administration unavailable")
	}
	normalized := normalizeRoles(roles)
	if len(normalized) == 0 {
		return Account{}, InvalidInputError{Msg: "at least one role is required"}
	}
	current, err := s.repo.FindByID(ctx, vt.PrincipalID)
	if err != nil {
		return Account{}, err
	}
	if current.RealmID != vt.Realm {
		return Account{}, ErrInvalidCredentials
	}
	if hasRole(current.Roles, "admin") && !hasRole(normalized, "admin") {
		count, countErr := store.CountAdministrators(ctx, vt.Realm)
		if countErr != nil {
			return Account{}, countErr
		}
		if count <= 1 {
			return Account{}, ErrLastAdministrator
		}
	}
	updated, err := store.SetRoles(ctx, vt.PrincipalID, normalized)
	if err != nil {
		return Account{}, err
	}
	s.logEvent(ctx, vt.PrincipalID, vt.Realm, "account.roles.updated", meta, true, map[string]any{
		"roles": normalized,
		"actor": vt.UserID,
	})
	return updated, nil
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
	if principalID != vt.UserID && !hasRole(vt.Roles, "admin") {
		return ValidatedToken{}, ErrInvalidCredentials
	}
	vt.PrincipalID = principalID
	return vt, nil
}

func hasRole(roles []string, want string) bool {
	for _, role := range roles {
		if role == want {
			return true
		}
	}
	return false
}

func normalizeRoles(roles []string) []string {
	seen := make(map[string]struct{}, len(roles))
	out := make([]string, 0, len(roles))
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		if _, exists := seen[role]; exists {
			continue
		}
		seen[role] = struct{}{}
		out = append(out, role)
	}
	sort.Strings(out)
	return out
}
