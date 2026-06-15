package proposal

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

type fakeProposalService struct {
	err error
}

func (f fakeProposalService) AnalyzeProposedScenario(req types.ProposedScenarioRequest) (map[string]interface{}, error) {
	if f.err != nil {
		return nil, f.err
	}
	return map[string]interface{}{"name": req.Name}, nil
}

func TestAnalyzeProposedRoute(t *testing.T) {
	router := proposalTestRouter(fakeProposalService{})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analyze/proposed", strings.NewReader(`{"name":"new-scenario"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response["name"] != "new-scenario" {
		t.Fatalf("name = %v, want new-scenario", response["name"])
	}
}

func TestAnalyzeProposedRejectsInvalidJSON(t *testing.T) {
	router := proposalTestRouter(fakeProposalService{})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analyze/proposed", strings.NewReader(`{`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", recorder.Code, recorder.Body.String())
	}
}

func TestAnalyzeProposedServiceError(t *testing.T) {
	router := proposalTestRouter(fakeProposalService{err: errors.New("proposal unavailable")})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analyze/proposed", strings.NewReader(`{"name":"new-scenario"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %s", recorder.Code, recorder.Body.String())
	}
}

func proposalTestRouter(proposals interface {
	AnalyzeProposedScenario(types.ProposedScenarioRequest) (map[string]interface{}, error)
},
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterHTTPRoutes(router.Group("/api/v1"), proposals)
	return router
}
