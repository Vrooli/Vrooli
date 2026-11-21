package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"landing-page-business-suite-api/internal/aidoor"
)

func TestAIDoorReceiptRouteRequiresIssuedBearerKey(t *testing.T) {
	server := &Server{router: mux.NewRouter()}
	registerAIDoorRoutes(server, &aidoor.Door{})
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/ai/receipt", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("registered recovery route status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unauthenticated receipt response is cacheable: %v", response.Header())
	}
}
