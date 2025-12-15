package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/vrooli/browser-automation-studio/internal/protoconv"
	workflowvalidator "github.com/vrooli/browser-automation-studio/workflow/validator"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
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

func normalizeWorkflowDefinitionV2Compat(doc map[string]any) {
	settings, ok := doc["settings"].(map[string]any)
	if !ok || settings == nil {
		// Continue; settings are optional, but node-level normalization still applies.
	} else {
		// test-genie may inject UI-oriented execution defaults (camelCase) that do not exist in the proto.
		// Translate what we can and drop the rest.
		if viewport, ok := settings["executionViewport"].(map[string]any); ok && viewport != nil {
			if width, ok := viewport["width"]; ok {
				switch v := width.(type) {
				case float64:
					settings["viewport_width"] = int32(v)
				case int:
					settings["viewport_width"] = int32(v)
				case int32:
					settings["viewport_width"] = v
				case int64:
					settings["viewport_width"] = int32(v)
				}
			}
			if height, ok := viewport["height"]; ok {
				switch v := height.(type) {
				case float64:
					settings["viewport_height"] = int32(v)
				case int:
					settings["viewport_height"] = int32(v)
				case int32:
					settings["viewport_height"] = v
				case int64:
					settings["viewport_height"] = int32(v)
				}
			}
			delete(settings, "executionViewport")
		}
		delete(settings, "defaultStepTimeoutMs")
	}

	// WorkflowDefinitionV2 expects some nested maps to be JsonValue-wrapped in proto JSON.
	// For compatibility with existing playbook JSON, wrap primitive subflow args into the
	// expected oneof shape (string_value/bool_value/int_value/object_value/etc).
	nodes, ok := doc["nodes"].([]any)
	if !ok || len(nodes) == 0 {
		return
	}
	for _, rawNode := range nodes {
		node, ok := rawNode.(map[string]any)
		if !ok || node == nil {
			continue
		}
		action, ok := node["action"].(map[string]any)
		if !ok || action == nil {
			continue
		}
		subflow, ok := action["subflow"].(map[string]any)
		if !ok || subflow == nil {
			continue
		}
		args, ok := subflow["args"].(map[string]any)
		if !ok || args == nil {
			continue
		}

		normalized := make(map[string]any, len(args))
		for k, v := range args {
			normalized[k] = jsonValueWrapper(v)
		}
		subflow["args"] = normalized
	}
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
	// validate it via proto decoding and return a compatible validation response. This keeps
	// playbooks and proto-first clients unblocked while the legacy map-based validator is phased out.
	if looksLikeWorkflowDefinitionV2(req.Workflow) {
		start := time.Now()
		normalizeWorkflowDefinitionV2Compat(req.Workflow)
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
		h.respondProto(w, http.StatusOK, &basapi.WorkflowValidationResult{
			Valid:         true,
			SchemaVersion: "workflow_definition_v2",
			DurationMs:    time.Since(start).Milliseconds(),
		})
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
		start := time.Now()
		normalizeWorkflowDefinitionV2Compat(req.Workflow)
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
		h.respondProto(w, http.StatusOK, &basapi.WorkflowValidationResult{
			Valid:         true,
			SchemaVersion: "workflow_definition_v2",
			DurationMs:    time.Since(start).Milliseconds(),
		})
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
