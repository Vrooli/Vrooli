package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	workflowservice "github.com/vrooli/browser-automation-studio/services/workflow"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	"google.golang.org/protobuf/encoding/protojson"
)

func (h *Handler) ExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if _, err := uuid.Parse(idStr); err != nil {
		h.respondError(w, ErrInvalidWorkflowID)
		return
	}

	var req basapi.ExecuteWorkflowRequest
	if err := decodeProtoJSONBody(r, &req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": err.Error()}))
		return
	}
	req.WorkflowId = idStr

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	resp, err := h.executionService.ExecuteWorkflowAPI(ctx, &req)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "execute_workflow", "error": err.Error()}))
		return
	}
	h.respondProto(w, http.StatusOK, resp)
}

func (h *Handler) ExecuteAdhocWorkflow(w http.ResponseWriter, r *http.Request) {
	var req basexecution.ExecuteAdhocRequest

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": fmt.Sprintf("read body: %v", err)}))
		return
	}
	if len(body) == 0 {
		body = []byte("{}")
	}

	// Backwards compatibility: older playbooks clients used `execution_params` instead of `parameters`.
	// Keep proto as the source-of-truth by translating into the canonical proto field name.
	var raw map[string]any
	if jsonErr := json.Unmarshal(body, &raw); jsonErr == nil {
		if execParams, ok := raw["execution_params"]; ok {
			if _, has := raw["parameters"]; !has {
				raw["parameters"] = execParams
			}
			delete(raw, "execution_params")
		}

		// Compatibility for clients that send plain JSON primitives into ExecutionParameters.{initial_params,initial_store,env}.
		// The proto expects map<string, common.v1.JsonValue> (i.e. {"string_value": "..."} wrappers).
		if params, ok := raw["parameters"].(map[string]any); ok && params != nil {
			normalizeJsonValueMaps(params, "initial_params", "initial_store", "env")
		}

		// Compatibility for test-genie's injected execution defaults (camelCase UI settings).
		if flowDef, ok := raw["flow_definition"].(map[string]any); ok && flowDef != nil {
			if settings, ok := flowDef["settings"].(map[string]any); ok && settings != nil {
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
		}

		if remapped, marshalErr := json.Marshal(raw); marshalErr == nil {
			body = remapped
		}
	}

	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(body, &req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": err.Error()}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	resp, err := h.executionService.ExecuteAdhocWorkflowAPI(ctx, &req)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "execute_adhoc", "error": err.Error()}))
		return
	}
	h.respondProto(w, http.StatusOK, resp)
}

func normalizeJsonValueMaps(params map[string]any, keys ...string) {
	for _, key := range keys {
		rawMap, ok := params[key].(map[string]any)
		if !ok || rawMap == nil {
			continue
		}
		normalized := make(map[string]any, len(rawMap))
		for k, v := range rawMap {
			normalized[k] = jsonValueWrapper(v)
		}
		params[key] = normalized
	}
}

func jsonValueWrapper(v any) any {
	if v == nil {
		return map[string]any{"null_value": "NULL_VALUE"}
	}
	switch vv := v.(type) {
	case map[string]any:
		// Already in JsonValue JSON form (oneof wrapper), keep as-is.
		if looksLikeJsonValueWrapper(vv) {
			return vv
		}
		fields := make(map[string]any, len(vv))
		for k, child := range vv {
			fields[k] = jsonValueWrapper(child)
		}
		return map[string]any{
			"object_value": map[string]any{
				"fields": fields,
			},
		}
	case []any:
		values := make([]any, 0, len(vv))
		for _, child := range vv {
			values = append(values, jsonValueWrapper(child))
		}
		return map[string]any{
			"list_value": map[string]any{
				"values": values,
			},
		}
	case string:
		return map[string]any{"string_value": vv}
	case bool:
		return map[string]any{"bool_value": vv}
	case float64:
		if isJSONIntegral(vv) {
			return map[string]any{"int_value": int64(vv)}
		}
		return map[string]any{"double_value": vv}
	case float32:
		f := float64(vv)
		if isJSONIntegral(f) {
			return map[string]any{"int_value": int64(f)}
		}
		return map[string]any{"double_value": f}
	case int:
		return map[string]any{"int_value": int64(vv)}
	case int32:
		return map[string]any{"int_value": int64(vv)}
	case int64:
		return map[string]any{"int_value": vv}
	case uint:
		return map[string]any{"int_value": int64(vv)}
	case uint32:
		return map[string]any{"int_value": int64(vv)}
	case uint64:
		if vv > math.MaxInt64 {
			return map[string]any{"double_value": float64(vv)}
		}
		return map[string]any{"int_value": int64(vv)}
	default:
		// Best-effort: preserve as string rather than failing hard.
		return map[string]any{"string_value": fmt.Sprintf("%v", v)}
	}
}

func looksLikeJsonValueWrapper(m map[string]any) bool {
	if m == nil {
		return false
	}
	// Accept both proto field names (snake_case) and JSON names (lowerCamel).
	for _, key := range []string{
		"bool_value", "boolValue",
		"int_value", "intValue",
		"double_value", "doubleValue",
		"string_value", "stringValue",
		"object_value", "objectValue",
		"list_value", "listValue",
		"null_value", "nullValue",
		"bytes_value", "bytesValue",
	} {
		if _, ok := m[key]; ok {
			return true
		}
	}
	return false
}

func isJSONIntegral(v float64) bool {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return false
	}
	if math.Trunc(v) != v {
		return false
	}
	if v < float64(math.MinInt64) || v > float64(math.MaxInt64) {
		return false
	}
	return true
}

func (h *Handler) ListWorkflowVersions(w http.ResponseWriter, r *http.Request) {
	workflowID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidWorkflowID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	list, err := h.catalogService.ListWorkflowVersionsAPI(ctx, workflowID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrWorkflowNotFound.WithDetails(map[string]string{"workflow_id": workflowID.String()}))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "list_workflow_versions", "error": err.Error()}))
		return
	}
	h.respondProto(w, http.StatusOK, list)
}

func (h *Handler) GetWorkflowVersion(w http.ResponseWriter, r *http.Request) {
	workflowID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidWorkflowID)
	if !ok {
		return
	}
	versionStr := strings.TrimSpace(chi.URLParam(r, "version"))
	versionInt, err := strconv.Atoi(versionStr)
	if err != nil || versionInt <= 0 {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "invalid version"}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	version, err := h.catalogService.GetWorkflowVersionAPI(ctx, workflowID, int32(versionInt))
	if err != nil {
		if errors.Is(err, workflowservice.ErrWorkflowVersionNotFound) {
			h.respondError(w, ErrWorkflowNotFound.WithDetails(map[string]string{"workflow_id": workflowID.String()}))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "get_workflow_version", "error": err.Error()}))
		return
	}
	h.respondProto(w, http.StatusOK, version)
}

func (h *Handler) RestoreWorkflowVersion(w http.ResponseWriter, r *http.Request) {
	workflowID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidWorkflowID)
	if !ok {
		return
	}
	versionStr := strings.TrimSpace(chi.URLParam(r, "version"))
	versionInt, err := strconv.Atoi(versionStr)
	if err != nil || versionInt <= 0 {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "invalid version"}))
		return
	}

	var body struct {
		ChangeDescription string `json:"change_description"`
	}
	_ = decodeJSONBodyAllowEmpty(w, r, &body)

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	resp, err := h.catalogService.RestoreWorkflowVersionAPI(ctx, workflowID, int32(versionInt), body.ChangeDescription)
	if err != nil {
		if errors.Is(err, workflowservice.ErrWorkflowVersionNotFound) {
			h.respondError(w, ErrWorkflowNotFound.WithDetails(map[string]string{"workflow_id": workflowID.String()}))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "restore_workflow_version", "error": err.Error()}))
		return
	}
	h.respondProto(w, http.StatusOK, resp)
}

func (h *Handler) ModifyWorkflow(w http.ResponseWriter, r *http.Request) {
	workflowID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidWorkflowID)
	if !ok {
		return
	}

	var body struct {
		Prompt      string         `json:"modification_prompt"`
		CurrentFlow map[string]any `json:"current_flow"`
	}
	if err := decodeJSONBody(w, r, &body); err != nil {
		h.respondError(w, ErrInvalidRequest)
		return
	}
	prompt := strings.TrimSpace(body.Prompt)
	if prompt == "" {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "modification_prompt is required"}))
		return
	}

	// Parse V2 flow definition directly - V1 format is no longer accepted.
	def, err := workflowservice.BuildFlowDefinitionV2ForWrite(body.CurrentFlow, nil, nil)
	if err != nil {
		if errors.Is(err, workflowservice.ErrInvalidWorkflowFormat) {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "Invalid workflow format: nodes must have 'action' field with typed action definitions"}))
			return
		}
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": err.Error()}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	resp, err := h.catalogService.ModifyWorkflowAPI(ctx, workflowID, prompt, def)
	if err != nil {
		var aiErr *workflowservice.AIWorkflowError
		if errors.As(err, &aiErr) {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": aiErr.Error()}))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "modify_workflow", "error": err.Error()}))
		return
	}

	// Keep response shape consistent with proto UpdateWorkflowResponse.
	h.respondProto(w, http.StatusOK, resp)
}
