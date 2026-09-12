package development

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

var (
	evidenceResolverPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:\.[a-z][a-z0-9]*)+$`)
	evidenceSchemaPattern   = regexp.MustCompile(`^v[0-9]+$`)
	evidenceCohortPattern   = regexp.MustCompile(`^[a-z0-9]+(?:[-_][a-z0-9]+)*$`)
)

var ErrEvidenceResolverUnavailable = fmt.Errorf("evidence resolver unavailable")

// EvidenceContract is encoded in the existing outcome evidence_source field
// as resolver@schema/cohort. It names the owner adapter, receipt schema, and
// exact tested cohort required for acceptance.
type EvidenceContract struct {
	ResolverID string
	Schema     string
	Cohort     string
}

func ParseEvidenceContract(raw string) (EvidenceContract, error) {
	parts := strings.Split(strings.TrimSpace(raw), "/")
	if len(parts) != 2 {
		return EvidenceContract{}, fmt.Errorf("evidence source %q must be resolver@schema/cohort: %w", raw, ErrInvalid)
	}
	identity := strings.Split(parts[0], "@")
	if len(identity) != 2 || !evidenceResolverPattern.MatchString(identity[0]) || !evidenceSchemaPattern.MatchString(identity[1]) || !evidenceCohortPattern.MatchString(parts[1]) {
		return EvidenceContract{}, fmt.Errorf("evidence source %q is not a registered resolver contract: %w", raw, ErrInvalid)
	}
	return EvidenceContract{ResolverID: identity[0], Schema: identity[1], Cohort: parts[1]}, nil
}

// ResolverRegistry is the owner-side registry for versioned evidence
// resolvers. A source contract selects the resolver by resolver@schema and
// independently binds the exact cohort in the receipt.
type ResolverRegistry struct {
	mu        sync.RWMutex
	resolvers map[string]EvidenceResolver
	scenarios map[string]string
}

func NewResolverRegistry() *ResolverRegistry {
	return &ResolverRegistry{resolvers: map[string]EvidenceResolver{}, scenarios: map[string]string{}}
}

func (r *ResolverRegistry) Register(identity string, resolver EvidenceResolver) error {
	identity = strings.TrimSpace(identity)
	if resolver == nil || !strings.Contains(identity, "@") {
		return fmt.Errorf("resolver identity and implementation are required: %w", ErrInvalid)
	}
	if _, err := ParseEvidenceContract(identity + "/cohort"); err != nil {
		return err
	}
	if r == nil {
		return fmt.Errorf("resolver registry is unavailable: %w", ErrInvalid)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.resolvers == nil {
		r.resolvers = map[string]EvidenceResolver{}
	}
	if r.scenarios == nil {
		r.scenarios = map[string]string{}
	}
	if _, exists := r.resolvers[identity]; exists {
		return fmt.Errorf("resolver %q is already registered: %w", identity, ErrConflict)
	}
	r.resolvers[identity] = resolver
	return nil
}

// RegisterForScenario binds a resolver identity to the owner of one scenario.
// CurrentRevision can then select by ownership rather than map iteration or
// registration order.
func (r *ResolverRegistry) RegisterForScenario(identity, scenario string, resolver EvidenceResolver) error {
	scenario = strings.TrimSpace(scenario)
	if !slugPattern.MatchString(scenario) {
		return fmt.Errorf("scenario %q is not a canonical slug: %w", scenario, ErrInvalid)
	}
	if err := r.Register(identity, resolver); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.scenarios == nil {
		r.scenarios = map[string]string{}
	}
	if prior := r.scenarios[scenario]; prior != "" && prior != strings.TrimSpace(identity) {
		return fmt.Errorf("scenario %q already has resolver %q: %w", scenario, prior, ErrConflict)
	}
	r.scenarios[scenario] = strings.TrimSpace(identity)
	return nil
}

func (r *ResolverRegistry) Resolve(ctx context.Context, source, receiptID string) (Evidence, error) {
	contract, err := ParseEvidenceContract(source)
	if err != nil {
		return Evidence{}, err
	}
	key := contract.ResolverID + "@" + contract.Schema
	r.mu.RLock()
	resolver := r.resolvers[key]
	r.mu.RUnlock()
	if resolver == nil {
		return Evidence{}, fmt.Errorf("resolver %q is not registered: %w", key, ErrEvidenceResolverUnavailable)
	}
	return resolver.Resolve(ctx, source, receiptID)
}

func (r *ResolverRegistry) CurrentRevision(ctx context.Context, scenario string) (string, error) {
	scenario = strings.TrimSpace(scenario)
	r.mu.RLock()
	if identity := r.scenarios[scenario]; identity != "" {
		resolver := r.resolvers[identity]
		r.mu.RUnlock()
		if resolver == nil {
			return "", ErrEvidenceResolverUnavailable
		}
		return resolver.CurrentRevision(ctx, scenario)
	}
	keys := make([]string, 0, len(r.resolvers))
	for key := range r.resolvers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	resolvers := make([]EvidenceResolver, 0, len(keys))
	for _, key := range keys {
		resolvers = append(resolvers, r.resolvers[key])
	}
	r.mu.RUnlock()
	if len(resolvers) != 1 {
		return "", ErrEvidenceResolverUnavailable
	}
	return resolvers[0].CurrentRevision(ctx, scenario)
}

var _ EvidenceResolver = (*ResolverRegistry)(nil)
