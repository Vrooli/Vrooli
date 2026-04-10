package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"brand-manager/handlers"
	"brand-manager/repository/mocks"
)

// Deeper tests for standards/audit-rules endpoint.
// [REQ:BM-REQ-API-STANDARDS] [REQ:BM-REQ-AUDIT-ENDPOINT] [REQ:BM-REQ-AUDIT-RULES]

func TestGetStandards_ResponseShape(t *testing.T) {
	h := handlers.New(&mocks.BrandRepository{}, &mocks.VersionRepository{}, &mocks.AssignmentRepository{})

	req := httptest.NewRequest("GET", "/api/v1/standards", nil)
	w := httptest.NewRecorder()
	h.GetStandards(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	// Verify Content-Type
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	// Verify the response is valid JSON with expected structure
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := raw["rules"]; !ok {
		t.Error("missing 'rules' key in response")
	}
	if _, ok := raw["count"]; !ok {
		t.Error("missing 'count' key in response")
	}
}

func TestGetStandards_RuleFields(t *testing.T) {
	h := handlers.New(&mocks.BrandRepository{}, &mocks.VersionRepository{}, &mocks.AssignmentRepository{})

	req := httptest.NewRequest("GET", "/api/v1/standards", nil)
	w := httptest.NewRecorder()
	h.GetStandards(w, req)

	var resp struct {
		Rules []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Severity    string `json:"severity"`
			Category    string `json:"category"`
		} `json:"rules"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	// Every rule must have all required fields populated
	for _, rule := range resp.Rules {
		if rule.ID == "" {
			t.Error("rule has empty ID")
		}
		if rule.Name == "" {
			t.Errorf("rule %s has empty Name", rule.ID)
		}
		if rule.Description == "" {
			t.Errorf("rule %s has empty Description", rule.ID)
		}
		if rule.Severity == "" {
			t.Errorf("rule %s has empty Severity", rule.ID)
		}
		validSeverities := map[string]bool{"error": true, "warning": true, "info": true}
		if !validSeverities[rule.Severity] {
			t.Errorf("rule %s has invalid severity %q, want error|warning|info", rule.ID, rule.Severity)
		}
		if rule.Category != "branding" {
			t.Errorf("rule %s has category %q, want 'branding'", rule.ID, rule.Category)
		}
	}
}

func TestGetStandards_CountMatchesArray(t *testing.T) {
	h := handlers.New(&mocks.BrandRepository{}, &mocks.VersionRepository{}, &mocks.AssignmentRepository{})

	req := httptest.NewRequest("GET", "/api/v1/standards", nil)
	w := httptest.NewRecorder()
	h.GetStandards(w, req)

	var resp struct {
		Rules []json.RawMessage `json:"rules"`
		Count int               `json:"count"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Count != len(resp.Rules) {
		t.Errorf("count = %d, array length = %d — mismatch", resp.Count, len(resp.Rules))
	}
}

func TestGetStandards_UniqueRuleIDs(t *testing.T) {
	h := handlers.New(&mocks.BrandRepository{}, &mocks.VersionRepository{}, &mocks.AssignmentRepository{})

	req := httptest.NewRequest("GET", "/api/v1/standards", nil)
	w := httptest.NewRecorder()
	h.GetStandards(w, req)

	var resp struct {
		Rules []struct {
			ID string `json:"id"`
		} `json:"rules"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	seen := map[string]bool{}
	for _, rule := range resp.Rules {
		if seen[rule.ID] {
			t.Errorf("duplicate rule ID: %s", rule.ID)
		}
		seen[rule.ID] = true
	}
}

func TestGetStandards_HasCriticalDisplayNameRule(t *testing.T) {
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

	// has-display-name must exist as the only "error" severity rule
	// (the most critical branding requirement)
	errorRules := 0
	found := false
	for _, rule := range resp.Rules {
		if rule.Severity == "error" {
			errorRules++
			if rule.ID == "has-display-name" {
				found = true
			}
		}
	}
	if !found {
		t.Error("has-display-name rule not found at error severity")
	}
	if errorRules != 1 {
		t.Errorf("expected exactly 1 error-severity rule, got %d", errorRules)
	}
}
