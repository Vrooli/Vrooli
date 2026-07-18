package registry

import (
	"context"
	"strings"
)

// Service is the application-layer surface the registry handlers depend on.
// Owns validation, default substitution, and cross-handler policy. The handler
// stays thin around it: decode → call service → translate errors → apply the
// presence overlay.
type Service interface {
	// Register validates in (name/os/arch required after trim) and persists a
	// new node. Returns ErrInvalidNode on validation failure.
	Register(ctx context.Context, in RegisterInput) (Node, error)

	// List returns every node, newest-first.
	List(ctx context.Context) ([]Node, error)

	// Get is a thin pass-through to Repository.Get; ErrNodeNotFound propagates.
	Get(ctx context.Context, id string) (Node, error)

	GetByPairingCorrelation(ctx context.Context, correlationID string) (Node, error)

	// Update validates (id + name required) and persists the editable surface.
	Update(ctx context.Context, in UpdateInput) (Node, error)

	// Revoke severs a node (idempotent). Returns ErrNodeNotFound when unknown.
	Revoke(ctx context.Context, id string) (Node, error)
}

type service struct {
	repo Repository
}

// NewService constructs the production Service.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) Register(ctx context.Context, in RegisterInput) (Node, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Node{}, ErrInvalidNode{Field: "name", Reason: "required"}
	}
	os := strings.TrimSpace(in.OS)
	if os == "" {
		return Node{}, ErrInvalidNode{Field: "os", Reason: "required"}
	}
	arch := strings.TrimSpace(in.Arch)
	if arch == "" {
		return Node{}, ErrInvalidNode{Field: "arch", Reason: "required"}
	}
	return s.repo.Create(ctx, Node{
		Name:                 name,
		OS:                   os,
		Arch:                 arch,
		Endpoint:             strings.TrimSpace(in.Endpoint),
		Capabilities:         trimAll(in.Capabilities),
		Scopes:               trimAll(in.Scopes),
		PairingCorrelationID: strings.TrimSpace(in.PairingCorrelationID),
	})
}

func (s *service) List(ctx context.Context) ([]Node, error) {
	return s.repo.List(ctx)
}

func (s *service) Get(ctx context.Context, id string) (Node, error) {
	return s.repo.Get(ctx, id)
}

func (s *service) GetByPairingCorrelation(ctx context.Context, correlationID string) (Node, error) {
	return s.repo.GetByPairingCorrelation(ctx, strings.TrimSpace(correlationID))
}

func (s *service) Update(ctx context.Context, in UpdateInput) (Node, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return Node{}, ErrInvalidNode{Field: "id", Reason: "required"}
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Node{}, ErrInvalidNode{Field: "name", Reason: "required"}
	}
	return s.repo.Update(ctx, Node{
		ID:           id,
		Name:         name,
		Endpoint:     strings.TrimSpace(in.Endpoint),
		Capabilities: trimAll(in.Capabilities),
		Scopes:       trimAll(in.Scopes),
		Revision:     strings.TrimSpace(in.Revision),
	})
}

func (s *service) Revoke(ctx context.Context, id string) (Node, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Node{}, ErrInvalidNode{Field: "id", Reason: "required"}
	}
	return s.repo.Revoke(ctx, id)
}

// trimAll trims each element and drops empties, normalising the
// capability/scope lists so storage and comparison are stable.
func trimAll(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, v := range in {
		if t := strings.TrimSpace(v); t != "" {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
