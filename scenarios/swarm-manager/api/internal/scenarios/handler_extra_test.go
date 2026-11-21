package scenarios

import (
	"github.com/vrooli/api-core/apihttptest"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestHandler_LoadAllAndLoad(t *testing.T) {
	root, sources := setupTestScenarios(t)
	handler := newTestHandler(root, sources)

	all, err := handler.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll error: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 scenarios, got %d", len(all))
	}

	scenario, err := handler.Load("test-scenario-1")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if scenario.Name != "test-scenario-1" {
		t.Fatalf("expected name test-scenario-1, got %q", scenario.Name)
	}
	if scenario.DisplayName == "" {
		t.Fatalf("expected display name to be set")
	}
}

func TestHandler_RegisterRoutes(t *testing.T) {
	root, sources := setupTestScenarios(t)
	handler := newTestHandler(root, sources)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	apihttptest.AssertStatus(t, rec.Result(), 200)
	resp := apihttptest.MustDecodeJSON[listScenariosResponse](t, rec.Body.Bytes())
	if len(resp.Scenarios) != 3 {
		t.Fatalf("expected 3 scenarios, got %d", len(resp.Scenarios))
	}
}
