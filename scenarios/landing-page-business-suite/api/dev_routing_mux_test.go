package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestDevRoutingMuxMountRoutesServiceProcedures(t *testing.T) {
	router := mux.NewRouter()
	devRoutingMux{router: router}.Mount("/vrooli.dev_routing.v1.routing.RoutingService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/vrooli.dev_routing.v1.routing.RoutingService/InstallTestPool", nil))

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("mounted Connect procedure status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}
