package catalog

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

type fakeScenarioService struct {
	summaries []types.ScenarioSummary
	detail    *types.ScenarioDetailResponse
	err       error
}

func (s fakeScenarioService) ListScenarios() ([]types.ScenarioSummary, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.summaries, nil
}

func (s fakeScenarioService) GetScenarioDetail(_ string) (*types.ScenarioDetailResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.detail, nil
}

func TestListScenarios(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), fakeScenarioService{
		summaries: []types.ScenarioSummary{
			{Name: "scenario-a", DisplayName: "Scenario A"},
		},
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d. body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response []types.ScenarioSummary
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(response) != 1 || response[0].Name != "scenario-a" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestGetScenarioDetailMapsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), fakeScenarioService{
		err: errors.New("scenario not found: missing-scenario"),
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/missing-scenario", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d. body: %s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
}

func TestGetScenarioDetailMapsUnexpectedError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), fakeScenarioService{
		err: errors.New("store unavailable"),
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/scenario-a", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d. body: %s", recorder.Code, http.StatusInternalServerError, recorder.Body.String())
	}
}
