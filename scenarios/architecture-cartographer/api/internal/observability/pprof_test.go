package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterPprof_DisabledDoesNotMountRoutes(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPprof(mux, false)

	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	if resp.Code != http.StatusNotFound {
		t.Fatalf("disabled pprof status=%d want 404", resp.Code)
	}
}

func TestRegisterPprof_EnabledMountsRoutes(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPprof(mux, true)

	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	if resp.Code != http.StatusOK {
		t.Fatalf("enabled pprof status=%d want 200", resp.Code)
	}
}
