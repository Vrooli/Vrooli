package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestSemanticSearchStubIsRemoved(t *testing.T) {
	router := mux.NewRouter()
	registerSearchRoutes(router)
	request := httptest.NewRequest(http.MethodPost, "/api/search/semantic", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("removed semantic search endpoint status = %d, want %d", response.Code, http.StatusNotFound)
	}
}
