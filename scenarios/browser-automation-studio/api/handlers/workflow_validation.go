package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/vrooli/browser-automation-studio/internal/protoconv"
	workflowvalidator "github.com/vrooli/browser-automation-studio/workflow/validator"
)

type workflowValidationRequest struct {
	Workflow map[string]any `json:"workflow"`
	Strict   bool           `json:"strict"`
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
		Strict: req.Strict,
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

	pb := protoconv.WorkflowValidationResultToProto(result)
	h.respondProto(w, http.StatusOK, pb)
}

// ValidateResolvedWorkflow validates workflow definitions that have already been resolved.
// This endpoint is designed for test-genie's pre-flight validation: it checks that all
// tokens (@fixture/, @selector/, @seed/, ${}, {{}}) have been properly substituted
// and that navigate nodes with destinationType=scenario have been resolved to URLs.
//
// Use this endpoint after resolving a workflow but before execution to catch
// resolution failures early with clear error messages.
func (h *Handler) ValidateResolvedWorkflow(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.workflowValidator.ValidateResolved(r.Context(), req.Workflow, workflowvalidator.Options{
		Strict: req.Strict,
	})
	if err != nil {
		h.respondError(w, &APIError{
			Status:  http.StatusInternalServerError,
			Code:    "WF_VALIDATE_RESOLVED_ERROR",
			Message: "Failed to validate resolved workflow",
			Details: err.Error(),
		})
		return
	}

	pb := protoconv.WorkflowValidationResultToProto(result)
	h.respondProto(w, http.StatusOK, pb)
}
