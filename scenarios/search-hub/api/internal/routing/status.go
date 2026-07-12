package routing

import (
	"context"
	"fmt"

	internalregistry "search-hub/internal/registry"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

// Status reports federation health (Phase 7): per-provider reachability plus
// whether the classifier and reranker models are available. It lists the ACTIVE
// leaves (capability_gap stubs carry no endpoint and are excluded), resolves
// each leaf's scenario URL to gauge reachability, and probes the optional
// Classifier/Reranker Available seams.
//
// It never fails on an individual provider — an unresolvable leaf is reported
// as unreachable+degraded rather than erroring the whole call. It returns an
// error only on a registry read failure.
//
// Freshness is currently a reachability note: the descriptor's optional
// status_endpoint probe (last sync / point count) is a deeper follow-up; the
// seams wired today are URL-resolution + model availability.
func (r *Router) Status(ctx context.Context) (*routingv1.StatusResponse, error) {
	active, err := r.deps.Lister.List(ctx, internalregistry.ListFilter{
		State: int32(registryv1.ProviderState_PROVIDER_STATE_ACTIVE),
	})
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}

	resp := &routingv1.StatusResponse{}
	for _, p := range active {
		resp.Providers = append(resp.Providers, r.providerHealth(ctx, p))
	}
	resp.ClassifierAvailable = r.deps.Classifier != nil && r.deps.Classifier.Available(ctx)
	resp.RerankerAvailable = r.deps.Reranker != nil && r.deps.Reranker.Available(ctx)
	return resp, nil
}

// providerHealth resolves one leaf's reachability. A leaf with no http_json
// endpoint, or whose scenario URL won't resolve, is reported unreachable and
// degraded with a human note in `freshness`.
func (r *Router) providerHealth(ctx context.Context, p *registryv1.ProviderDescriptor) *routingv1.ProviderHealth {
	h := &routingv1.ProviderHealth{ProviderId: p.GetProviderId()}
	if open, note := r.providerBreakers.status(p.GetProviderId(), r.deps.Now()); open {
		h.Reachable = false
		h.Degraded = true
		h.Freshness = note
		return h
	}

	hj := p.GetEndpoint().GetHttpJson()
	if hj == nil {
		h.Reachable = false
		h.Degraded = true
		h.Freshness = "no http endpoint registered"
		return h
	}

	if _, err := r.deps.Resolver.ResolveScenarioURL(ctx, hj.GetScenarioId()); err != nil {
		h.Reachable = false
		h.Degraded = true
		h.Freshness = fmt.Sprintf("scenario %q unreachable: %s", hj.GetScenarioId(), oneLine(err.Error()))
		return h
	}

	h.Reachable = true
	h.Freshness = "endpoint resolved"
	return h
}
