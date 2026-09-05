package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/domain"
	"github.com/vrooli/browser-automation-studio/internal/protoconv"
	"github.com/vrooli/browser-automation-studio/performance"
	livecapture "github.com/vrooli/browser-automation-studio/services/live-capture"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
	workflowservice "github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/websocket"
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
			h.log.WithError(err).WithFields(map[string]interface{}{
				"session_id": result.SessionID,
				"profile_id": profileID,
				"tab_count":  len(openTabs),
			}).Warn("Failed to restore tabs from profile")
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

	// Capture storage state and open tabs before closing (for session profile persistence)
	var storageState json.RawMessage
	var openTabs []sessionprofilepersistence.TabState
	profileID := h.getActiveSessionProfile(sessionID)
	if profileID != "" && h.sessionProfileService != nil {
		// Capture storage state
		if state, err := h.recordModeService.GetStorageState(ctx, sessionID); err != nil {
			h.log.WithError(err).WithFields(map[string]interface{}{
				"session_id": sessionID,
				"profile_id": profileID,
			}).Warn("Failed to capture storage state before closing session")
		} else {
			storageState = state
		}

		// Capture open tabs for restoration
		if pages, activePageID, err := h.recordModeService.GetOpenPages(sessionID); err != nil {
			h.log.WithError(err).WithFields(map[string]interface{}{
				"session_id": sessionID,
				"profile_id": profileID,
			}).Warn("Failed to capture open tabs before closing session")
		} else {
			openTabs = make([]sessionprofilepersistence.TabState, 0, len(pages))
			for i, page := range pages {
				openTabs = append(openTabs, sessionprofilepersistence.TabState{
					URL:      page.URL,
					Title:    page.Title,
					IsActive: page.ID == activePageID,
					Order:    i,
				})
			}
		}
	}

	// Delegate to recordmode service
	if err := h.recordModeService.CloseSession(ctx, sessionID); err != nil {
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

	// Persist storage state and open tabs to profile after successful close
	if profileID != "" && h.sessionProfileService != nil {
		if len(storageState) > 0 {
			if _, err := h.sessionProfileService.SaveStorageState(sessionprofilepersistence.ProfileID(profileID), storageState); err != nil {
				h.log.WithError(err).WithFields(map[string]interface{}{
					"profile_id": profileID,
					"session_id": sessionID,
				}).Warn("Failed to persist session profile storage state")
			}
		}

		// Save open tabs for restoration on next session start
		if len(openTabs) > 0 {
			if _, err := h.sessionProfileService.SaveOpenTabs(sessionprofilepersistence.ProfileID(profileID), openTabs); err != nil {
				h.log.WithError(err).WithFields(map[string]interface{}{
					"profile_id": profileID,
					"session_id": sessionID,
					"tab_count":  len(openTabs),
				}).Warn("Failed to persist session profile open tabs")
			}
		}
	}

	h.clearActiveSessionProfile(sessionID)

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"session_id": sessionID,
		"status":     "closed",
	})
}

// GetRecordingDebug handles GET /api/v1/recordings/live/{sessionId}/debug
// Gets live debugging info for an active recording session.
// This proxies directly to the playwright-driver's debug endpoint.
func (h *Handler) GetRecordingDebug(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	driverURL, err := getPlaywrightDriverURL()
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "resolve_playwright_driver"}))
		return
	}
	targetURL := fmt.Sprintf("%s/session/%s/record/debug", driverURL, sessionID)

	// #nosec G704 -- driverURL is validated by driver.ResolveEndpoint; sessionID is path data only.
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	// #nosec G704 -- request target is restricted to the validated Playwright driver endpoint.
	resp, err := client.Do(req)
	if err != nil {
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

	// Delegate directly to driver client (no service-layer business logic needed)
	resp, err := h.recordModeService.DriverClient().GetRecordedActions(ctx, sessionID, clearActions)
	if err != nil {
		h.log.WithError(err).Error("Failed to get recorded actions")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Map service response to handler response type
	driverResp := &GetActionsResponse{
		SessionID:   resp.SessionID,
		IsRecording: resp.IsRecording,
		Actions:     resp.Actions,
		Count:       len(resp.Actions),
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

	// Build V2 flow definition for storage
	v2, err := workflowservice.BuildFlowDefinitionV2ForWrite(genResult.FlowDefinition, nil, nil)
	if err != nil {
		h.respondError(w, ErrInvalidWorkflowPayload.WithDetails(map[string]string{"error": err.Error()}))
		return
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

// HandleDriverFrameStream handles WebSocket connection for binary frame streaming from playwright-driver.
// GET /ws/recording/{sessionId}/frames
// This is more efficient than HTTP POST as it:
// 1. Uses a persistent connection (no per-frame TCP overhead)
// 2. Sends raw binary JPEG data (no base64 encoding = 33% smaller)
// 3. Pass-through to browser clients (no JSON parsing/re-encoding)
//
// When performance mode is enabled, frames may include a performance header:
// [4 bytes: header length (uint32 big-endian)][N bytes: JSON perf header][remaining: JPEG data]
// Detection: If first 2 bytes are 0xFF 0xD8 (JPEG magic), no header present.
func (h *Handler) HandleDriverFrameStream(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.log.Error("Missing sessionId in driver frame stream request")
		http.Error(w, "Missing sessionId", http.StatusBadRequest)
		return
	}

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.WithError(err).Error("Failed to upgrade driver frame stream connection")
		return
	}
	defer conn.Close()

	h.log.WithField("session_id", sessionID).Info("Driver frame stream connected")

	// Get performance config (used for logging/broadcast intervals)
	cfg := config.Load()

	// Get or create performance collector for this session (always, for potential runtime enabling)
	collector := h.perfRegistry.GetOrCreate(sessionID)

	// Read binary frames from driver and broadcast to browser clients
	for {
		receiveStart := time.Now()

		messageType, data, err := conn.ReadMessage()
		if err != nil {
			// Check for normal closure
			if websocket.IsCloseError(err) {
				h.log.WithField("session_id", sessionID).Debug("Driver frame stream closed normally")
			} else {
				h.log.WithError(err).WithField("session_id", sessionID).Warn("Driver frame stream read error")
			}
			break
		}

		// Only process binary messages (JPEG data)
		if messageType != websocket.BinaryMessage {
			continue
		}

		receiveMs := float64(time.Since(receiveStart).Microseconds()) / 1000.0

		frameData, driverHeader, decodeErr := decodeDriverFrame(data)
		if decodeErr != nil {
			h.log.WithError(decodeErr).Debug("Failed to parse frame perf header")
		}

		// Broadcast binary frame to subscribed browser clients
		broadcastStart := time.Now()
		if h.wsHub.HasRecordingSubscribers(sessionID) {
			h.wsHub.BroadcastBinaryFrame(sessionID, frameData)
		}
		broadcastMs := float64(time.Since(broadcastStart).Microseconds()) / 1000.0

		// Record performance data if driver sent a perf header
		// (presence of header indicates per-session perf mode is enabled)
		if driverHeader != nil {
			timing := &performance.FrameTimings{
				FrameID:         driverHeader.FrameID,
				SessionID:       sessionID,
				Timestamp:       time.Now(),
				DriverCaptureMs: driverHeader.CaptureMs,
				DriverCompareMs: driverHeader.CompareMs,
				DriverWsSendMs:  driverHeader.WsSendMs,
				DriverTotalMs:   driverHeader.CaptureMs + driverHeader.CompareMs + driverHeader.WsSendMs,
				APIReceiveMs:    receiveMs,
				APIBroadcastMs:  broadcastMs,
				APITotalMs:      receiveMs + broadcastMs,
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
	h.perfRegistry.Remove(sessionID)

	h.log.WithField("session_id", sessionID).Info("Driver frame stream disconnected")
}

// ReloadRecordingSession handles POST /api/v1/recordings/live/{sessionId}/reload
// Reloads the current page in the recording session.
func (h *Handler) ReloadRecordingSession(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	var req ReloadRecordingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	// Delegate directly to driver client (no service-layer business logic needed)
	resp, err := h.recordModeService.DriverClient().Reload(ctx, sessionID, &driver.ReloadRequest{
		WaitUntil: req.WaitUntil,
		TimeoutMs: req.TimeoutMs,
	})
	if err != nil {
		h.log.WithError(err).Error("Failed to reload recording session")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Create a reload action for the recording timeline
	if sess, ok := h.recordModeService.GetSession(sessionID); ok && sess.Pages() != nil {
		correlationID := h.generateCorrelationID(sessionID)
		pages := sess.Pages()
		activePageID := pages.GetActivePageID()

		now := time.Now()
		reloadAction := driver.RecordedAction{
			ID:         uuid.NewString(),
			SessionID:  sessionID,
			Timestamp:  now.Format(time.RFC3339Nano),
			ActionType: "reload",
			Confidence: 1.0,
			URL:        resp.URL,
			PageID:     activePageID.String(),
			PageTitle:  resp.Title,
		}

		h.recordModeService.AddTimelineAction(sessionID, &reloadAction, activePageID)

		// Broadcast unified timeline entry
		broadcastResult := h.wsHub.BroadcastTimelineEntry(sessionID, h.createTimelineEntry(&reloadAction))

		h.log.WithFields(map[string]interface{}{
			"correlation_id":   correlationID,
			"session_id":       sessionID,
			"action_type":      "reload",
			"action_id":        reloadAction.ID,
			"url":              resp.URL,
			"persisted":        true,
			"broadcast_sent":   broadcastResult.SentCount > 0,
			"subscriber_count": broadcastResult.SubscriberCount,
		}).Debug("Reload action recorded")
	}

	driverResp := ReloadRecordingResponse{
		SessionID:    sessionID,
		URL:          resp.URL,
		Title:        resp.Title,
		CanGoBack:    resp.CanGoBack,
		CanGoForward: resp.CanGoForward,
	}

	h.respondSuccess(w, http.StatusOK, driverResp)
}

// GoBackRecordingSession handles POST /api/v1/recordings/live/{sessionId}/go-back
// Navigates back in browser history.
func (h *Handler) GoBackRecordingSession(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	var req GoBackRecordingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	// Delegate directly to driver client (no service-layer business logic needed)
	resp, err := h.recordModeService.DriverClient().GoBack(ctx, sessionID, &driver.GoBackRequest{
		WaitUntil: req.WaitUntil,
		TimeoutMs: req.TimeoutMs,
	})
	if err != nil {
		h.log.WithError(err).Error("Failed to go back in recording session")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Create a goBack action for the recording timeline
	if sess, ok := h.recordModeService.GetSession(sessionID); ok && sess.Pages() != nil {
		correlationID := h.generateCorrelationID(sessionID)
		pages := sess.Pages()
		activePageID := pages.GetActivePageID()

		// Update page info
		pages.UpdatePageInfo(activePageID, resp.URL, resp.Title)

		now := time.Now()
		goBackAction := driver.RecordedAction{
			ID:         uuid.NewString(),
			SessionID:  sessionID,
			Timestamp:  now.Format(time.RFC3339Nano),
			ActionType: "goBack",
			Confidence: 1.0,
			URL:        resp.URL,
			PageID:     activePageID.String(),
			PageTitle:  resp.Title,
		}

		h.recordModeService.AddTimelineAction(sessionID, &goBackAction, activePageID)

		// Broadcast unified timeline entry
		broadcastResult := h.wsHub.BroadcastTimelineEntry(sessionID, h.createTimelineEntry(&goBackAction))

		// Broadcast page_navigated event
		pageEvent := &domain.PageEvent{
			ID:        uuid.New(),
			Type:      domain.PageEventNavigated,
			PageID:    activePageID,
			URL:       resp.URL,
			Title:     resp.Title,
			Timestamp: now,
		}
		h.wsHub.BroadcastPageEvent(sessionID, pageEvent)

		h.log.WithFields(map[string]interface{}{
			"correlation_id":   correlationID,
			"session_id":       sessionID,
			"action_type":      "goBack",
			"action_id":        goBackAction.ID,
			"url":              resp.URL,
			"persisted":        true,
			"broadcast_sent":   broadcastResult.SentCount > 0,
			"subscriber_count": broadcastResult.SubscriberCount,
		}).Debug("GoBack action recorded")
	}

	driverResp := GoBackRecordingResponse{
		SessionID:    sessionID,
		URL:          resp.URL,
		Title:        resp.Title,
		CanGoBack:    resp.CanGoBack,
		CanGoForward: resp.CanGoForward,
	}

	h.respondSuccess(w, http.StatusOK, driverResp)
}

// GoForwardRecordingSession handles POST /api/v1/recordings/live/{sessionId}/go-forward
// Navigates forward in browser history.
func (h *Handler) GoForwardRecordingSession(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	var req GoForwardRecordingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	// Delegate directly to driver client (no service-layer business logic needed)
	resp, err := h.recordModeService.DriverClient().GoForward(ctx, sessionID, &driver.GoForwardRequest{
		WaitUntil: req.WaitUntil,
		TimeoutMs: req.TimeoutMs,
	})
	if err != nil {
		h.log.WithError(err).Error("Failed to go forward in recording session")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Create a goForward action for the recording timeline
	if sess, ok := h.recordModeService.GetSession(sessionID); ok && sess.Pages() != nil {
		correlationID := h.generateCorrelationID(sessionID)
		pages := sess.Pages()
		activePageID := pages.GetActivePageID()

		// Update page info
		pages.UpdatePageInfo(activePageID, resp.URL, resp.Title)

		now := time.Now()
		goForwardAction := driver.RecordedAction{
			ID:         uuid.NewString(),
			SessionID:  sessionID,
			Timestamp:  now.Format(time.RFC3339Nano),
			ActionType: "goForward",
			Confidence: 1.0,
			URL:        resp.URL,
			PageID:     activePageID.String(),
			PageTitle:  resp.Title,
		}

		h.recordModeService.AddTimelineAction(sessionID, &goForwardAction, activePageID)

		// Broadcast unified timeline entry
		broadcastResult := h.wsHub.BroadcastTimelineEntry(sessionID, h.createTimelineEntry(&goForwardAction))

		// Broadcast page_navigated event
		pageEvent := &domain.PageEvent{
			ID:        uuid.New(),
			Type:      domain.PageEventNavigated,
			PageID:    activePageID,
			URL:       resp.URL,
			Title:     resp.Title,
			Timestamp: now,
		}
		h.wsHub.BroadcastPageEvent(sessionID, pageEvent)

		h.log.WithFields(map[string]interface{}{
			"correlation_id":   correlationID,
			"session_id":       sessionID,
			"action_type":      "goForward",
			"action_id":        goForwardAction.ID,
			"url":              resp.URL,
			"persisted":        true,
			"broadcast_sent":   broadcastResult.SentCount > 0,
			"subscriber_count": broadcastResult.SubscriberCount,
		}).Debug("GoForward action recorded")
	}

	driverResp := GoForwardRecordingResponse{
		SessionID:    sessionID,
		URL:          resp.URL,
		Title:        resp.Title,
		CanGoBack:    resp.CanGoBack,
		CanGoForward: resp.CanGoForward,
	}

	h.respondSuccess(w, http.StatusOK, driverResp)
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

	resp, err := h.recordModeService.DriverClient().CaptureScreenshot(ctx, sessionID, svcReq)
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

	// Delegate directly to driver client (no service-layer business logic needed)
	resp, err := h.recordModeService.DriverClient().UpdateViewport(ctx, sessionID, &driver.UpdateViewportRequest{
		Width:  reqBody.Width,
		Height: reqBody.Height,
	})
	if err != nil {
		h.log.WithError(err).Error("Failed to update recording viewport")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Map service response to handler response type
	var width, height int
	if resp.ActualViewport != nil {
		width = resp.ActualViewport.Width
		height = resp.ActualViewport.Height
	}
	driverResp := struct {
		SessionID string `json:"session_id"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	}{
		SessionID: resp.SessionID,
		Width:     width,
		Height:    height,
	}

	h.respondSuccess(w, http.StatusOK, driverResp)
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

	resp, err := h.recordModeService.DriverClient().UpdateStreamSettings(ctx, sessionID, svcReq)
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

	// Delegate directly to driver client (no service-layer business logic needed)
	err = h.recordModeService.DriverClient().ForwardInput(ctx, sessionID, bodyBytes)
	if err != nil {
		h.log.WithError(err).Error("Failed to forward recording input")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
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

	// Delegate directly to driver client (no service-layer business logic needed)
	resp, err := h.recordModeService.DriverClient().GetFrame(ctx, sessionID, r.URL.RawQuery)
	if err != nil {
		h.log.WithError(err).Error("Failed to get frame")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Map service response to handler response
	driverResp := RecordingFrameResponse{
		SessionID:   sessionID,
		Mime:        resp.MediaType,
		Image:       resp.Data,
		Width:       resp.Width,
		Height:      resp.Height,
		CapturedAt:  resp.CapturedAt,
		ContentHash: resp.ContentHash,
		PageTitle:   resp.PageTitle,
		PageURL:     resp.PageURL,
	}

	// Generate ETag from content hash provided by playwright-driver.
	// The driver computes MD5 hash of raw JPEG buffer, which is a reliable
	// content fingerprint that changes if and only if the frame content changes.
	var etag string
	if driverResp.ContentHash != "" {
		etag = fmt.Sprintf(`"%s"`, driverResp.ContentHash)
	} else {
		// Fallback for older driver versions without content_hash field
		etag = fmt.Sprintf(`"%s"`, driverResp.CapturedAt)
	}

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

	h.respondSuccess(w, http.StatusOK, driverResp)
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
func (h *Handler) CreateInputForwarder() func(sessionID string, input map[string]any) error {
	// Shared HTTP client with connection pooling for all input forwarding.
	// This dramatically reduces latency vs creating a new client per request.
	// Keep-alive connections mean subsequent requests reuse existing TCP connections.
	transport := &http.Transport{
		MaxIdleConns:        100,              // Total pool size
		MaxIdleConnsPerHost: 10,               // Per-driver connections (usually just one driver)
		IdleConnTimeout:     90 * time.Second, // Keep connections warm
		DisableKeepAlives:   false,            // Explicitly enable keep-alive
	}
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
	}

	return func(sessionID string, input map[string]any) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second) // Tighter timeout for input
		defer cancel()

		driverBaseURL, err := getPlaywrightDriverURL()
		if err != nil {
			return fmt.Errorf("resolve Playwright driver: %w", err)
		}
		driverURL := fmt.Sprintf("%s/session/%s/record/input", driverBaseURL, sessionID)

		jsonBody, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("marshal input: %w", err)
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, driverURL, bytes.NewReader(jsonBody))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(httpReq)
		if err != nil {
			return fmt.Errorf("forward input: %w", err)
		}
		defer resp.Body.Close()

		// Drain the response body to enable connection reuse
		_, _ = io.Copy(io.Discard, resp.Body)

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("driver returned %d", resp.StatusCode)
		}

		return nil
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
