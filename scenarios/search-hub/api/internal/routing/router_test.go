package routing_test

import (
	"context"
	"errors"
	"io"
	"net/http"
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
	// No --all, no types, no group: Phase 4 has no classifier, so this is an
	// honest InvalidArgument rather than a silent widen.
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

func groupsByID(groups []*routingv1.ProviderResultGroup) map[string]*routingv1.ProviderResultGroup {
	out := make(map[string]*routingv1.ProviderResultGroup, len(groups))
	for _, g := range groups {
		out[g.GetProviderId()] = g
	}
	return out
}
