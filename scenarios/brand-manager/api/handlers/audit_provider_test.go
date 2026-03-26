package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"brand-manager/domain"
)

// [REQ:BM-REQ-AUDIT-PROVIDER] [REQ:BM-REQ-AUDIT-RULES] [REQ:BM-REQ-AUDIT-ENDPOINT]

func TestGetAuditRules(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	req := httptest.NewRequest("GET", "/api/v1/audit/rules", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Provider string `json:"provider"`
		Version  string `json:"version"`
		Rules    []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Severity string `json:"severity"`
			Category string `json:"category"`
		} `json:"rules"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Provider != "brand-manager" {
		t.Errorf("Provider = %q, want %q", resp.Provider, "brand-manager")
	}
	if len(resp.Rules) != 5 {
		t.Errorf("Rules count = %d, want 5", len(resp.Rules))
	}

	// Verify expected rule IDs
	expectedIDs := map[string]bool{
		"has-logo": true, "has-favicon": true, "has-color-system": true,
		"has-display-name": true, "has-typography": true,
	}
	for _, r := range resp.Rules {
		if !expectedIDs[r.ID] {
			t.Errorf("unexpected rule ID: %q", r.ID)
		}
	}
}

func TestEvaluateScenarioNoAssignment(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	req := httptest.NewRequest("POST", "/api/v1/audit/evaluate/unbranded-scenario", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Scenario string `json:"scenario"`
		Results  []struct {
			RuleID  string `json:"rule_id"`
			Pass    bool   `json:"pass"`
			Message string `json:"message"`
		} `json:"results"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Scenario != "unbranded-scenario" {
		t.Errorf("Scenario = %q, want %q", resp.Scenario, "unbranded-scenario")
	}

	for _, r := range resp.Results {
		if r.Pass {
			t.Errorf("rule %q should fail when no brand assigned", r.RuleID)
		}
	}
}

func TestEvaluateScenarioWithBrand(t *testing.T) {
	_, router, brandRepo, _, assignRepo := setupMockServer(t)

	brandRepo.Seed(&domain.Brand{
		ID:      "b1",
		Name:    "Full Brand",
		Version: 1,
		Identity: &domain.Identity{
			DisplayName: "Full Brand",
			LogoPath:    "/logo.png",
			FaviconPath: "/favicon.ico",
		},
		Colors: &domain.Colors{
			Primary:    "#000",
			Background: "#fff",
			Surface:    "#eee",
			Text:       "#111",
		},
		Typography: &domain.Typography{
			HeadingFont: "Inter",
			BodyFont:    "Open Sans",
		},
	})
	assignRepo.Create(nil, &domain.Assignment{
		ID:           "a1",
		BrandID:      "b1",
		ScenarioName: "branded-scenario",
		BrandVersion: 1,
	})

	req := httptest.NewRequest("POST", "/api/v1/audit/evaluate/branded-scenario", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Results []struct {
			RuleID string `json:"rule_id"`
			Pass   bool   `json:"pass"`
		} `json:"results"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	for _, r := range resp.Results {
		if !r.Pass {
			t.Errorf("rule %q should pass for fully branded scenario", r.RuleID)
		}
	}
}

// [REQ:BM-REQ-AUDIT-PROVIDER] Tests response Content-Type and interface contract fields
func TestGetAuditRulesResponseFormat(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	req := httptest.NewRequest("GET", "/api/v1/audit/rules", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify JSON content type
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	// Verify every rule has required fields for externalRuleProvider contract
	var resp struct {
		Provider string `json:"provider"`
		Version  string `json:"version"`
		Rules    []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Severity string `json:"severity"`
			Category string `json:"category"`
		} `json:"rules"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Provider == "" {
		t.Error("provider field must not be empty")
	}
	if resp.Version == "" {
		t.Error("version field must not be empty")
	}
	for _, r := range resp.Rules {
		if r.ID == "" {
			t.Error("rule ID must not be empty")
		}
		if r.Name == "" {
			t.Errorf("rule %q name must not be empty", r.ID)
		}
		if r.Severity == "" {
			t.Errorf("rule %q severity must not be empty", r.ID)
		}
		if r.Category == "" {
			t.Errorf("rule %q category must not be empty", r.ID)
		}
	}
}

func TestEvaluateScenarioPartialBrand(t *testing.T) {
	_, router, brandRepo, _, assignRepo := setupMockServer(t)

	// Brand with only display name (no logo, favicon, colors, typography)
	brandRepo.Seed(&domain.Brand{
		ID:      "b2",
		Name:    "Partial Brand",
		Version: 1,
		Identity: &domain.Identity{
			DisplayName: "Partial",
		},
	})
	assignRepo.Create(nil, &domain.Assignment{
		ID:           "a2",
		BrandID:      "b2",
		ScenarioName: "partial-scenario",
		BrandVersion: 1,
	})

	req := httptest.NewRequest("POST", "/api/v1/audit/evaluate/partial-scenario", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Results []struct {
			RuleID string `json:"rule_id"`
			Pass   bool   `json:"pass"`
		} `json:"results"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	// Only has-display-name should pass
	for _, r := range resp.Results {
		switch r.RuleID {
		case "has-display-name":
			if !r.Pass {
				t.Error("has-display-name should pass")
			}
		default:
			if r.Pass {
				t.Errorf("rule %q should fail for partial brand", r.RuleID)
			}
		}
	}
}
