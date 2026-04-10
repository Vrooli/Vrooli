package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGraphEndpoint verifies graph API endpoint behavior. [REQ:KO-KG-002]
func TestGraphEndpoint(t *testing.T) {
	srv := newTestServerWithServices()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge/graph?center_concept=alpha", nil)
	w := httptest.NewRecorder()

	srv.handleGraph(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["center"] != "alpha" {
		t.Fatalf("expected center alpha, got %v", resp["center"])
	}
}
