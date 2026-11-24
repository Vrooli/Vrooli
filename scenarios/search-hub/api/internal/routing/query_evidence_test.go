package routing

import (
	"testing"

	"github.com/stretchr/testify/require"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

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

func TestSearchDegradationsRetainIndependentComponents(t *testing.T) {
	got := searchDegradations([]*routingv1.ProviderResultGroup{
		{ProviderId: "one", Degraded: true, Note: "timed out"},
		{ProviderId: "two", Degraded: true, Degradations: []*routingv1.ProviderDegradation{{Reason: "http_error", Leg: "semantic", Detail: "unavailable", Retryable: true}}},
	}, true, "cross_encoder_timeout", "routing_index_unavailable", true, "reranker_unavailable", true, 1)

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

func TestQueryPartTargetIDsAreStable(t *testing.T) {
	state := &queryPartState{providers: make(map[string]struct{})}
	addQueryPartTargets(state, []*registryv1.ProviderDescriptor{{ProviderId: "z"}, {ProviderId: "a"}, {ProviderId: "z"}})
	require.Equal(t, []string{"a", "z"}, queryPartProviderIDs(state))
}
