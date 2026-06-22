package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"landing-manager/errors"
	"landing-manager/internal/agentmanager"
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

	if h.AgentManager == nil {
		appErr := errors.NewAgentManagerError("connect", fmt.Errorf("agent-manager client not configured"))
		appErr.Message = "Agent runner is not available"
		appErr.Suggestion = "Ensure agent-manager is running. Start it with: vrooli scenario start agent-manager"
		h.RespondAppError(w, appErr)
		return
	}

	// Build the full customization prompt: brief + persona guidance + styling
	// guardrail + asset hints. This is the agent's complete instruction set.
	personaPrompt := ""
	if req.PersonaID != "" {
		persona, err := h.PersonaService.GetPersona(req.PersonaID)
		if err == nil {
			personaPrompt = fmt.Sprintf("\n\nPersona: %s\nGuidance:\n%s", persona.Name, persona.Prompt)
		}
	}

	prompt := fmt.Sprintf(
		"Requested customization for landing page scenario.\n\nScenario: %s\nBrief:\n%s\nAssets: %v\nPreview: %t%s\nSource: landing-manager factory\nTimestamp: %s\n\nExpected deliverables:\n- Apply brief to template-safe areas (content, design tokens, imagery)\n- Run A/B variant setup if applicable\n- Regenerate preview links and summarize changes\n- Return next steps and validation status.%s",
		req.ScenarioID, req.Brief, req.Assets, req.Preview, personaPrompt, time.Now().UTC().Format(time.RFC3339), buildStylingAppendix(req.ScenarioID),
	)

	title := fmt.Sprintf("Customize landing page scenario: %s", strings.TrimSpace(req.ScenarioID))

	runID, err := h.AgentManager.CreateRun(r.Context(), agentmanager.RunRequest{
		ScenarioID: req.ScenarioID,
		Title:      title,
		Prompt:     prompt,
	})
	if err != nil {
		appErr := errors.NewAgentManagerError("create run", err)
		appErr.Suggestion = "agent-manager may be temporarily unavailable. Try again in a few moments."
		h.RespondAppError(w, appErr)
		return
	}

	h.RespondJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":  "queued",
		"run_id":  runID,
		"message": "Customization run started",
	})
}

// HandleCustomizeStatus returns the current status of a customization run so the
// UI can poll progress after triggering a customize request.
func (h *Handler) HandleCustomizeStatus(w http.ResponseWriter, r *http.Request) {
	runID := strings.TrimSpace(mux.Vars(r)["run_id"])
	if runID == "" {
		appErr := errors.NewMissingFieldError("run_id")
		h.RespondAppError(w, appErr)
		return
	}

	if h.AgentManager == nil {
		appErr := errors.NewAgentManagerError("connect", fmt.Errorf("agent-manager client not configured"))
		appErr.Message = "Agent runner is not available"
		h.RespondAppError(w, appErr)
		return
	}

	run, err := h.AgentManager.GetRun(r.Context(), runID)
	if err != nil {
		appErr := errors.NewAgentManagerError("get run", err)
		h.RespondAppError(w, appErr)
		return
	}
	if run == nil {
		appErr := errors.NewValidationError("run_id", "run not found")
		appErr.Code = errors.ErrCodeScenarioNotFound
		h.RespondAppError(w, appErr)
		return
	}

	h.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"run_id": run.Id,
		"status": run.Status.String(),
	})
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
	b.WriteString("Reference docs: templates/scenarios/landing-page-react-vite/docs/DESIGN_SYSTEM.md\n")
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
