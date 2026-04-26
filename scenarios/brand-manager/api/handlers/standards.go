package handlers

import (
	"net/http"
)

// BrandingRule represents a single branding compliance rule for scenario-auditor integration.
// [REQ:BM-REQ-AUDIT-RULES] [REQ:BM-REQ-AUDIT-ENDPOINT]
type BrandingRule struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	Severity            string   `json:"severity"`
	Category            string   `json:"category"`
	TargetFiles         []string `json:"target_files"`
	DetailedDescription string   `json:"detailed_description"`
	PassingExample      string   `json:"passing_example"`
	FailingExample      string   `json:"failing_example"`
	FixInstructions     string   `json:"fix_instructions"`
	SeverityRationale   string   `json:"severity_rationale"`
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
		TargetFiles: []string{"ui/public/logo.*", "brand identity assets"},
		DetailedDescription: "Validates that the brand assigned to the scenario defines a non-empty " +
			"`identity.logo_path`. The audit checks the scenario's current brand assignment, then inspects " +
			"the brand record's `Identity.LogoPath` field. A missing or empty value fails the rule.",
		PassingExample: "```json\n{\n  \"identity\": {\n    \"logo_path\": \"/public/logo.svg\"\n  }\n}\n```",
		FailingExample: "```json\n{\n  \"identity\": {\n    \"logo_path\": \"\"\n  }\n}\n```",
		FixInstructions: "1. Add a logo asset under `ui/public/` (SVG preferred).\n" +
			"2. Edit the brand: set `Identity → Logo Path` to the asset's URL.\n" +
			"3. Re-run the audit to confirm the rule passes.",
		SeverityRationale: "**Warning** — a missing logo degrades brand presentation but does not block " +
			"core scenario functionality.",
	},
	{
		ID:          "has-favicon",
		Name:        "Favicon Present",
		Description: "Scenario has a favicon asset assigned via brand-manager",
		Severity:    "warning",
		Category:    "branding",
		TargetFiles: []string{"ui/public/favicon.*", "brand identity assets"},
		DetailedDescription: "Validates that the brand defines a non-empty `identity.favicon_path`. " +
			"Browsers and OS task switchers use this asset; without it the scenario renders with a generic icon.",
		PassingExample: "```json\n{\n  \"identity\": {\n    \"favicon_path\": \"/public/favicon.ico\"\n  }\n}\n```",
		FailingExample: "```json\n{\n  \"identity\": {\n    \"favicon_path\": \"\"\n  }\n}\n```",
		FixInstructions: "1. Place a `favicon.ico` (or `favicon.svg`) in `ui/public/`.\n" +
			"2. Set the brand's `Identity → Favicon Path` to the asset URL.",
		SeverityRationale: "**Warning** — visual polish only; functional flows still work without a favicon.",
	},
	{
		ID:          "has-color-system",
		Name:        "Color System Defined",
		Description: "Scenario brand includes primary, background, surface, and text colors",
		Severity:    "warning",
		Category:    "branding",
		TargetFiles: []string{"brand colors record", "ui/src/index.css (token consumers)"},
		DetailedDescription: "Validates that the brand's `Colors` block defines all four core tokens: " +
			"`primary`, `background`, `surface`, and `text`. Any missing value fails the rule because " +
			"theme generation cannot produce a complete palette without them.",
		PassingExample: "```json\n{\n  \"colors\": {\n    \"primary\": \"#4f46e5\",\n" +
			"    \"background\": \"#0f172a\",\n    \"surface\": \"#1e293b\",\n    \"text\": \"#f8fafc\"\n  }\n}\n```",
		FailingExample: "```json\n{\n  \"colors\": {\n    \"primary\": \"#4f46e5\",\n" +
			"    \"background\": \"\",\n    \"surface\": \"\",\n    \"text\": \"\"\n  }\n}\n```",
		FixInstructions: "1. Open the brand and fill `Colors → Primary / Background / Surface / Text`.\n" +
			"2. Run the contrast check to confirm WCAG AA pairs.\n" +
			"3. Re-evaluate the rule.",
		SeverityRationale: "**Warning** — a partial palette still renders, but theme tokens may fall back " +
			"to defaults inconsistently.",
	},
	{
		ID:          "has-display-name",
		Name:        "Display Name Set",
		Description: "Scenario brand includes a display name in identity",
		Severity:    "error",
		Category:    "branding",
		TargetFiles: []string{"brand identity record", "ui chrome (page titles, manifest)"},
		DetailedDescription: "Validates that the brand's `identity.display_name` is non-empty. The display " +
			"name surfaces in page titles, manifests, and any branded chrome — without it the scenario " +
			"appears unbranded to users.",
		PassingExample: "```json\n{\n  \"identity\": {\n    \"display_name\": \"Acme Brand Manager\"\n  }\n}\n```",
		FailingExample: "```json\n{\n  \"identity\": {\n    \"display_name\": \"\"\n  }\n}\n```",
		FixInstructions: "1. Edit the brand and set `Identity → Display Name`.\n" +
			"2. The name should match the user-facing product name, not the internal slug.",
		SeverityRationale: "**Error** — a missing display name is a visible identity defect; pages without " +
			"it look broken or unbranded to end users.",
	},
	{
		ID:          "has-typography",
		Name:        "Typography Defined",
		Description: "Scenario brand includes heading and body font definitions",
		Severity:    "info",
		Category:    "branding",
		TargetFiles: []string{"brand typography record"},
		DetailedDescription: "Validates that the brand's `Typography` block defines both `heading_font` " +
			"and `body_font`. Without these values the theme falls back to system defaults.",
		PassingExample: "```json\n{\n  \"typography\": {\n    \"heading_font\": \"Inter\",\n" +
			"    \"body_font\": \"Open Sans\"\n  }\n}\n```",
		FailingExample: "```json\n{\n  \"typography\": {\n    \"heading_font\": \"\",\n" +
			"    \"body_font\": \"\"\n  }\n}\n```",
		FixInstructions: "1. Set `Typography → Heading Font` and `Typography → Body Font` on the brand.\n" +
			"2. Make sure the font is loaded by the scenario UI (e.g., `<link>` in `index.html`).",
		SeverityRationale: "**Info** — system fallbacks render legibly; this is a polish-level rule.",
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
