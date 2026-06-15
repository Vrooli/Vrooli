package deployment

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	types "scenario-dependency-analyzer/internal/types"
)

type fakeReporter struct {
	report      *types.DeploymentAnalysisReport
	err         error
	lastName    string
	lastRefresh bool
}

func (r *fakeReporter) GetDeploymentReport(name string, refresh bool) (*types.DeploymentAnalysisReport, error) {
	r.lastName = name
	r.lastRefresh = refresh
	if r.err != nil {
		return nil, r.err
	}
	return r.report, nil
}

func TestGetDeploymentReport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reporter := &fakeReporter{report: testDeploymentReport()}
	router := gin.New()
	RegisterHTTPRoutes(router.Group("/api/v1"), reporter)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/sample/deployment?refresh=true", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d. body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if reporter.lastName != "sample" {
		t.Fatalf("scenario = %q, want sample", reporter.lastName)
	}
	if !reporter.lastRefresh {
		t.Fatalf("expected refresh query to be parsed as true")
	}
}

func TestExportDAGFlattensNonRecursiveResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reporter := &fakeReporter{report: testDeploymentReport()}
	router := gin.New()
	RegisterHTTPRoutes(router.Group("/api/v1"), reporter)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/sample/dag/export?recursive=false", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d. body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Scenario  string                           `json:"scenario"`
		Recursive bool                             `json:"recursive"`
		DAG       []types.DeploymentDependencyNode `json:"dag"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Scenario != "sample" || response.Recursive {
		t.Fatalf("unexpected response metadata: %+v", response)
	}
	if len(response.DAG) != 1 {
		t.Fatalf("dag length = %d, want 1", len(response.DAG))
	}
	if len(response.DAG[0].Children) != 0 {
		t.Fatalf("expected non-recursive DAG to omit children, got %+v", response.DAG[0].Children)
	}
}

func TestExportDAGRejectsUnsupportedFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reporter := &fakeReporter{report: testDeploymentReport()}
	router := gin.New()
	RegisterHTTPRoutes(router.Group("/api/v1"), reporter)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/sample/dag/export?format=dot", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d. body: %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func TestGetBundleManifest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reporter := &fakeReporter{report: testDeploymentReport()}
	router := gin.New()
	RegisterHTTPRoutes(router.Group("/api/v1"), reporter)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/sample/bundle/manifest", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d. body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Scenario string               `json:"scenario"`
		Manifest types.BundleManifest `json:"manifest"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Scenario != "sample" {
		t.Fatalf("scenario = %q, want sample", response.Scenario)
	}
	if response.Manifest.Scenario != "sample" {
		t.Fatalf("manifest scenario = %q, want sample", response.Manifest.Scenario)
	}
}

func TestDeploymentAdapterMapsReporterError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reporter := &fakeReporter{err: errors.New("report unavailable")}
	router := gin.New()
	RegisterHTTPRoutes(router.Group("/api/v1"), reporter)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/sample/deployment", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d. body: %s", recorder.Code, http.StatusInternalServerError, recorder.Body.String())
	}
}

func testDeploymentReport() *types.DeploymentAnalysisReport {
	generatedAt := time.Date(2026, 6, 14, 21, 0, 0, 0, time.UTC)
	return &types.DeploymentAnalysisReport{
		Scenario:    "sample",
		GeneratedAt: generatedAt,
		Dependencies: []types.DeploymentDependencyNode{
			{
				Name: "postgres",
				Type: "resource",
				Children: []types.DeploymentDependencyNode{
					{Name: "postgres-image", Type: "artifact"},
				},
			},
		},
		BundleManifest: types.BundleManifest{
			Scenario:    "sample",
			GeneratedAt: generatedAt,
			Files: []types.BundleFileEntry{
				{Path: ".vrooli/service.json", Type: "manifest", Exists: true},
			},
		},
	}
}
