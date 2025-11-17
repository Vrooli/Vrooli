package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	workflowvalidator "github.com/vrooli/browser-automation-studio/workflow/validator"
)

type workflowValidationRequest struct {
	Workflow map[string]any `json:"workflow"`
	Strict   bool           `json:"strict"`
	Selector string         `json:"selector_root"`
}

// ValidateWorkflow validates ad-hoc workflow definitions via schema + lint rules.
func (h *Handler) ValidateWorkflow(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.workflowValidator == nil {
		RespondError(w, ErrInternalServer)
		return
	}

	var req workflowValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, ErrInvalidRequest)
		return
	}

	if req.Workflow == nil {
		h.respondError(w, ErrInvalidWorkflowPayload)
		return
	}

	result, err := h.workflowValidator.Validate(r.Context(), req.Workflow, workflowvalidator.Options{
		Strict:       req.Strict,
		SelectorRoot: strings.TrimSpace(req.Selector),
	})
	if err != nil {
		h.respondError(w, &APIError{
			Status:  http.StatusInternalServerError,
			Code:    "WF_VALIDATE_ERROR",
			Message: "Failed to validate workflow",
			Details: err.Error(),
		})
		return
	}

	h.respondSuccess(w, http.StatusOK, result)
}
