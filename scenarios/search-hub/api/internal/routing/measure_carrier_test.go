package routing_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"

	"search-hub/internal/routing"
)

// measureQuery builds an explicit --type measure request (no classifier needed).
func measureQuery(text string) *routingv1.QueryRequest {
	return &routingv1.QueryRequest{Query: text, Types: []string{"measure"}, Limit: 10}
}

// referenceMeasureProvider is the Phase-3 "thin reference provider": a real HTTP
// endpoint that answers like the measures provider (measures-health, Phase 4)
// will, so search-hub's routing + measure carrier are testable end-to-end and
// independently of that scenario. It returns the fixed MeasureHit-contract JSON
// for three canonical cases keyed off the query text:
//   - a clean read-only measure → executed (answer + executed_query);
//   - a question missing a required param → needs[], no answer;
//   - a write measure → resolved params but never executed.
//
// The provider is intentionally deterministic (no LLM / qdrant) — the live
// match/extract/execute brain is exercised in packages/measures-go; here we
// prove the router carries the structured measure through, thinly.
func referenceMeasureProvider() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Query string `json:"query"`
		}
		_ = json.Unmarshal(body, &req)
		q := strings.ToLower(req.Query)

		var measure string
		switch {
		case strings.Contains(q, "archive"):
			measure = `{
				"measure_id":"backlog.archive","scenario":"swarm-manager",
				"params":{"window":"last_month"},"effect":"write","confidence":1.0
			}`
		case strings.Contains(q, "initiative"):
			measure = `{
				"measure_id":"backlog.completed","scenario":"swarm-manager",
				"needs":["initiative"],"effect":"read","confidence":1.0
			}`
		default:
			measure = `{
				"measure_id":"backlog.completed","scenario":"swarm-manager",
				"params":{"window":"this_week"},
				"answer":"42 backlog items completed (this_week)",
				"effect":"read","executed_query":"SELECT count(*) ...","confidence":1.0
			}`
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"results":[{"score":0.92,"measure":`+measure+`}]}`)
	}))
}

func measureProviderDescriptor() *registryv1.ProviderDescriptor {
	return &registryv1.ProviderDescriptor{
		ProviderId:    "measures-health.measures",
		ProviderGroup: "measures-health",
		Bucket:        registryv1.Bucket_BUCKET_STATE,
		Type:          "measure",
		Description:   "Analytical measures: how many / what's the rate / what's next.",
		State:         registryv1.ProviderState_PROVIDER_STATE_ACTIVE,
		Endpoint:      httpJSON("measures-health", "/api/v1/measures/query", `{"query":"{{query}}","limit":{{limit}}}`),
		ResultMapping: &registryv1.ResultMapping{
			ResultsPath:  "results",
			IdField:      "measure.measure_id",
			TitleField:   "measure.measure_id",
			SnippetField: "measure.answer",
			ScoreField:   "score",
			ScoreScale:   registryv1.ScoreScale_SCORE_SCALE_COSINE_0_1,
			MeasureField: "measure",
		},
	}
}

// routerOverMeasureProvider wires a real Router at the reference provider via an
// explicit --type measure selector (no classifier needed) and a real HTTP doer.
func routerOverMeasureProvider(t *testing.T, srv *httptest.Server) *routing.Router {
	t.Helper()
	return routing.NewRouter(routing.Deps{
		Lister:   &fakeLister{providers: []*registryv1.ProviderDescriptor{measureProviderDescriptor()}},
		Resolver: staticResolver{urls: map[string]string{"measures-health": srv.URL}},
		Doer:     srv.Client(),
	})
}

func TestRouter_CarriesExecutedMeasure(t *testing.T) {
	srv := referenceMeasureProvider()
	defer srv.Close()
	r := routerOverMeasureProvider(t, srv)

	resp, err := r.Query(context.Background(), measureQuery("how many backlog items did we complete this week"))
	require.NoError(t, err)
	require.Len(t, resp.GetGroups(), 1)
	require.Len(t, resp.GetGroups()[0].GetHits(), 1)

	m := resp.GetGroups()[0].GetHits()[0].GetMeasure()
	require.NotNil(t, m, "measure must carry end-to-end through the router")
	require.Equal(t, "backlog.completed", m.GetMeasureId())
	require.Equal(t, "swarm-manager", m.GetScenario())
	require.Equal(t, "42 backlog items completed (this_week)", m.GetAnswer())
	require.Equal(t, "SELECT count(*) ...", m.GetExecutedQuery())
	require.Equal(t, "this_week", m.GetParams()["window"])
	require.Empty(t, m.GetNeeds())
}

func TestRouter_CarriesNeedsMeasure(t *testing.T) {
	srv := referenceMeasureProvider()
	defer srv.Close()
	r := routerOverMeasureProvider(t, srv)

	resp, err := r.Query(context.Background(), measureQuery("completed work in the initiative"))
	require.NoError(t, err)
	m := resp.GetGroups()[0].GetHits()[0].GetMeasure()
	require.NotNil(t, m)
	require.Equal(t, []string{"initiative"}, m.GetNeeds())
	require.Empty(t, m.GetAnswer(), "an unresolved measure carries no answer through the router")
}

func TestRouter_NeverCarriesAutoExecutedWriteMeasure(t *testing.T) {
	srv := referenceMeasureProvider()
	defer srv.Close()
	r := routerOverMeasureProvider(t, srv)

	resp, err := r.Query(context.Background(), measureQuery("archive old backlog items"))
	require.NoError(t, err)
	m := resp.GetGroups()[0].GetHits()[0].GetMeasure()
	require.NotNil(t, m)
	require.Equal(t, "write", m.GetEffect())
	require.Empty(t, m.GetAnswer(), "a write measure must never be auto-executed")
	require.Equal(t, "last_month", m.GetParams()["window"])
}
