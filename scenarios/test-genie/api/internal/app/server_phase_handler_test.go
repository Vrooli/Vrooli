package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"test-genie/internal/suite"
)

func TestHandleListPhases(t *testing.T) {
	tmp := t.TempDir()
	orchestrator, err := suite.NewSuiteOrchestrator(tmp)
	if err != nil {
		t.Fatalf("failed to initialize orchestrator: %v", err)
	}
	server := &Server{
		router:       mux.NewRouter(),
		orchestrator: orchestrator,
	}
	server.setupRoutes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/phases", nil)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var payload struct {
		Items []suite.PhaseDescriptor `json:"items"`
		Count int                     `json:"count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Count == 0 || len(payload.Items) == 0 {
		t.Fatalf("expected phase descriptors in response")
	}
}
