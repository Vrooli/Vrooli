package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/vrooli/browser-automation-studio/internal/compat"
	"github.com/vrooli/browser-automation-studio/internal/protoconv"
	workflowvalidator "github.com/vrooli/browser-automation-studio/workflow/validator"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/encoding/protojson"
)

type workflowValidationRequest struct {
	Workflow map[string]any `json:"workflow"`
	Strict   bool           `json:"strict"`
}

func looksLikeWorkflowDefinitionV2(doc map[string]any) bool {
	nodes, ok := doc["nodes"].([]any)
	if !ok || len(nodes) == 0 {
		return false
	}
	first, ok := nodes[0].(map[string]any)
	if !ok || first == nil {
		return false
	}
	_, ok = first["action"]
	return ok
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

	// If the client is already sending a canonical WorkflowDefinitionV2 proto JSON payload,
	// validate it via proto decoding + semantic lint and return a compatible validation response.
	if looksLikeWorkflowDefinitionV2(req.Workflow) {
		compat.NormalizeWorkflowDefinitionV2(req.Workflow)
		body, err := json.Marshal(req.Workflow)
		if err != nil {
			h.respondError(w, ErrInvalidWorkflowPayload)
			return
		}
		var parsed basworkflows.WorkflowDefinitionV2
		if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(body, &parsed); err != nil {
			h.respondError(w, ErrInvalidWorkflowPayload.WithDetails(map[string]string{"error": err.Error()}))
			return
		}
		// Run semantic validation on V2 workflows
		v2Result := h.workflowValidator.ValidateV2(&parsed)
		h.respondProto(w, http.StatusOK, protoconv.WorkflowValidationResultToProto(v2Result))
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

	// See ValidateWorkflow() for rationale. Allow proto-first WorkflowDefinitionV2 payloads.
	if looksLikeWorkflowDefinitionV2(req.Workflow) {
		compat.NormalizeWorkflowDefinitionV2(req.Workflow)
		body, err := json.Marshal(req.Workflow)
		if err != nil {
			h.respondError(w, ErrInvalidWorkflowPayload)
			return
		}
		var parsed basworkflows.WorkflowDefinitionV2
		if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(body, &parsed); err != nil {
			h.respondError(w, ErrInvalidWorkflowPayload.WithDetails(map[string]string{"error": err.Error()}))
			return
		}
		// Run semantic validation on V2 workflows
		v2Result := h.workflowValidator.ValidateV2(&parsed)
		h.respondProto(w, http.StatusOK, protoconv.WorkflowValidationResultToProto(v2Result))
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
