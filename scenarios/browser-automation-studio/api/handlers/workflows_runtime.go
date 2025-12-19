package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/compat"
	workflowservice "github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/websocket"
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

	// Check for frame streaming query parameter
	// Usage: POST /workflows/{id}/execute?frame_streaming=true
	var opts *workflowservice.ExecuteOptions
	if r.URL.Query().Get("frame_streaming") == "true" {
		opts = &workflowservice.ExecuteOptions{
			EnableFrameStreaming: true,
		}
		// Optional: parse quality and fps from query params
		if q := r.URL.Query().Get("frame_streaming_quality"); q != "" {
			if quality, err := strconv.Atoi(q); err == nil && quality > 0 && quality <= 100 {
				opts.FrameStreamingQuality = quality
			}
		}
		if f := r.URL.Query().Get("frame_streaming_fps"); f != "" {
			if fps, err := strconv.Atoi(f); err == nil && fps > 0 && fps <= 30 {
				opts.FrameStreamingFPS = fps
			}
		}
	}

	resp, err := h.executionService.ExecuteWorkflowAPIWithOptions(ctx, &req, opts)
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

	// Apply backwards compatibility transformations via centralized compat adapter.
	normalizedBody, err := compat.NormalizeExecuteAdhocRequest(body)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": fmt.Sprintf("normalize: %v", err)}))
		return
	}

	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(normalizedBody, &req); err != nil {
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

// ReceiveExecutionFrame receives frames from the playwright-driver during workflow execution.
// This endpoint is called by the driver when frame streaming is enabled for an execution.
// Frames are broadcast to WebSocket clients subscribed to the execution's frame stream.
func (h *Handler) ReceiveExecutionFrame(w http.ResponseWriter, r *http.Request) {
	executionID := chi.URLParam(r, "executionId")
	if executionID == "" {
		http.Error(w, "missing execution_id", http.StatusBadRequest)
		return
	}

	// Check if anyone is subscribed to this execution's frames
	// This optimization avoids decoding frames when no one is watching
	if !h.wsHub.HasExecutionFrameSubscribers(executionID) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Decode frame data from driver
	var frame struct {
		Data      string `json:"data"`       // Base64 encoded image data
		MediaType string `json:"media_type"` // e.g., "image/jpeg"
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	}
	if err := json.NewDecoder(r.Body).Decode(&frame); err != nil {
		http.Error(w, "invalid frame data", http.StatusBadRequest)
		return
	}

	// Broadcast to subscribers
	h.wsHub.BroadcastExecutionFrame(executionID, &websocket.ExecutionFrame{
		ExecutionID: executionID,
		Data:        frame.Data,
		MediaType:   frame.MediaType,
		Width:       frame.Width,
		Height:      frame.Height,
		CapturedAt:  time.Now().UTC().Format(time.RFC3339Nano),
	})

	w.WriteHeader(http.StatusOK)
}
