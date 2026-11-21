package optimization

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

type fakeOptimizationService struct {
	err     error
	request *types.OptimizationRequest
}

func (f fakeOptimizationService) RunOptimization(req types.OptimizationRequest) (map[string]*types.OptimizationResult, error) {
	if f.request != nil {
		*f.request = req
	}
	if f.err != nil {
		return nil, f.err
	}
	scenario := req.Scenario
	if scenario == "" {
		scenario = "all"
	}
	return map[string]*types.OptimizationResult{
		scenario: {Scenario: scenario},
	}, nil
}

func TestSelectedOptimizationRejectsMissingScopeAndPreservesSelection(t *testing.T) {
	for _, body := range []string{`{}`, `{"scenario":"sample"}`, `{"scenario":"all","only_scenarios":["old"]}`, `{"scenario":"sample","only_scenarios":[""]}`, `{"scenario":"sample","only_scenarios":["../old"]}`} {
		captured := types.OptimizationRequest{}
		router := optimizationTestRouter(fakeOptimizationService{request: &captured})
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/optimize-scenarios", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || captured.Scenario != "" {
			t.Fatalf("unbounded request admitted: %s status=%d captured=%+v", body, response.Code, captured)
		}
	}
	captured := types.OptimizationRequest{}
	router := optimizationTestRouter(fakeOptimizationService{request: &captured})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/optimize-scenarios", strings.NewReader(`{"scenario":"sample","only_scenarios":["old"],"apply":true,"type":"dependency_reduction"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !captured.Apply || captured.Scenario != "sample" || captured.Type != "dependency_reduction" || len(captured.OnlyScenarios) != 1 || captured.OnlyScenarios[0] != "old" {
		t.Fatalf("scope lost: status=%d request=%+v", response.Code, captured)
	}
}

func TestOptimizeRoute(t *testing.T) {
	router := optimizationTestRouter(fakeOptimizationService{})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/optimize", strings.NewReader(`{"scenario":"sample"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response["results"] == nil || response["generated_at"] == nil {
		t.Fatalf("expected results and generated_at, got %+v", response)
	}
}

func TestOptimizeRejectsInvalidJSON(t *testing.T) {
	router := optimizationTestRouter(fakeOptimizationService{})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/optimize", strings.NewReader(`{`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", recorder.Code, recorder.Body.String())
	}
}

func TestOptimizeServiceError(t *testing.T) {
	router := optimizationTestRouter(fakeOptimizationService{err: errors.New("no workspace")})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/optimize", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %s", recorder.Code, recorder.Body.String())
	}
}

func optimizationTestRouter(optimization interface {
	RunOptimization(types.OptimizationRequest) (map[string]*types.OptimizationResult, error)
},
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterHTTPRoutes(router.Group("/api/v1"), optimization)
	return router
}
