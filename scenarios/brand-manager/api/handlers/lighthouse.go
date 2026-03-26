// Package handlers - Lighthouse WCAG accessibility audit integration.
// [REQ:BM-REQ-LIGHTHOUSE]
package handlers

import (
	"encoding/json"
	"net/http"

	"brand-manager/apierr"
	"brand-manager/domain"

	"github.com/gorilla/mux"
)

// LighthouseRequest specifies the scenario and optional URL to audit.
type LighthouseRequest struct {
	ScenarioName string `json:"scenario_name"`
	URL          string `json:"url,omitempty"` // if empty, resolve from scenario port
}

// LighthouseResult holds the accessibility audit outcome.
type LighthouseResult struct {
	Scenario     string            `json:"scenario"`
	BrandID      string            `json:"brand_id"`
	URL          string            `json:"url"`
	Score        float64           `json:"score"`     // 0-100
	Passed       bool              `json:"passed"`    // score >= threshold
	Threshold    float64           `json:"threshold"` // configurable pass threshold
	Violations   []AccessViolation `json:"violations,omitempty"`
	BrandRelated []AccessViolation `json:"brand_related,omitempty"` // violations caused by brand colors/typography
	Status       string            `json:"status"`                  // "completed", "pending", "error"
	ErrorMessage string            `json:"error_message,omitempty"`
}

// AccessViolation represents a single WCAG accessibility violation.
type AccessViolation struct {
	ID          string `json:"id"`
	Impact      string `json:"impact"` // "critical", "serious", "moderate", "minor"
	Description string `json:"description"`
	HelpURL     string `json:"help_url,omitempty"`
	Element     string `json:"element,omitempty"` // CSS selector of violating element
	BrandCaused bool   `json:"brand_caused"`      // true if violation relates to brand colors/fonts
}

// brandRelatedRuleIDs are axe/Lighthouse rule IDs that may be caused by brand choices.
var brandRelatedRuleIDs = map[string]bool{
	"color-contrast":          true,
	"color-contrast-enhanced": true,
	"link-in-text-block":      true,
	"meta-viewport":           true,
}

// defaultLighthouseThreshold is the minimum accessibility score to pass.
const defaultLighthouseThreshold = 90.0

// LighthouseAudit handles POST /api/v1/brands/{id}/lighthouse.
// It initiates a Lighthouse WCAG accessibility audit on the scenario where
// the brand has been applied. Returns brand-related violations separately.
// [REQ:BM-REQ-LIGHTHOUSE]
func (h *Handlers) LighthouseAudit(w http.ResponseWriter, r *http.Request) {
	brandID := mux.Vars(r)["id"]

	_, done := getOrNotFound(w, func() (*domain.Brand, error) {
		return h.brands.GetByID(r.Context(), brandID)
	}, "brand")
	if done {
		return
	}

	var req LighthouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.Write(w, apierr.Validation("invalid request body"))
		return
	}
	if req.ScenarioName == "" {
		apierr.Write(w, apierr.Validation("scenario_name is required"))
		return
	}

	// Resolve scenario URL if not provided
	url := req.URL
	if url == "" {
		url = "http://localhost" // placeholder - in production, resolve via vrooli scenario port
	}

	if isDryRun(r) {
		result := LighthouseResult{
			Scenario:  req.ScenarioName,
			BrandID:   brandID,
			URL:       url,
			Threshold: defaultLighthouseThreshold,
			Status:    "pending",
		}
		writeJSON(w, http.StatusOK, result)
		return
	}

	// In production, this would call test-genie's Lighthouse API.
	// For now, return a pending status indicating the audit needs to be triggered externally.
	// Future: POST to test-genie /api/v1/lighthouse with the scenario URL.
	result := LighthouseResult{
		Scenario:  req.ScenarioName,
		BrandID:   brandID,
		URL:       url,
		Threshold: defaultLighthouseThreshold,
		Status:    "pending",
	}

	writeJSON(w, http.StatusAccepted, result)
}

// ClassifyBrandViolations separates violations into brand-related and other.
func ClassifyBrandViolations(violations []AccessViolation) (brandRelated, other []AccessViolation) {
	for _, v := range violations {
		if brandRelatedRuleIDs[v.ID] {
			v.BrandCaused = true
			brandRelated = append(brandRelated, v)
		} else {
			other = append(other, v)
		}
	}
	return
}
