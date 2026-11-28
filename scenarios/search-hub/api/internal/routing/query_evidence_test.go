package routing

import (
	"testing"

	"github.com/stretchr/testify/require"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

func TestAutomaticReserveOutcomeClassifierKeepsSimpleAndCompoundSemanticsAligned(t *testing.T) {
	hit := &routingv1.SearchHit{ProviderId: "reserve", Id: "hit-1", Type: "skill", Path: "skills/one"}
	for _, tc := range []struct {
		name          string
		primaryFailed bool
		groups        []*routingv1.ProviderResultGroup
		want          automaticReserveOutcome
	}{
		{name: "healthy evidence after empty primary", groups: []*routingv1.ProviderResultGroup{{ProviderId: "reserve", Hits: []*routingv1.SearchHit{hit}}}, want: automaticReserveRecoveredAfterEmpty},
		{name: "healthy evidence after failed primary", primaryFailed: true, groups: []*routingv1.ProviderResultGroup{{ProviderId: "reserve", Hits: []*routingv1.SearchHit{hit}}}, want: automaticReserveRecoveredAfterFailure},
		{name: "provider failure after empty primary", groups: []*routingv1.ProviderResultGroup{{ProviderId: "reserve", Degraded: true}}, want: automaticReserveFailedAfterEmpty},
		{name: "provider failure after failed primary", primaryFailed: true, groups: []*routingv1.ProviderResultGroup{{ProviderId: "reserve", Degraded: true}}, want: automaticReserveFailedAfterFailure},
		{name: "healthy empty after failed primary", primaryFailed: true, groups: []*routingv1.ProviderResultGroup{{ProviderId: "reserve"}}, want: automaticReserveEmptyAfterFailure},
		{name: "healthy empty after healthy empty primary", groups: []*routingv1.ProviderResultGroup{{ProviderId: "reserve"}}, want: automaticReserveNoEvidence},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, classifyAutomaticReserveOutcome(tc.primaryFailed, tc.groups))
		})
	}
}

func TestQueryPartEvidenceUsesGroupHitReferenceWithoutClaimingRelevance(t *testing.T) {
	hit := &routingv1.SearchHit{ProviderId: "skills", Id: "design-skill", Type: "skill", Path: "design-skill"}
	groups := []*routingv1.ProviderResultGroup{{ProviderId: "skills", Hits: []*routingv1.SearchHit{hit}}}
	state := &queryPartState{
		plan:      queryPartPlan{id: "part-1", label: "design a UX", source: routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT},
		providers: map[string]struct{}{"skills": {}}, attempted: true,
	}
	recordQueryPartHits(state, groups, routingv1.SearchHitReference_RETRIEVAL_SOURCE_DERIVED_SEGMENT)
	parts := queryPartEvidence(&routingv1.QueryResponse{
		Groups: groups,
	}, []*queryPartState{state})

	require.Len(t, parts, 1)
	require.Equal(t, routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED, parts[0].GetStatus())
	require.Equal(t, "part-1", parts[0].GetPartId())
	require.Len(t, parts[0].GetEvidence(), 1)
	ref := parts[0].GetEvidence()[0]
	require.Equal(t, "skills", ref.GetProviderId())
	require.Equal(t, "design-skill", ref.GetHitId())
	require.Equal(t, []routingv1.SearchHitReference_RetrievalSource{routingv1.SearchHitReference_RETRIEVAL_SOURCE_DERIVED_SEGMENT}, ref.GetRetrievalSources())
	require.NotNil(t, ref.GroupIndex)
	require.NotNil(t, ref.HitIndex)
}

func TestPrimaryResultsRetainEligibleEvidenceAcrossSingleAndCompoundParts(t *testing.T) {
	selected := []*registryv1.ProviderDescriptor{{ProviderId: "ui-health.surfaces"}, {ProviderId: "cli-health.commands"}}
	uiHit := &routingv1.SearchHit{ProviderId: "ui-health.surfaces", Id: "surface-1", Type: "surface", Path: "surface.tsx"}
	unrelatedHit := &routingv1.SearchHit{ProviderId: "catalog.other", Id: "other-1", Type: "skill", Path: "other.md"}
	groups := []*routingv1.ProviderResultGroup{
		{ProviderId: "ui-health.surfaces", Hits: []*routingv1.SearchHit{uiHit}},
		{ProviderId: "cli-health.commands", Degraded: true, Note: "connection refused"},
		{ProviderId: "catalog.other", Hits: []*routingv1.SearchHit{unrelatedHit}},
	}
	for _, source := range []routingv1.QueryPartEvidence_Source{
		routingv1.QueryPartEvidence_SOURCE_USER_QUERY,
		routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT,
	} {
		t.Run(source.String(), func(t *testing.T) {
			state := &queryPartState{
				plan:      queryPartPlan{id: "part-1", label: "design a surface", source: source},
				providers: make(map[string]struct{}),
			}
			recordQueryPartPrimaryResults(state, selected, groups)
			parts := queryPartEvidence(&routingv1.QueryResponse{Groups: groups}, []*queryPartState{state})

			require.Len(t, parts, 1)
			part := parts[0]
			require.Equal(t, routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED, part.GetStatus())
			require.True(t, part.GetDegraded(), "the selected command owner failed during primary fan-out")
			require.True(t, part.GetPartial(), "eligible primary evidence remains visible despite the selected peer failure")
			require.ElementsMatch(t, []string{"ui-health.surfaces", "cli-health.commands"}, part.GetSearchedProviderIds())
			require.Len(t, part.GetEvidence(), 1, "unselected provider hits cannot be attributed to this part")
			require.Equal(t, "ui-health.surfaces", part.GetEvidence()[0].GetProviderId())
			require.Equal(t, []routingv1.SearchHitReference_RetrievalSource{routingv1.SearchHitReference_RETRIEVAL_SOURCE_USER_QUERY}, part.GetEvidence()[0].GetRetrievalSources())
		})
	}
}

func TestQueryPartEvidenceUsesRankedReferenceWhenReranked(t *testing.T) {
	hit := &routingv1.SearchHit{ProviderId: "skills", Id: "design-skill", Type: "skill", Path: "design-skill"}
	state := &queryPartState{
		plan:      queryPartPlan{id: "part-1", label: "design-skill", source: routingv1.QueryPartEvidence_SOURCE_USER_QUERY},
		providers: map[string]struct{}{"skills": {}}, attempted: true,
	}
	recordQueryPartHits(state, []*routingv1.ProviderResultGroup{{ProviderId: "skills", Hits: []*routingv1.SearchHit{hit}}}, routingv1.SearchHitReference_RETRIEVAL_SOURCE_USER_QUERY)
	parts := queryPartEvidence(&routingv1.QueryResponse{Ranked: []*routingv1.SearchHit{hit}}, []*queryPartState{state})

	require.Len(t, parts, 1)
	require.Len(t, parts[0].GetEvidence(), 1)
	require.Equal(t, int32(0), parts[0].GetEvidence()[0].GetRankedIndex())
	require.Nil(t, parts[0].GetEvidence()[0].GroupIndex)
}

func TestQueryPartEvidenceUsesGroupProviderForRerankedHitWithoutProviderID(t *testing.T) {
	hit := &routingv1.SearchHit{Id: "skill-1", Type: "skill", Path: "skills/design"}
	group := &routingv1.ProviderResultGroup{ProviderId: "catalog", Hits: []*routingv1.SearchHit{hit}}
	state := &queryPartState{
		plan:      queryPartPlan{id: "part-1", label: "design skill", source: routingv1.QueryPartEvidence_SOURCE_USER_QUERY},
		providers: map[string]struct{}{"catalog": {}}, attempted: true,
	}
	recordQueryPartHits(state, []*routingv1.ProviderResultGroup{group}, routingv1.SearchHitReference_RETRIEVAL_SOURCE_USER_QUERY)
	parts := queryPartEvidence(&routingv1.QueryResponse{Groups: []*routingv1.ProviderResultGroup{group}, Ranked: []*routingv1.SearchHit{hit}}, []*queryPartState{state})

	require.Len(t, parts, 1)
	require.Len(t, parts[0].GetEvidence(), 1)
	require.Equal(t, "catalog", parts[0].GetEvidence()[0].GetProviderId())
	require.Equal(t, "skill-1", parts[0].GetEvidence()[0].GetHitId())
	require.Equal(t, []routingv1.SearchHitReference_RetrievalSource{routingv1.SearchHitReference_RETRIEVAL_SOURCE_USER_QUERY}, parts[0].GetEvidence()[0].GetRetrievalSources())
}

func TestQueryPartEvidenceTracksAnonymousHitsAcrossRerankerOrder(t *testing.T) {
	first := &routingv1.SearchHit{ProviderId: "catalog", Title: "first anonymous result"}
	second := &routingv1.SearchHit{ProviderId: "catalog", Title: "second anonymous result"}
	states := []*queryPartState{
		{plan: queryPartPlan{id: "part-1", label: "first request", source: routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT}, providers: map[string]struct{}{"catalog": {}}, attempted: true},
		{plan: queryPartPlan{id: "part-2", label: "second request", source: routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT}, providers: map[string]struct{}{"catalog": {}}, attempted: true},
	}
	recordQueryPartHits(states[0], []*routingv1.ProviderResultGroup{{ProviderId: "catalog", Hits: []*routingv1.SearchHit{first}}}, routingv1.SearchHitReference_RETRIEVAL_SOURCE_DERIVED_SEGMENT)
	recordQueryPartHits(states[1], []*routingv1.ProviderResultGroup{{ProviderId: "catalog", Hits: []*routingv1.SearchHit{second}}}, routingv1.SearchHitReference_RETRIEVAL_SOURCE_DERIVED_SEGMENT)
	parts := queryPartEvidence(&routingv1.QueryResponse{Ranked: []*routingv1.SearchHit{second, first}}, states)

	require.Len(t, parts, 2)
	require.Len(t, parts[0].GetEvidence(), 1)
	require.Equal(t, int32(1), parts[0].GetEvidence()[0].GetRankedIndex())
	require.Len(t, parts[1].GetEvidence(), 1)
	require.Equal(t, int32(0), parts[1].GetEvidence()[0].GetRankedIndex())
}

func TestQueryPartEvidenceDoesNotBleedAcrossTypesSharingProviderPath(t *testing.T) {
	retrievedForPart := &routingv1.SearchHit{ProviderId: "catalog", Id: "skill-1", Type: "skill", Path: "guides/design"}
	otherTypeAtSamePath := &routingv1.SearchHit{ProviderId: "catalog", Id: "workflow-1", Type: "workflow", Path: "guides/design"}
	state := &queryPartState{
		plan:      queryPartPlan{id: "part-1", label: "design skill", source: routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT},
		providers: map[string]struct{}{"catalog": {}}, attempted: true,
	}
	recordQueryPartHits(state, []*routingv1.ProviderResultGroup{{ProviderId: "catalog", Hits: []*routingv1.SearchHit{retrievedForPart}}}, routingv1.SearchHitReference_RETRIEVAL_SOURCE_DERIVED_SEGMENT)
	parts := queryPartEvidence(&routingv1.QueryResponse{
		Groups: []*routingv1.ProviderResultGroup{{ProviderId: "catalog", Hits: []*routingv1.SearchHit{retrievedForPart, otherTypeAtSamePath}}},
	}, []*queryPartState{state})

	require.Len(t, parts, 1)
	require.Len(t, parts[0].GetEvidence(), 1)
	require.Equal(t, "skill-1", parts[0].GetEvidence()[0].GetHitId())
	require.Equal(t, "skill", parts[0].GetEvidence()[0].GetType())
	require.Equal(t, []routingv1.SearchHitReference_RetrievalSource{routingv1.SearchHitReference_RETRIEVAL_SOURCE_DERIVED_SEGMENT}, parts[0].GetEvidence()[0].GetRetrievalSources())
}

func TestQueryPartEvidenceReportsEmptyAndDegradedStatesSeparately(t *testing.T) {
	states := []*queryPartState{
		{plan: queryPartPlan{id: "part-1", label: "first", source: routingv1.QueryPartEvidence_SOURCE_USER_QUERY}, providers: map[string]struct{}{"skills": {}}, attempted: true},
		{plan: queryPartPlan{id: "part-2", label: "second", source: routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT}, providers: map[string]struct{}{"skills": {}}, attempted: true, degraded: true},
		{plan: queryPartPlan{id: "part-3", label: "third", source: routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT}},
	}
	parts := queryPartEvidence(&routingv1.QueryResponse{}, states)
	require.Equal(t, routingv1.QueryPartEvidence_STATUS_SEARCHED_EMPTY, parts[0].GetStatus())
	require.Equal(t, routingv1.QueryPartEvidence_STATUS_PROVIDER_DEGRADED, parts[1].GetStatus())
	require.True(t, parts[1].GetDegraded())
	require.Equal(t, routingv1.QueryPartEvidence_STATUS_NOT_ROUTED, parts[2].GetStatus())
}

func TestQueryWidePartialDoesNotDegradeHealthyPartEvidence(t *testing.T) {
	hit := &routingv1.SearchHit{ProviderId: "ui-health.surfaces", Id: "surface-1", Type: "surface", Path: "s.tsx"}
	partialHit := &routingv1.SearchHit{ProviderId: "cli-health.commands", Id: "command-1", Type: "command", Path: "scenario restart"}
	states := []*queryPartState{
		{plan: queryPartPlan{id: "part-1", label: "design a surface", source: routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT}, providers: map[string]struct{}{"ui-health.surfaces": {}}, attempted: true},
		{plan: queryPartPlan{id: "part-2", label: "run a command", source: routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT}, providers: map[string]struct{}{"cli-health.commands": {}}, attempted: true, degraded: true, partial: true, reason: "provider_timeout"},
	}
	recordQueryPartHits(states[0], []*routingv1.ProviderResultGroup{{ProviderId: "ui-health.surfaces", Hits: []*routingv1.SearchHit{hit}}}, routingv1.SearchHitReference_RETRIEVAL_SOURCE_DERIVED_SEGMENT)
	recordQueryPartHits(states[1], []*routingv1.ProviderResultGroup{{ProviderId: "cli-health.commands", Degraded: true, Hits: []*routingv1.SearchHit{partialHit}}}, routingv1.SearchHitReference_RETRIEVAL_SOURCE_DERIVED_SEGMENT)
	parts := queryPartEvidence(&routingv1.QueryResponse{
		Groups: []*routingv1.ProviderResultGroup{
			{ProviderId: "ui-health.surfaces", Hits: []*routingv1.SearchHit{hit}},
			{ProviderId: "cli-health.commands", Degraded: true, Hits: []*routingv1.SearchHit{partialHit}},
		},
		Partial: true,
	}, states)

	require.Len(t, parts, 2)
	require.Equal(t, routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED, parts[0].GetStatus())
	require.False(t, parts[0].GetDegraded())
	require.False(t, parts[0].GetPartial())
	require.Len(t, parts[0].GetEvidence(), 1)
	require.Equal(t, routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED, parts[1].GetStatus())
	require.True(t, parts[1].GetDegraded())
	require.True(t, parts[1].GetPartial())
}

func TestQueryPartPartialRequiresReturnedEvidence(t *testing.T) {
	state := &queryPartState{
		plan:      queryPartPlan{id: "part-1", label: "search", source: routingv1.QueryPartEvidence_SOURCE_USER_QUERY},
		providers: map[string]struct{}{"catalog": {}}, attempted: true, degraded: true, partial: true, reason: "provider_timeout",
	}
	parts := queryPartEvidence(&routingv1.QueryResponse{}, []*queryPartState{state})
	require.Len(t, parts, 1)
	require.Equal(t, routingv1.QueryPartEvidence_STATUS_PROVIDER_DEGRADED, parts[0].GetStatus())
	require.True(t, parts[0].GetDegraded())
	require.False(t, parts[0].GetPartial(), "a failure with no candidates is degraded, not partial evidence")
}

func TestDerivedOwnerFailureSurfacesWhenPrimaryGroupRemainsHealthy(t *testing.T) {
	hit := &routingv1.SearchHit{ProviderId: "ui-health.surfaces", Id: "surface-1", Type: "surface", Path: "surface.tsx"}
	groups := []*routingv1.ProviderResultGroup{{ProviderId: "ui-health.surfaces", Hits: []*routingv1.SearchHit{hit}}}
	state := &queryPartState{
		plan:      queryPartPlan{id: "part-1", label: "design a surface", source: routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT},
		providers: map[string]struct{}{"ui-health.surfaces": {}}, attempted: true, degraded: true,
		reason: "supplemental_primary_provider_degraded",
	}
	resp := &routingv1.QueryResponse{Groups: groups}
	resp.QueryParts = queryPartEvidence(resp, []*queryPartState{state})
	resp.Degradations = searchDegradations(groups, false, "", "", false, "", false, 0, 0, false, false)
	applyQueryPartDegradations(resp, []*queryPartState{state})

	require.True(t, resp.GetDegraded(), "a derived request can fail even when the primary full-query group stayed healthy")
	require.False(t, groups[0].GetDegraded(), "the earlier healthy result group must not be rewritten as a provider outage")
	require.Len(t, resp.GetDegradations(), 1)
	require.Equal(t, "query_part", resp.GetDegradations()[0].GetComponent())
	require.Equal(t, "supplemental_primary_provider_degraded", resp.GetDegradations()[0].GetReason())
	require.Contains(t, resp.GetDegradations()[0].GetDetail(), "part-1")
}

func TestPartFailureCoveredByProviderGroupDoesNotDuplicateResponseDegradation(t *testing.T) {
	groups := []*routingv1.ProviderResultGroup{{ProviderId: "cli-health.commands", Degraded: true, Note: "connection refused"}}
	state := &queryPartState{
		plan:      queryPartPlan{id: "part-1", label: "run a command", source: routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT},
		providers: map[string]struct{}{"cli-health.commands": {}}, attempted: true, degraded: true,
		reason: "selected_provider_degraded_on_primary_query",
	}
	resp := &routingv1.QueryResponse{Groups: groups}
	resp.QueryParts = queryPartEvidence(resp, []*queryPartState{state})
	resp.Degradations = searchDegradations(groups, false, "", "", false, "", false, 0, 0, false, false)
	applyQueryPartDegradations(resp, []*queryPartState{state})

	require.True(t, resp.GetDegraded())
	require.Len(t, resp.GetDegradations(), 1)
	require.Equal(t, "provider", resp.GetDegradations()[0].GetComponent())
}

func TestCallerDeadlinePartialIncludesPendingCompoundParts(t *testing.T) {
	groups := []*routingv1.ProviderResultGroup{{ProviderId: "ui-health.surfaces", Hits: []*routingv1.SearchHit{{Id: "surface-1", Type: "surface"}}}}
	states := []*queryPartState{{
		plan:      queryPartPlan{id: "part-2", label: "run a command", source: routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT},
		providers: map[string]struct{}{"cli-health.commands": {}}, attempted: true, degraded: true,
		reason: "query_deadline_exhausted",
	}}
	usable, pendingProviders, pendingParts, partial := queryDeadlineCoverage(groups, states)
	require.Equal(t, 1, usable)
	require.Zero(t, pendingProviders, "the original full-query provider group remains healthy")
	require.Equal(t, 1, pendingParts)
	require.True(t, partial, "caller deadline expiry leaves one compound part incomplete after usable evidence returned")

	degradations := searchDegradations(groups, false, "", "", false, "", partial, pendingProviders, pendingParts, false, false)
	require.Len(t, degradations, 1)
	require.Equal(t, "query_deadline_partial", degradations[0].GetReason())
	require.Contains(t, degradations[0].GetDetail(), "0 provider(s) and 1 derived part(s) pending")
}

func TestCallerDeadlineWithoutUsableEvidenceIsNotPartial(t *testing.T) {
	states := []*queryPartState{{
		plan:      queryPartPlan{id: "part-1", label: "design a surface", source: routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT},
		attempted: true, degraded: true, reason: "query_deadline_exhausted",
	}}
	_, _, pendingParts, partial := queryDeadlineCoverage([]*routingv1.ProviderResultGroup{{ProviderId: "ui-health.surfaces"}}, states)
	require.Equal(t, 1, pendingParts)
	require.False(t, partial, "partial remains reserved for caller-deadline expiry after usable evidence")
}

func TestCallerDeadlineTreatsDegradedGroupHitsAsUsablePartialEvidence(t *testing.T) {
	groups := []*routingv1.ProviderResultGroup{{
		ProviderId: "cli-health.commands", Degraded: true,
		Hits: []*routingv1.SearchHit{{ProviderId: "cli-health.commands", Id: "restart", Type: "command", Path: "scenario restart"}},
	}}
	usable, pendingProviders, pendingParts, partial := queryDeadlineCoverage(groups, nil)
	require.Equal(t, 1, usable, "degraded groups can still carry valid returned candidate hits")
	require.Equal(t, 1, pendingProviders)
	require.Zero(t, pendingParts)
	require.True(t, partial)
}

func TestSearchDegradationsRetainIndependentComponents(t *testing.T) {
	got := searchDegradations([]*routingv1.ProviderResultGroup{
		{ProviderId: "one", Degraded: true, Note: "timed out"},
		{ProviderId: "two", Degraded: true, Degradations: []*routingv1.ProviderDegradation{{Reason: "http_error", Leg: "semantic", Detail: "unavailable", Retryable: true}}},
	}, true, "cross_encoder_timeout", "routing_index_unavailable", true, "reranker_unavailable", true, 1, 0, false, false)

	require.Len(t, got, 6)
	require.Equal(t, "selector", got[0].GetComponent())
	require.Equal(t, "routing_index_unavailable", got[0].GetReason())
	require.True(t, got[0].GetFallbackActive())
	require.Equal(t, "selector", got[1].GetComponent())
	require.Equal(t, "reranker", got[2].GetComponent())
	require.Equal(t, "one", got[3].GetProviderId())
	require.Equal(t, "http_error", got[4].GetReason())
	require.Equal(t, "query_deadline_partial", got[5].GetReason())
}

func TestSupplementalBudgetDegradationDoesNotClaimQueryDeadline(t *testing.T) {
	got := searchDegradations(nil, false, "", "", false, "", false, 0, 0, true, false)
	require.Len(t, got, 1)
	require.Equal(t, "query", got[0].GetComponent())
	require.Equal(t, "supplemental_query_budget_exhausted", got[0].GetReason())
	require.NotEqual(t, "query_deadline_partial", got[0].GetReason())
	require.True(t, got[0].GetFallbackActive())
}

func TestSupplementalPartBudgetDegradationIsDistinctFromGlobalBudget(t *testing.T) {
	got := searchDegradations(nil, false, "", "", false, "", false, 0, 0, false, true)
	require.Len(t, got, 1)
	require.Equal(t, "supplemental_part_budget_exhausted", got[0].GetReason())
}

func TestQueryPartTargetIDsAreStable(t *testing.T) {
	state := &queryPartState{providers: make(map[string]struct{})}
	addQueryPartTargets(state, []*registryv1.ProviderDescriptor{{ProviderId: "z"}, {ProviderId: "a"}, {ProviderId: "z"}})
	require.Equal(t, []string{"a", "z"}, queryPartProviderIDs(state))
}
