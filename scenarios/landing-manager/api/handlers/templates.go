package handlers

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"landing-manager/errors"
	"landing-manager/validation"
)

// HandleTemplateList returns all available templates
func (h *Handler) HandleTemplateList(w http.ResponseWriter, r *http.Request) {
	templates, err := h.Registry.ListTemplates()
	if err != nil {
		errMsg := err.Error()

		// Provide specific guidance based on error type
		var appErr *errors.AppError
		switch {
		case strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "no such file"):
			appErr = errors.NewFileSystemError("read templates", "templates/", err)
			appErr.Message = "Template directory not found"
			appErr.Suggestion = "Ensure the templates directory exists in the scenario root"
		case strings.Contains(errMsg, "permission"):
			appErr = errors.NewFileSystemError("read templates", "templates/", err)
			appErr.Suggestion = "Check file permissions for the templates directory"
		default:
			appErr = errors.NewInternalError("Failed to load templates", err)
			appErr.Suggestion = "Check the scenario logs for more details or restart the scenario"
		}

		h.RespondAppError(w, appErr)
		return
	}

	h.RespondJSON(w, http.StatusOK, templates)
}

// HandleTemplateShow returns a specific template by ID
func (h *Handler) HandleTemplateShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := validation.ValidateTemplateID(id); err != nil {
		appErr := errors.NewValidationError("id", err.Error())
		h.RespondAppError(w, appErr)
		return
	}

	template, err := h.Registry.GetTemplate(id)
	if err != nil {
		appErr := errors.NewNotFoundError("template", id)
		appErr.Suggestion = "Use GET /templates to list all available template IDs"
		h.RespondAppError(w, appErr)
		return
	}

	h.RespondJSON(w, http.StatusOK, template)
}
