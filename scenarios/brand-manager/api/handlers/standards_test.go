package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"brand-manager/handlers"
	"brand-manager/repository/mocks"
)

// [REQ:BM-REQ-API-STANDARDS] [REQ:BM-REQ-AUDIT-ENDPOINT] [REQ:BM-REQ-AUDIT-RULES]
func TestGetStandardsEndpoint(t *testing.T) {
	h := handlers.New(&mocks.BrandRepository{}, &mocks.VersionRepository{}, &mocks.AssignmentRepository{})

	req := httptest.NewRequest("GET", "/api/v1/standards", nil)
	w := httptest.NewRecorder()
	h.GetStandards(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Rules []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Severity string `json:"severity"`
			Category string `json:"category"`
		} `json:"rules"`
		Count int `json:"count"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Count != 5 {
		t.Errorf("expected 5 rules, got %d", resp.Count)
	}
	if len(resp.Rules) != 5 {
		t.Fatalf("expected 5 rules in array, got %d", len(resp.Rules))
	}

	// Verify required branding rules are present
	expectedIDs := map[string]bool{
		"has-logo":         false,
		"has-favicon":      false,
		"has-color-system": false,
		"has-display-name": false,
		"has-typography":   false,
	}
	for _, rule := range resp.Rules {
		if _, ok := expectedIDs[rule.ID]; ok {
			expectedIDs[rule.ID] = true
		}
		if rule.Category != "branding" {
			t.Errorf("rule %s: expected category 'branding', got %q", rule.ID, rule.Category)
		}
	}
	for id, found := range expectedIDs {
		if !found {
			t.Errorf("expected rule %q not found in response", id)
		}
	}
}

// [REQ:BM-REQ-AUDIT-RULES]
func TestStandardRuleSeverities(t *testing.T) {
	h := handlers.New(&mocks.BrandRepository{}, &mocks.VersionRepository{}, &mocks.AssignmentRepository{})

	req := httptest.NewRequest("GET", "/api/v1/standards", nil)
	w := httptest.NewRecorder()
	h.GetStandards(w, req)

	var resp struct {
		Rules []struct {
			ID       string `json:"id"`
			Severity string `json:"severity"`
		} `json:"rules"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	severityMap := map[string]string{}
	for _, r := range resp.Rules {
		severityMap[r.ID] = r.Severity
	}

	// has-display-name is critical (error), assets are warnings, typography is info
	if s := severityMap["has-display-name"]; s != "error" {
		t.Errorf("has-display-name severity: expected 'error', got %q", s)
	}
	if s := severityMap["has-logo"]; s != "warning" {
		t.Errorf("has-logo severity: expected 'warning', got %q", s)
	}
	if s := severityMap["has-typography"]; s != "info" {
		t.Errorf("has-typography severity: expected 'info', got %q", s)
	}
}
