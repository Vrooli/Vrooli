package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestSetupRoutesKeepsDesktopBundleExportCompatibilitySeam(t *testing.T) {
	router := mux.NewRouter()
	registerBundleExportCompatibilityRoute(router, func(http.ResponseWriter, *http.Request) {})
	matched := false
	if err := router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		path, err := route.GetPathTemplate()
		if err == nil && path == "/api/v1/bundles/export" {
			matched = true
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Fatal("bundle export compatibility route is not registered")
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/bundles/export", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET bundle export status = %d, want 405", response.Code)
	}
}
