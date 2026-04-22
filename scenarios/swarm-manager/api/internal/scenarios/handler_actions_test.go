package scenarios

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"swarm-manager/internal/testutil"

	"github.com/gorilla/mux"
)

func TestScenarioLifecycle_Start(t *testing.T) {
	root, sources := setupTestScenarios(t)
	lifecycle := &stubLifecycle{}
	handler := NewHandlerWithDeps(
		filepath.Join(root, "scenarios"),
		stubSource{scenarios: sources},
		lifecycle,
		stubCompleteness{scores: map[string]int{}},
	)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/test-scenario-1/start", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatusOK(t, rec)
	resp := testutil.DecodeJSON[scenarioResponse](t, rec)
	if resp.Scenario.Name != "test-scenario-1" {
		t.Fatalf("expected scenario test-scenario-1, got %q", resp.Scenario.Name)
	}
	if len(lifecycle.startCalls) != 1 || lifecycle.startCalls[0] != "test-scenario-1" {
		t.Fatalf("expected lifecycle start to be called with test-scenario-1")
	}
}

func TestScenarioLifecycle_Stop_NotFound(t *testing.T) {
	root, sources := setupTestScenarios(t)
	lifecycle := &stubLifecycle{}
	handler := NewHandlerWithDeps(
		filepath.Join(root, "scenarios"),
		stubSource{scenarios: sources},
		lifecycle,
		stubCompleteness{scores: map[string]int{}},
	)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/missing-scenario/stop", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatusNotFound(t, rec)
	if len(lifecycle.stopCalls) != 0 {
		t.Fatalf("expected lifecycle stop not to be called for missing scenario")
	}
}

func TestScenarioLifecycle_Restart_Error(t *testing.T) {
	root, sources := setupTestScenarios(t)
	lifecycle := &stubLifecycle{err: errScenarioNameRequired}
	handler := NewHandlerWithDeps(
		filepath.Join(root, "scenarios"),
		stubSource{scenarios: sources},
		lifecycle,
		stubCompleteness{scores: map[string]int{}},
	)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/test-scenario-2/restart", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatusBadRequest(t, rec)
	if len(lifecycle.restartCalls) != 1 {
		t.Fatalf("expected lifecycle restart to be called once")
	}
}
