package handlers

import (
	"context"
	"encoding/base64"
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
	gorillawebsocket "github.com/gorilla/websocket"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/compat"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	"github.com/vrooli/browser-automation-studio/services/testgenie"
	workflowservice "github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/websocket"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

const seedCleanupTimeout = 30 * time.Second

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
	if parseBoolQuery(r, "requires_video", "record_video") {
		if opts == nil {
			opts = &workflowservice.ExecuteOptions{}
		}
		opts.RequiresVideo = true
	}
	if parseBoolQuery(r, "requires_trace", "record_trace") {
		if opts == nil {
			opts = &workflowservice.ExecuteOptions{}
		}
		opts.RequiresTrace = true
	}
	if parseBoolQuery(r, "requires_har", "record_har") {
		if opts == nil {
			opts = &workflowservice.ExecuteOptions{}
		}
		opts.RequiresHAR = true
	}

	seedPlan, seedErr := applySeedIfNeeded(ctx, r, &req.Parameters)
	if seedErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": seedErr.Error()}))
		return
	}

	resp, err := h.executionService.ExecuteWorkflowAPIWithOptions(ctx, &req, opts)
	if err != nil {
		if seedPlan != nil && seedPlan.cleanup != nil {
			_ = seedPlan.cleanup(context.Background())
		}
		var seedErr *workflowservice.SeedRequirementError
		if errors.As(err, &seedErr) {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
				"error":        seedErr.Error(),
				"missing_keys": strings.Join(seedErr.MissingKeys, ", "),
				"hint":         "run the BAS seed script or test-genie before executing; or pass parameters.env.seed_applied=true",
			}))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "execute_workflow", "error": err.Error()}))
		return
	}

	if seedPlan != nil {
		if req.WaitForCompletion {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), seedCleanupTimeout)
			defer cancel()
			if err := seedPlan.cleanup(cleanupCtx); err != nil {
				h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
					"error":         "seed cleanup failed",
					"details":       err.Error(),
					"execution_id":  resp.GetExecutionId(),
					"seed_scenario": seedPlan.seedScenario,
				}))
				return
			}
		} else if h.seedCleanupManager != nil {
			if err := h.seedCleanupManager.Schedule(resp.GetExecutionId(), seedPlan.seedScenario, seedPlan.cleanupToken); err != nil {
				h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
					"error":         "seed cleanup scheduling failed",
					"details":       err.Error(),
					"execution_id":  resp.GetExecutionId(),
					"seed_scenario": seedPlan.seedScenario,
				}))
				return
			}
		} else {
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
				"error":         "seed cleanup manager unavailable",
				"execution_id":  resp.GetExecutionId(),
				"seed_scenario": seedPlan.seedScenario,
			}))
			return
		}
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

	var opts *workflowservice.ExecuteOptions
	if parseBoolQuery(r, "requires_video", "record_video") {
		opts = &workflowservice.ExecuteOptions{RequiresVideo: true}
	}
	if parseBoolQuery(r, "requires_trace", "record_trace") {
		if opts == nil {
			opts = &workflowservice.ExecuteOptions{}
		}
		opts.RequiresTrace = true
	}
	if parseBoolQuery(r, "requires_har", "record_har") {
		if opts == nil {
			opts = &workflowservice.ExecuteOptions{}
		}
		opts.RequiresHAR = true
	}

	seedPlan, seedErr := applySeedIfNeeded(ctx, r, &req.Parameters)
	if seedErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": seedErr.Error()}))
		return
	}
	resp, err := h.executionService.ExecuteAdhocWorkflowAPIWithOptions(ctx, &req, opts)
	if err != nil {
		if seedPlan != nil && seedPlan.cleanup != nil {
			_ = seedPlan.cleanup(context.Background())
		}
		var seedErr *workflowservice.SeedRequirementError
		if errors.As(err, &seedErr) {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
				"error":        seedErr.Error(),
				"missing_keys": strings.Join(seedErr.MissingKeys, ", "),
				"hint":         "run the BAS seed script or test-genie before executing; or pass parameters.env.seed_applied=true",
			}))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "execute_adhoc", "error": err.Error()}))
		return
	}

	if seedPlan != nil {
		if req.WaitForCompletion {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), seedCleanupTimeout)
			defer cancel()
			if err := seedPlan.cleanup(cleanupCtx); err != nil {
				h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
					"error":         "seed cleanup failed",
					"details":       err.Error(),
					"execution_id":  resp.GetExecutionId(),
					"seed_scenario": seedPlan.seedScenario,
				}))
				return
			}
		} else if h.seedCleanupManager != nil {
			if err := h.seedCleanupManager.Schedule(resp.GetExecutionId(), seedPlan.seedScenario, seedPlan.cleanupToken); err != nil {
				h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
					"error":         "seed cleanup scheduling failed",
					"details":       err.Error(),
					"execution_id":  resp.GetExecutionId(),
					"seed_scenario": seedPlan.seedScenario,
				}))
				return
			}
		} else {
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
				"error":         "seed cleanup manager unavailable",
				"execution_id":  resp.GetExecutionId(),
				"seed_scenario": seedPlan.seedScenario,
			}))
			return
		}
	}
	h.respondProto(w, http.StatusOK, resp)
}

func parseBoolQuery(r *http.Request, keys ...string) bool {
	if r == nil {
		return false
	}
	query := r.URL.Query()
	for _, key := range keys {
		raw := strings.TrimSpace(query.Get(key))
		if raw == "" {
			continue
		}
		switch strings.ToLower(raw) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return false
}

type seedCleanupPlan struct {
	cleanup      func(context.Context) error
	cleanupToken string
	seedScenario string
}

func applySeedIfNeeded(ctx context.Context, r *http.Request, params **basexecution.ExecutionParameters) (*seedCleanupPlan, error) {
	mode := seedModeFromRequest(r)
	if mode == "" {
		return nil, nil
	}
	if mode != "needs-applying" {
		return nil, fmt.Errorf("unsupported seed mode %q", mode)
	}

	seedScenario := seedScenarioFromRequest(r)
	if strings.EqualFold(seedScenario, "browser-automation-studio") {
		return nil, fmt.Errorf("seed=needs-applying is not supported when targeting browser-automation-studio directly; run via test-genie or the CLI handshake")
	}
	client := testgenie.NewClient(nil, nil)
	applyResp, err := client.ApplySeed(ctx, seedScenario, false)
	if err != nil {
		return nil, err
	}

	mergeSeedState(params, applyResp.SeedState)

	return &seedCleanupPlan{
		cleanup: func(cleanupCtx context.Context) error {
			_, err := client.CleanupSeed(cleanupCtx, seedScenario, applyResp.CleanupToken)
			return err
		},
		cleanupToken: applyResp.CleanupToken,
		seedScenario: seedScenario,
	}, nil
}

func seedModeFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	raw := strings.TrimSpace(r.URL.Query().Get("seed"))
	if raw == "" {
		raw = strings.TrimSpace(r.URL.Query().Get("seed_mode"))
	}
	return strings.ToLower(raw)
}

func seedScenarioFromRequest(r *http.Request) string {
	if r == nil {
		return "browser-automation-studio"
	}
	raw := strings.TrimSpace(r.URL.Query().Get("seed_scenario"))
	if raw == "" {
		return "browser-automation-studio"
	}
	return raw
}

func mergeSeedState(params **basexecution.ExecutionParameters, seedState map[string]any) {
	if params == nil {
		return
	}
	if *params == nil {
		*params = &basexecution.ExecutionParameters{}
	}
	if (*params).InitialParams == nil {
		(*params).InitialParams = map[string]*commonv1.JsonValue{}
	}
	for key, value := range seedState {
		if _, exists := (*params).InitialParams[key]; exists {
			continue
		}
		(*params).InitialParams[key] = typeconv.AnyToJsonValue(value)
	}
	if (*params).Env == nil {
		(*params).Env = map[string]*commonv1.JsonValue{}
	}
	if _, exists := (*params).Env["seed_applied"]; !exists {
		(*params).Env["seed_applied"] = typeconv.AnyToJsonValue(true)
	}
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

// HandleDriverExecutionFrameStream handles WebSocket connection for binary frame streaming from playwright-driver during workflow execution.
// GET /ws/execution/{executionId}/frames
// This mirrors HandleDriverFrameStream (for recording) but for execution mode.
// It receives raw binary JPEG frames from the playwright-driver and broadcasts them to UI clients
// who are subscribed to this execution via subscribe_execution_frames.
func (h *Handler) HandleDriverExecutionFrameStream(w http.ResponseWriter, r *http.Request) {
	executionID := chi.URLParam(r, "executionId")
	if executionID == "" {
		h.log.Error("Missing executionId in driver execution frame stream request")
		http.Error(w, "Missing executionId", http.StatusBadRequest)
		return
	}

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.WithError(err).Error("Failed to upgrade driver execution frame stream connection")
		return
	}
	defer conn.Close()

	h.log.WithField("execution_id", executionID).Info("Driver execution frame stream connected")

	// Read binary frames from driver and broadcast to browser clients
	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			// Check for normal closure
			if gorillawebsocket.IsCloseError(err, gorillawebsocket.CloseNormalClosure, gorillawebsocket.CloseGoingAway) {
				h.log.WithField("execution_id", executionID).Debug("Driver execution frame stream closed normally")
			} else {
				h.log.WithError(err).WithField("execution_id", executionID).Warn("Driver execution frame stream read error")
			}
			break
		}

		// Only process binary messages (JPEG data)
		if messageType != gorillawebsocket.BinaryMessage {
			continue
		}

		// Check if anyone is subscribed before processing
		if !h.wsHub.HasExecutionFrameSubscribers(executionID) {
			continue
		}

		// Convert binary frame to base64 for broadcast via the existing JSON-based hub
		// Note: This is less efficient than binary WebSocket, but works with existing infrastructure
		frameData := base64.StdEncoding.EncodeToString(data)

		h.wsHub.BroadcastExecutionFrame(executionID, &websocket.ExecutionFrame{
			ExecutionID: executionID,
			Data:        frameData,
			MediaType:   "image/jpeg",
			Width:       0, // Not available in raw binary frame
			Height:      0, // Not available in raw binary frame
			CapturedAt:  time.Now().UTC().Format(time.RFC3339Nano),
		})
	}

	h.log.WithField("execution_id", executionID).Info("Driver execution frame stream disconnected")
}
