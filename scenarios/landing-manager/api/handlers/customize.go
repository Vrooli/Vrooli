package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"landing-manager/errors"
	"landing-manager/util"
	"landing-manager/validation"
)

// HandleCustomize triggers agent-based customization for a scenario
func (h *Handler) HandleCustomize(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScenarioID string   `json:"scenario_id"`
		Brief      string   `json:"brief"`
		Assets     []string `json:"assets"`
		Preview    bool     `json:"preview"`
		PersonaID  string   `json:"persona_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appErr := errors.NewValidationError("request_body", "Invalid JSON in request body")
		appErr.Suggestion = "Ensure the request body is valid JSON with required fields: scenario_id, brief"
		h.RespondAppError(w, appErr)
		return
	}

	// Validate inputs
	if req.ScenarioID == "" {
		appErr := errors.NewMissingFieldError("scenario_id")
		h.RespondAppError(w, appErr)
		return
	}
	if err := validation.ValidateScenarioSlug(req.ScenarioID); err != nil {
		appErr := errors.NewValidationError("scenario_id", err.Error())
		h.RespondAppError(w, appErr)
		return
	}
	if err := validation.ValidateBrief(req.Brief); err != nil {
		appErr := errors.NewValidationError("brief", err.Error())
		appErr.Suggestion = "Provide a description of how you want to customize the landing page"
		h.RespondAppError(w, appErr)
		return
	}
	if err := validation.ValidatePersonaID(req.PersonaID); err != nil {
		appErr := errors.NewValidationError("persona_id", err.Error())
		appErr.Suggestion = "Use GET /personas to list available persona IDs"
		h.RespondAppError(w, appErr)
		return
	}

	issueTrackerBase, err := h.resolveIssueTrackerBase()
	if err != nil {
		appErr := errors.NewIssueTrackerError("connect", err)
		appErr.Message = "Issue tracker is not available"
		appErr.Suggestion = "Ensure app-issue-tracker is running. Start it with: vrooli scenario start app-issue-tracker"
		h.RespondAppError(w, appErr)
		return
	}

	issueTitle := fmt.Sprintf("Customize landing page scenario: %s", strings.TrimSpace(req.ScenarioID))
	if strings.TrimSpace(req.ScenarioID) == "" {
		issueTitle = "Customize landing page scenario (unnamed)"
	}

	// Build description with persona prompt if provided
	personaPrompt := ""
	if req.PersonaID != "" {
		persona, err := h.PersonaService.GetPersona(req.PersonaID)
		if err == nil {
			personaPrompt = fmt.Sprintf("\n\nPersona: %s\nGuidance:\n%s", persona.Name, persona.Prompt)
		}
	}

	description := fmt.Sprintf(
		"Requested customization for landing page scenario.\n\nScenario: %s\nBrief:\n%s\nAssets: %v\nPreview: %t%s\nSource: landing-manager factory\nTimestamp: %s\n\nExpected deliverables:\n- Apply brief to template-safe areas (content, design tokens, imagery)\n- Run A/B variant setup if applicable\n- Regenerate preview links and summarize changes\n- Return next steps and validation status.%s",
		req.ScenarioID, req.Brief, req.Assets, req.Preview, personaPrompt, time.Now().UTC().Format(time.RFC3339), buildStylingAppendix(req.ScenarioID),
	)

	issuePayload := map[string]interface{}{
		"title":       issueTitle,
		"description": description,
		"type":        "feature",
		"priority":    "high",
		"app_id":      "landing-manager",
		"tags":        []string{"landing-manager", "landing-page", "customization", "automation"},
		"environment": map[string]interface{}{
			"scenario_id":  req.ScenarioID,
			"template_id":  "saas-landing-page",
			"requested_by": "landing-manager",
			"preview_mode": req.Preview,
			"asset_hints":  req.Assets,
		},
	}

	issueResp := struct {
		Success bool                   `json:"success"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}{}

	if err := h.postJSON(issueTrackerBase+"/issues", issuePayload, &issueResp); err != nil {
		appErr := errors.NewIssueTrackerError("create issue", err)
		appErr.Suggestion = "The issue tracker may be temporarily unavailable. Try again in a few moments."
		h.RespondAppError(w, appErr)
		return
	}

	issueID := ""
	if issueResp.Data != nil {
		if v, ok := issueResp.Data["issue_id"].(string); ok {
			issueID = v
		}
		if issueID == "" {
			if nested, ok := issueResp.Data["issue"].(map[string]interface{}); ok {
				if v, ok := nested["id"].(string); ok {
					issueID = v
				}
			}
		}
	}

	// Trigger investigation
	investigation := struct {
		RunID   string `json:"run_id,omitempty"`
		Status  string `json:"status,omitempty"`
		AgentID string `json:"agent_id,omitempty"`
		Error   string `json:"error,omitempty"`
	}{}

	var investigationWarning string
	if issueID != "" {
		investigatePayload := map[string]interface{}{
			"issue_id": issueID,
			"priority": "high",
		}
		investigateResp := struct {
			Success bool                   `json:"success"`
			Data    map[string]interface{} `json:"data"`
		}{}
		if err := h.postJSON(issueTrackerBase+"/investigate", investigatePayload, &investigateResp); err == nil && investigateResp.Data != nil {
			if v, ok := investigateResp.Data["run_id"].(string); ok {
				investigation.RunID = v
			}
			if v, ok := investigateResp.Data["status"].(string); ok {
				investigation.Status = v
			}
			if v, ok := investigateResp.Data["agent_id"].(string); ok {
				investigation.AgentID = v
			}
		} else if err != nil {
			// Log the error but don't fail the request - the issue was created successfully
			h.Log("issue_tracker_investigate_failed", map[string]interface{}{"error": err.Error(), "issue_id": issueID})
			investigation.Status = "pending"
			investigation.Error = "Agent scheduling failed - issue created but investigation not started"
			investigationWarning = "The customization request was logged, but automatic agent investigation could not be triggered. You can manually trigger investigation from the issue tracker."
		}
	} else {
		investigationWarning = "Issue was created but no issue ID was returned. Agent investigation could not be triggered."
	}

	response := map[string]interface{}{
		"status":      "queued",
		"issue_id":    issueID,
		"tracker_url": issueTrackerBase,
		"agent":       investigation.AgentID,
		"run_id":      investigation.RunID,
		"message":     issueResp.Message,
	}

	// Surface warnings to user for graceful degradation
	if investigationWarning != "" {
		response["warning"] = investigationWarning
		response["status"] = "partially_queued"
	}

	h.RespondJSON(w, http.StatusAccepted, response)
}

func (h *Handler) resolveIssueTrackerBase() (string, error) {
	if raw := strings.TrimSpace(os.Getenv("APP_ISSUE_TRACKER_API_BASE")); raw != "" {
		return strings.TrimSuffix(raw, "/"), nil
	}

	if port := strings.TrimSpace(os.Getenv("APP_ISSUE_TRACKER_API_PORT")); port != "" {
		return fmt.Sprintf("http://localhost:%s/api/v1", port), nil
	}

	// Fallback: discover via CLI (not implemented here for simplicity)
	return "", fmt.Errorf("issue tracker not configured")
}

func (h *Handler) postJSON(url string, payload interface{}, out interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("issue tracker request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("issue tracker responded %d: %s", resp.StatusCode, string(body))
	}

	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

const stylingSnippetLimit = 6000

var antiSlopChecklist = []string{
	"Use the palette tokens exactly as defined (background, surface_primary, accent_primary) — no ad-hoc gradients or rainbow glass.",
	"Hero and story sections must include real artifacts (layered UI panels, brand strips, wireframes, download previews) instead of abstract blobs.",
	"Reuse the provided component kits (hero panels, process timeline, brand guidelines strip) to keep the layout coherent.",
	"CTA buttons stay pill-shaped, solid, and paired with outline/text variants. No dual-gradient fills.",
	"Alternate dense UI clusters with whitespace per layout.section_sequence to preserve the case-study cadence.",
	"Respect the typography scale pairings (Space Grotesk + Inter) unless you also update `typography.scale` and explain why.",
}

func buildStylingAppendix(scenarioID string) string {
	snippet, source := loadStylingSnippet(scenarioID)
	var b strings.Builder

	if snippet != "" {
		ref := source
		if ref == "" {
			ref = ".vrooli/styling.json"
		}
		b.WriteString("\n\nStyling guardrail (")
		b.WriteString(ref)
		b.WriteString("):\n```json\n")
		b.WriteString(snippet)
		b.WriteString("\n```\n")
	} else {
		b.WriteString("\n\nStyling guardrail: .vrooli/styling.json could not be attached automatically. Instruct the agent to read it from the scenario root.\n")
	}

	b.WriteString("Anti-slop checklist:\n")
	for _, item := range antiSlopChecklist {
		b.WriteString("- ")
		b.WriteString(item)
		b.WriteString("\n")
	}
	b.WriteString("Reference docs: scripts/scenarios/templates/landing-page-react-vite/docs/DESIGN_SYSTEM.md\n")
	return b.String()
}

func loadStylingSnippet(scenarioID string) (string, string) {
	for _, candidate := range candidateStylingPaths(scenarioID) {
		if candidate == "" {
			continue
		}

		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		trimmed := strings.TrimSpace(string(bytes.TrimSpace(data)))
		if trimmed == "" {
			continue
		}
		if len(trimmed) > stylingSnippetLimit {
			return trimmed[:stylingSnippetLimit] + "\n... (truncated)", candidate
		}
		return trimmed, candidate
	}
	return "", ""
}

func candidateStylingPaths(scenarioID string) []string {
	var paths []string
	if scenarioID != "" {
		if loc := util.ResolveScenarioPath(scenarioID); loc.Found {
			paths = append(paths, filepath.Join(loc.Path, ".vrooli", "styling.json"))
		}
	}

	vrooliRoot := strings.TrimSpace(util.GetVrooliRoot())
	if vrooliRoot != "" {
		templateRoot := filepath.Join(vrooliRoot, "scripts", "scenarios", "templates", "landing-page-react-vite", ".vrooli")
		paths = append(paths,
			filepath.Join(templateRoot, "styling.json"),
			filepath.Join(templateRoot, "style-packs", "clause-case-study.json"),
		)
	}

	return paths
}
