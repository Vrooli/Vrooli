package handlers

import (
	"net/http"
)

// BrandingRule represents a single branding compliance rule for scenario-auditor integration.
// [REQ:BM-REQ-AUDIT-RULES] [REQ:BM-REQ-AUDIT-ENDPOINT]
type BrandingRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Category    string `json:"category"`
}

// standardRules defines the branding compliance rules reported to scenario-auditor.
// [REQ:BM-REQ-AUDIT-RULES]
var standardRules = []BrandingRule{
	{
		ID:          "has-logo",
		Name:        "Logo Present",
		Description: "Scenario has a logo asset assigned via brand-manager",
		Severity:    "warning",
		Category:    "branding",
	},
	{
		ID:          "has-favicon",
		Name:        "Favicon Present",
		Description: "Scenario has a favicon asset assigned via brand-manager",
		Severity:    "warning",
		Category:    "branding",
	},
	{
		ID:          "has-color-system",
		Name:        "Color System Defined",
		Description: "Scenario brand includes primary, background, surface, and text colors",
		Severity:    "warning",
		Category:    "branding",
	},
	{
		ID:          "has-display-name",
		Name:        "Display Name Set",
		Description: "Scenario brand includes a display name in identity",
		Severity:    "error",
		Category:    "branding",
	},
	{
		ID:          "has-typography",
		Name:        "Typography Defined",
		Description: "Scenario brand includes heading and body font definitions",
		Severity:    "info",
		Category:    "branding",
	},
}

// GetStandards handles GET /api/v1/standards.
// Returns the list of branding compliance rules for scenario-auditor integration.
// [REQ:BM-REQ-API-STANDARDS] [REQ:BM-REQ-AUDIT-ENDPOINT]
func (h *Handlers) GetStandards(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"rules": standardRules,
		"count": len(standardRules),
	})
}
