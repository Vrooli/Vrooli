package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWatchdogTemplate(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/watchdog/template", nil)
	w := httptest.NewRecorder()

	h.WatchdogTemplate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("WatchdogTemplate() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check required fields
	if _, ok := resp["platform"]; !ok {
		t.Error("response should have platform field")
	}

	if _, ok := resp["template"]; !ok {
		t.Error("response should have template field")
	}

	if _, ok := resp["instructions"]; !ok {
		t.Error("response should have instructions field")
	}

	// Template should have substantial content
	template, ok := resp["template"].(string)
	if !ok || len(template) < 50 {
		t.Errorf("template should be a substantial string, got length %d", len(template))
	}
}

// --- Additional Handler Tests ---
