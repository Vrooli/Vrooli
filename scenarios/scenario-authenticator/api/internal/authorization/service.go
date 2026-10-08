// Package authorization owns assignment of opaque scopes to principals. It
// does not derive, validate, or enforce scope meaning; relying parties own
// those decisions.
package authorization

import (
	"context"
	"errors"
	"strings"

	"scenario-authenticator/internal/audit"
)

var (
	ErrInvalidScope      = errors.New("scope must not be empty")
	ErrPrincipalNotFound = errors.New("principal not found")
)

// ScopeStore is the persistence seam for opaque assignments.
type ScopeStore interface {
	GrantScope(ctx context.Context, principalID, scope string) ([]string, error)
	RevokeScope(ctx context.Context, principalID, scope string) ([]string, error)
	ListScopes(ctx context.Context, principalID string) ([]string, error)
}

// Meta carries request context for the audit record without coupling this
// domain to the accounts package.
type Meta struct {
	RealmID   string
	IPAddress string
	UserAgent string
}

// Service assigns opaque scope strings and records each effective mutation.
type Service struct {
	store ScopeStore
	audit audit.Logger
}

func NewService(store ScopeStore, logger audit.Logger) *Service {
	return &Service{store: store, audit: logger}
}

func (s *Service) Grant(ctx context.Context, principalID, scope string, meta Meta) ([]string, error) {
	if strings.TrimSpace(scope) == "" {
		return nil, ErrInvalidScope
	}
	scopes, err := s.store.GrantScope(ctx, principalID, scope)
	if err != nil {
		return nil, err
	}
	s.log(ctx, principalID, meta, "scope.granted", scope)
	return scopes, nil
}

func (s *Service) Revoke(ctx context.Context, principalID, scope string, meta Meta) ([]string, error) {
	if strings.TrimSpace(scope) == "" {
		return nil, ErrInvalidScope
	}
	scopes, err := s.store.RevokeScope(ctx, principalID, scope)
	if err != nil {
		return nil, err
	}
	s.log(ctx, principalID, meta, "scope.revoked", scope)
	return scopes, nil
}

func (s *Service) List(ctx context.Context, principalID string) ([]string, error) {
	return s.store.ListScopes(ctx, principalID)
}

func (s *Service) log(ctx context.Context, principalID string, meta Meta, action, scope string) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Log(ctx, audit.Event{
		UserID: principalID, RealmID: meta.RealmID, Action: action,
		IPAddress: meta.IPAddress, UserAgent: meta.UserAgent, Success: true,
		Metadata: map[string]any{"scope": scope},
	})
}
