package routing

import (
	"fmt"
	"sort"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

type queryPartPlan struct {
	id     string
	label  string
	source routingv1.QueryPartEvidence_Source
}

type queryPartState struct {
	plan      queryPartPlan
	providers map[string]struct{}
	hits      map[queryPartHitKey]map[routingv1.SearchHitReference_RetrievalSource]struct{}
	attempted bool
	degraded  bool
	partial   bool
	reason    string
}

type queryPartHitKey struct {
	providerID string
	document   string
}

func planQueryParts(query string, variants []string) []queryPartPlan {
	if len(variants) > 1 {
		parts := make([]queryPartPlan, 0, len(variants))
		for i, variant := range variants {
			parts = append(parts, queryPartPlan{
				id: fmt.Sprintf("part-%d", i+1), label: variant,
				source: routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT,
			})
		}
		return parts
	}
	return []queryPartPlan{{id: "part-1", label: query, source: routingv1.QueryPartEvidence_SOURCE_USER_QUERY}}
}

func newQueryPartStates(query string, variants []string, targets []*registryv1.ProviderDescriptor) []*queryPartState {
	plans := planQueryParts(query, variants)
	states := make([]*queryPartState, 0, len(plans))
	for _, plan := range plans {
		state := &queryPartState{
			plan: plan, providers: make(map[string]struct{}),
			hits: make(map[queryPartHitKey]map[routingv1.SearchHitReference_RetrievalSource]struct{}),
		}
		if len(variants) <= 1 {
			addQueryPartTargets(state, targets)
		}
		states = append(states, state)
	}
	return states
}

func queryPartForVariant(states []*queryPartState, variantIndex, variantCount int) *queryPartState {
	if len(states) == 0 {
		return nil
	}
	if variantCount > 1 && variantIndex >= 0 && variantIndex < len(states) {
		return states[variantIndex]
	}
	return states[0]
}

func addQueryPartTargets(state *queryPartState, targets []*registryv1.ProviderDescriptor) {
	if state == nil {
		return
	}
	for _, target := range targets {
		if target == nil || target.GetProviderId() == "" {
			continue
		}
		state.providers[target.GetProviderId()] = struct{}{}
	}
}

func queryPartProviderIDs(state *queryPartState) []string {
	if state == nil {
		return nil
	}
	ids := make([]string, 0, len(state.providers))
	for id := range state.providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func hasDegradedGroups(groups []*routingv1.ProviderResultGroup) bool {
	for _, group := range groups {
		if group.GetDegraded() {
			return true
		}
	}
	return false
}

func recordQueryPartHits(state *queryPartState, groups []*routingv1.ProviderResultGroup, source routingv1.SearchHitReference_RetrievalSource) {
	if state == nil {
		return
	}
	if state.hits == nil {
		state.hits = make(map[queryPartHitKey]map[routingv1.SearchHitReference_RetrievalSource]struct{})
	}
	for _, group := range groups {
		if group == nil {
			continue
		}
		for index, hit := range group.GetHits() {
			if hit == nil {
				continue
			}
			providerID := hit.GetProviderId()
			if providerID == "" {
				providerID = group.GetProviderId()
			}
			key := queryPartHitKey{providerID: providerID, document: documentKey(hit, index)}
			if state.hits[key] == nil {
				state.hits[key] = make(map[routingv1.SearchHitReference_RetrievalSource]struct{})
			}
			state.hits[key][source] = struct{}{}
		}
	}
}

func queryPartHitSources(state *queryPartState, providerID string, hit *routingv1.SearchHit, index int) []routingv1.SearchHitReference_RetrievalSource {
	if state == nil || hit == nil {
		return nil
	}
	if providerID == "" {
		providerID = hit.GetProviderId()
	}
	sourceSet := state.hits[queryPartHitKey{providerID: providerID, document: documentKey(hit, index)}]
	sources := make([]routingv1.SearchHitReference_RetrievalSource, 0, len(sourceSet))
	for source := range sourceSet {
		sources = append(sources, source)
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i] < sources[j] })
	return sources
}

func queryPartEvidence(resp *routingv1.QueryResponse, states []*queryPartState) []*routingv1.QueryPartEvidence {
	if resp == nil {
		return nil
	}
	parts := make([]*routingv1.QueryPartEvidence, 0, len(states))
	for _, state := range states {
		if state == nil {
			continue
		}
		part := &routingv1.QueryPartEvidence{
			PartId: state.plan.id, Label: state.plan.label, Source: state.plan.source,
			SearchedProviderIds: queryPartProviderIDs(state),
			Degraded:            state.degraded, Partial: state.partial || resp.GetPartial(),
			Reason: state.reason,
		}
		if resp.GetPartial() {
			part.Degraded = true
		}
		if len(resp.GetRanked()) > 0 {
			for rank, hit := range resp.GetRanked() {
				if sources := queryPartHitSources(state, hit.GetProviderId(), hit, rank); len(sources) > 0 {
					part.Evidence = append(part.Evidence, &routingv1.SearchHitReference{
						ProviderId: hit.GetProviderId(), RankedIndex: int32Pointer(int32(rank)),
						HitId: hit.GetId(), Type: hit.GetType(), Path: hit.GetPath(), RetrievalSources: sources,
					})
				}
			}
		} else {
			for groupIndex, group := range resp.GetGroups() {
				for hitIndex, hit := range group.GetHits() {
					providerID := hit.GetProviderId()
					if providerID == "" {
						providerID = group.GetProviderId()
					}
					if sources := queryPartHitSources(state, providerID, hit, hitIndex); len(sources) > 0 {
						part.Evidence = append(part.Evidence, &routingv1.SearchHitReference{
							ProviderId: providerID, GroupIndex: int32Pointer(int32(groupIndex)), HitIndex: int32Pointer(int32(hitIndex)),
							HitId: hit.GetId(), Type: hit.GetType(), Path: hit.GetPath(), RetrievalSources: sources,
						})
					}
				}
			}
		}
		switch {
		case len(part.Evidence) > 0:
			part.Status = routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED
		case state.degraded || resp.GetPartial():
			part.Status = routingv1.QueryPartEvidence_STATUS_PROVIDER_DEGRADED
		case !state.attempted:
			part.Status = routingv1.QueryPartEvidence_STATUS_NOT_ROUTED
		default:
			part.Status = routingv1.QueryPartEvidence_STATUS_SEARCHED_EMPTY
		}
		parts = append(parts, part)
	}
	return parts
}

func searchDegradations(groups []*routingv1.ProviderResultGroup, selectionFallback bool, selectionReason, routingIndexReason string, rerankDegraded bool, rerankReason string, partial bool, pendingProviders int) []*routingv1.SearchDegradation {
	var degradations []*routingv1.SearchDegradation
	if routingIndexReason != "" {
		degradations = append(degradations, &routingv1.SearchDegradation{
			Component: "selector", Reason: routingIndexReason,
			Detail: "provider-description selection used its lexical fallback", FallbackActive: true,
		})
	}
	if selectionFallback {
		degradations = append(degradations, &routingv1.SearchDegradation{
			Component: "selector", Reason: "selector_fallback", Detail: selectionReason, FallbackActive: true, Retryable: true,
		})
	}
	if rerankDegraded {
		reason := rerankReason
		if reason == "" {
			reason = "reranker_unavailable"
		}
		degradations = append(degradations, &routingv1.SearchDegradation{
			Component: "reranker", Reason: reason, Detail: "provider-grouped results remain available", FallbackActive: true, Retryable: true,
		})
	}
	for _, group := range groups {
		if !group.GetDegraded() {
			continue
		}
		if len(group.GetDegradations()) == 0 {
			degradations = append(degradations, &routingv1.SearchDegradation{
				Component: "provider", Reason: "provider_degraded", ProviderId: group.GetProviderId(), Leg: "provider",
				Detail: group.GetNote(), Retryable: true,
			})
			continue
		}
		for _, detail := range group.GetDegradations() {
			degradations = append(degradations, &routingv1.SearchDegradation{
				Component: "provider", Reason: detail.GetReason(), ProviderId: group.GetProviderId(), Leg: detail.GetLeg(),
				Detail: detail.GetDetail(), Retryable: detail.GetRetryable(),
			})
		}
	}
	if partial {
		degradations = append(degradations, &routingv1.SearchDegradation{
			Component: "query", Reason: "query_deadline_partial",
			Detail: fmt.Sprintf("query deadline expired with %d provider(s) pending", pendingProviders), Retryable: true, FallbackActive: true,
		})
	}
	return degradations
}

func int32Pointer(value int32) *int32 { return &value }
