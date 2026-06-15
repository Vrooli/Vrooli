package analysis

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"scenario-dependency-analyzer/internal/app/services"
	types "scenario-dependency-analyzer/internal/types"
)

type fakeAnalysisService struct {
	all map[string]*types.DependencyAnalysisResponse
	err error
}

func (f fakeAnalysisService) AnalyzeScenario(name string) (*types.DependencyAnalysisResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &types.DependencyAnalysisResponse{Scenario: name}, nil
}

func (f fakeAnalysisService) AnalyzeAllScenarios() (map[string]*types.DependencyAnalysisResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.all != nil {
		return f.all, nil
	}
	return map[string]*types.DependencyAnalysisResponse{
		"one": {Scenario: "one"},
	}, nil
}

type fakeScanService struct {
	result *services.ScanResult
	err    error
}

func (f fakeScanService) ScanScenario(name string, _ types.ScanRequest) (*services.ScanResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.result != nil {
		return f.result, nil
	}
	return &services.ScanResult{
		Analysis: &types.DependencyAnalysisResponse{Scenario: name},
		Applied:  true,
		ApplySummary: map[string]interface{}{
			"changed": true,
		},
	}, nil
}

func TestAnalyzeScenarioRoute(t *testing.T) {
	router := analysisTestRouter(fakeAnalysisService{}, fakeScanService{})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analyze/sample", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}

	var response types.DependencyAnalysisResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Scenario != "sample" {
		t.Fatalf("scenario = %q, want sample", response.Scenario)
	}
}

func TestAnalyzeScenarioNotFound(t *testing.T) {
	router := analysisTestRouter(fakeAnalysisService{err: errors.New("missing")}, fakeScanService{})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analyze/sample", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", recorder.Code, recorder.Body.String())
	}
}

func TestScanScenarioRoute(t *testing.T) {
	router := analysisTestRouter(fakeAnalysisService{}, fakeScanService{})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/sample/scan", strings.NewReader(`{"apply":true}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response["applied"] != true {
		t.Fatalf("applied = %v, want true", response["applied"])
	}
	if response["analysis"] == nil {
		t.Fatalf("expected analysis payload")
	}
}

func TestScanScenarioRejectsInvalidJSON(t *testing.T) {
	router := analysisTestRouter(fakeAnalysisService{}, fakeScanService{})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/sample/scan", strings.NewReader(`{`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", recorder.Code, recorder.Body.String())
	}
}

func analysisTestRouter(analysis services.AnalysisService, scans services.ScanService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterHTTPRoutes(router.Group("/api/v1"), analysis, scans)
	return router
}
