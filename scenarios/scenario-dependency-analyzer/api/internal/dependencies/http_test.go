package dependencies

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	types "scenario-dependency-analyzer/internal/types"
)

type fakeDependencyService struct {
	stored map[string][]types.ScenarioDependency
	report *types.DependencyImpactReport
	err    error
}

func (f fakeDependencyService) StoredDependencies(string) (map[string][]types.ScenarioDependency, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.stored, nil
}

func (f fakeDependencyService) DependencyImpact(name string) (*types.DependencyImpactReport, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.report != nil {
		return f.report, nil
	}
	return &types.DependencyImpactReport{DependencyName: name, TotalAffected: 1}, nil
}

func TestGetDependenciesRoute(t *testing.T) {
	router := dependenciesTestRouter(fakeDependencyService{
		stored: map[string][]types.ScenarioDependency{
			"resources": {{ScenarioName: "sample", DependencyName: "postgres"}},
		},
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/sample/dependencies", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response["scenario"] != "sample" {
		t.Fatalf("scenario = %v, want sample", response["scenario"])
	}
	if response["resources"] == nil {
		t.Fatalf("expected resources array")
	}
}

func TestGetDependencyImpactRoute(t *testing.T) {
	router := dependenciesTestRouter(fakeDependencyService{})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dependencies/postgres/impact", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}

	var response types.DependencyImpactReport
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.DependencyName != "postgres" {
		t.Fatalf("dependency name = %q, want postgres", response.DependencyName)
	}
}

func TestDependencyRouteErrors(t *testing.T) {
	router := dependenciesTestRouter(fakeDependencyService{err: errors.New("store unavailable")})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/sample/dependencies", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %s", recorder.Code, recorder.Body.String())
	}
}

func dependenciesTestRouter(dependencies Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterHTTPRoutes(router.Group("/api/v1"), dependencies)
	return router
}
