package dependencyrationale

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	repocontract "github.com/vrooli/repo-contract-go"
)

func TestSearchReturnsOwnerDeclarationRationaleAndDistinctEvidenceFields(t *testing.T) {
	root := t.TempDir()
	writeServiceManifest(t, root, "consumer", `{
  "dependencies": {"scenarios": {"provider": {
    "type":"scenario", "enabled":true, "required":false, "runtime_only":true,
    "description":"Generated Connect service boundary",
    "purpose":"Fetches current workflow evidence",
    "runtime_only_rationale":"This optional owner is reached over Connect and has no in-process import.",
    "degraded_behavior":"Return partial evidence when the owner is unavailable."
  }}}
}`)
	response := searchRequest(t, root, `{"query":"why Connect has no in process import","scenario":"consumer","limit":5}`)
	if !response.Scan.Complete || response.Scan.ScenariosScanned != 1 || response.Scan.ScenariosFailed != 0 {
		t.Fatalf("scan evidence = %+v", response.Scan)
	}
	if response.Scan.RuntimeStatusKnown {
		t.Fatal("manifest-only projection claimed runtime status")
	}
	if len(response.Results) != 1 {
		t.Fatalf("results = %+v, want one rationale", response.Results)
	}
	result := response.Results[0]
	if result.ID != "dependency-rationale:consumer:provider" || result.Consumer != "consumer" || result.Dependency != "provider" {
		t.Fatalf("identity = %+v", result)
	}
	if result.Evidence.RuntimeOnlyRationale == "" || result.Evidence.Description == "" || result.Evidence.Purpose == "" || result.Evidence.DegradedBehavior == "" {
		t.Fatalf("evidence fields were flattened or lost: %+v", result.Evidence)
	}
	if !strings.Contains(result.Snippet, "Connect") || result.Source.Owner != "consumer" || result.Source.Basis != "service_manifest.dependency_declaration" || !strings.HasPrefix(result.Source.Fingerprint, "sha256:") {
		t.Fatalf("snippet or provenance missing: %+v", result)
	}
	if result.Source.Path != "consumer/.vrooli/service.json" {
		t.Fatalf("source path = %q", result.Source.Path)
	}
}

func TestSearchDoesNotInventRationaleAndKeepsManifestFailuresVisible(t *testing.T) {
	root := t.TempDir()
	writeServiceManifest(t, root, "no-rationale", `{"dependencies":{"scenarios":{"provider":{"type":"scenario","enabled":true}}}}`)
	writeServiceManifest(t, root, "broken", `{ "dependencies": {`)
	response := searchRequest(t, root, `{"query":"why provider"}`)
	if response.Scan.Complete || response.Scan.ScenariosFailed != 1 || len(response.Degradations) != 1 || response.Degradations[0].Scenario != "broken" {
		t.Fatalf("partial source coverage was hidden: %+v", response)
	}
	if len(response.Results) != 0 {
		t.Fatalf("a graph/dependency edge was promoted to rationale: %+v", response.Results)
	}
}

func TestSearchLimitsScopeAndRejectsMalformedRequests(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"one", "two"} {
		writeServiceManifest(t, root, name, `{"dependencies":{"scenarios":{"provider":{"type":"scenario","enabled":true,"runtime_only":true,"runtime_only_rationale":"Uses a runtime API boundary and avoids in process imports."}}}}`)
	}
	response := searchRequest(t, root, `{"query":"runtime API boundary","scenario":"one"}`)
	if len(response.Results) != 1 || response.Results[0].Consumer != "one" || response.Scenario != "one" {
		t.Fatalf("scenario filter not honored: %+v", response)
	}

	for _, body := range []string{`{`, `{"query":"x","unknown":true}`, `{"query":"x"} {"query":"y"}`, `{"query":"x","limit":-1}`} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/dependencies/rationale/search", strings.NewReader(body))
		testRouter(root).ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("body %q status = %d, want 400: %s", body, recorder.Code, recorder.Body.String())
		}
	}
}

func TestSearchMakesUnknownAndMissingScopedManifestsExplicit(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "manifest-less"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		want string
	}{
		{name: "missing scenario", want: "scenario_not_found"},
		{name: "manifest-less", want: "manifest_unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := searchRequest(t, root, `{"query":"why dependency","scenario":"`+tc.name+`"}`)
			if response.Scan.Complete || len(response.Degradations) != 1 || response.Degradations[0].Code != tc.want {
				t.Fatalf("scope response = %+v; want explicit degradation %q", response, tc.want)
			}
		})
	}
}

func TestProviderDirectExamplesAgainstRepositoryDeclarations(t *testing.T) {
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	searchPath := filepath.Join(repoRoot, "scenarios", "scenario-dependency-analyzer", ".vrooli", "search.json")
	raw, err := os.ReadFile(searchPath)
	if err != nil {
		t.Fatalf("read provider descriptor: %v", err)
	}
	var descriptor struct {
		Providers []struct {
			ProviderID string `json:"provider_id"`
			Tests      struct {
				Cases []struct {
					ID                string   `json:"id"`
					Query             string   `json:"query"`
					Scope             string   `json:"scope"`
					ExpectedIDs       []string `json:"expect_ids"`
					ExpectNoStrongHit bool     `json:"expect_no_strong_hit"`
					ExpectMaxScore    float64  `json:"expect_max_score"`
				} `json:"cases"`
			} `json:"tests"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(raw, &descriptor); err != nil {
		t.Fatalf("decode provider descriptor: %v", err)
	}
	var cases []struct {
		ID                string   `json:"id"`
		Query             string   `json:"query"`
		Scope             string   `json:"scope"`
		ExpectedIDs       []string `json:"expect_ids"`
		ExpectNoStrongHit bool     `json:"expect_no_strong_hit"`
		ExpectMaxScore    float64  `json:"expect_max_score"`
	}
	for _, provider := range descriptor.Providers {
		if provider.ProviderID == "scenario-dependency-analyzer.dependency-rationale" {
			cases = provider.Tests.Cases
			break
		}
	}
	if len(cases) < 5 {
		t.Fatalf("dependency rationale provider cases = %d, want positive and negative coverage", len(cases))
	}
	for _, tc := range cases {
		t.Run(tc.ID, func(t *testing.T) {
			scenario := strings.TrimPrefix(tc.Scope, "scenario:")
			body, err := json.Marshal(SearchRequest{Query: tc.Query, Scenario: scenario, Limit: maximumLimit})
			if err != nil {
				t.Fatal(err)
			}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/dependencies/rationale/search", strings.NewReader(string(body)))
			testRouter(filepath.Join(repoRoot, "scenarios")).ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d: %s", recorder.Code, recorder.Body.String())
			}
			var response SearchResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			found := map[string]bool{}
			for _, result := range response.Results {
				found[result.ID] = true
				if tc.ExpectNoStrongHit && result.Score > tc.ExpectMaxScore {
					t.Fatalf("negative query returned strong hit: %+v", result)
				}
			}
			for _, id := range tc.ExpectedIDs {
				if !found[id] {
					t.Fatalf("expected %q, got results %+v (scan %+v)", id, response.Results, response.Scan)
				}
			}
		})
	}
}

func searchRequest(t *testing.T, root, body string) SearchResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/dependencies/rationale/search", strings.NewReader(body))
	testRouter(root).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	var response SearchResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return response
}

func testRouter(root string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterHTTPRoutes(router.Group("/api/v1"), func() string { return root })
	return router
}

func writeServiceManifest(t *testing.T, root, scenario, content string) {
	t.Helper()
	manifestDir := filepath.Join(root, scenario, ".vrooli")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "service.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
