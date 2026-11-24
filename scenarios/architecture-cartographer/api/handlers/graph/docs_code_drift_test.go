package graph_test

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	graphh "architecture-cartographer/handlers/graph"
	intdomains "architecture-cartographer/internal/domains"
	domainmocks "architecture-cartographer/internal/domains/mocks"
	"architecture-cartographer/internal/graph"
	graphmocks "architecture-cartographer/internal/graph/mocks"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestDocCodeDriftRouteReturnsJoinedAndNegativeCases(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o600))
	}
	write("docs/concepts/DOMAINS.md", "## Domain Inventory\n| Domain | Responsibility | Source Paths |\n|---|---|---|\n| billing | Own invoice behavior. | `api/internal/billing/` |\n")
	write("api/internal/billing/service.go", "package billing\n\nfunc Charge() {}\n")
	domainsSvc := &domainmocks.FakeService{Map: intdomains.DerivedDomainMap{Scenario: "demo", Authority: intdomains.SourceDomainsDoc, AuthorityConfidence: intdomains.ConfidenceHigh, Domains: []intdomains.DerivedDomain{{Name: "billing", Paths: []string{"api/internal/billing/"}, Responsibility: "Own invoice behavior.", Provenance: []intdomains.Source{intdomains.SourceDomainsDoc}}}}}
	api := graphh.ModuleWithLocator(&graphmocks.FakeService{}, domainsSvc, &domainmocks.FakeLocator{Dir: root})
	router := mux.NewRouter()
	api.Mount(router)
	positive := httptest.NewRequest("POST", "/api/v1/doc-code-drift/search", strings.NewReader(`{"query":"billing api source","scope":"scenario:demo","limit":5}`))
	positive.Header.Set("Content-Type", "application/json")
	got := httptest.NewRecorder()
	router.ServeHTTP(got, positive)
	require.Equal(t, 200, got.Code, got.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(got.Body.Bytes(), &body))
	rows, _ := body["results"].([]any)
	require.NotEmpty(t, rows)
	first := rows[0].(map[string]any)
	require.Equal(t, "STRUCTURAL_PATH_MATCH", first["classification"])
	negative := httptest.NewRequest("POST", "/api/v1/doc-code-drift/search", strings.NewReader(`{"query":"underwater penguin","scope":"scenario:demo","limit":5}`))
	negative.Header.Set("Content-Type", "application/json")
	miss := httptest.NewRecorder()
	router.ServeHTTP(miss, negative)
	require.Equal(t, 200, miss.Code, miss.Body.String())
	require.NoError(t, json.Unmarshal(miss.Body.Bytes(), &body))
	rows, _ = body["results"].([]any)
	require.Empty(t, rows)
}

var _ graph.Service = (*graphmocks.FakeService)(nil)
