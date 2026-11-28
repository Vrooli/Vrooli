package httpserver

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestHandleSearchScenarioRequirementsReturnsScopedProvenanceAndRecordedStatus(t *testing.T) {
	root := t.TempDir()
	demo := filepath.Join(root, "demo")
	other := filepath.Join(root, "demonstration")
	writeRequirementSearchFixture(t, demo, "demo", "DEMO-001", "Find the owner requirements search endpoint", "complete", "critical")
	writeRequirementSearchFixture(t, other, "demonstration", "OTHER-001", "Find the owner requirements search endpoint", "pending", "high")
	server := requirementsSearchTestServer(root)

	response := runRequirementsSearch(t, server, requirementsSearchRequest{Query: "owner requirements endpoint", Scenario: "demo", Limit: 3})
	if len(response.Results) != 2 {
		t.Fatalf("results = %d, want matching row and aggregate summary: %+v", len(response.Results), response.Results)
	}
	result := response.Results[0]
	if result.ResultType != "requirement" {
		t.Fatalf("highest-ranked result type = %q, want requirement", result.ResultType)
	}
	if result.ID != "DEMO-001" || result.Status != "complete" || result.LiveStatus != "passed" {
		t.Fatalf("requirement status was not preserved: %+v", result)
	}
	if response.Scope != "scenario:demo" {
		t.Fatalf("scope = %q, want exact scenario scope", response.Scope)
	}
	if result.Criticality != "critical" || result.Module != "01-search/module" || result.Path != "requirements/01-search/module.json" {
		t.Fatalf("requirement source was not preserved: %+v", result)
	}
	if result.Score <= 0 || result.Score > 1 || result.Source.Owner != "test-genie" || result.Source.Scenario != "demo" {
		t.Fatalf("invalid relevance/provenance: %+v", result)
	}
	if result.Source.CurrentRunVerified || !result.Source.LiveStatusKnown || result.Source.GeneratedAt.IsZero() {
		t.Fatalf("snapshot status was misrepresented: %+v", result.Source)
	}
	if result.Source.Freshness != "owner view generatedAt; this timestamp is not current test freshness" {
		t.Fatalf("freshness metadata = %q", result.Source.Freshness)
	}
	if response.Summary == nil || response.Summary.TotalRequirements != 1 {
		t.Fatalf("summary was not preserved: %+v", response.Summary)
	}
	if response.Results[1].ResultType != "summary" || response.Results[1].Source.CurrentRunVerified || response.Results[1].Summary == nil {
		t.Fatalf("aggregate summary provenance missing: %+v", response.Results[1])
	}
}

func TestSearchScenarioRequirementsIsMountedOnVersionedAPIRouter(t *testing.T) {
	root := t.TempDir()
	writeRequirementSearchFixture(t, filepath.Join(root, "demo"), "demo", "DEMO-001", "Search requirement status", "complete", "critical")
	server := requirementsSearchTestServer(root)
	server.router = mux.NewRouter()
	server.setupRoutes()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/scenario-requirements", bytes.NewBufferString(`{"query":"requirements status","scenario":"demo"}`))
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var response requirementsSearchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Scope != "scenario:demo" || len(response.Results) != 2 {
		t.Fatalf("mounted route response = %+v", response)
	}
}

func TestHandleSearchScenarioRequirementsCanMatchOnlyAggregateSummary(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "demo_app")
	writeRequirementSearchFixture(t, scenarioDir, "demo_app", "DEMO-001", "A title without state words", "complete", "critical")
	response := runRequirementsSearch(t, requirementsSearchTestServer(root), requirementsSearchRequest{Query: "requirements status", Scenario: "demo_app"})
	if response.Scope != "scenario:demo_app" || len(response.Results) != 1 || response.Results[0].ResultType != "summary" {
		t.Fatalf("aggregate-only scoped response = %+v", response)
	}
	if response.Results[0].Summary == nil || response.Results[0].Source.LiveStatusKnown || response.Results[0].Source.CurrentRunVerified {
		t.Fatalf("aggregate summary overstated evidence: %+v", response.Results[0])
	}
}

func TestHandleSearchScenarioRequirementsPreservesUnknownLiveStatus(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "demo")
	writeRequirementSearchFixture(t, scenarioDir, "demo", "DEMO-UNKNOWN", "Unobserved requirement", "unknown", "normal", false)
	response := runRequirementsSearch(t, requirementsSearchTestServer(root), requirementsSearchRequest{Query: "unobserved requirement", Scenario: "demo"})
	if len(response.Results) == 0 {
		t.Fatalf("results = %+v", response.Results)
	}
	if response.Results[0].LiveStatus != "unknown" || response.Results[0].Source.LiveStatusKnown {
		t.Fatalf("unknown owner status was not preserved: %+v", response.Results[0])
	}
}

func TestHandleSearchScenarioRequirementsEmptyAndUnrecognizedScopesAreExplicit(t *testing.T) {
	root := t.TempDir()
	server := requirementsSearchTestServer(root)
	for _, tc := range []struct {
		name     string
		scenario string
		want     string
	}{
		{name: "empty", want: "scenario_scope_required"},
		{name: "ambiguous substring", scenario: "demo-other", want: "scenario_not_found"},
		{name: "invalid scope", scenario: "demo other", want: "scenario_scope_invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := runRequirementsSearch(t, server, requirementsSearchRequest{Query: "requirements", Scenario: tc.scenario})
			if len(response.Results) != 0 || len(response.Degradations) != 1 || response.Degradations[0].Reason != tc.want || response.Degradations[0].Leg != "scope" {
				t.Fatalf("empty scope response = %+v; want degradation %q", response, tc.want)
			}
		})
	}
}

func TestHandleSearchScenarioRequirementsReportsUnavailableSource(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	response := runRequirementsSearch(t, requirementsSearchTestServer(root), requirementsSearchRequest{Query: "anything", Scenario: "demo"})
	if len(response.Results) != 0 || len(response.Degradations) != 1 || response.Degradations[0].Reason != "requirements_source_unavailable" || response.Degradations[0].Leg != "requirements" || !response.Degradations[0].Retryable {
		t.Fatalf("unavailable source response = %+v", response)
	}
}

func TestRequirementLexicalScoreIsDeterministicAndNotStatusBased(t *testing.T) {
	req := RequirementItem{ID: "REQ-1", Title: "Owner endpoint", Status: "pending", LiveStatus: "unknown", Description: "Search by scenario"}
	a := requirementLexicalScore("owner endpoint", req, "module")
	b := requirementLexicalScore("owner endpoint", req, "module")
	if a != 1 || a != b {
		t.Fatalf("score = %v then %v, want deterministic full lexical match", a, b)
	}
	req.Status = "complete"
	req.LiveStatus = "passed"
	if got := requirementLexicalScore("owner endpoint", req, "module"); got != a {
		t.Fatalf("status changed query relevance: before=%v after=%v", a, got)
	}
}

func requirementsSearchTestServer(root string) *Server {
	return &Server{
		scenarios: &stubScenarioDirectory{scenarioRoot: root},
		logger:    log.New(io.Discard, "", 0),
	}
}

func runRequirementsSearch(t *testing.T, server *Server, request requirementsSearchRequest) requirementsSearchResponse {
	t.Helper()
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/scenario-requirements", bytes.NewReader(body))
	response := httptest.NewRecorder()
	server.handleSearchScenarioRequirements(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var decoded requirementsSearchResponse
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return decoded
}

func writeRequirementSearchFixture(t *testing.T, scenarioDir, scenario, id, title, status, criticality string, withValidation ...bool) {
	t.Helper()
	requirementsDir := filepath.Join(scenarioDir, "requirements")
	moduleDir := filepath.Join(requirementsDir, "01-search")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	index := `{"imports":["01-search/module.json"]}`
	if err := os.WriteFile(filepath.Join(requirementsDir, "index.json"), []byte(index), 0o600); err != nil {
		t.Fatal(err)
	}
	requirement := map[string]any{
		"id": id, "title": title, "description": "Search endpoint owned by this scenario.", "status": status, "criticality": criticality,
	}
	includeValidation := len(withValidation) == 0 || withValidation[0]
	if includeValidation {
		requirement["validation"] = []map[string]any{{"type": "test", "ref": "test/requirements.sh", "status": "implemented"}}
	}
	module := map[string]any{"requirements": []map[string]any{requirement}}
	encoded, err := json.Marshal(module)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "module.json"), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	generatedAt := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	paths := newRequirementsPaths(scenarioDir)
	if err := os.MkdirAll(filepath.Dir(paths.snapshotPath), 0o755); err != nil {
		t.Fatal(err)
	}
	snapshot := map[string]any{
		"generated_at":  generatedAt,
		"scenario_name": scenario,
		"summary":       map[string]any{"total_requirements": 1},
		"modules":       []map[string]any{{"name": "01-search/module", "file_path": "requirements/01-search/module.json", "total": 1, "complete": 1}},
	}
	encoded, err = json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.snapshotPath, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
}
