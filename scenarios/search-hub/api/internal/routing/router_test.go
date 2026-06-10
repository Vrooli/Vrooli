package routing_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"

	internalregistry "search-hub/internal/registry"
	"search-hub/internal/routing"
)

// --- test seams -------------------------------------------------------------

// fakeLister returns a fixed provider set, recording the filter it was asked
// for so tests can assert the router lists only ACTIVE leaves.
type fakeLister struct {
	providers  []*registryv1.ProviderDescriptor
	err        error
	lastFilter internalregistry.ListFilter
}

func (f *fakeLister) List(_ context.Context, filter internalregistry.ListFilter) ([]*registryv1.ProviderDescriptor, error) {
	f.lastFilter = filter
	if f.err != nil {
		return nil, f.err
	}
	// Echo the ACTIVE-only contract the router relies on (real store filters in
	// SQL; the fake mirrors it so selection tests stay honest).
	if filter.State == int32(registryv1.ProviderState_PROVIDER_STATE_ACTIVE) {
		out := make([]*registryv1.ProviderDescriptor, 0, len(f.providers))
		for _, p := range f.providers {
			if p.GetState() == registryv1.ProviderState_PROVIDER_STATE_ACTIVE {
				out = append(out, p)
			}
		}
		return out, nil
	}
	return f.providers, nil
}

// staticResolver maps scenario_id → base URL (or an error to simulate a
// stopped/unreachable scenario).
type staticResolver struct {
	urls map[string]string
	errs map[string]error
}

func (s staticResolver) ResolveScenarioURL(_ context.Context, scenarioID string) (string, error) {
	if err, ok := s.errs[scenarioID]; ok {
		return "", err
	}
	if u, ok := s.urls[scenarioID]; ok {
		return u, nil
	}
	return "", errors.New("scenario not running")
}

// routeDoer answers by full request URL, so it is robust to the router's
// concurrent fan-out (unlike an ordered fake). A canned entry carries either a
// status+body or a transport error.
type routeDoer struct {
	byURL map[string]cannedResponse
}

type cannedResponse struct {
	status int
	body   string
	err    error
}

func (d routeDoer) Do(req *http.Request) (*http.Response, error) {
	key := req.URL.String()
	c, ok := d.byURL[key]
	if !ok {
		return nil, errors.New("routeDoer: no canned response for " + key)
	}
	if c.err != nil {
		return nil, c.err
	}
	return &http.Response{
		StatusCode: c.status,
		Body:       io.NopCloser(strings.NewReader(c.body)),
		Header:     make(http.Header),
	}, nil
}

// --- descriptor fixtures ----------------------------------------------------

func cliHealthCommands() *registryv1.ProviderDescriptor {
	return &registryv1.ProviderDescriptor{
		ProviderId:    "cli-health.commands",
		ProviderGroup: "cli-health",
		Bucket:        registryv1.Bucket_BUCKET_DO,
		Type:          "command",
		Description:   "CLI commands.",
		State:         registryv1.ProviderState_PROVIDER_STATE_ACTIVE,
		Endpoint: httpJSON("cli-health", "/vrooli.cli_health.v1.search.SearchService/Search",
			`{"query":"{{query}}","limit":{{limit}},"mode":"MODE_AI"}`),
		ResultMapping: &registryv1.ResultMapping{
			ResultsPath: "results", IdField: "name", TitleField: "name", ScoreField: "score",
			SnippetField: "description", PathField: "name",
			ScoreScale: registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1,
		},
	}
}

func swarmRecords() *registryv1.ProviderDescriptor {
	return &registryv1.ProviderDescriptor{
		ProviderId:    "swarm-manager.records",
		ProviderGroup: "swarm-manager",
		Bucket:        registryv1.Bucket_BUCKET_KNOW,
		Type:          "record",
		Description:   "Records.",
		State:         registryv1.ProviderState_PROVIDER_STATE_ACTIVE,
		Endpoint: httpJSON("swarm-manager", "/api/v1/search/ai",
			`{"query":"{{query}}","entity":"record","limit":{{limit}}}`),
		ResultMapping: &registryv1.ResultMapping{
			ResultsPath: "results", IdField: "id", TitleField: "payload.record_id", ScoreField: "score",
			SnippetField: "payload.scenario", PathField: "payload.record_id",
			ScoreScale: registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1,
		},
	}
}

// uiSurfacesFiltered is a synthetic multi-leaf descriptor used to exercise the
// filter_field/filter_value path end-to-end through the live fan-out.
func uiSurfacesFiltered() *registryv1.ProviderDescriptor {
	return &registryv1.ProviderDescriptor{
		ProviderId:    "ui-health.surfaces",
		ProviderGroup: "ui-health",
		Bucket:        registryv1.Bucket_BUCKET_REUSE,
		Type:          "component",
		Description:   "Surfaces.",
		State:         registryv1.ProviderState_PROVIDER_STATE_ACTIVE,
		Endpoint:      httpJSON("ui-health", "/vrooli.ui_health.v1.search.SearchService/Search", `{"query":"{{query}}","limit":{{limit}}}`),
		ResultMapping: &registryv1.ResultMapping{
			ResultsPath: "results", IdField: "filePath", TitleField: "displayName", ScoreField: "score",
			SnippetField: "description", PathField: "filePath",
			FilterField: "kind", FilterValue: "surface",
			ScoreScale: registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1,
		},
	}
}

func httpJSON(scenario, path, body string) *registryv1.Endpoint {
	return &registryv1.Endpoint{Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{
		ScenarioId:   scenario,
		Path:         path,
		Method:       registryv1.HttpMethod_HTTP_METHOD_POST,
		BodyTemplate: body,
		Headers:      map[string]string{"Content-Type": "application/json"},
	}}}
}

// --- tests ------------------------------------------------------------------

func TestQueryRejectsEmptyText(t *testing.T) {
	r := routing.NewRouter(routing.Deps{
		Lister: &fakeLister{}, Resolver: staticResolver{}, Doer: routeDoer{},
	})
	_, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "   ", All: true})
	require.ErrorAs(t, err, &routing.ErrInvalidQuery{})
}

func TestQueryRejectsNoSelector(t *testing.T) {
	r := routing.NewRouter(routing.Deps{
		Lister: &fakeLister{}, Resolver: staticResolver{}, Doer: routeDoer{},
	})
	// No --all, no types, no group, and no Classifier wired in Deps: this is an
	// honest InvalidArgument rather than a silent widen. (With a Classifier set,
	// the same request routes automatically — see TestAutoRouteUsesClassifierTypes.)
	_, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "restart a scenario"})
	require.ErrorAs(t, err, &routing.ErrInvalidQuery{})
}

func TestExplicitTypeFanOut(t *testing.T) {
	lister := &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands(), swarmRecords()}}
	resolver := staticResolver{urls: map[string]string{
		"cli-health":    "http://cli-health.test",
		"swarm-manager": "http://swarm-manager.test",
	}}
	doer := routeDoer{byURL: map[string]cannedResponse{
		"http://cli-health.test/vrooli.cli_health.v1.search.SearchService/Search": {status: 200, body: `{"results":[{"name":"scenario restart","description":"Restart a scenario","score":0.91}]}`},
		"http://swarm-manager.test/api/v1/search/ai":                              {status: 200, body: `{"results":[{"id":"pt-1","score":0.77,"payload":{"record_id":"rec-abc","scenario":"agent-manager"}}]}`},
	}}
	r := routing.NewRouter(routing.Deps{Lister: lister, Resolver: resolver, Doer: doer})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{
		Query: "restart a scenario", Types: []string{"command", "record"}, Limit: 5,
	})
	require.NoError(t, err)
	require.False(t, resp.GetReranked(), "pre-rerank: ranked list is empty, grouping only")
	require.False(t, resp.GetDegraded())
	require.Empty(t, resp.GetRanked())
	require.Equal(t, int32(registryv1.ProviderState_PROVIDER_STATE_ACTIVE), lister.lastFilter.State,
		"router must list only ACTIVE providers (gap stubs are not callable)")

	require.Len(t, resp.GetGroups(), 2)
	// Deterministic provider_id order.
	require.Equal(t, "cli-health.commands", resp.GetGroups()[0].GetProviderId())
	require.Equal(t, "swarm-manager.records", resp.GetGroups()[1].GetProviderId())
	require.Equal(t, []string{"cli-health.commands", "swarm-manager.records"}, resp.GetCorporaSearched())

	cli := resp.GetGroups()[0]
	require.Equal(t, int32(1), cli.GetCount())
	require.False(t, cli.GetDegraded())
	require.Equal(t, "scenario restart", cli.GetHits()[0].GetTitle())
	require.Equal(t, "cli-health", cli.GetHits()[0].GetProviderGroup())
	require.Equal(t, "command", cli.GetHits()[0].GetType())
	require.InDelta(t, 0.91, cli.GetHits()[0].GetScore(), 1e-9)

	sm := resp.GetGroups()[1]
	require.Equal(t, int32(1), sm.GetCount())
	require.Equal(t, "rec-abc", sm.GetHits()[0].GetTitle())
	require.Equal(t, "agent-manager", sm.GetHits()[0].GetSnippet())
	require.Equal(t, "pt-1", sm.GetHits()[0].GetId())
}

func TestDegradedProviderDoesNotFailQuery(t *testing.T) {
	lister := &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands(), swarmRecords()}}
	resolver := staticResolver{urls: map[string]string{
		"cli-health":    "http://cli-health.test",
		"swarm-manager": "http://swarm-manager.test",
	}}
	doer := routeDoer{byURL: map[string]cannedResponse{
		"http://cli-health.test/vrooli.cli_health.v1.search.SearchService/Search": {status: 200, body: `{"results":[{"name":"scenario logs","score":0.5}]}`},
		"http://swarm-manager.test/api/v1/search/ai":                              {status: 500, body: `boom`},
	}}
	r := routing.NewRouter(routing.Deps{Lister: lister, Resolver: resolver, Doer: doer})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "logs", All: true})
	require.NoError(t, err)
	require.True(t, resp.GetDegraded(), "one degraded provider flags the whole response degraded")

	bySvc := groupsByID(resp.GetGroups())
	require.False(t, bySvc["cli-health.commands"].GetDegraded())
	require.Equal(t, int32(1), bySvc["cli-health.commands"].GetCount())

	bad := bySvc["swarm-manager.records"]
	require.True(t, bad.GetDegraded())
	require.Contains(t, bad.GetNote(), "HTTP 500")
	require.Empty(t, bad.GetHits())
}

func TestUnreachableProviderDegrades(t *testing.T) {
	lister := &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands()}}
	resolver := staticResolver{errs: map[string]error{"cli-health": errors.New("scenario not running")}}
	r := routing.NewRouter(routing.Deps{Lister: lister, Resolver: resolver, Doer: routeDoer{}})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "x", All: true})
	require.NoError(t, err)
	require.True(t, resp.GetDegraded())
	require.Len(t, resp.GetGroups(), 1)
	require.True(t, resp.GetGroups()[0].GetDegraded())
	require.Contains(t, resp.GetGroups()[0].GetNote(), "unreachable")
}

func TestTransportErrorDegrades(t *testing.T) {
	lister := &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands()}}
	resolver := staticResolver{urls: map[string]string{"cli-health": "http://cli-health.test"}}
	doer := routeDoer{byURL: map[string]cannedResponse{
		"http://cli-health.test/vrooli.cli_health.v1.search.SearchService/Search": {err: errors.New("dial tcp: connection refused")},
	}}
	r := routing.NewRouter(routing.Deps{Lister: lister, Resolver: resolver, Doer: doer})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "x", All: true})
	require.NoError(t, err)
	require.True(t, resp.GetGroups()[0].GetDegraded())
	require.Contains(t, resp.GetGroups()[0].GetNote(), "request failed")
}

func TestFilterFieldAppliedThroughFanOut(t *testing.T) {
	lister := &fakeLister{providers: []*registryv1.ProviderDescriptor{uiSurfacesFiltered()}}
	resolver := staticResolver{urls: map[string]string{"ui-health": "http://ui-health.test"}}
	// Mixed kinds in one response; only kind=="surface" rows survive the filter.
	doer := routeDoer{byURL: map[string]cannedResponse{
		"http://ui-health.test/vrooli.ui_health.v1.search.SearchService/Search": {status: 200, body: `{"results":[
			{"displayName":"Button","filePath":"a.tsx","kind":"surface","score":0.8},
			{"displayName":"WidgetX","filePath":"w.tsx","kind":"widget","score":0.9},
			{"displayName":"Card","filePath":"b.tsx","kind":"surface","score":0.7}
		]}`},
	}}
	r := routing.NewRouter(routing.Deps{Lister: lister, Resolver: resolver, Doer: doer})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "button", Types: []string{"component"}, Limit: 10})
	require.NoError(t, err)
	g := resp.GetGroups()[0]
	require.Equal(t, int32(2), g.GetCount(), "the widget row is filtered out by filter_field=kind/filter_value=surface")
	require.Equal(t, "Button", g.GetHits()[0].GetTitle())
	require.Equal(t, "Card", g.GetHits()[1].GetTitle())
}

func TestGroupScopingLimitsToScenario(t *testing.T) {
	lister := &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands(), swarmRecords()}}
	resolver := staticResolver{urls: map[string]string{"swarm-manager": "http://swarm-manager.test"}}
	doer := routeDoer{byURL: map[string]cannedResponse{
		"http://swarm-manager.test/api/v1/search/ai": {status: 200, body: `{"results":[{"id":"pt-1","score":0.6,"payload":{"record_id":"rec-x","scenario":"s"}}]}`},
	}}
	r := routing.NewRouter(routing.Deps{Lister: lister, Resolver: resolver, Doer: doer})

	// --group with no narrower selector ⇒ every active leaf in that group only.
	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "x", Group: "swarm-manager"})
	require.NoError(t, err)
	require.Len(t, resp.GetGroups(), 1)
	require.Equal(t, "swarm-manager.records", resp.GetGroups()[0].GetProviderId())
}

func TestPerProviderLimitCapsHits(t *testing.T) {
	lister := &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands()}}
	resolver := staticResolver{urls: map[string]string{"cli-health": "http://cli-health.test"}}
	doer := routeDoer{byURL: map[string]cannedResponse{
		"http://cli-health.test/vrooli.cli_health.v1.search.SearchService/Search": {status: 200, body: `{"results":[
			{"name":"a","score":0.9},{"name":"b","score":0.8},{"name":"c","score":0.7}
		]}`},
	}}
	r := routing.NewRouter(routing.Deps{Lister: lister, Resolver: resolver, Doer: doer})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "x", All: true, Limit: 2})
	require.NoError(t, err)
	require.Equal(t, int32(2), resp.GetGroups()[0].GetCount(), "hits are capped to the per-provider limit")
}

func TestExplainListsSelectorAndProviders(t *testing.T) {
	lister := &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands()}}
	resolver := staticResolver{urls: map[string]string{"cli-health": "http://cli-health.test"}}
	doer := routeDoer{byURL: map[string]cannedResponse{
		"http://cli-health.test/vrooli.cli_health.v1.search.SearchService/Search": {status: 200, body: `{"results":[]}`},
	}}
	r := routing.NewRouter(routing.Deps{Lister: lister, Resolver: resolver, Doer: doer})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "x", Types: []string{"command"}, Explain: true})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetRoutingExplanation())
	joined := strings.Join(resp.GetRoutingExplanation(), "\n")
	require.Contains(t, joined, "command")
	require.Contains(t, joined, "cli-health.commands")
}

func TestListerErrorPropagates(t *testing.T) {
	lister := &fakeLister{err: errors.New("db down")}
	r := routing.NewRouter(routing.Deps{Lister: lister, Resolver: staticResolver{}, Doer: routeDoer{}})
	_, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "x", All: true})
	require.Error(t, err)
	require.NotErrorIs(t, err, routing.ErrInvalidQuery{}, "a registry failure is not a caller error")
}

// --- classifier (Phase 5 automatic routing) --------------------------------

// fakeClassifier records whether it was consulted and returns a canned result
// (or error). called lets a test assert the explicit-selector path bypasses it.
type fakeClassifier struct {
	result    routing.ClassifyResult
	err       error
	called    bool
	gotQuery  string
	gotalltyp []string
}

func (f *fakeClassifier) Classify(_ context.Context, query string, profiles []routing.ProviderProfile) (routing.ClassifyResult, error) {
	f.called = true
	f.gotQuery = query
	for _, p := range profiles {
		f.gotalltyp = append(f.gotalltyp, p.Type)
	}
	return f.result, f.err
}

func (f *fakeClassifier) Available(context.Context) bool { return f.err == nil }

func threeProviderLister() *fakeLister {
	return &fakeLister{providers: []*registryv1.ProviderDescriptor{
		cliHealthCommands(), uiSurfacesFiltered(), swarmRecords(),
	}}
}

func threeProviderResolver() staticResolver {
	return staticResolver{urls: map[string]string{
		"cli-health":    "http://cli-health.test",
		"ui-health":     "http://ui-health.test",
		"swarm-manager": "http://swarm-manager.test",
	}}
}

func threeProviderDoer() routeDoer {
	return routeDoer{byURL: map[string]cannedResponse{
		"http://cli-health.test/vrooli.cli_health.v1.search.SearchService/Search": {status: 200, body: `{"results":[{"name":"scenario restart","description":"Restart","score":0.9}]}`},
		"http://ui-health.test/vrooli.ui_health.v1.search.SearchService/Search":   {status: 200, body: `{"results":[{"displayName":"Settings","filePath":"s.tsx","kind":"surface","score":0.6}]}`},
		"http://swarm-manager.test/api/v1/search/ai":                              {status: 200, body: `{"results":[{"id":"pt-1","score":0.7,"payload":{"record_id":"rec-a","scenario":"x"}}]}`},
	}}
}

func TestAutoRouteUsesClassifierTypes(t *testing.T) {
	clf := &fakeClassifier{result: routing.ClassifyResult{Types: []string{"command"}, Confidence: 0.9, Rationale: "CLI op"}}
	r := routing.NewRouter(routing.Deps{
		Lister: threeProviderLister(), Resolver: threeProviderResolver(), Doer: threeProviderDoer(), Classifier: clf,
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "restart a scenario", Explain: true})
	require.NoError(t, err)
	require.True(t, clf.called)
	require.Equal(t, "restart a scenario", clf.gotQuery)
	require.Subset(t, clf.gotalltyp, []string{"command", "component", "record"}, "every active type's description is offered to the classifier")

	require.Len(t, resp.GetGroups(), 1, "confident single-type route hits only that provider")
	require.Equal(t, "cli-health.commands", resp.GetGroups()[0].GetProviderId())
	require.False(t, resp.GetDegraded())

	joined := strings.Join(resp.GetRoutingExplanation(), "\n")
	require.Contains(t, joined, "automatic routing via classifier")
	require.Contains(t, joined, "CLI op")
	require.Contains(t, joined, "routed to types: command")
	require.Contains(t, joined, "cli-health.commands")
}

func TestAutoRouteWidensOnLowConfidence(t *testing.T) {
	clf := &fakeClassifier{result: routing.ClassifyResult{Types: []string{"command"}, Confidence: 0.2}}
	r := routing.NewRouter(routing.Deps{
		Lister: threeProviderLister(), Resolver: threeProviderResolver(), Doer: threeProviderDoer(), Classifier: clf,
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "something vague", Explain: true})
	require.NoError(t, err)
	require.Len(t, resp.GetGroups(), 3, "uncertain ⇒ widen to every active provider (recall over precision)")
	require.False(t, resp.GetDegraded(), "widening is not degradation")
	require.Contains(t, strings.Join(resp.GetRoutingExplanation(), "\n"), "widened on uncertainty")
}

func TestAutoRouteClassifierErrorWidensAndFlagsDegraded(t *testing.T) {
	clf := &fakeClassifier{err: errors.New("model unreachable")}
	r := routing.NewRouter(routing.Deps{
		Lister: threeProviderLister(), Resolver: threeProviderResolver(), Doer: threeProviderDoer(), Classifier: clf,
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "anything", Explain: true})
	require.NoError(t, err, "a classifier failure never fails the query")
	require.Len(t, resp.GetGroups(), 3, "degrade ⇒ fall back to all active providers")
	require.True(t, resp.GetDegraded(), "classifier failure flags the response degraded")
	require.Contains(t, strings.Join(resp.GetRoutingExplanation(), "\n"), "classifier unavailable")
}

func TestExplicitSelectorBypassesClassifier(t *testing.T) {
	clf := &fakeClassifier{result: routing.ClassifyResult{Types: []string{"record"}, Confidence: 0.9}}
	r := routing.NewRouter(routing.Deps{
		Lister: threeProviderLister(), Resolver: threeProviderResolver(), Doer: threeProviderDoer(), Classifier: clf,
	})

	// --type given: the classifier must NOT be consulted.
	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "x", Types: []string{"command"}})
	require.NoError(t, err)
	require.False(t, clf.called, "explicit --type overrides the classifier")
	require.Len(t, resp.GetGroups(), 1)
	require.Equal(t, "cli-health.commands", resp.GetGroups()[0].GetProviderId())
}

func groupsByID(groups []*routingv1.ProviderResultGroup) map[string]*routingv1.ProviderResultGroup {
	out := make(map[string]*routingv1.ProviderResultGroup, len(groups))
	for _, g := range groups {
		out[g.GetProviderId()] = g
	}
	return out
}

// --- reranker (Phase 6 unified ranking) ------------------------------------

// fakeReranker records the candidates it was handed and orders them by a fixed
// score map keyed on hit id (or errors). It lets the router's rerank wiring be
// tested without a model.
type fakeReranker struct {
	scoreByID map[string]float64
	err       error
	called    bool
	gotQuery  string
	gotCount  int
}

func (f *fakeReranker) Rerank(_ context.Context, query string, candidates []*routingv1.SearchHit) ([]*routingv1.SearchHit, error) {
	f.called = true
	f.gotQuery = query
	f.gotCount = len(candidates)
	if f.err != nil {
		return nil, f.err
	}
	scores := make([]float64, len(candidates))
	for i, h := range candidates {
		scores[i] = f.scoreByID[h.GetId()]
	}
	return applyRerankForTest(candidates, scores), nil
}

func (f *fakeReranker) Available(context.Context) bool { return f.err == nil }

// applyRerankForTest mirrors the production applyRerank ordering (descending,
// stable) for the fake — the routing package's own applyRerank is unexported, so
// the external test reconstructs the contract it relies on.
func applyRerankForTest(candidates []*routingv1.SearchHit, scores []float64) []*routingv1.SearchHit {
	ranked := make([]*routingv1.SearchHit, len(candidates))
	copy(ranked, candidates)
	for i, h := range ranked {
		h.RerankScore = scores[i]
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].GetRerankScore() > ranked[j].GetRerankScore()
	})
	return ranked
}

func TestRerankProducesUnifiedRankedList(t *testing.T) {
	// swarm-manager's record (raw provider score 0.7) is the best answer; the
	// reranker must float it above cli-health's command (raw 0.9) — proving the
	// unified list is NOT just the per-provider scores interleaved.
	rr := &fakeReranker{scoreByID: map[string]float64{
		"scenario restart": 0.4, // cli-health hit (its PathField=name → id="scenario restart")
		"pt-1":             0.95,
	}}
	r := routing.NewRouter(routing.Deps{
		Lister:   &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands(), swarmRecords()}},
		Resolver: threeProviderResolver(),
		Doer:     threeProviderDoer(),
		Reranker: rr,
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "restart a scenario", All: true, Explain: true})
	require.NoError(t, err)
	require.True(t, rr.called)
	require.Equal(t, "restart a scenario", rr.gotQuery)
	require.Equal(t, 2, rr.gotCount, "every group's hit is fused into the shortlist")

	require.True(t, resp.GetReranked(), "a successful rerank flags the response reranked")
	require.False(t, resp.GetDegraded())
	require.Len(t, resp.GetRanked(), 2)
	require.Equal(t, "pt-1", resp.GetRanked()[0].GetId(), "highest rerank_score ranks first across providers")
	require.InDelta(t, 0.95, resp.GetRanked()[0].GetRerankScore(), 1e-9)

	// Groups stay populated for provenance.
	require.Len(t, resp.GetGroups(), 2)
	require.Contains(t, strings.Join(resp.GetRoutingExplanation(), "\n"), "reranked 2 candidate")
}

func TestRerankDegradesToGroupingOnError(t *testing.T) {
	rr := &fakeReranker{err: errors.New("model unreachable")}
	r := routing.NewRouter(routing.Deps{
		Lister:   &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands()}},
		Resolver: threeProviderResolver(),
		Doer:     threeProviderDoer(),
		Reranker: rr,
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "x", All: true, Explain: true})
	require.NoError(t, err, "a reranker failure never fails the query")
	require.False(t, resp.GetReranked(), "degraded ⇒ keep honest by-provider grouping")
	require.Empty(t, resp.GetRanked())
	require.True(t, resp.GetDegraded(), "reranker failure flags the response degraded")
	require.NotEmpty(t, resp.GetGroups(), "the grouping is still returned")
	require.Contains(t, strings.Join(resp.GetRoutingExplanation(), "\n"), "reranker unavailable")
}

func TestNoRerankerKeepsGroupingOnly(t *testing.T) {
	// No Reranker wired ⇒ the Phase-4/5 behavior is preserved exactly.
	r := routing.NewRouter(routing.Deps{
		Lister:   &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands()}},
		Resolver: threeProviderResolver(),
		Doer:     threeProviderDoer(),
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "x", All: true})
	require.NoError(t, err)
	require.False(t, resp.GetReranked())
	require.Empty(t, resp.GetRanked())
	require.False(t, resp.GetDegraded())
	require.NotEmpty(t, resp.GetGroups())
}

func TestRerankSkippedWhenNoHits(t *testing.T) {
	// All providers return empty: there is nothing to rerank, so reranked stays
	// false WITHOUT flagging degraded (an empty result set is not a failure).
	rr := &fakeReranker{scoreByID: map[string]float64{}}
	doer := routeDoer{byURL: map[string]cannedResponse{
		"http://cli-health.test/vrooli.cli_health.v1.search.SearchService/Search": {status: 200, body: `{"results":[]}`},
	}}
	r := routing.NewRouter(routing.Deps{
		Lister:   &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands()}},
		Resolver: staticResolver{urls: map[string]string{"cli-health": "http://cli-health.test"}},
		Doer:     doer,
		Reranker: rr,
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "x", All: true})
	require.NoError(t, err)
	require.False(t, rr.called, "reranker is not invoked when there are zero candidates")
	require.False(t, resp.GetReranked())
	require.False(t, resp.GetDegraded(), "an empty result set is not a degradation")
}

// --- scope-aware routing (external providers withheld from auto) ------------

// webSearchLive is a SCOPE_EXTERNAL provider (live web search). It must be
// withheld from automatic/classifier routing and reachable only via the
// explicit --all / --type web selectors.
func webSearchLive() *registryv1.ProviderDescriptor {
	return &registryv1.ProviderDescriptor{
		ProviderId:    "web-search.live",
		ProviderGroup: "web-search",
		Bucket:        registryv1.Bucket_BUCKET_KNOW,
		Type:          "web",
		Description:   "Live web search.",
		Scope:         registryv1.Scope_SCOPE_EXTERNAL,
		State:         registryv1.ProviderState_PROVIDER_STATE_ACTIVE,
		Endpoint: httpJSON("web-search", "/vrooli.web_search.v1.livesearch.LiveSearchService/Search",
			`{"query":"{{query}}","limit":{{limit}}}`),
		ResultMapping: &registryv1.ResultMapping{
			ResultsPath: "results", IdField: "url", TitleField: "title", ScoreField: "score",
			SnippetField: "snippet", PathField: "url",
			ScoreScale: registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1,
		},
	}
}

func TestAutoRouteWithholdsExternalScope(t *testing.T) {
	clf := &fakeClassifier{result: routing.ClassifyResult{Types: []string{"command"}, Confidence: 0.9}}
	lister := &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands(), webSearchLive()}}
	resolver := staticResolver{urls: map[string]string{
		"cli-health": "http://cli-health.test",
		"web-search": "http://web-search.test",
	}}
	doer := routeDoer{byURL: map[string]cannedResponse{
		"http://cli-health.test/vrooli.cli_health.v1.search.SearchService/Search": {status: 200, body: `{"results":[{"name":"scenario restart","description":"Restart","score":0.9}]}`},
	}}
	r := routing.NewRouter(routing.Deps{Lister: lister, Resolver: resolver, Doer: doer, Classifier: clf})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "restart a scenario", Explain: true})
	require.NoError(t, err)
	require.NotContains(t, clf.gotalltyp, "web", "an external provider's type is not offered to the classifier")
	require.Contains(t, resp.GetCorporaSearched(), "cli-health.commands")
	require.NotContains(t, resp.GetCorporaSearched(), "web-search.live", "automatic routing must never hit an external provider")
	require.Contains(t, strings.Join(resp.GetRoutingExplanation(), "\n"), "withheld 1 external")
}

func TestAutoRouteWidenStillExcludesExternalScope(t *testing.T) {
	// Even on the widen-on-low-confidence path, the external provider is never
	// in the candidate set, so widening cannot reach it.
	clf := &fakeClassifier{result: routing.ClassifyResult{Types: []string{"command"}, Confidence: 0.1}}
	lister := &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands(), webSearchLive()}}
	resolver := staticResolver{urls: map[string]string{
		"cli-health": "http://cli-health.test",
		"web-search": "http://web-search.test",
	}}
	doer := routeDoer{byURL: map[string]cannedResponse{
		"http://cli-health.test/vrooli.cli_health.v1.search.SearchService/Search": {status: 200, body: `{"results":[]}`},
	}}
	r := routing.NewRouter(routing.Deps{Lister: lister, Resolver: resolver, Doer: doer, Classifier: clf})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "something vague"})
	require.NoError(t, err)
	require.NotContains(t, resp.GetCorporaSearched(), "web-search.live")
}

func TestExplicitAllReachesExternalScope(t *testing.T) {
	lister := &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands(), webSearchLive()}}
	resolver := staticResolver{urls: map[string]string{
		"cli-health": "http://cli-health.test",
		"web-search": "http://web-search.test",
	}}
	doer := routeDoer{byURL: map[string]cannedResponse{
		"http://cli-health.test/vrooli.cli_health.v1.search.SearchService/Search":         {status: 200, body: `{"results":[]}`},
		"http://web-search.test/vrooli.web_search.v1.livesearch.LiveSearchService/Search": {status: 200, body: `{"results":[{"url":"https://x","title":"X","snippet":"s","score":0.5}]}`},
	}}
	r := routing.NewRouter(routing.Deps{Lister: lister, Resolver: resolver, Doer: doer})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "x", All: true})
	require.NoError(t, err)
	require.Contains(t, resp.GetCorporaSearched(), "web-search.live", "--all reaches external providers")
	require.Contains(t, resp.GetCorporaSearched(), "cli-health.commands")
}

func TestExplicitTypeWebReachesExternalScope(t *testing.T) {
	lister := &fakeLister{providers: []*registryv1.ProviderDescriptor{cliHealthCommands(), webSearchLive()}}
	resolver := staticResolver{urls: map[string]string{"web-search": "http://web-search.test"}}
	doer := routeDoer{byURL: map[string]cannedResponse{
		"http://web-search.test/vrooli.web_search.v1.livesearch.LiveSearchService/Search": {status: 200, body: `{"results":[{"url":"https://x","title":"X","snippet":"s","score":0.5}]}`},
	}}
	r := routing.NewRouter(routing.Deps{Lister: lister, Resolver: resolver, Doer: doer})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "x", Types: []string{"web"}})
	require.NoError(t, err)
	require.Equal(t, []string{"web-search.live"}, resp.GetCorporaSearched())
}
