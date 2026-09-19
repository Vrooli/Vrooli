package capabilities

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// TestModuleServiceUnavailableWithoutRegistry pins the degraded path: a nil
// registry must answer 503 rather than panicking the whole API. This is the
// branch that runs when an optional dependency was never wired.
func TestModuleServiceUnavailableWithoutRegistry(t *testing.T) {
	router := mux.NewRouter()
	Module(nil).Mount(router)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/capabilities/describe", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

// TestModuleDeclaresItsEndpoints keeps the module's declared endpoint surface
// non-empty, since cli-health and ui-health discover commands and widgets from
// it and an empty declaration is indistinguishable from an absent feature.
func TestModuleDeclaresItsEndpoints(t *testing.T) {
	m := Module(nil)
	if m.Name != "capabilities" {
		t.Errorf("module name = %q, want %q", m.Name, "capabilities")
	}
	if len(m.Endpoints) == 0 {
		t.Error("module declares no endpoints; discovery surfaces will not see it")
	}
}
