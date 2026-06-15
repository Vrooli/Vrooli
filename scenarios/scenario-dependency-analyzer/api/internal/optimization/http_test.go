package optimization

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	types "scenario-dependency-analyzer/internal/types"
)

type fakeOptimizationService struct {
	err error
}

func (f fakeOptimizationService) RunOptimization(req types.OptimizationRequest) (map[string]*types.OptimizationResult, error) {
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
