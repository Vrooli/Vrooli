package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/vrooli/api-core/scopecatalog"
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
	Remove(ctx context.Context, id string) error
}

type service struct {
	repo          Repository
	validateGrant func([]string) error
}

type Option func(*service)

// WithGrantValidator supplies the catalog-derived write-time authority gate.
func WithGrantValidator(validate func([]string) error) Option {
	return func(s *service) { s.validateGrant = validate }
}

// NewService constructs the production Service. Non-empty grants fail closed
// unless the caller supplies the repository catalog validator.
func NewService(repo Repository, opts ...Option) Service {
	s := &service{repo: repo}
	for _, opt := range opts {
		opt(s)
	}
	return s
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
	kind := strings.TrimSpace(in.Kind)
	if kind == "" {
		kind = KindAgent
	}
	if !ValidKind(kind) {
		return Node{}, ErrInvalidNode{Field: "kind", Reason: "must be agent, ssh, attached, or control_plane"}
	}
	if err := s.validateScopes(in.Scopes); err != nil {
		return Node{}, err
	}
	return s.repo.Create(ctx, Node{
		Name:                 name,
		Kind:                 kind,
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
	if err := s.validateScopes(in.Scopes); err != nil {
		return Node{}, err
	}
	kind := strings.TrimSpace(in.Kind)
	if kind == "" {
		current, err := s.repo.Get(ctx, id)
		if err != nil {
			return Node{}, err
		}
		kind = current.Kind
	}
	if !ValidKind(kind) {
		return Node{}, ErrInvalidNode{Field: "kind", Reason: "must be agent, ssh, attached, or control_plane"}
	}
	return s.repo.Update(ctx, Node{
		ID:           id,
		Name:         name,
		Endpoint:     strings.TrimSpace(in.Endpoint),
		Capabilities: trimAll(in.Capabilities),
		Scopes:       trimAll(in.Scopes),
		Revision:     strings.TrimSpace(in.Revision),
		Kind:         kind,
	})
}

func (s *service) validateScopes(scopes []string) error {
	if len(scopes) == 0 {
		return nil
	}
	if s.validateGrant == nil {
		return ErrInvalidGrant{Scope: scopes[0], Reason: "catalog grant validator is not configured"}
	}
	return s.validateGrant(scopes)
}

// NewCatalogGrantValidator creates the one write-time vocabulary gate used by
// registry and pairing. Catalog scopes and their two wildcard forms are valid;
// Bridge's session transport capability is the sole non-command grant.
func NewCatalogGrantValidator(catalog scopecatalog.Catalog) func([]string) error {
	namespaces := make(map[string]struct{})
	for _, scope := range catalog.Scopes {
		namespaces[scope.Scenario] = struct{}{}
	}
	for _, omitted := range catalog.OmittedResolutions {
		namespaces[omitted.Scenario] = struct{}{}
	}
	return func(scopes []string) error {
		for _, scope := range scopes {
			if err := validateGrant(scope, catalog, namespaces); err != nil {
				return err
			}
		}
		return nil
	}
}

func validateGrant(scope string, catalog scopecatalog.Catalog, namespaces map[string]struct{}) error {
	if scope == "" {
		return ErrInvalidGrant{Scope: scope, Reason: "empty grants are not allowed"}
	}
	if scope != strings.TrimSpace(scope) {
		return ErrInvalidGrant{Scope: scope, Reason: "leading or trailing whitespace is not allowed"}
	}
	if strings.Contains(scope, " ") {
		return ErrInvalidGrant{Scope: scope, Reason: "command-named grants are not allowed; use <namespace>:<effect>"}
	}
	if scope == "*" || scope == "vrooli-bridge:session" || catalog.HasScope(scope) {
		return nil
	}
	namespace, effect, ok := strings.Cut(scope, ":")
	if !ok || namespace == "" || effect == "" || strings.Contains(effect, ":") {
		return ErrInvalidGrant{Scope: scope, Reason: "must be *, <namespace>:<effect>, <namespace>:*, or *:<effect>"}
	}
	if effect == "*" {
		if _, exists := namespaces[namespace]; exists && namespace != "*" {
			return nil
		}
		return ErrInvalidGrant{Scope: scope, Reason: fmt.Sprintf("namespace %q does not exist in the derived catalog", namespace)}
	}
	if namespace == "*" && validEffect(effect) {
		return nil
	}
	return ErrInvalidGrant{Scope: scope, Reason: "scope is not present in the derived catalog"}
}

func validEffect(effect string) bool {
	return effect == string(scopecatalog.EffectRead) || effect == string(scopecatalog.EffectWrite) || effect == string(scopecatalog.EffectDestructive)
}

func (s *service) Revoke(ctx context.Context, id string) (Node, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Node{}, ErrInvalidNode{Field: "id", Reason: "required"}
	}
	return s.repo.Revoke(ctx, id)
}

func (s *service) Remove(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrInvalidNode{Field: "id", Reason: "required"}
	}
	return s.repo.Remove(ctx, id)
}

// trimAll trims each element and drops empties. Scope inputs have already
// passed the strict non-normalizing validator before they reach this helper.
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
