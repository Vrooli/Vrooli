package module_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/module"
)

func TestHealthMountsBothGETRoutes(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	m := module.Health(handler, nil)
	if m.Name != "health" || m.Mount == nil {
		t.Fatalf("unexpected health module: %+v", m)
	}
	router := mux.NewRouter()
	m.Mount(router)

	for _, path := range []string{"/health", "/api/v1/health"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusNoContent {
			t.Errorf("GET %s status = %d, want %d", path, recorder.Code, http.StatusNoContent)
		}
	}
	for _, path := range []string{"/health", "/api/v1/health"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, nil))
		if recorder.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST %s status = %d, want %d", path, recorder.Code, http.StatusMethodNotAllowed)
		}
	}
}
