package resourceinventory

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
	"github.com/vrooli/vrooli/internal/resources/catalog"
)

func TestSearchRejectsInvalidRequestsAndUsesSafeEmptyResponse(t *testing.T) {
	for _, body := range []string{`{`, `{"query":"x","extra":1}`, `{"query":"x"} {"query":"y"}`, `{"query":"x","limit":-1}`, `{"query":" "}`} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/resources/inventory/search", strings.NewReader(body))
		testRouter(t.TempDir()).ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("body %q status = %d, want 400: %s", body, recorder.Code, recorder.Body.String())
		}
	}
}

func TestSearchMakesCatalogAndManifestFailuresExplicit(t *testing.T) {
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	root := t.TempDir()
	manifestDir := filepath.Join(root, "resources", "broken")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "resource.json"), []byte(`{"name":"broken","driver":`), 0o600); err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/resources/inventory/search", strings.NewReader(`{"query":"broken resource driver"}`))
	testRouter(root).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", recorder.Code, recorder.Body.String())
	}
	var response SearchResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Scan.Complete || response.Scan.ResourcesFailed != 1 || len(response.Degradations) != 1 || response.Degradations[0].Resource != "broken" {
		t.Fatalf("manifest failure was hidden: %+v", response)
	}

	catalogRoot := t.TempDir()
	configDir := filepath.Join(catalogRoot, ".vrooli")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "service.json"), []byte(`{"dependencies":`), 0o600); err != nil {
		t.Fatal(err)
	}
	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/resources/inventory/search", strings.NewReader(`{"query":"resource ports"}`))
	testRouter(catalogRoot).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("catalog status = %d: %s", recorder.Code, recorder.Body.String())
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Scan.Complete || len(response.Results) != 0 || len(response.Degradations) != 1 || response.Degradations[0].Code != "resource_catalog_unavailable" {
		t.Fatalf("catalog failure was hidden: %+v", response)
	}
}

func TestSearchReportsUnavailableRepositoryRoot(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/resources/inventory/search", strings.NewReader(`{"query":"resource ports"}`))
	testRouter("").ServeHTTP(recorder, request)
	var response SearchResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK || response.Scan.Complete || len(response.Degradations) != 1 || response.Degradations[0].Code != "repository_root_unavailable" {
		t.Fatalf("empty root response = status %d, %+v", recorder.Code, response)
	}
}

func TestProviderDirectExamplesAgainstCanonicalResourceCatalog(t *testing.T) {
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	if _, err := catalog.New(repoRoot).DiscoverManifestAnatomy(catalog.DiscoverOptions{}); err != nil {
		t.Fatalf("canonical catalog read: %v", err)
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
		ExpectedIDs       []string `json:"expect_ids"`
		ExpectNoStrongHit bool     `json:"expect_no_strong_hit"`
		ExpectMaxScore    float64  `json:"expect_max_score"`
	}
	for _, provider := range descriptor.Providers {
		if provider.ProviderID == "scenario-dependency-analyzer.resource-inventory" {
			cases = provider.Tests.Cases
			break
		}
	}
	if len(cases) < 5 {
		t.Fatalf("resource inventory cases = %d, want positive and negative coverage", len(cases))
	}
	for _, tc := range cases {
		t.Run(tc.ID, func(t *testing.T) {
			body, err := json.Marshal(SearchRequest{Query: tc.Query, Limit: maximumLimit})
			if err != nil {
				t.Fatal(err)
			}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/resources/inventory/search", strings.NewReader(string(body)))
			testRouter(repoRoot).ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d: %s", recorder.Code, recorder.Body.String())
			}
			var response SearchResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if !response.Scan.Complete || response.Scan.RuntimeStatusKnown {
				t.Fatalf("scan evidence = %+v; degradations=%+v", response.Scan, response.Degradations)
			}
			found := map[string]bool{}
			for _, result := range response.Results {
				found[result.ID] = true
				if tc.ExpectNoStrongHit && result.Score > tc.ExpectMaxScore {
					t.Fatalf("negative query returned strong hit: %+v", result)
				}
				if result.RuntimeStatusKnown {
					t.Fatalf("manifest result claimed runtime status: %+v", result)
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

func testRouter(root string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterHTTPRoutes(router.Group("/api/v1"), func() string { return root })
	return router
}
