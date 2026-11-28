package routing

import (
	"fmt"
	"sort"
	"strings"

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
	// Anonymous hits have no stable document identity. Keep evidence attached
	// to the provider and returned hit object so ranking cannot confuse offsets.
	anonymousHits map[queryPartAnonymousHitKey]map[routingv1.SearchHitReference_RetrievalSource]struct{}
	attempted     bool
	degraded      bool
	partial       bool
	reason        string
}

type queryPartHitKey struct {
	providerID string
	document   string
}

type queryPartAnonymousHitKey struct {
	providerID string
	hit        *routingv1.SearchHit
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

// recordQueryPartPrimaryResults preserves evidence from the initial full-query
// fan-out for owners selected for a compound part. This matters when the
// bounded derived request cannot run or times out; only an owner already
// eligible for that part may contribute its full-query hits to its evidence.
func recordQueryPartPrimaryResults(state *queryPartState, selected []*registryv1.ProviderDescriptor, groups []*routingv1.ProviderResultGroup) {
	if state == nil {
		return
	}
	byProvider := make(map[string]*routingv1.ProviderResultGroup, len(groups))
	for _, group := range groups {
		if group != nil {
			byProvider[group.GetProviderId()] = group
		}
	}
	for _, target := range selected {
		if target == nil || target.GetProviderId() == "" {
			continue
		}
		group := byProvider[target.GetProviderId()]
		if group == nil {
			continue
		}
		addQueryPartTargets(state, []*registryv1.ProviderDescriptor{target})
		state.attempted = true
		if group.GetDegraded() {
			state.degraded = true
			if state.reason == "" {
				state.reason = "selected_provider_degraded_on_primary_query"
			}
		}
		recordQueryPartHits(state, []*routingv1.ProviderResultGroup{group}, routingv1.SearchHitReference_RETRIEVAL_SOURCE_USER_QUERY)
	}
}

func hasDegradedGroups(groups []*routingv1.ProviderResultGroup) bool {
	for _, group := range groups {
		if group.GetDegraded() {
			return true
		}
	}
	return false
}

func hasHealthyPartEvidence(groups []*routingv1.ProviderResultGroup) bool {
	for _, group := range groups {
		if group != nil && !group.GetDegraded() && len(group.GetHits()) > 0 {
			return true
		}
	}
	return false
}

type automaticReserveOutcome uint8

const (
	automaticReserveNoEvidence automaticReserveOutcome = iota
	automaticReserveRecoveredAfterEmpty
	automaticReserveRecoveredAfterFailure
	automaticReserveFailedAfterEmpty
	automaticReserveFailedAfterFailure
	automaticReserveEmptyAfterFailure
)

// classifyAutomaticReserveOutcome is shared by single-query and per-part
// automatic recovery so a healthy-empty reserve cannot be reported as
// partial evidence in one route family but not the other.
func classifyAutomaticReserveOutcome(primaryFailed bool, reserveGroups []*routingv1.ProviderResultGroup) automaticReserveOutcome {
	if hasHealthyPartEvidence(reserveGroups) {
		if primaryFailed {
			return automaticReserveRecoveredAfterFailure
		}
		return automaticReserveRecoveredAfterEmpty
	}
	if hasDegradedGroups(reserveGroups) {
		if primaryFailed {
			return automaticReserveFailedAfterFailure
		}
		return automaticReserveFailedAfterEmpty
	}
	if primaryFailed {
		return automaticReserveEmptyAfterFailure
	}
	return automaticReserveNoEvidence
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
			if !hasStableHitIdentity(hit) {
				if state.anonymousHits == nil {
					state.anonymousHits = make(map[queryPartAnonymousHitKey]map[routingv1.SearchHitReference_RetrievalSource]struct{})
				}
				providerID := hit.GetProviderId()
				if providerID == "" {
					providerID = group.GetProviderId()
				}
				key := queryPartAnonymousHitKey{providerID: providerID, hit: hit}
				if state.anonymousHits[key] == nil {
					state.anonymousHits[key] = make(map[routingv1.SearchHitReference_RetrievalSource]struct{})
				}
				state.anonymousHits[key][source] = struct{}{}
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
	if !hasStableHitIdentity(hit) {
		return sortedRetrievalSources(state.anonymousHits[queryPartAnonymousHitKey{providerID: providerID, hit: hit}])
	}
	sourceSet := state.hits[queryPartHitKey{providerID: providerID, document: documentKey(hit, index)}]
	return sortedRetrievalSources(sourceSet)
}

func hasStableHitIdentity(hit *routingv1.SearchHit) bool {
	return hit != nil && (hit.GetType() != "" || hit.GetId() != "" || hit.GetPath() != "")
}

func queryPartHasRecordedHits(state *queryPartState) bool {
	return state != nil && (len(state.hits) > 0 || len(state.anonymousHits) > 0)
}

func sortedRetrievalSources(sourceSet map[routingv1.SearchHitReference_RetrievalSource]struct{}) []routingv1.SearchHitReference_RetrievalSource {
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
	rankedProviderIDs := make(map[*routingv1.SearchHit]string, len(resp.GetRanked()))
	if len(resp.GetRanked()) > 0 {
		for _, group := range resp.GetGroups() {
			for _, hit := range group.GetHits() {
				if hit == nil {
					continue
				}
				providerID := hit.GetProviderId()
				if providerID == "" {
					providerID = group.GetProviderId()
				}
				rankedProviderIDs[hit] = providerID
			}
		}
	}
	parts := make([]*routingv1.QueryPartEvidence, 0, len(states))
	for _, state := range states {
		if state == nil {
			continue
		}
		part := &routingv1.QueryPartEvidence{
			PartId: state.plan.id, Label: state.plan.label, Source: state.plan.source,
			SearchedProviderIds: queryPartProviderIDs(state),
			Degraded:            state.degraded,
			Reason:              state.reason,
		}
		if len(resp.GetRanked()) > 0 {
			for rank, hit := range resp.GetRanked() {
				providerID := rankedProviderIDs[hit]
				if providerID == "" {
					providerID = hit.GetProviderId()
				}
				if sources := queryPartHitSources(state, providerID, hit, rank); len(sources) > 0 {
					part.Evidence = append(part.Evidence, &routingv1.SearchHitReference{
						ProviderId: providerID, RankedIndex: int32Pointer(int32(rank)),
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
		case state.degraded:
			part.Status = routingv1.QueryPartEvidence_STATUS_PROVIDER_DEGRADED
		case !state.attempted:
			part.Status = routingv1.QueryPartEvidence_STATUS_NOT_ROUTED
		default:
			part.Status = routingv1.QueryPartEvidence_STATUS_SEARCHED_EMPTY
		}
		// Part-level partial describes incomplete candidate coverage. A failed
		// provider without any returned evidence is degraded, but has no partial
		// evidence to expose. The caller-deadline-only response Partial field is
		// computed separately by the router.
		part.Partial = len(part.GetEvidence()) > 0 && (state.partial || state.degraded)
		parts = append(parts, part)
	}
	return parts
}

// applyQueryPartDegradations projects per-part failures onto the response even
// when a provider's full-query group was healthy and therefore remains healthy
// after a later derived request failed. Existing provider-group degradations
// and shared budget reasons already have their own response-level entries.
func applyQueryPartDegradations(resp *routingv1.QueryResponse, states []*queryPartState) {
	if resp == nil {
		return
	}
	for _, part := range resp.GetQueryParts() {
		if part.GetDegraded() {
			resp.Degraded = true
		}
	}
	for _, state := range states {
		if state == nil || !state.degraded || state.reason == "" || state.reason == "supplemental_query_budget_exhausted" || state.reason == "supplemental_part_budget_exhausted" || state.reason == "query_deadline_exhausted" && resp.GetPartial() {
			continue
		}
		coveredByProvider := false
		for _, group := range resp.GetGroups() {
			if group == nil || !group.GetDegraded() {
				continue
			}
			if _, selected := state.providers[group.GetProviderId()]; selected {
				coveredByProvider = true
				break
			}
		}
		if coveredByProvider {
			continue
		}
		providerIDs := queryPartProviderIDs(state)
		resp.Degradations = append(resp.Degradations, &routingv1.SearchDegradation{
			Component: "query_part", Reason: state.reason,
			Detail:    fmt.Sprintf("part %s (%s) has incomplete evidence from selected owner(s): %s", state.plan.id, state.plan.label, strings.Join(providerIDs, ", ")),
			Retryable: true, FallbackActive: true,
		})
	}
}

// queryDeadlineCoverage accounts for provider and derived-part work that was
// still incomplete when the caller's query context expired. PendingProviders
// remains provider-only; pending part count is reported separately in the
// response degradation detail.
func queryDeadlineCoverage(groups []*routingv1.ProviderResultGroup, states []*queryPartState) (usable, pendingProviders, pendingParts int, partial bool) {
	for _, group := range groups {
		if group == nil {
			continue
		}
		if group.GetDegraded() {
			pendingProviders++
		}
		if len(group.GetHits()) > 0 {
			usable++
		}
	}
	for _, state := range states {
		if state != nil && state.reason == "query_deadline_exhausted" {
			pendingParts++
		}
	}
	partial = usable > 0 && (pendingProviders > 0 || pendingParts > 0)
	return usable, pendingProviders, pendingParts, partial
}

func searchDegradations(groups []*routingv1.ProviderResultGroup, selectionFallback bool, selectionReason, routingIndexReason string, rerankDegraded bool, rerankReason string, queryDeadlinePartial bool, pendingProviders, pendingParts int, supplementalBudgetExhausted, supplementalPartBudgetExhausted bool) []*routingv1.SearchDegradation {
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
	if queryDeadlinePartial {
		degradations = append(degradations, &routingv1.SearchDegradation{
			Component: "query", Reason: "query_deadline_partial",
			Detail: fmt.Sprintf("query deadline expired with %d provider(s) and %d derived part(s) pending", pendingProviders, pendingParts), Retryable: true, FallbackActive: true,
		})
	}
	if supplementalBudgetExhausted {
		degradations = append(degradations, &routingv1.SearchDegradation{
			Component: "query", Reason: "supplemental_query_budget_exhausted",
			Detail: "the bounded supplemental retrieval budget expired before automatic recovery or all selected derived parts completed", Retryable: true, FallbackActive: true,
		})
	}
	if supplementalPartBudgetExhausted {
		degradations = append(degradations, &routingv1.SearchDegradation{
			Component: "query", Reason: "supplemental_part_budget_exhausted",
			Detail: "one or more derived parts reached their fair share of the bounded supplemental query budget", Retryable: true, FallbackActive: true,
		})
	}
	return degradations
}

func int32Pointer(value int32) *int32 { return &value }
