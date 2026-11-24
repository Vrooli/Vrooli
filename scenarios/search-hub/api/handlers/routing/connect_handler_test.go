package routing_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing/routing_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	handler "search-hub/handlers/routing"
	internalregistry "search-hub/internal/registry"
	internalrouting "search-hub/internal/routing"
)

// fakeRouter is a hand-written Querier fake: arrange via fields, assert against
// the recorded request. Keeps the handler test off real provider fan-out (that
// path is covered by internal/routing/router_test.go).
type fakeRouter struct {
	out               *routingv1.QueryResponse
	err               error
	lastReq           *routingv1.QueryRequest
	statusOut         *routingv1.StatusResponse
	statusErr         error
	statusCall        bool
	providerStatusOut *routingv1.ProviderStatusResponse
	providerStatusErr error
	providerStatusID  string
	repromoteErr      error
	repromoteID       string
}

func (f *fakeRouter) Query(_ context.Context, req *routingv1.QueryRequest) (*routingv1.QueryResponse, error) {
	f.lastReq = req
	return f.out, f.err
}

func (f *fakeRouter) Status(context.Context) (*routingv1.StatusResponse, error) {
	f.statusCall = true
	return f.statusOut, f.statusErr
}

func (f *fakeRouter) ProviderStatus(_ context.Context, providerID string) (*routingv1.ProviderStatusResponse, error) {
	f.providerStatusID = providerID
	return f.providerStatusOut, f.providerStatusErr
}

func (f *fakeRouter) RepromoteProvider(_ context.Context, providerID string) error {
	f.repromoteID = providerID
	return f.repromoteErr
}

func newClient(t *testing.T, router handler.Querier) routingconnect.RoutingServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, h := routingconnect.NewRoutingServiceHandler(handler.NewConnectHandler(handler.Deps{Router: router, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: h})
	return routingconnect.NewRoutingServiceClient(server.Client(), server.URL)
}

type apiTestLister struct {
	providers []*registryv1.ProviderDescriptor
}

func (l apiTestLister) List(_ context.Context, filter internalregistry.ListFilter) ([]*registryv1.ProviderDescriptor, error) {
	matched := make([]*registryv1.ProviderDescriptor, 0, len(l.providers))
	for _, provider := range l.providers {
		if filter.ProviderID != "" && provider.GetProviderId() != filter.ProviderID {
			continue
		}
		if filter.State != 0 && int32(provider.GetState()) != filter.State {
			continue
		}
		matched = append(matched, provider)
	}
	return matched, nil
}

type apiTestResolver struct{ urls map[string]string }

func (r apiTestResolver) ResolveScenarioURL(_ context.Context, scenarioID string) (string, error) {
	if url := r.urls[scenarioID]; url != "" {
		return url, nil
	}
	return "http://" + scenarioID + ".test", nil
}

type apiTestDoer struct {
	mu     sync.Mutex
	body   string
	bodies []string
	urls   []string
}

func (d *apiTestDoer) Do(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	d.mu.Lock()
	d.body = string(body)
	d.bodies = append(d.bodies, string(body))
	d.urls = append(d.urls, req.URL.String())
	d.mu.Unlock()
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"results":[{"name":"run tests","description":"Run scenario tests","score":0.9}]}`)),
		Header:     make(http.Header),
	}, nil
}

func apiTestProvider(id, group, description, scenarioID string) *registryv1.ProviderDescriptor {
	return &registryv1.ProviderDescriptor{
		ProviderId: id, ProviderGroup: group, Type: "skill", Description: description,
		Bucket: registryv1.Bucket_BUCKET_DO, State: registryv1.ProviderState_PROVIDER_STATE_ACTIVE,
		Endpoint: &registryv1.Endpoint{Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{
			ScenarioId: scenarioID, Path: "/search", Method: registryv1.HttpMethod_HTTP_METHOD_POST,
			BodyTemplate: `{"query":"{{query}}","limit":{{limit}}}`,
		}}},
		ResultMapping: &registryv1.ResultMapping{ResultsPath: "results", IdField: "name", TitleField: "name", ScoreField: "score", SnippetField: "description", PathField: "name", ScoreScale: registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1},
	}
}

// This exercises the ordinary Connect API through the real router and a local
// provider seam. It proves the additive evidence contract is produced by the
// serving route, rather than only being transportable from a fake response.
func TestOrdinaryAPIEmitsEvidenceForUniqueTypoRoute(t *testing.T) {
	provider := &registryv1.ProviderDescriptor{
		ProviderId: "cli-health.commands", ProviderGroup: "cli-health", Type: "command",
		Description: "CLI commands.", Bucket: registryv1.Bucket_BUCKET_DO,
		State: registryv1.ProviderState_PROVIDER_STATE_ACTIVE,
		Endpoint: &registryv1.Endpoint{Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{
			ScenarioId: "cli-health", Path: "/search", Method: registryv1.HttpMethod_HTTP_METHOD_POST,
			BodyTemplate: `{"query":"{{query}}","limit":{{limit}}}`,
		}}},
		ResultMapping: &registryv1.ResultMapping{ResultsPath: "results", IdField: "name", TitleField: "name", ScoreField: "score", SnippetField: "description", PathField: "name", ScoreScale: registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1},
	}
	strategy := internalrouting.RetrievalStrategy{
		Name:   "lexical-typo-fallback",
		Stages: []internalrouting.RetrievalStage{{Kind: internalrouting.StageLexical, Params: map[string]interface{}{"shortlist_width": float64(1), "typo_policy": "one_edit_unique"}}},
	}
	doer := &apiTestDoer{}
	router := internalrouting.NewRouter(internalrouting.Deps{
		Lister:   apiTestLister{providers: []*registryv1.ProviderDescriptor{provider}},
		Resolver: apiTestResolver{}, Doer: doer, Strategy: &strategy,
	})
	client := newClient(t, router)
	resp, err := client.Query(context.Background(), connect.NewRequest(&routingv1.QueryRequest{Query: "commnad", Explain: true}))
	require.NoError(t, err)
	require.Contains(t, doer.body, `"query":"commnad"`, "the provider request retains the exact caller text")
	require.Equal(t, []string{"cli-health.commands"}, resp.Msg.GetCorporaSearched())
	require.Len(t, resp.Msg.GetQueryParts(), 1)
	part := resp.Msg.GetQueryParts()[0]
	require.Equal(t, routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED, part.GetStatus())
	require.Equal(t, []string{"cli-health.commands"}, part.GetSearchedProviderIds())
	require.Len(t, part.GetEvidence(), 1)
	require.Equal(t, "run tests", part.GetEvidence()[0].GetHitId())
	require.Contains(t, part.GetEvidence()[0].GetRetrievalSources(), routingv1.SearchHitReference_RETRIEVAL_SOURCE_USER_QUERY)
	require.NotEmpty(t, resp.Msg.GetRoutingExplanation())

	exactQuery := "use CLI Health Commands"
	exactResp, err := client.Query(context.Background(), connect.NewRequest(&routingv1.QueryRequest{Query: exactQuery, Explain: true}))
	require.NoError(t, err)
	require.Contains(t, doer.body, `"query":"`+exactQuery+`"`, "the exact-identity route must preserve caller text")
	require.Equal(t, []string{"cli-health.commands"}, exactResp.Msg.GetCorporaSearched())
	require.Contains(t, strings.Join(exactResp.Msg.GetRoutingExplanation(), "\n"), "exact eligible provider identity matched")
}

func TestOrdinaryAPIShortAndElaboratedTaskQueriesUseSameOwner(t *testing.T) {
	ui := apiTestProvider("ui-health.surfaces", "ui-health", "Mobile iOS fitness tracker interface, weekly activity charts, recovery tracking, pace zones for runners", "ui-health")
	commands := apiTestProvider("cli-health.commands", "cli-health", "CLI commands, test automation, scenario deploy workflows and operational checks", "cli-health")
	strategy := internalrouting.RetrievalStrategy{
		Name:   "lexical-dev-pair",
		Stages: []internalrouting.RetrievalStage{{Kind: internalrouting.StageLexical, Params: map[string]interface{}{"shortlist_width": float64(1)}}},
	}
	doer := &apiTestDoer{}
	router := internalrouting.NewRouter(internalrouting.Deps{
		Lister:   apiTestLister{providers: []*registryv1.ProviderDescriptor{ui, commands}},
		Resolver: apiTestResolver{urls: map[string]string{"ui-health": "http://ui-health.test", "cli-health": "http://cli-health.test"}}, Doer: doer, Strategy: &strategy,
	})
	client := newClient(t, router)

	queries := []string{
		"design an iOS fitness tracker interface",
		"design a polished iOS fitness tracker interface with weekly activity charts, recovery tracking, and clear pace zones for runners",
	}
	for _, query := range queries {
		doer.mu.Lock()
		startIndex := len(doer.bodies)
		doer.mu.Unlock()
		resp, err := client.Query(context.Background(), connect.NewRequest(&routingv1.QueryRequest{Query: query}))
		require.NoError(t, err)
		require.Equal(t, []string{"ui-health.surfaces"}, resp.Msg.GetCorporaSearched(), "query class should resolve to the UI owner")
		for _, part := range resp.Msg.GetQueryParts() {
			require.Equal(t, []string{"ui-health.surfaces"}, part.GetSearchedProviderIds(), "each emitted part must retain the owner selected for that need")
		}
		doer.mu.Lock()
		require.Greater(t, len(doer.bodies), startIndex)
		firstRequest := doer.bodies[startIndex]
		doer.mu.Unlock()
		require.Contains(t, firstRequest, `"query":"`+query+`"`, "the primary provider call sees the original query and its platform/output constraints")
	}
}

func TestOrdinaryAPIExposesMissingProviderAsUnroutedPart(t *testing.T) {
	doer := &apiTestDoer{}
	router := internalrouting.NewRouter(internalrouting.Deps{
		Lister: apiTestLister{}, Resolver: apiTestResolver{}, Doer: doer,
	})
	client := newClient(t, router)
	resp, err := client.Query(context.Background(), connect.NewRequest(&routingv1.QueryRequest{Query: "design an unavailable named resource"}))
	require.NoError(t, err)
	require.Empty(t, resp.Msg.GetCorporaSearched())
	require.Len(t, resp.Msg.GetQueryParts(), 1)
	part := resp.Msg.GetQueryParts()[0]
	require.Equal(t, routingv1.QueryPartEvidence_STATUS_NOT_ROUTED, part.GetStatus())
	require.Empty(t, part.GetSearchedProviderIds())
	require.Empty(t, doer.bodies, "no provider call should be fabricated when the eligible provider set is empty")
	require.NotEmpty(t, resp.Msg.GetDegradations())
	require.Equal(t, "no_eligible_provider", resp.Msg.GetDegradations()[0].GetReason())
}

func TestQueryReturnsGroups(t *testing.T) {
	router := &fakeRouter{out: &routingv1.QueryResponse{
		Groups: []*routingv1.ProviderResultGroup{
			{ProviderId: "cli-health.commands", Count: 1, Hits: []*routingv1.SearchHit{{ProviderId: "cli-health.commands", Title: "scenario restart"}}},
		},
		CorporaSearched: []string{"cli-health.commands"},
	}}
	client := newClient(t, router)

	resp, err := client.Query(context.Background(), connect.NewRequest(&routingv1.QueryRequest{
		Query: "restart a scenario", Types: []string{"command"}, Limit: 5,
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetGroups(), 1)
	require.Equal(t, "cli-health.commands", resp.Msg.GetGroups()[0].GetProviderId())
	// The handler forwards the request verbatim to the router seam.
	require.Equal(t, "restart a scenario", router.lastReq.GetQuery())
	require.Equal(t, []string{"command"}, router.lastReq.GetTypes())
	require.Equal(t, int32(5), router.lastReq.GetLimit())
}

func TestQueryTransportsPartCandidatesAndStructuredDegradation(t *testing.T) {
	rankedIndex := int32(0)
	router := &fakeRouter{out: &routingv1.QueryResponse{
		Ranked: []*routingv1.SearchHit{{ProviderId: "skills", Id: "ux-skill"}},
		QueryParts: []*routingv1.QueryPartEvidence{{
			PartId: "part-1", Label: "design a UX", Source: routingv1.QueryPartEvidence_SOURCE_USER_QUERY,
			SearchedProviderIds: []string{"skills"}, Status: routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED,
			Evidence: []*routingv1.SearchHitReference{{
				ProviderId: "skills", RankedIndex: &rankedIndex, HitId: "ux-skill",
				RetrievalSources: []routingv1.SearchHitReference_RetrievalSource{routingv1.SearchHitReference_RETRIEVAL_SOURCE_USER_QUERY},
			}},
		}},
		Degradations: []*routingv1.SearchDegradation{{
			Component: "reranker", Reason: "reranker_unavailable", FallbackActive: true, Retryable: true,
		}},
	}}
	client := newClient(t, router)

	resp, err := client.Query(context.Background(), connect.NewRequest(&routingv1.QueryRequest{Query: "design a UX"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetQueryParts(), 1)
	require.Equal(t, routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED, resp.Msg.GetQueryParts()[0].GetStatus())
	require.Equal(t, int32(0), resp.Msg.GetQueryParts()[0].GetEvidence()[0].GetRankedIndex())
	require.Equal(t, routingv1.SearchHitReference_RETRIEVAL_SOURCE_USER_QUERY, resp.Msg.GetQueryParts()[0].GetEvidence()[0].GetRetrievalSources()[0])
	require.Len(t, resp.Msg.GetDegradations(), 1)
	require.Equal(t, "reranker", resp.Msg.GetDegradations()[0].GetComponent())
	require.True(t, resp.Msg.GetDegradations()[0].GetFallbackActive())
}

func TestQueryInvalidArgument(t *testing.T) {
	router := &fakeRouter{err: internalrouting.ErrInvalidQuery{Reason: "query text is required"}}
	client := newClient(t, router)

	_, err := client.Query(context.Background(), connect.NewRequest(&routingv1.QueryRequest{All: true}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestQueryInternalErrorIsOpaque(t *testing.T) {
	router := &fakeRouter{err: errors.New("registry db on fire: /var/lib/secret")}
	client := newClient(t, router)

	_, err := client.Query(context.Background(), connect.NewRequest(&routingv1.QueryRequest{Query: "x", All: true}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.NotContains(t, err.Error(), "secret", "internal details must not leak to the client")
}

func TestStatusReturnsHealth(t *testing.T) {
	router := &fakeRouter{statusOut: &routingv1.StatusResponse{
		Providers: []*routingv1.ProviderHealth{
			{ProviderId: "cli-health.commands", Reachable: true, Reachability: "endpoint resolved", IndexAge: "not_applicable: provider has no status_endpoint"},
		},
		ClassifierAvailable: true,
		RerankerAvailable:   false,
	}}
	client := newClient(t, router)

	resp, err := client.Status(context.Background(), connect.NewRequest(&routingv1.StatusRequest{}))
	require.NoError(t, err)
	require.True(t, router.statusCall)
	require.Len(t, resp.Msg.GetProviders(), 1)
	require.True(t, resp.Msg.GetProviders()[0].GetReachable())
	require.True(t, resp.Msg.GetClassifierAvailable())
	require.False(t, resp.Msg.GetRerankerAvailable())
}

func TestStatusInternalErrorIsOpaque(t *testing.T) {
	router := &fakeRouter{statusErr: errors.New("registry db on fire: /var/lib/secret")}
	client := newClient(t, router)

	_, err := client.Status(context.Background(), connect.NewRequest(&routingv1.StatusRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.NotContains(t, err.Error(), "secret", "internal details must not leak to the client")
}

func TestProviderStatusForwardsExactProviderAndHealth(t *testing.T) {
	health := &routingv1.ProviderHealth{ProviderId: "skills", Reachable: true, IndexAge: "5m"}
	router := &fakeRouter{providerStatusOut: &routingv1.ProviderStatusResponse{ProviderId: "skills", Health: health}}
	client := newClient(t, router)

	resp, err := client.ProviderStatus(context.Background(), connect.NewRequest(&routingv1.ProviderStatusRequest{ProviderId: "skills"}))
	require.NoError(t, err)
	require.Equal(t, "skills", router.providerStatusID)
	require.Equal(t, "skills", resp.Msg.GetProviderId())
	require.True(t, resp.Msg.GetHealth().GetReachable())
	require.Equal(t, "5m", resp.Msg.GetHealth().GetIndexAge())
}

func TestProviderStatusDeliversAdditiveSelectedOwnerProjectionHeaders(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	health := &routingv1.ProviderHealth{
		ProviderId: "selected.synthetic", Reachable: true, Reachability: "endpoint resolved",
		IndexAge: "5m", LastIndexedAt: timestamppb.New(now.Add(-5 * time.Minute)), FreshnessBudget: "1h",
	}
	router := &fakeRouter{providerStatusOut: &routingv1.ProviderStatusResponse{
		ProviderId: "selected.synthetic", Health: health, ObservedAt: timestamppb.New(now),
	}}
	client := newClient(t, router)
	response, err := client.ProviderStatus(context.Background(), connect.NewRequest(&routingv1.ProviderStatusRequest{ProviderId: "selected.synthetic"}))
	require.NoError(t, err)
	require.Equal(t, "selected.synthetic", router.providerStatusID, "the API must probe the exact selected identity")
	for header, want := range map[string]string{
		"X-Search-Hub-Selected-Owner-Identity-Match":          "MATCHED",
		"X-Search-Hub-Selected-Owner-Status-RPC-Reachability": "REACHABLE",
		"X-Search-Hub-Selected-Owner-Endpoint-Reachability":   "REACHABLE",
		"X-Search-Hub-Selected-Owner-Freshness":               "CURRENT",
		"X-Search-Hub-Selected-Owner-Supported-Fallback":      "UNKNOWN",
	} {
		require.Equal(t, want, response.Header().Get(header), "header %s", header)
	}
}

func TestProviderStatusFailsClosedOnMismatchedReadinessIdentity(t *testing.T) {
	cases := []struct {
		name string
		resp *routingv1.ProviderStatusResponse
	}{
		{name: "response names another provider", resp: &routingv1.ProviderStatusResponse{ProviderId: "other", Health: &routingv1.ProviderHealth{ProviderId: "selected", Reachable: true, IndexAge: "5m"}}},
		{name: "health names another provider", resp: &routingv1.ProviderStatusResponse{ProviderId: "selected", Health: &routingv1.ProviderHealth{ProviderId: "other", Reachable: true, IndexAge: "5m"}}},
		{name: "health missing", resp: &routingv1.ProviderStatusResponse{ProviderId: "selected"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newClient(t, &fakeRouter{providerStatusOut: tc.resp})
			_, err := client.ProviderStatus(context.Background(), connect.NewRequest(&routingv1.ProviderStatusRequest{ProviderId: "selected"}))
			require.Error(t, err)
			require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
			require.NotContains(t, err.Error(), "other", "router internals and mismatched owner IDs stay out of caller-visible readiness")
		})
	}
}

func TestProviderStatusErrorsKeepCallerVersusOwnerStateDistinct(t *testing.T) {
	client := newClient(t, &fakeRouter{providerStatusErr: internalrouting.ErrInvalidProviderStatusRequest{}})
	_, err := client.ProviderStatus(context.Background(), connect.NewRequest(&routingv1.ProviderStatusRequest{}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	client = newClient(t, &fakeRouter{providerStatusErr: internalrouting.ErrProviderStatusNotFound{ProviderID: "missing"}})
	_, err = client.ProviderStatus(context.Background(), connect.NewRequest(&routingv1.ProviderStatusRequest{ProviderId: "missing"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestProviderStatusConnectProbesOnlyTheSelectedOwnerStatusRoute(t *testing.T) {
	selected := apiTestProvider("selected.skills", "skills", "selected owner", "selected-owner")
	selected.StatusEndpoint = &registryv1.Endpoint{Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{
		ScenarioId: "selected-owner", Path: "/status", Method: registryv1.HttpMethod_HTTP_METHOD_POST,
		BodyTemplate: "{}",
	}}}
	other := apiTestProvider("other.skills", "skills", "other owner", "other-owner")
	other.StatusEndpoint = &registryv1.Endpoint{Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{
		ScenarioId: "other-owner", Path: "/status", Method: registryv1.HttpMethod_HTTP_METHOD_POST,
		BodyTemplate: "{}",
	}}}
	doer := &apiTestDoer{}
	router := internalrouting.NewRouter(internalrouting.Deps{
		Lister:   apiTestLister{providers: []*registryv1.ProviderDescriptor{selected, other}},
		Resolver: apiTestResolver{},
		Doer:     doer,
	})
	client := newClient(t, router)

	resp, err := client.ProviderStatus(context.Background(), connect.NewRequest(&routingv1.ProviderStatusRequest{ProviderId: "selected.skills"}))
	require.NoError(t, err)
	require.Equal(t, "selected.skills", resp.Msg.GetProviderId())
	require.Equal(t, "selected.skills", resp.Msg.GetHealth().GetProviderId())
	require.True(t, resp.Msg.GetHealth().GetReachable())
	require.Equal(t, []string{"http://selected-owner.test/status"}, doer.urls)
}

func TestRepromoteForwardsProviderID(t *testing.T) {
	router := &fakeRouter{}
	client := newClient(t, router)

	resp, err := client.Repromote(context.Background(), connect.NewRequest(&routingv1.RepromoteRequest{
		ProviderId: "dense.docs",
	}))
	require.NoError(t, err)
	require.Equal(t, "dense.docs", router.repromoteID)
	require.Equal(t, "dense.docs", resp.Msg.GetProviderId())
	require.True(t, resp.Msg.GetReset_())
}

func TestRepromoteInternalErrorIsOpaque(t *testing.T) {
	router := &fakeRouter{repromoteErr: errors.New("registry db on fire: /var/lib/secret")}
	client := newClient(t, router)

	_, err := client.Repromote(context.Background(), connect.NewRequest(&routingv1.RepromoteRequest{
		ProviderId: "dense.docs",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.NotContains(t, err.Error(), "secret", "internal details must not leak to the client")
}
