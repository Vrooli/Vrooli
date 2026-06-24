package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
)

func TestMountDebugRoutesDevelopment(t *testing.T) {
	r := mux.NewRouter()
	mountDebugRoutes(&config.Config{Server: config.ServerConfig{Environment: "development"}}, r)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /debug/pprof/ = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMountDebugRoutesProduction(t *testing.T) {
	r := mux.NewRouter()
	mountDebugRoutes(&config.Config{Server: config.ServerConfig{Environment: "production"}}, r)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /debug/pprof/ = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestServerWriteTimeoutAllowsDevelopmentProfiles(t *testing.T) {
	dev := serverWriteTimeout(&config.Config{Server: config.ServerConfig{Environment: "development"}})
	if dev < 60*time.Second {
		t.Fatalf("development write timeout = %v, want at least 60s", dev)
	}

	prod := serverWriteTimeout(&config.Config{Server: config.ServerConfig{Environment: "production"}})
	if prod != 15*time.Second {
		t.Fatalf("production write timeout = %v, want 15s", prod)
	}
}
