package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"landing-manager/errors"
	"landing-manager/validation"
)

// HandleGenerate creates a new landing page scenario from a template
func (h *Handler) HandleGenerate(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	var req struct {
		TemplateID string                 `json:"template_id"`
		Name       string                 `json:"name"`
		Slug       string                 `json:"slug"`
		Options    map[string]interface{} `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate all inputs
	if err := validation.ValidateTemplateID(req.TemplateID); err != nil {
		h.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validation.ValidateScenarioName(req.Name); err != nil {
		h.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validation.ValidateScenarioSlug(req.Slug); err != nil {
		h.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	response, err := h.Generator.GenerateScenario(req.TemplateID, req.Name, req.Slug, req.Options)
	durationMs := time.Since(startTime).Milliseconds()

	// Determine if this was a dry-run
	isDryRun := false
	if req.Options != nil {
		if dr, ok := req.Options["dry_run"].(bool); ok {
			isDryRun = dr
		}
	}

	if err != nil {
		// Record failed generation
		if h.AnalyticsService != nil {
			h.AnalyticsService.RecordGeneration(req.TemplateID, req.Slug, isDryRun, false, err.Error(), durationMs)
		}

		// Create structured error with helpful context
		appErr := errors.NewGenerationError(req.TemplateID, req.Slug, err)

		// Provide specific suggestions based on error type
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "template") && strings.Contains(errMsg, "not found"):
			appErr = errors.NewNotFoundError("template", req.TemplateID)
			appErr.Suggestion = "Check the template ID and try again. Use GET /templates to list available templates."
		case strings.Contains(errMsg, "already exists"):
			appErr = errors.NewConflictError(req.Slug, "A scenario with this slug already exists")
			appErr.Suggestion = "Choose a different slug or delete the existing scenario first."
		case strings.Contains(errMsg, "permission"):
			appErr.Suggestion = "Check file system permissions for the generated directory."
		default:
			appErr.Suggestion = "Check the template configuration and try with different parameters."
		}

		h.RespondAppError(w, appErr)
		return
	}

	// Record successful generation
	if h.AnalyticsService != nil {
		h.AnalyticsService.RecordGeneration(req.TemplateID, req.Slug, isDryRun, true, "", durationMs)
	}

	h.RespondJSON(w, http.StatusCreated, response)
}

// HandleGeneratedList returns all generated scenarios
func (h *Handler) HandleGeneratedList(w http.ResponseWriter, r *http.Request) {
	scenarios, err := h.Generator.ListGeneratedScenarios()
	if err != nil {
		appErr := errors.NewFileSystemError("list generated scenarios", "generated/", err)
		appErr.Suggestion = "Check that the generated directory exists and is readable."
		h.RespondAppError(w, appErr)
		return
	}

	h.RespondJSON(w, http.StatusOK, scenarios)
}

// HandlePreviewLinks returns preview URLs for a generated scenario
func (h *Handler) HandlePreviewLinks(w http.ResponseWriter, r *http.Request) {
	vars := extractVars(r)
	scenarioID := vars["scenario_id"]

	if err := validation.ValidateScenarioSlug(scenarioID); err != nil {
		appErr := errors.NewValidationError("scenario_id", "Invalid scenario ID format")
		h.RespondAppError(w, appErr)
		return
	}

	preview, err := h.PreviewService.GetPreviewLinks(scenarioID)
	if err != nil {
		errMsg := err.Error()

		// Determine error type based on error message
		var appErr *errors.AppError
		switch {
		case strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "not exist"):
			appErr = errors.NewNotFoundError("scenario", scenarioID)
			appErr.Suggestion = "Make sure the scenario exists and is started before requesting preview links."
		case strings.Contains(errMsg, "not running"):
			appErr = errors.NewExternalServiceError("scenario", err)
			appErr.Message = "Scenario is not running"
			appErr.Suggestion = "Start the scenario first using the lifecycle controls."
		default:
			appErr = errors.NewExternalServiceError("preview service", err)
			appErr.Suggestion = "Try starting the scenario or check scenario logs for errors."
		}

		h.RespondAppError(w, appErr)
		return
	}

	h.RespondJSON(w, http.StatusOK, preview)
}

// extractVars is a helper to get route variables
func extractVars(r *http.Request) map[string]string {
	return mux.Vars(r)
}
