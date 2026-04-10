// Package handlers - agent-assisted branding application via agent-manager API.
// [REQ:BM-REQ-AGENT-SPAWN] [REQ:BM-REQ-AGENT-INSTRUCT] [REQ:BM-REQ-AGENT-VALIDATE]
package handlers

import (
	"encoding/json"
	"net/http"

	"brand-manager/apierr"
	"brand-manager/domain"

	"github.com/gorilla/mux"
)

// AgentApplyRequest specifies the brand to apply and the target scenario
// for agent-assisted application.
type AgentApplyRequest struct {
	ScenarioName string   `json:"scenario_name"`
	Elements     []string `json:"elements,omitempty"` // empty = all
	Prompt       string   `json:"prompt,omitempty"`   // additional instructions for the agent
}

// AgentApplyResult reports the outcome of an agent-assisted application.
type AgentApplyResult struct {
	Scenario     string   `json:"scenario"`
	BrandID      string   `json:"brand_id"`
	BrandVersion int      `json:"brand_version"`
	AgentID      string   `json:"agent_id,omitempty"` // populated after spawn
	Status       string   `json:"status"`             // "pending", "running", "completed", "failed"
	Elements     []string `json:"elements"`
	Instructions string   `json:"instructions"` // generated agent instructions with marker mandates
	DryRun       bool     `json:"dry_run,omitempty"`
}

// AgentValidateRequest asks the system to validate that markers are present
// after an agent has applied branding.
type AgentValidateRequest struct {
	ScenarioName string   `json:"scenario_name"`
	Elements     []string `json:"elements,omitempty"` // which elements to check
}

// AgentValidateResult reports marker validation results post-agent application.
type AgentValidateResult struct {
	Scenario   string             `json:"scenario"`
	Valid      bool               `json:"valid"`
	Expected   []string           `json:"expected"` // markers that should be present
	Found      []string           `json:"found"`    // markers actually found
	Missing    []string           `json:"missing"`  // markers not found
	ScanReport *domain.ScanReport `json:"scan_report,omitempty"`
}

// markerElements maps brand elements to expected CSS marker names.
var markerElements = map[string][]string{
	"colors":     {"primary", "secondary", "accent", "background", "surface", "text"},
	"typography": {"heading-font", "body-font", "mono-font", "base-font-size"},
	"identity":   {"display-name", "tagline"},
	"favicon":    {"favicon"},
	"logo":       {"logo"},
}

// BuildAgentInstructions generates agent instructions that mandate inline marker usage.
// [REQ:BM-REQ-AGENT-INSTRUCT]
func BuildAgentInstructions(brand *domain.Brand, elements []string, extraPrompt string) string {
	instructions := "Apply branding to the scenario using inline markers.\n\n"
	instructions += "MANDATORY: All brand values MUST be applied using inline markers.\n"
	instructions += "CSS custom properties must use /* brand-manager:<element> */ comments.\n"
	instructions += "JSON values must use \"_brand<key>\" key prefixes.\n\n"

	instructions += "Brand: " + brand.Name + "\n"

	if brand.Colors != nil {
		instructions += "\nColors:\n"
		if brand.Colors.Primary != "" {
			instructions += "  primary: " + brand.Colors.Primary + " (marker: /* brand-manager:primary */)\n"
		}
		if brand.Colors.Secondary != "" {
			instructions += "  secondary: " + brand.Colors.Secondary + " (marker: /* brand-manager:secondary */)\n"
		}
		if brand.Colors.Accent != "" {
			instructions += "  accent: " + brand.Colors.Accent + " (marker: /* brand-manager:accent */)\n"
		}
		if brand.Colors.Background != "" {
			instructions += "  background: " + brand.Colors.Background + " (marker: /* brand-manager:background */)\n"
		}
		if brand.Colors.Surface != "" {
			instructions += "  surface: " + brand.Colors.Surface + " (marker: /* brand-manager:surface */)\n"
		}
		if brand.Colors.Text != "" {
			instructions += "  text: " + brand.Colors.Text + " (marker: /* brand-manager:text */)\n"
		}
	}

	if brand.Typography != nil {
		instructions += "\nTypography:\n"
		if brand.Typography.HeadingFont != "" {
			instructions += "  heading-font: " + brand.Typography.HeadingFont + " (marker: /* brand-manager:heading-font */)\n"
		}
		if brand.Typography.BodyFont != "" {
			instructions += "  body-font: " + brand.Typography.BodyFont + " (marker: /* brand-manager:body-font */)\n"
		}
	}

	instructions += "\nElements to apply: "
	for i, e := range elements {
		if i > 0 {
			instructions += ", "
		}
		instructions += e
	}
	instructions += "\n"

	if extraPrompt != "" {
		instructions += "\nAdditional instructions: " + extraPrompt + "\n"
	}

	return instructions
}

// ExpectedMarkers returns the marker names expected for the given elements.
func ExpectedMarkers(elements []string) []string {
	var markers []string
	seen := map[string]bool{}
	for _, elem := range elements {
		for _, m := range markerElements[elem] {
			if !seen[m] {
				markers = append(markers, m)
				seen[m] = true
			}
		}
	}
	return markers
}

// AgentApply handles POST /api/v1/brands/{id}/agent-apply.
// It prepares agent instructions with mandatory marker constraints and returns
// the instructions (dry-run) or spawns an agent via agent-manager API.
// [REQ:BM-REQ-AGENT-SPAWN] [REQ:BM-REQ-AGENT-INSTRUCT]
func (h *Handlers) AgentApply(w http.ResponseWriter, r *http.Request) {
	brandID := mux.Vars(r)["id"]

	brand, done := getOrNotFound(w, func() (*domain.Brand, error) {
		return h.brands.GetByID(r.Context(), brandID)
	}, "brand")
	if done {
		return
	}

	var req AgentApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.Write(w, apierr.Validation("invalid request body"))
		return
	}
	if req.ScenarioName == "" {
		apierr.Write(w, apierr.Validation("scenario_name is required"))
		return
	}

	elements := req.Elements
	if len(elements) == 0 {
		elements = allApplyElements
	}
	for _, e := range elements {
		if !isValidElement(e) {
			apierr.Write(w, apierr.Validation("unknown element: "+e))
			return
		}
	}

	instructions := BuildAgentInstructions(brand, elements, req.Prompt)

	result := AgentApplyResult{
		Scenario:     req.ScenarioName,
		BrandID:      brandID,
		BrandVersion: brand.Version,
		Status:       "pending",
		Elements:     elements,
		Instructions: instructions,
		DryRun:       isDryRun(r),
	}

	if isDryRun(r) {
		writeJSON(w, http.StatusOK, result)
		return
	}

	// In production, this would call the agent-manager API to spawn a sandboxed agent.
	// For now, return the instructions with pending status so the caller can spawn manually.
	// Future: POST to agent-manager /api/v1/agents with instructions payload.
	result.Status = "pending"
	writeJSON(w, http.StatusAccepted, result)
}

// AgentValidate handles POST /api/v1/brands/{id}/agent-validate.
// It scans the scenario for inline markers and validates that all expected
// markers are present after an agent has applied branding.
// [REQ:BM-REQ-AGENT-VALIDATE]
func (h *Handlers) AgentValidate(w http.ResponseWriter, r *http.Request) {
	_ = mux.Vars(r)["id"] // brand ID for context (future: verify markers match brand values)

	var req AgentValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.Write(w, apierr.Validation("invalid request body"))
		return
	}
	if req.ScenarioName == "" {
		apierr.Write(w, apierr.Validation("scenario_name is required"))
		return
	}

	scenarioDir, done := h.resolveScenarioDir(w, req.ScenarioName)
	if done {
		return
	}

	elements := req.Elements
	if len(elements) == 0 {
		elements = allApplyElements
	}

	// Scan the scenario for existing markers
	report := domain.ScanReport{Scenario: req.ScenarioName}
	walkScenarioDir(scenarioDir, func(path, relPath, ext string) {
		var results []domain.ScanResult
		switch ext {
		case ".css", ".scss", ".less":
			results = scanFileForCSS(path, relPath)
			report.CSSMarkers += len(results)
		case ".json":
			results = scanFileForJSON(path, relPath)
			report.JSONKeys += len(results)
		}
		report.Results = append(report.Results, results...)
	})
	report.Total = report.CSSMarkers + report.JSONKeys

	// Build found markers set from scan results
	foundSet := map[string]bool{}
	for _, r := range report.Results {
		foundSet[r.Element] = true
	}

	expected := ExpectedMarkers(elements)
	var found, missing []string
	for _, m := range expected {
		if foundSet[m] {
			found = append(found, m)
		} else {
			missing = append(missing, m)
		}
	}

	result := AgentValidateResult{
		Scenario:   req.ScenarioName,
		Valid:      len(missing) == 0,
		Expected:   expected,
		Found:      found,
		Missing:    missing,
		ScanReport: &report,
	}

	writeJSON(w, http.StatusOK, result)
}

// isValidElement checks if the element name is recognized.
func isValidElement(name string) bool {
	for _, e := range allApplyElements {
		if e == name {
			return true
		}
	}
	return false
}
