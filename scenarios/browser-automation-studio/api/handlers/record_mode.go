package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/uiselectors"
	"github.com/vrooli/browser-automation-studio/internal/scenarioport"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/internal/protoconv"
	"github.com/vrooli/browser-automation-studio/performance"
	livecapture "github.com/vrooli/browser-automation-studio/services/live-capture"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
	"github.com/vrooli/browser-automation-studio/websocket"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
)

// Request/response types are defined in record_mode_types.go

const recordModeTimeout = 30 * time.Second

func getPlaywrightDriverURL() (string, error) {
	return driver.ResolveEndpoint(os.Getenv(driver.PlaywrightDriverEnv))
}

// CreateRecordingSession handles POST /api/v1/recordings/live/session
// Creates a new browser session for recording user actions.
func (h *Handler) CreateRecordingSession(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	var req CreateRecordingSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	// Resolve session profile for authentication persistence and browser configuration
	var profileID, profileName, profileLastUsed string
	var storageState json.RawMessage
	var browserProfile *sessionprofilepersistence.BrowserProfile
	var openTabs []sessionprofilepersistence.TabState
	if h.sessionProfileService != nil {
		profile, err := h.resolveSessionProfile(req.SessionProfileID)
		if err != nil {
			h.respondError(w, err)
			return
		}
		if profile != nil {
			profileID = string(profile.ID)
			profileName = profile.Name
			profileLastUsed = profile.LastUsedAt.Format(time.RFC3339)
			storageState = profile.StorageState
			browserProfile = profile.BrowserProfile
			openTabs = profile.OpenTabs
		}
	}

	// Apply stream settings with defaults from config
	// Note: These defaults should be centralized in config.go, not hardcoded here
	appCfg := config.Load()
	streamQuality := appCfg.Recording.DefaultStreamQuality
	if streamQuality <= 0 || streamQuality > 100 {
		streamQuality = 55 // Fallback if config invalid
	}
	if req.StreamQuality != nil && *req.StreamQuality >= 1 && *req.StreamQuality <= 100 {
		streamQuality = *req.StreamQuality
	}
	streamFPS := appCfg.Recording.DefaultStreamFPS
	if streamFPS <= 0 || streamFPS > 60 {
		streamFPS = 30 // Fallback if config invalid
	}
	if req.StreamFPS != nil && *req.StreamFPS >= 1 && *req.StreamFPS <= 60 {
		streamFPS = *req.StreamFPS
	}
	streamScale := "css"
	if req.StreamScale == "device" {
		streamScale = "device"
	}

	// Delegate to recordmode service
	cfg := &livecapture.SessionConfig{
		ViewportWidth:  req.ViewportWidth,
		ViewportHeight: req.ViewportHeight,
		InitialURL:     req.InitialURL,
		StreamQuality:  streamQuality,
		StreamFPS:      streamFPS,
		StreamScale:    streamScale,
		StorageState:   storageState,
		APIHost:        os.Getenv("API_HOST"),
		APIPort:        os.Getenv("API_PORT"),
		BrowserProfile: browserProfile,
	}

	result, err := h.recordModeService.CreateSession(ctx, cfg)
	if err != nil {
		h.log.WithError(err).Error("Failed to create recording session")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Restore tabs if requested (default: true for recording sessions)
	var restoredTabs []RestoredTabInfo
	var initialURL string
	restoreTabs := req.RestoreTabs == nil || *req.RestoreTabs // Default to true
	h.log.WithFields(map[string]interface{}{
		"session_id":    result.SessionID,
		"profile_id":    profileID,
		"restore_tabs":  restoreTabs,
		"open_tabs_len": len(openTabs),
	}).Info("Tab restoration check")
	if restoreTabs && len(openTabs) > 0 {
		// Log the tabs we're about to restore
		for i, tab := range openTabs {
			h.log.WithFields(map[string]interface{}{
				"index":     i,
				"url":       tab.URL,
				"title":     tab.Title,
				"is_active": tab.IsActive,
				"order":     tab.Order,
			}).Info("Tab to restore")
		}
		restorationResult, err := h.recordModeService.RestoreTabs(ctx, result.SessionID, openTabs)
		if err != nil {
			// The saved profile remains authoritative until restoration succeeds.
			// Cleanup must run even if the admission request was cancelled.
			cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), recordModeTimeout)
			closeErr := result.Close(cleanupCtx)
			cleanupCancel()
			details := map[string]string{"error": err.Error()}
			if closeErr != nil {
				details["cleanup_error"] = closeErr.Error()
				details["session_id"] = result.SessionID
			}
			h.respondError(w, ErrServiceUnavailable.WithMessage("Saved tabs could not be restored; the saved profile is unchanged.").WithDetails(details))
			return
		} else if restorationResult != nil {
			h.log.WithFields(map[string]interface{}{
				"restored_count": len(restorationResult.Tabs),
				"initial_url":    restorationResult.InitialURL,
			}).Info("Tabs restored successfully")
			initialURL = restorationResult.InitialURL
			restoredTabs = make([]RestoredTabInfo, 0, len(restorationResult.Tabs))
			for _, tab := range restorationResult.Tabs {
				restoredTabs = append(restoredTabs, RestoredTabInfo{
					PageID:   tab.PageID,
					URL:      tab.URL,
					IsActive: tab.IsActive,
				})
			}

			// Save history entries from tab restoration
			if profileID != "" && h.sessionProfileService != nil && len(restorationResult.HistoryEntries) > 0 {
				for _, histEntry := range restorationResult.HistoryEntries {
					entry := sessionprofilepersistence.HistoryEntry{
						ID:        uuid.NewString(),
						URL:       histEntry.URL,
						Title:     histEntry.Title,
						Timestamp: time.Now().UTC().Format(time.RFC3339),
					}
					if _, err := h.sessionProfileService.AddHistoryEntry(sessionprofilepersistence.ProfileID(profileID), entry); err != nil {
						h.log.WithError(err).WithFields(map[string]interface{}{
							"profile_id": profileID,
							"url":        histEntry.URL,
						}).Warn("Failed to add history entry from tab restoration")
					}
				}
				h.log.WithFields(map[string]interface{}{
					"profile_id":    profileID,
					"history_count": len(restorationResult.HistoryEntries),
				}).Debug("Saved history entries from tab restoration")
			}
		}
	} else if profileID != "" && h.sessionProfileService != nil && result.InitialNavigation != nil {
		// No tab restoration, but there was an initial URL navigation - capture it as history
		entry := sessionprofilepersistence.HistoryEntry{
			ID:        uuid.NewString(),
			URL:       result.InitialNavigation.URL,
			Title:     result.InitialNavigation.Title,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		if _, err := h.sessionProfileService.AddHistoryEntry(sessionprofilepersistence.ProfileID(profileID), entry); err != nil {
			h.log.WithError(err).WithFields(map[string]interface{}{
				"profile_id": profileID,
				"url":        result.InitialNavigation.URL,
			}).Warn("Failed to add history entry from initial navigation")
		} else {
			h.log.WithFields(map[string]interface{}{
				"profile_id": profileID,
				"url":        result.InitialNavigation.URL,
			}).Debug("Saved history entry from initial navigation")
		}
	}

	// Update session profile usage tracking
	if profileID != "" && h.sessionProfileService != nil {
		if updated, err := h.sessionProfileService.Touch(sessionprofilepersistence.ProfileID(profileID)); err != nil {
			h.log.WithError(err).WithField("profile_id", profileID).Warn("Failed to update session profile usage")
		} else if updated != nil {
			profileName = updated.Name
			profileLastUsed = updated.LastUsedAt.Format(time.RFC3339)
		}
		h.setActiveSessionProfile(result.SessionID, profileID)
	}

	// Convert actual viewport with source attribution if present
	var actualViewport *ActualViewportWithSource
	if result.ActualViewport != nil {
		actualViewport = &ActualViewportWithSource{
			Width:  result.ActualViewport.Width,
			Height: result.ActualViewport.Height,
			Source: ViewportSource(result.ActualViewport.Source),
			Reason: result.ActualViewport.Reason,
		}
	}

	response := CreateRecordingSessionResponse{
		SessionID:          result.SessionID,
		CreatedAt:          result.CreatedAt.Format(time.RFC3339),
		SessionProfileID:   profileID,
		SessionProfileName: profileName,
		LastUsedAt:         profileLastUsed,
		ActualViewport:     actualViewport,
		RestoredTabs:       restoredTabs,
		InitialURL:         initialURL,
	}

	if pb, err := protoconv.RecordingSessionToProto(response); err == nil && pb != nil {
		h.respondProto(w, http.StatusOK, pb)
		return
	}
	h.respondSuccess(w, http.StatusOK, response)
}

// CloseRecordingSession handles POST /api/v1/recordings/live/session/{sessionId}/close
// Closes a recording session and cleans up resources.
func (h *Handler) CloseRecordingSession(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	profileBound := h.getActiveSessionProfile(sessionID) != ""
	profilePersisted := true
	// Keep the browser available for retry until its complete profile snapshot
	// has committed. Closing first destroys the only recoverable live state.
	// If the browser has already disappeared, however, retrying the snapshot
	// cannot recover it and must not strand the profile binding forever.
	if err := h.persistSessionProfile(ctx, sessionID); err != nil {
		if isGoneSessionError(err) {
			profilePersisted = false
			h.log.WithError(err).WithField("session_id", sessionID).Warn("Session already gone before profile snapshot; continuing idempotent close")
		} else {
			h.log.WithError(err).WithField("session_id", sessionID).Warn("Failed to save profile before close")
			h.respondError(w, ErrInternalServer.WithMessage("Session profile could not be saved; the browser remains open. Retry closing after resolving the save failure.").WithDetails(map[string]string{"error": err.Error()}))
			return
		}
	}

	// Delegate to recordmode service
	if err := h.recordModeService.CloseSession(ctx, sessionID); err != nil {
		if !profileBound || !isGoneSessionError(err) {
			h.log.WithError(err).Error("Failed to close recording session")
			// Check for not found error
			if driverErr, ok := err.(*driver.Error); ok && driverErr.Status == 404 {
				h.respondError(w, ErrExecutionNotFound.WithMessage("Session not found"))
				return
			}
			h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
				"error": err.Error(),
			}))
			return
		}
		h.log.WithError(err).WithField("session_id", sessionID).Warn("Session was already gone during idempotent close")
	}

	if frameSlots, ok := h.wsHub.(interface{ ClearRecordingFrames(string) }); ok {
		frameSlots.ClearRecordingFrames(sessionID)
	}
	h.clearActiveSessionProfile(sessionID)

	h.respondSuccess(w, http.StatusOK, map[string]interface{}{
		"session_id":        sessionID,
		"status":            "closed",
		"profile_persisted": profilePersisted,
	})
}

// isGoneSessionError identifies terminal/absent browser state. Transient
// storage or driver failures must still preserve the browser and its profile
// binding for retry.
func isGoneSessionError(err error) bool {
	if err == nil {
		return false
	}
	var driverErr *driver.Error
	if errors.As(err, &driverErr) {
		return driverErr.Status == http.StatusNotFound || strings.Contains(strings.ToUpper(driverErr.Message), "SESSION_NOT_FOUND")
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "session closed") || strings.Contains(message, "session not found")
}

// GetRecordingDebug handles GET /api/v1/recordings/live/{sessionId}/debug
// Gets live debugging info for an active recording session.
// This proxies the broker-owned driver's debug endpoint.
func (h *Handler) GetRecordingDebug(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	resp, err := h.recordModeService.GetRecordingDebug(r.Context(), sessionID)
	if err != nil {
		var driverErr *driver.Error
		if errors.As(err, &driverErr) && driverErr.Status > 0 {
			http.Error(w, driverErr.Message, driverErr.Status)
			return
		}
		http.Error(w, "Failed to reach playwright-driver", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		if h.log != nil {
			h.log.WithError(err).Warn("record_mode: failed to proxy response")
		}
	}
}

// GetRecordedActions handles GET /api/v1/recordings/live/{sessionId}/actions
// Gets all recorded actions for a session.
func (h *Handler) GetRecordedActions(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	// Check for clear query param
	clearActions := r.URL.Query().Get("clear") == "true"
	owner, owned := h.recordModeService.GetSession(sessionID)
	if clearActions && (!owned || owner == nil) {
		h.respondError(w, ErrExecutionNotFound.WithMessage("Session not found"))
		return
	}

	// Read without mutation; acknowledgement follows journal commit.
	resp, err := h.recordModeService.GetRecordedActions(ctx, sessionID)
	if err != nil {
		h.log.WithError(err).Error("Failed to get recorded actions")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	if clearActions && len(resp.TimelineEntries) > 0 {
		ids := make([]string, 0, len(resp.TimelineEntries))
		for _, entry := range resp.TimelineEntries {
			if entry == nil {
				continue
			}
			if err := h.commitRecordingEntry(ctx, sessionID, entry); err != nil {
				h.respondError(w, err)
				return
			}
			ids = append(ids, entry.GetId())
		}
		if err := owner.AcknowledgeRecordedActions(ctx, ids); err != nil {
			h.respondError(w, ErrServiceUnavailable.WithMessage("Recording committed but driver acknowledgement failed").WithDetails(map[string]string{"error": err.Error()}))
			return
		}
	} else if clearActions && len(resp.Actions) > 0 {
		ids := make([]string, 0, len(resp.Actions))
		for i := range resp.Actions {
			action := &resp.Actions[i]
			if err := h.commitRecordingAction(ctx, sessionID, action); err != nil {
				h.respondError(w, err)
				return
			}
			ids = append(ids, action.ID)
		}
		if err := owner.AcknowledgeRecordedActions(ctx, ids); err != nil {
			h.respondError(w, ErrServiceUnavailable.WithMessage("Recording committed but driver acknowledgement failed").WithDetails(map[string]string{"error": err.Error()}))
			return
		}
	}

	// Current driver receipts carry canonical proto entries. The raw Actions
	// count is only a fallback for older drivers that have not adopted them.
	count := len(resp.Actions)
	if len(resp.TimelineEntries) > 0 {
		count = len(resp.TimelineEntries)
	}

	// Map service response to handler response type
	driverResp := &GetActionsResponse{
		SessionID:   resp.SessionID,
		IsRecording: resp.IsRecording,
		Actions:     resp.Actions,
		Count:       count,
		Entries:     resp.Entries,
	}

	h.respondSuccess(w, http.StatusOK, driverResp)
}

// GenerateWorkflowFromRecording handles POST /api/v1/recordings/live/{sessionId}/generate-workflow
// Converts recorded actions into a workflow.
func (h *Handler) GenerateWorkflowFromRecording(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	var req GenerateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	if req.Name == "" {
		req.Name = fmt.Sprintf("Recorded Workflow %s", time.Now().Format("2006-01-02 15:04"))
	}

	if req.ProjectID == nil || *req.ProjectID == uuid.Nil {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "project_id"}))
		return
	}
	projectID := *req.ProjectID

	// Persist session profile before workflow generation
	if err := h.persistSessionProfile(ctx, sessionID); err != nil {
		h.log.WithError(err).WithField("session_id", sessionID).Warn("Failed to persist session profile before workflow generation")
	}

	// Delegate workflow generation to the recordmode service
	// The service handles action fetching, merging, and smart wait insertion
	var actionRange *livecapture.ActionRange
	if req.ActionRange != nil {
		actionRange = &livecapture.ActionRange{
			Start: req.ActionRange.Start,
			End:   req.ActionRange.End,
		}
	}

	genResult, err := h.recordModeService.GenerateWorkflow(ctx, sessionID, &livecapture.GenerateWorkflowConfig{
		Name:        req.Name,
		Actions:     req.Actions, // Pass through user-edited actions if provided
		ActionRange: actionRange,
	})
	if err != nil {
		h.log.WithError(err).WithField("session_id", sessionID).Error("Failed to generate workflow from recording")
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	v2 := genResult.FlowDefinition

	// Bind recorded locators to this project's contract when recording its own UI.
	if project, lookupErr := h.repo.GetProject(ctx, projectID); lookupErr == nil && project != nil {
		if manifest, _, manifestErr := uiselectors.Load(project.FolderPath); manifestErr == nil {
			if origin, _, originErr := scenarioport.ResolveURLAtPath(ctx, filepath.Base(project.FolderPath), project.FolderPath, "/"); originErr == nil {
				symbolizeRecordingSelectors(v2, manifest, origin)
			}
		}
	}
	// Create the workflow via catalog service
	createResp, err := h.catalogService.CreateWorkflow(ctx, &basapi.CreateWorkflowRequest{
		ProjectId:      projectID.String(),
		Name:           req.Name,
		FolderPath:     "/",
		FlowDefinition: v2,
	})
	if err != nil || createResp == nil || createResp.Workflow == nil {
		h.log.WithError(err).Error("Failed to create workflow from recording")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to create workflow: " + err.Error(),
		}))
		return
	}

	respPayload := GenerateWorkflowResponse{
		WorkflowID:  uuid.MustParse(createResp.Workflow.Id),
		ProjectID:   projectID,
		Name:        createResp.Workflow.Name,
		NodeCount:   genResult.NodeCount,
		ActionCount: genResult.ActionCount,
	}
	if pb, err := protoconv.GenerateWorkflowToProto(respPayload); err == nil && pb != nil {
		h.respondProto(w, http.StatusCreated, pb)
		return
	}
	h.respondSuccess(w, http.StatusCreated, respPayload)
}

// HandleDriverFrameStream accepts the single source-bearing driver frame protocol.
// Recording frames are checked against the current lease/page; execution frames
// are associated with their owned execution before entering the viewer hub.
func (h *Handler) HandleDriverFrameStream(w http.ResponseWriter, r *http.Request) {
	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.WithError(err).Error("Failed to upgrade driver frame stream connection")
		return
	}
	defer conn.Close()

	h.log.Info("Driver frame stream connected")

	// Get performance config (used for logging/broadcast intervals)
	cfg := config.Load()

	// Allocate telemetry only for accepted frames carrying timing data.
	var collector *performance.Collector
	var streamSessionID string

	// Read binary frames from driver and broadcast to browser clients
	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			// Check for normal closure
			if websocket.IsCloseError(err) {
				h.log.Debug("Driver frame stream closed normally")
			} else {
				h.log.WithError(err).Warn("Driver frame stream read error")
			}
			break
		}

		// Only process binary messages (JPEG data)
		if messageType != websocket.BinaryMessage {
			continue
		}

		processingStart := time.Now()

		frameData, driverHeader, decodeErr := decodeDriverFrame(data)
		if decodeErr != nil {
			h.log.WithError(decodeErr).Debug("Rejected invalid recording frame")
			continue
		}
		source := driverHeader.Source
		sessionID := source.SessionID
		streamSessionID = sessionID
		broadcastStart := time.Now()
		if source.StreamKind == "recording" {
			pageID, title, pageURL, accepted := h.framePageMetadata(sessionID, source)
			if !accepted {
				continue
			}
			h.wsHub.BroadcastBinaryFrame(sessionID, viewerFrame(sessionID, pageID, title, pageURL, driverHeader.CapturedAt, frameData))
		} else if source.StreamKind == "execution" {
			executionID, parseErr := uuid.Parse(source.ExecutionID)
			if parseErr != nil || h.executionService == nil || source.SessionID == "" || source.LeaseID == "" || source.PageID == "" {
				continue
			}
			if _, ownerErr := h.executionService.GetExecution(r.Context(), executionID); ownerErr != nil {
				continue
			}
			width, height := jpegDimensions(frameData)
			h.wsHub.BroadcastExecutionFrame(executionID.String(), &wsHub.ExecutionFrame{
				ExecutionID: executionID.String(), Data: base64.StdEncoding.EncodeToString(frameData),
				MediaType: "image/jpeg", Width: width, Height: height,
				CapturedAt: driverHeader.CapturedAt.UTC().Format(time.RFC3339Nano),
			})
		} else {
			continue
		}
		broadcastMs := float64(time.Since(broadcastStart).Microseconds()) / 1000.0

		// Record performance data if driver sent a perf header
		// (presence of header indicates per-session perf mode is enabled)
		if driverHeader.Timing != nil {
			if collector == nil {
				collector = h.perfRegistry.GetOrCreate(sessionID)
			}
			timing := &performance.FrameTimings{
				FrameID:         driverHeader.Timing.FrameID,
				SessionID:       sessionID,
				Timestamp:       time.Now(),
				DriverCaptureMs: driverHeader.Timing.CaptureMs,
				DriverCompareMs: driverHeader.Timing.CompareMs,
				DriverWsSendMs:  driverHeader.Timing.WsSendMs,
				DriverTotalMs:   driverHeader.Timing.CaptureMs + driverHeader.Timing.CompareMs + driverHeader.Timing.WsSendMs,
				APIBroadcastMs:  broadcastMs,
				APITotalMs:      float64(time.Since(processingStart).Microseconds()) / 1000.0,
				FrameBytes:      len(frameData),
				Skipped:         false,
			}
			collector.Record(timing)

			// Broadcast perf stats periodically (every 60 frames by default)
			if cfg.Performance.StreamToWebSocket && collector.ShouldBroadcast() {
				stats := collector.GetAggregated()
				h.wsHub.BroadcastPerfStats(sessionID, stats)

				// Log summary if enabled
				if cfg.Performance.LogSummaryInterval > 0 {
					h.log.WithFields(map[string]interface{}{
						"session_id":      sessionID,
						"frame_count":     stats.FrameCount,
						"capture_p50_ms":  stats.CaptureP50Ms,
						"capture_p90_ms":  stats.CaptureP90Ms,
						"e2e_p50_ms":      stats.E2EP50Ms,
						"e2e_p90_ms":      stats.E2EP90Ms,
						"actual_fps":      stats.ActualFps,
						"target_fps":      stats.TargetFps,
						"bottleneck":      stats.PrimaryBottleneck,
						"bandwidth_bps":   stats.BandwidthBytesPerSec,
						"avg_frame_bytes": stats.AvgFrameBytes,
					}).Info("recording: frame perf summary")
				}
			}
		}
	}

	// Cleanup collector when stream disconnects
	if collector != nil {
		h.perfRegistry.Remove(streamSessionID)
	}

	h.log.WithField("session_id", streamSessionID).Info("Driver frame stream disconnected")
}

// ReloadRecordingSession handles POST /api/v1/recordings/live/{sessionId}/reload.
func (h *Handler) ReloadRecordingSession(w http.ResponseWriter, r *http.Request) {
	h.navigateRecordingHistory(w, r, driver.HistoryReload)
}

// GoBackRecordingSession handles POST /api/v1/recordings/live/{sessionId}/go-back.
func (h *Handler) GoBackRecordingSession(w http.ResponseWriter, r *http.Request) {
	h.navigateRecordingHistory(w, r, driver.HistoryBack)
}

// GoForwardRecordingSession handles POST /api/v1/recordings/live/{sessionId}/go-forward.
func (h *Handler) GoForwardRecordingSession(w http.ResponseWriter, r *http.Request) {
	h.navigateRecordingHistory(w, r, driver.HistoryForward)
}

func (h *Handler) navigateRecordingHistory(w http.ResponseWriter, r *http.Request, operation driver.HistoryNavigation) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "sessionId"}))
		return
	}
	var req struct {
		driver.HistoryNavigationRequest
		PageID string `json:"page_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "Invalid JSON body: " + err.Error()}))
		return
	}
	owner, ok := h.recordModeService.GetSession(sessionID)
	if !ok || owner == nil {
		h.respondError(w, ErrExecutionNotFound.WithMessage("Session not found"))
		return
	}
	expectedPage, apiErr := navigationRequestPage(owner, req.PageID)
	if apiErr != nil {
		h.respondError(w, apiErr)
		return
	}
	req.ExpectedPageID = expectedPage
	resp, err := owner.NavigateHistory(ctx, operation, &req.HistoryNavigationRequest)
	if err != nil {
		h.log.WithError(err).Error("Failed to navigate recording history")
		h.respondError(w, recordingNavigationError(err))
		return
	}
	actionType := "reload"
	if operation == driver.HistoryBack {
		actionType = "goBack"
	}
	if operation == driver.HistoryForward {
		actionType = "goForward"
	}
	if apiErr := h.recordCompletedNavigation(ctx, owner, actionType, resp); apiErr != nil {
		h.respondError(w, apiErr)
		return
	}
	h.respondSuccess(w, http.StatusOK, NavigationStateResponse{SessionID: sessionID, URL: resp.URL, Title: resp.Title, CanGoBack: resp.CanGoBack, CanGoForward: resp.CanGoForward})
}

// CaptureRecordingScreenshot handles POST /api/v1/recordings/live/{sessionId}/screenshot
// Captures a screenshot from the current recording page.
func (h *Handler) CaptureRecordingScreenshot(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	// Parse optional request body for format/quality
	var reqBody struct {
		Format  string `json:"format,omitempty"`
		Quality int    `json:"quality,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil && err.Error() != "EOF" {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	// Delegate directly to driver client (no service-layer business logic needed)
	svcReq := &driver.CaptureScreenshotRequest{
		Format:  reqBody.Format,
		Quality: reqBody.Quality,
	}

	resp, err := h.recordModeService.CaptureScreenshot(ctx, sessionID, svcReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to capture screenshot")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Map service response to handler response
	driverResp := RecordingScreenshotResponse{
		SessionID:  sessionID,
		Screenshot: resp.Data,
	}

	h.respondSuccess(w, http.StatusOK, driverResp)
}

// UpdateRecordingViewport handles POST /api/v1/recordings/live/{sessionId}/viewport
// Updates the viewport dimensions for the active recording session.
func (h *Handler) UpdateRecordingViewport(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	var reqBody RecordingViewportRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	if reqBody.Width <= 0 || reqBody.Height <= 0 {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "width and height must be positive integers",
		}))
		return
	}

	owner, ok := h.recordModeService.GetSession(sessionID)
	if !ok || owner == nil {
		h.respondError(w, ErrExecutionNotFound.WithMessage("Session not found"))
		return
	}
	expected, apiErr := recordingSelectedPage(owner, reqBody.PageID)
	if apiErr != nil {
		h.respondError(w, apiErr)
		return
	}
	resp, err := owner.UpdateViewport(ctx, reqBody.Width, reqBody.Height, expected)
	if err != nil {
		h.respondError(w, recordingNavigationError(err))
		return
	}
	current, ok := h.recordModeService.GetSession(sessionID)
	if !ok || current != owner {
		h.respondError(w, ErrConflict.WithMessage("Recording session changed during resize"))
		return
	}
	selected, apiErr := recordingSelectedPage(owner, reqBody.PageID)
	if apiErr != nil {
		h.respondError(w, apiErr)
		return
	}
	if selected != expected {
		h.respondError(w, ErrConflict.WithMessage("Recording tab changed during resize"))
		return
	}
	h.respondSuccess(w, http.StatusOK, struct {
		SessionID string `json:"session_id"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	}{sessionID, resp.Width, resp.Height})
}

// UpdateStreamSettings handles POST /api/v1/recordings/live/{sessionId}/stream-settings
// Updates stream settings (quality, fps) for an active recording session.
// Quality and FPS can be updated immediately. Scale changes require a new session.
func (h *Handler) UpdateStreamSettings(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	var reqBody UpdateStreamSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil && err.Error() != "EOF" {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	// Delegate directly to driver client (no service-layer business logic needed)
	svcReq := &driver.UpdateStreamSettingsRequest{
		Quality:  reqBody.Quality,
		FPS:      reqBody.FPS,
		Scale:    reqBody.Scale,
		PerfMode: reqBody.PerfMode,
	}

	resp, err := h.recordModeService.UpdateStreamSettings(ctx, sessionID, svcReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to update stream settings")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Map service response to handler response
	driverResp := UpdateStreamSettingsResponse{
		SessionID:    resp.SessionID,
		Quality:      resp.Quality,
		FPS:          resp.FPS,
		CurrentFPS:   resp.CurrentFPS,
		Scale:        resp.Scale,
		IsStreaming:  resp.IsStreaming,
		Updated:      resp.Updated,
		ScaleWarning: resp.ScaleWarning,
		PerfMode:     resp.PerfMode,
	}

	h.log.WithFields(map[string]interface{}{
		"session_id":  sessionID,
		"quality":     driverResp.Quality,
		"fps":         driverResp.FPS,
		"current_fps": driverResp.CurrentFPS,
		"updated":     driverResp.Updated,
	}).Debug("Stream settings updated")

	h.respondSuccess(w, http.StatusOK, driverResp)
}

// ForwardRecordingInput handles POST /api/v1/recordings/live/{sessionId}/input
// Forwards pointer/keyboard/wheel events to the Playwright driver.
func (h *Handler) ForwardRecordingInput(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Failed to read request body: " + err.Error(),
		}))
		return
	}
	if len(bodyBytes) == 0 {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Empty request body",
		}))
		return
	}

	// The service carries the immutable session lease for this mutation.
	receipt, err := h.recordModeService.ForwardInput(ctx, sessionID, bodyBytes)
	if err != nil {
		h.log.WithError(err).Error("Failed to forward recording input")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusOK, receipt)
}

// GetRecordingFrame handles GET /api/v1/recordings/live/{sessionId}/frame
// Retrieves a lightweight frame preview from the driver.
// Supports ETag-based caching to skip identical frames (If-None-Match header).
func (h *Handler) GetRecordingFrame(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	owner, ok := h.recordModeService.GetSession(sessionID)
	if !ok || owner == nil {
		h.respondError(w, ErrExecutionNotFound.WithMessage("Session not found"))
		return
	}
	query := r.URL.Query()
	requestedPage := query.Get("page_id")
	if _, apiErr := navigationRequestPage(owner, requestedPage); apiErr != nil {
		h.respondError(w, apiErr)
		return
	}
	pages := owner.Pages()
	if pages == nil {
		h.respondError(w, ErrConflict.WithMessage("No recording tab is selected"))
		return
	}
	page := pages.GetActivePage()
	if page == nil || page.DriverPageID == "" || (requestedPage != "" && requestedPage != page.ID.String()) {
		h.respondError(w, ErrConflict.WithMessage("The selected recording tab changed"))
		return
	}
	query.Set("page_id", page.DriverPageID)
	resp, err := h.recordModeService.GetFrame(ctx, sessionID, query.Encode())
	if err != nil {
		var driverError *driver.Error
		if errors.As(err, &driverError) && driverError.Status == http.StatusConflict {
			h.respondError(w, ErrConflict.WithMessage("The selected recording tab changed during capture"))
		} else {
			h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{"error": err.Error()}))
		}
		return
	}
	if resp == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Missing recording frame"))
		return
	}
	canonical, accepted := h.framePage(sessionID, resp.Source)
	if !accepted || canonical != page.ID || resp.SessionID != sessionID {
		h.respondError(w, ErrConflict.WithMessage("The recording frame source changed during capture"))
		return
	}
	// Do not expose producer credentials or mutate a response owned by the client.
	frame := *resp
	frame.Source = nil
	frame.PageID = canonical.String()

	etag := fmt.Sprintf(`"%s"`, resp.ContentHash)

	// Check If-None-Match header for conditional request
	clientETag := r.Header.Get("If-None-Match")
	if clientETag != "" && clientETag == etag {
		// Frame hasn't changed, return 304 Not Modified
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Set ETag header for client caching
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-cache")

	h.respondSuccess(w, http.StatusOK, &frame)
}

// PersistRecordingSession handles POST /api/v1/recordings/live/{sessionId}/persist
// Captures current storage state and saves it to the active session profile without closing the session.
// CreateInputForwarder returns a function that forwards input events to the playwright-driver.
// This is used by the WebSocket hub to forward input messages without going through HTTP.
//
// Performance optimizations:
// - Uses a shared HTTP client with connection pooling (reuses TCP connections)
// - Keep-alive connections reduce latency by avoiding TCP handshake per request
// - Connection pool sized for concurrent input events across sessions
func (h *Handler) CreateInputForwarder() func(sessionID string, input map[string]any) (*driver.ForwardInputResponse, error) {
	return func(sessionID string, input map[string]any) (*driver.ForwardInputResponse, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		body, err := json.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("marshal input: %w", err)
		}
		return h.recordModeService.ForwardInput(ctx, sessionID, body)
	}
}

// generateCorrelationID creates a correlation ID for tracing actions through the pipeline.
// Format: rec-{short_session_id}-{timestamp_ns}
func (h *Handler) generateCorrelationID(sessionID string) string {
	shortSession := sessionID
	if len(shortSession) > 8 {
		shortSession = shortSession[:8]
	}
	return fmt.Sprintf("rec-%s-%d", shortSession, time.Now().UnixNano())
}
