package inference

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/rolecatalog"

	"github.com/vrooli/ai-provider-go/policy"
)

// ErrRoleForbidsSampling reports a caller-supplied sampling control sent to a
// role that has not declared itself overridable. It is a request defect, not
// provider incapacity, and the two get different error codes: the caller asked
// for something this role forbids, which no amount of re-routing would fix.
var ErrRoleForbidsSampling = errors.New("role does not admit caller-supplied sampling")

// ErrUnsupportedSampling reports that no remaining candidate's provider can
// honour a caller-supplied control. A caller-explicit control is a promise the
// gateway keeps or refuses; it is never silently downgraded, matching how a
// rejected schema is never degraded to unconstrained generation.
var ErrUnsupportedSampling = errors.New("no candidate can honor the requested sampling")

// ErrContextOverflow is returned before provider dispatch when provider policy
// declares a window that the assembled prompt and output cap cannot fit.
var ErrContextOverflow = errors.New("resolved model context window would be exceeded")

type ResourceRepository struct {
	catalog       rolecatalog.RoleCatalog
	adapters      map[string]providers.Adapter
	ledger        AttemptLedger
	rightsContext func(context.Context) (policy.RightsContext, error)
	consent       RemoteConsentVerifier

	// embedCache holds recent candidate authorizations for embedding requests;
	// authorizationTTL and clock exist so tests can control it.
	embedMu          sync.Mutex
	embedCache       map[string]embeddingCacheEntry
	authorizationTTL *time.Duration
	clock            func() time.Time
}

// RemoteConsentVerifier is the Gateway-owned, revocation-aware consent
// authority. It is consulted at candidate eligibility and again before dispatch.
type RemoteConsentVerifier interface {
	Verify(context.Context, string, string) error
}

// LookupCodeRole returns only a code role declared in this repository's
// validated inference catalog. The provider-neutral service uses this to
// dispatch bounded deterministic handlers without constructing a provider.
func (r *ResourceRepository) LookupCodeRole(role string) (rolecatalog.CodeRoleContract, bool) {
	if r == nil {
		return rolecatalog.CodeRoleContract{}, false
	}
	contract, ok := r.catalog.CodeRoles[strings.TrimSpace(role)]
	return contract, ok
}

func NewResourceRepository(catalog rolecatalog.RoleCatalog, adapters []providers.Adapter, ledgers ...AttemptLedger) (*ResourceRepository, error) {
	if err := catalog.Validate(); err != nil {
		return nil, err
	}
	repository := &ResourceRepository{catalog: catalog, adapters: make(map[string]providers.Adapter, len(adapters))}
	if len(ledgers) > 0 {
		repository.ledger = ledgers[0]
	}
	for _, adapter := range adapters {
		repository.adapters[strings.ToLower(strings.TrimSpace(adapter.Provider))] = adapter
	}
	return repository, nil
}

// NewResourceRepositoryWithRightsContextSource composes the typed route with
// the same operator-owned context source used by the general Gateway router.
// Local model candidates fail closed if the source is absent or unreadable.
func NewResourceRepositoryWithRightsContextSource(catalog rolecatalog.RoleCatalog, adapters []providers.Adapter, source func(context.Context) (policy.RightsContext, error), ledgers ...AttemptLedger) (*ResourceRepository, error) {
	repository, err := NewResourceRepository(catalog, adapters, ledgers...)
	if err != nil {
		return nil, err
	}
	repository.rightsContext = source
	return repository, nil
}

func (r *ResourceRepository) SetRemoteConsentVerifier(verifier RemoteConsentVerifier) {
	if r != nil {
		r.consent = verifier
	}
}
