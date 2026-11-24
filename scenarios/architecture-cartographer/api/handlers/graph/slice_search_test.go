package graph_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	graphh "architecture-cartographer/handlers/graph"
	intdomains "architecture-cartographer/internal/domains"
	domainmocks "architecture-cartographer/internal/domains/mocks"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/graph/mocks"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestSearchSlicesHTTP_RanksWithinScenarioAndUsesCachedEvidence(t *testing.T) {
	snap := sampleSnap()
	domainsSvc := &domainmocks.FakeService{Map: intdomains.DerivedDomainMap{Scenario: "demo", Domains: []intdomains.DerivedDomain{
		{Name: "billing", Responsibility: "invoice billing and payment records", Purpose: "charge customers", Paths: []string{"api/internal/billing/**"}},
		{Name: "orders", Responsibility: "order fulfillment", Purpose: "track shipments", Paths: []string{"api/internal/orders/**"}},
	}}}
	svc := &mocks.FakeService{Snapshots: []graph.GraphSnapshot{snap}}
	h := graphh.NewHandler(svc, domainsSvc)
	req := httptest.NewRequest("POST", "/api/v1/slices/search", strings.NewReader(`{"query":"billing payment implementation","scope":"scenario:demo","limit":2}`))
	res := httptest.NewRecorder()
	h.SearchSlicesHTTP()(res, req)

	require.Equal(t, 200, res.Code)
	var body sliceSearchResponseFixture
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))
	require.Len(t, body.Results, 1)
	require.Equal(t, "demo/billing", body.Results[0].ID)
	require.Contains(t, body.Results[0].Text, "DERIVED/PARTIAL")
	require.Equal(t, "snap:demo:h1", body.Results[0].SnapshotID)
	require.Equal(t, int64(1), svc.ListCalls.Load())
	require.Zero(t, svc.ExtractCalls.Load(), "search must not trigger graph extraction")
	require.Equal(t, int64(1), domainsSvc.GetCalls.Load())
}

func TestSearchSlicesHTTP_ExplicitlyDegradesWhenSnapshotIsMissing(t *testing.T) {
	domainsSvc := &domainmocks.FakeService{Map: intdomains.DerivedDomainMap{Scenario: "demo"}}
	svc := &mocks.FakeService{}
	h := graphh.NewHandler(svc, domainsSvc)
	req := httptest.NewRequest("POST", "/api/v1/slices/search", strings.NewReader(`{"query":"billing","scope":"scenario:demo"}`))
	res := httptest.NewRecorder()
	h.SearchSlicesHTTP()(res, req)

	require.Equal(t, 200, res.Code)
	var body sliceSearchResponseFixture
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))
	require.Empty(t, body.Results)
	require.Equal(t, "graph_snapshot_unavailable", body.Degradations[0].Code)
	require.Zero(t, svc.ExtractCalls.Load())
}

func TestSearchSlicesHTTP_RejectsOutOfScopeAndUnboundedRequests(t *testing.T) {
	h := graphh.NewHandler(&mocks.FakeService{}, &domainmocks.FakeService{})
	for _, body := range []string{
		`{"query":"billing","scope":"global"}`,
		`{"query":"billing","scope":"scenario:demo","limit":6}`,
	} {
		req := httptest.NewRequest("POST", "/api/v1/slices/search", strings.NewReader(body))
		res := httptest.NewRecorder()
		h.SearchSlicesHTTP()(res, req)
		require.Equal(t, 400, res.Code)
	}
	svc := &mocks.FakeService{Snapshots: []graph.GraphSnapshot{{ID: "snap:other:h1", Scenario: "other"}}}
	h = graphh.NewHandler(svc, &domainmocks.FakeService{})
	req := httptest.NewRequest("POST", "/api/v1/slices/search", strings.NewReader(`{"query":"billing","scope":"scenario:demo","snapshot_id":"snap:other:h1"}`))
	res := httptest.NewRecorder()
	h.SearchSlicesHTTP()(res, req)
	require.Equal(t, 400, res.Code)
}

func TestSearchSlicesHTTP_IsMountedAtDocumentedPath(t *testing.T) {
	module := graphh.Module(&mocks.FakeService{}, &domainmocks.FakeService{})
	router := mux.NewRouter()
	module.Mount(router)
	req := httptest.NewRequest("POST", "/api/v1/slices/search", strings.NewReader(`{"query":"billing","scope":"scenario:demo"}`))
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	require.Equal(t, 200, res.Code)
	require.Contains(t, res.Body.String(), "graph_snapshot_unavailable")
}

type sliceSearchResponseFixture struct {
	Results []struct {
		ID         string `json:"id"`
		Text       string `json:"text"`
		SnapshotID string `json:"snapshot_id"`
	} `json:"results"`
	Degradations []struct {
		Code string `json:"code"`
	} `json:"degradations"`
}
