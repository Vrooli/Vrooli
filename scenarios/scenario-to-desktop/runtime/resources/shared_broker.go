package resources

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

// SharedServiceBinding is an ephemeral, app-scoped connection handed to the
// desktop runtime after the broker has verified the shared instance. Its
// environment is intentionally not exposed through status or telemetry.
type SharedServiceBinding struct {
	Endpoint    string
	Environment map[string]string
	ExpiresAt   time.Time
	// Provider is an internal, metadata-only label used to prove which
	// already-running authority won provider selection. It is never copied into
	// a service environment or returned with credential material.
	Provider string
}

// SharedServiceResolver is supplied by an embedding application only after a
// user has explicitly consented to shared-resource reuse. A nil resolver keeps
// the bundle fully private.
type SharedServiceResolver interface {
	ResolveSharedService(context.Context, Item) (SharedServiceBinding, error)
}

// SharedProviderTier is the precedence of an already-running provider for a
// bundled resource. A lower-ranked provider is considered only when every
// higher-ranked provider is unavailable or rejects this resource.
type SharedProviderTier string

const (
	// SharedProviderTierLocalVrooli is the locally running Tier-1 control plane.
	SharedProviderTierLocalVrooli SharedProviderTier = "tier1-local-vrooli"
	// SharedProviderTierDesktopPeer is another running desktop application's
	// authenticated broker on this host.
	SharedProviderTierDesktopPeer SharedProviderTier = "tier2-desktop-peer"
)

// SharedServiceCandidate pairs one authenticated resolver with its provider
// tier. The resolver owns discovery, authentication, and user consent; this
// type only makes the precedence explicit and testable.
type SharedServiceCandidate struct {
	Tier     SharedProviderTier
	Resolver SharedServiceResolver
}

// PrioritySharedServiceResolver composes the two external provider classes
// into the required selection order. It never starts, stops, or discovers a
// process itself. If all candidates fail, the service supervisor starts the
// verified private bundle artifact.
type PrioritySharedServiceResolver struct {
	candidates []SharedServiceCandidate
}

// NewPrioritySharedServiceResolver validates and orders provider candidates.
// Exactly one candidate per provider tier is accepted so a caller cannot
// accidentally make peer selection nondeterministic.
func NewPrioritySharedServiceResolver(candidates ...SharedServiceCandidate) (*PrioritySharedServiceResolver, error) {
	ordered := make([]SharedServiceCandidate, 0, len(candidates))
	seen := make(map[SharedProviderTier]bool, len(candidates))
	for _, candidate := range candidates {
		if candidate.Tier != SharedProviderTierLocalVrooli && candidate.Tier != SharedProviderTierDesktopPeer {
			return nil, fmt.Errorf("unknown shared provider tier %q", candidate.Tier)
		}
		if candidate.Resolver == nil {
			return nil, fmt.Errorf("shared provider resolver is required for %s", candidate.Tier)
		}
		if seen[candidate.Tier] {
			return nil, fmt.Errorf("duplicate shared provider tier %s", candidate.Tier)
		}
		seen[candidate.Tier] = true
		ordered = append(ordered, candidate)
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		return sharedProviderRank(ordered[i].Tier) < sharedProviderRank(ordered[j].Tier)
	})
	return &PrioritySharedServiceResolver{candidates: ordered}, nil
}

func sharedProviderRank(tier SharedProviderTier) int {
	if tier == SharedProviderTierLocalVrooli {
		return 0
	}
	return 1
}

// ResolveSharedService tries providers in the documented precedence order.
// A provider failure is an availability result, not a reason to prevent the
// private bundle from starting; the supervisor owns that final fallback.
func (r *PrioritySharedServiceResolver) ResolveSharedService(ctx context.Context, item Item) (SharedServiceBinding, error) {
	if r == nil || len(r.candidates) == 0 {
		return SharedServiceBinding{}, fmt.Errorf("no shared service providers are configured")
	}
	failures := make([]string, 0, len(r.candidates))
	for _, candidate := range r.candidates {
		binding, err := candidate.Resolver.ResolveSharedService(ctx, item)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", candidate.Tier, err))
			continue
		}
		if strings.TrimSpace(binding.Endpoint) == "" || binding.ExpiresAt.IsZero() || !binding.ExpiresAt.After(time.Now()) {
			failures = append(failures, fmt.Sprintf("%s: invalid or expired binding", candidate.Tier))
			continue
		}
		if strings.TrimSpace(binding.Provider) == "" {
			binding.Provider = string(candidate.Tier)
		}
		return binding, nil
	}
	return SharedServiceBinding{}, fmt.Errorf("shared service unavailable for %s: %s", item.Resource, strings.Join(failures, "; "))
}

// SharedBrokerGrant is the narrow result a control-plane broker adapter may
// return after it has acquired and authorized a scoped use lease.
type SharedBrokerGrant struct {
	Endpoint   string
	Credential string
	ExpiresAt  time.Time
}

// SharedBrokerClient is implemented by the Vrooli control-plane adapter. The
// desktop runtime deliberately cannot pass a raw endpoint or choose a scope;
// both remain bound to the authenticated broker credential held by that
// adapter.
type SharedBrokerClient interface {
	GrantSharedService(context.Context, string, time.Duration) (SharedBrokerGrant, error)
}

// BrokerSharedServiceConfig binds one resource to an already scoped broker
// client. Environment values may reference ${RESOURCE_ENDPOINT} and
// ${RESOURCE_CREDENTIAL}; the latter is never persisted or reported.
type BrokerSharedServiceConfig struct {
	Client      SharedBrokerClient
	Environment map[string]string
}

// BrokerSharedServiceResolver obtains a use lease and resource-native scoped
// credential from the loopback Vrooli broker. It has no lifecycle authority.
type BrokerSharedServiceResolver struct {
	Resources            map[string]BrokerSharedServiceConfig
	LeaseTTL             time.Duration
	SharedReuseConsented bool
}

func (r BrokerSharedServiceResolver) ResolveSharedService(ctx context.Context, item Item) (SharedServiceBinding, error) {
	if item.Service == nil {
		return SharedServiceBinding{}, fmt.Errorf("resource %s has no managed service declaration", item.Resource)
	}
	if _, err := item.Service.ProviderPolicy.ResolveProvider(resourcedeployment.ProviderRequest{
		Target: resourcedeployment.ProviderTargetDesktopBundle, Mode: resourcedeployment.ProviderManagedShared, SharedConsented: r.SharedReuseConsented,
	}); err != nil {
		return SharedServiceBinding{}, err
	}
	configuration, ok := r.Resources[item.Resource]
	if !ok {
		return SharedServiceBinding{}, fmt.Errorf("no shared broker configuration for %s", item.Resource)
	}
	if configuration.Client == nil {
		return SharedServiceBinding{}, fmt.Errorf("shared broker client is required for %s", item.Resource)
	}
	ttl := r.LeaseTTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	grant, err := configuration.Client.GrantSharedService(ctx, item.Resource, ttl)
	if err != nil {
		return SharedServiceBinding{}, fmt.Errorf("obtain shared %s broker grant: %w", item.Resource, err)
	}
	if strings.TrimSpace(grant.Endpoint) == "" || strings.TrimSpace(grant.Credential) == "" || grant.ExpiresAt.IsZero() || !grant.ExpiresAt.After(time.Now()) {
		return SharedServiceBinding{}, fmt.Errorf("broker returned an invalid scoped %s grant", item.Resource)
	}
	environment := make(map[string]string, len(configuration.Environment))
	for key, value := range configuration.Environment {
		key = strings.TrimSpace(key)
		if key == "" {
			return SharedServiceBinding{}, fmt.Errorf("shared %s environment contains an empty key", item.Resource)
		}
		value = strings.ReplaceAll(value, "${RESOURCE_ENDPOINT}", grant.Endpoint)
		value = strings.ReplaceAll(value, "${RESOURCE_CREDENTIAL}", grant.Credential)
		environment[key] = value
	}
	return SharedServiceBinding{Endpoint: grant.Endpoint, Environment: environment, ExpiresAt: grant.ExpiresAt}, nil
}
