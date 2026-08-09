package identity

import (
	"errors"
	"maps"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrMissingParent     = errors.New("parent identity is required")
	ErrScopeWidening     = errors.New("delegated scope is not held by the parent")
	ErrExpiryWidening    = errors.New("delegated expiry exceeds the parent expiry")
	ErrExpiredDelegation = errors.New("delegated expiry is not in the future")
)

// Attenuate creates the claims for a child run. Every requested capability must
// already be covered by the parent, and the child can never outlive it. The
// returned scope list is explicit: wildcard requests are expanded against the
// parent's already-materialized concrete scope list.
func Attenuate(parent *Claims, childRunID, childTaskID uuid.UUID, requested []string, expiresAt, now time.Time) (*Claims, error) {
	if parent == nil {
		return nil, ErrMissingParent
	}
	if expiresAt.IsZero() {
		expiresAt = time.Unix(parent.ExpiresAt, 0)
	}
	if !expiresAt.After(now) {
		return nil, ErrExpiredDelegation
	}
	parentExpiry := time.Unix(parent.ExpiresAt, 0)
	if expiresAt.After(parentExpiry) {
		return nil, ErrExpiryWidening
	}

	scopes, err := attenuateScopes(parent.Scopes, requested)
	if err != nil {
		return nil, err
	}
	return &Claims{
		RunID:      childRunID,
		TaskID:     childTaskID,
		Subject:    parent.Subject,
		Scopes:     scopes,
		ProfileKey: parent.ProfileKey,
		ScopePath:  parent.ScopePath,
		IssuedAt:   now.Unix(),
		ExpiresAt:  expiresAt.Unix(),
		Meta:       maps.Clone(parent.Meta),
	}, nil
}

func attenuateScopes(parent, requested []string) ([]string, error) {
	if requested == nil {
		return uniqueScopes(parent), nil
	}
	result := make([]string, 0, len(requested))
	seen := map[string]struct{}{}
	for _, raw := range requested {
		scope := strings.TrimSpace(raw)
		if scope == "" {
			continue
		}
		matched := false
		for _, held := range parent {
			held = strings.TrimSpace(held)
			if scopeCovers(held, scope) {
				matched = true
				if strings.HasSuffix(scope, "*") || scope == "*" {
					if strings.HasSuffix(held, "*") || held == "*" {
						return nil, ErrScopeWidening
					}
					if _, ok := seen[held]; !ok {
						seen[held] = struct{}{}
						result = append(result, held)
					}
				} else if _, ok := seen[scope]; !ok {
					seen[scope] = struct{}{}
					result = append(result, scope)
				}
			}
		}
		if !matched {
			return nil, ErrScopeWidening
		}
	}
	return result, nil
}

func scopeCovers(held, requested string) bool {
	if held == "*" || held == requested {
		return true
	}
	if strings.HasSuffix(held, "*") && strings.HasPrefix(requested, strings.TrimSuffix(held, "*")) {
		return true
	}
	return strings.HasSuffix(requested, "*") && strings.HasPrefix(held, strings.TrimSuffix(requested, "*"))
}

func uniqueScopes(scopes []string) []string {
	result := make([]string, 0, len(scopes))
	seen := map[string]struct{}{}
	for _, raw := range scopes {
		scope := strings.TrimSpace(raw)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		result = append(result, scope)
	}
	return result
}
