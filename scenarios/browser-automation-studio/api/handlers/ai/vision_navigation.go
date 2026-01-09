package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/internal/httpjson"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

// VisionHTTPDoer is an interface for making HTTP requests.
type VisionHTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// ============================
// Vision Navigation Handler
// ============================
//
// This handler orchestrates AI-driven browser navigation:
// 1. Validates entitlements (user must have AI credits)
// 2. Forwards navigation request to playwright-driver
// 3. Receives step callbacks and broadcasts via WebSocket
// 4. Deducts credits for each step

// VisionNavigationHandler handles AI-powered vision navigation.
type VisionNavigationHandler struct {
	log              *logrus.Logger
	driverBaseURL    string
	entitlementSvc   *entitlement.Service
	aiCreditsTracker *entitlement.AICreditsTracker
	wsHub            wsHub.HubInterface
	httpClient       VisionHTTPDoer

	// Track active navigations for status queries
	mu                 sync.RWMutex
	activeNavigations  map[string]*NavigationSession
}

// NavigationSession tracks an active navigation.
type NavigationSession struct {
	NavigationID      string
	SessionID         string
	UserID            string
	Model             string
	StartedAt         time.Time
	StepCount         int
	TotalTokens       int
	Status            string // "navigating", "completed", "failed", "aborted", "awaiting_human"
	AwaitingHuman     bool
	HumanIntervention *HumanInterventionInfo
}

// HumanInterventionInfo contains details about human intervention.
type HumanInterventionInfo struct {
	Reason           string `json:"reason"`
	Instructions     string `json:"instructions,omitempty"`
	InterventionType string `json:"interventionType"`
	Trigger          string `json:"trigger"` // "programmatic" or "ai_requested"
}

// VisionNavigationHandlerOption configures VisionNavigationHandler.
type VisionNavigationHandlerOption func(*VisionNavigationHandler)

// WithVisionNavigationHTTPClient sets a custom HTTP client.
func WithVisionNavigationHTTPClient(client VisionHTTPDoer) VisionNavigationHandlerOption {
	return func(h *VisionNavigationHandler) {
		h.httpClient = client
	}
}

// WithVisionNavigationHub sets the WebSocket hub.
func WithVisionNavigationHub(hub wsHub.HubInterface) VisionNavigationHandlerOption {
	return func(h *VisionNavigationHandler) {
		h.wsHub = hub
	}
}

// WithVisionNavigationEntitlement sets the entitlement service.
func WithVisionNavigationEntitlement(svc *entitlement.Service, tracker *entitlement.AICreditsTracker) VisionNavigationHandlerOption {
	return func(h *VisionNavigationHandler) {
		h.entitlementSvc = svc
		h.aiCreditsTracker = tracker
	}
}

// NewVisionNavigationHandler creates a new vision navigation handler.
func NewVisionNavigationHandler(log *logrus.Logger, opts ...VisionNavigationHandlerOption) *VisionNavigationHandler {
	driverURL := resolveDriverURL()

	h := &VisionNavigationHandler{
		log:               log,
		driverBaseURL:     driverURL,
		activeNavigations: make(map[string]*NavigationSession),
		httpClient:        &http.Client{Timeout: 30 * time.Second},
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// resolveDriverURL gets the playwright-driver URL from environment.
func resolveDriverURL() string {
	url := strings.TrimSpace(os.Getenv("PLAYWRIGHT_DRIVER_URL"))
	if url == "" {
		url = "http://127.0.0.1:39400"
	}
	return strings.TrimRight(url, "/")
}

// ============================
// Request/Response Types
// ============================

// AINavigateRequest is the request body for POST /api/v1/ai-navigate.
type AINavigateRequest struct {
	SessionID string `json:"session_id"`
	Prompt    string `json:"prompt"`
	Model     string `json:"model"`
	MaxSteps  int    `json:"max_steps,omitempty"`
	APIKey    string `json:"api_key,omitempty"` // Optional: use BYOK
}

// AINavigateResponse is returned when navigation starts.
type AINavigateResponse struct {
	NavigationID  string  `json:"navigation_id"`
	Status        string  `json:"status"`
	Model         string  `json:"model"`
	MaxSteps      int     `json:"max_steps"`
	EstimatedCost float64 `json:"estimated_cost,omitempty"`
}

// NavigationStepCallback is received from playwright-driver.
type NavigationStepCallback struct {
	NavigationID         string                 `json:"navigationId"`
	StepNumber           int                    `json:"stepNumber"`
	Action               map[string]interface{} `json:"action"`
	Reasoning            string                 `json:"reasoning"`
	Screenshot           string                 `json:"screenshot"` // base64
	AnnotatedScreenshot  string                 `json:"annotatedScreenshot,omitempty"`
	CurrentURL           string                 `json:"currentUrl"`
	TokensUsed           TokenUsage             `json:"tokensUsed"`
	DurationMs           int64                  `json:"durationMs"`
	GoalAchieved         bool                   `json:"goalAchieved"`
	Error                string                 `json:"error,omitempty"`
	ElementLabels        []interface{}          `json:"elementLabels,omitempty"`
	AwaitingHuman        bool                   `json:"awaitingHuman,omitempty"`
	HumanIntervention    *HumanInterventionInfo `json:"humanIntervention,omitempty"`
}

// NavigationCompleteCallback is received when navigation ends.
type NavigationCompleteCallback struct {
	NavigationID    string `json:"navigationId"`
	Status          string `json:"status"` // completed, failed, aborted, max_steps_reached
	TotalSteps      int    `json:"totalSteps"`
	TotalTokens     int    `json:"totalTokens"`
	TotalDurationMs int64  `json:"totalDurationMs"`
	FinalURL        string `json:"finalUrl"`
	Error           string `json:"error,omitempty"`
	Summary         string `json:"summary,omitempty"`
}

// TokenUsage tracks token consumption.
type TokenUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// NavigationStatusResponse is returned by GET /api/v1/ai-navigate/:id/status.
type NavigationStatusResponse struct {
	NavigationID string `json:"navigation_id"`
	SessionID    string `json:"session_id"`
	Status       string `json:"status"`
	StepCount    int    `json:"step_count"`
	TotalTokens  int    `json:"total_tokens"`
	StartedAt    string `json:"started_at"`
}

// ============================
// HTTP Handlers
// ============================

// HandleAINavigate handles POST /api/v1/ai-navigate.
// Starts AI-driven navigation for a browser session.
func (h *VisionNavigationHandler) HandleAINavigate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Decode request
	var req AINavigateRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		h.log.WithError(err).Warn("vision_navigation: failed to decode request")
		RespondError(w, ErrInvalidRequest)
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.SessionID) == "" {
		RespondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "session_id"}))
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		RespondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "prompt"}))
		return
	}
	if strings.TrimSpace(req.Model) == "" {
		RespondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "model"}))
		return
	}

	// Default max steps
	maxSteps := req.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 20
	}
	if maxSteps > 100 {
		maxSteps = 100
	}

	// Check entitlements if service is available
	var userID string
	if h.entitlementSvc != nil {
		userID = entitlement.UserIdentityFromContext(ctx)
		if userID == "" {
			userID = "anonymous"
		}

		// Check tier allows vision navigation
		ent, err := h.entitlementSvc.GetEntitlement(ctx, userID)
		if err != nil {
			h.log.WithError(err).Warn("vision_navigation: failed to get entitlement")
		} else {
			// Check if tier supports AI (tier limits defined elsewhere)
			if !h.entitlementSvc.TierCanUseAI(ent.Tier) {
				RespondError(w, &APIError{
					Status:  http.StatusForbidden,
					Code:    "AI_NOT_AVAILABLE",
					Message: "AI navigation not available for your tier",
				})
				return
			}
		}

		// Check AI credits if tracker is available
		if h.aiCreditsTracker != nil {
			creditsLimit := h.entitlementSvc.GetAICreditsLimit(ent.Tier)
			canUse, remaining, err := h.aiCreditsTracker.CanUseCredits(ctx, userID, creditsLimit)
			if err != nil {
				h.log.WithError(err).Warn("vision_navigation: failed to check credits")
			} else if !canUse {
				RespondError(w, &APIError{
					Status:  http.StatusPaymentRequired,
					Code:    "INSUFFICIENT_CREDITS",
					Message: fmt.Sprintf("Insufficient AI credits. Remaining: %d", remaining),
					Details: map[string]string{"remaining": fmt.Sprintf("%d", remaining)},
				})
				return
			}
		}
	}

	// Resolve API key
	apiKey := req.APIKey
	if apiKey == "" {
		// Try to get from server environment (for server-side AI access)
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if apiKey == "" {
		RespondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "api_key",
			"hint":  "Provide api_key in request or set OPENROUTER_API_KEY environment variable",
		}))
		return
	}

	// Generate navigation ID
	navigationID := fmt.Sprintf("nav_%s", uuid.New().String()[:12])

	// Build callback URL (this API server receives step events)
	// The callback URL should point to this server's callback endpoint
	callbackURL := h.resolveCallbackURL(r, navigationID)

	// Track the navigation session
	session := &NavigationSession{
		NavigationID: navigationID,
		SessionID:    req.SessionID,
		UserID:       userID,
		Model:        req.Model,
		StartedAt:    time.Now(),
		Status:       "navigating",
	}
	h.mu.Lock()
	h.activeNavigations[navigationID] = session
	h.mu.Unlock()

	// Forward to playwright-driver
	driverReq := map[string]interface{}{
		"prompt":       req.Prompt,
		"model":        req.Model,
		"api_key":      apiKey,
		"max_steps":    maxSteps,
		"callback_url": callbackURL,
	}

	driverURL := fmt.Sprintf("%s/session/%s/ai-navigate", h.driverBaseURL, req.SessionID)

	body, err := json.Marshal(driverReq)
	if err != nil {
		h.log.WithError(err).Error("vision_navigation: failed to marshal driver request")
		h.removeNavigation(navigationID)
		RespondError(w, ErrInternalServer)
		return
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, driverURL, bytes.NewReader(body))
	if err != nil {
		h.log.WithError(err).Error("vision_navigation: failed to create request")
		h.removeNavigation(navigationID)
		RespondError(w, ErrInternalServer)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(httpReq)
	if err != nil {
		h.log.WithError(err).Error("vision_navigation: driver request failed")
		h.removeNavigation(navigationID)
		RespondError(w, &APIError{
			Status:  http.StatusBadGateway,
			Code:    "DRIVER_UNAVAILABLE",
			Message: "Failed to connect to browser driver",
			Details: map[string]string{"error": err.Error()},
		})
		return
	}
	defer resp.Body.Close()

	// Read driver response
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		h.log.WithField("status", resp.StatusCode).WithField("body", string(respBody)).
			Warn("vision_navigation: driver returned error")
		h.removeNavigation(navigationID)

		// Parse driver error
		var driverErr map[string]interface{}
		if json.Unmarshal(respBody, &driverErr) == nil {
			if msg, ok := driverErr["message"].(string); ok {
				RespondError(w, &APIError{
					Status:  resp.StatusCode,
					Code:    "DRIVER_ERROR",
					Message: msg,
				})
				return
			}
		}
		RespondError(w, &APIError{
			Status:  resp.StatusCode,
			Code:    "DRIVER_ERROR",
			Message: "Driver returned an error",
		})
		return
	}

	// Parse driver response
	var driverResp struct {
		NavigationID string `json:"navigation_id"`
		Status       string `json:"status"`
		Model        string `json:"model"`
		MaxSteps     int    `json:"max_steps"`
	}
	if err := json.Unmarshal(respBody, &driverResp); err != nil {
		h.log.WithError(err).Error("vision_navigation: failed to parse driver response")
		h.removeNavigation(navigationID)
		RespondError(w, ErrInternalServer)
		return
	}

	// Update navigation ID from driver (they may differ)
	if driverResp.NavigationID != "" {
		h.mu.Lock()
		delete(h.activeNavigations, navigationID)
		navigationID = driverResp.NavigationID
		session.NavigationID = navigationID
		h.activeNavigations[navigationID] = session
		h.mu.Unlock()
	}

	h.log.WithFields(logrus.Fields{
		"navigation_id": navigationID,
		"session_id":    req.SessionID,
		"model":         req.Model,
		"max_steps":     maxSteps,
	}).Info("vision_navigation: started")

	// Return response
	response := AINavigateResponse{
		NavigationID: navigationID,
		Status:       "started",
		Model:        req.Model,
		MaxSteps:     maxSteps,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

// HandleAINavigateCallback handles POST /api/v1/internal/ai-navigate/callback.
// Receives step events from playwright-driver.
func (h *VisionNavigationHandler) HandleAINavigateCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.WithError(err).Warn("vision_navigation_callback: failed to read body")
		RespondError(w, ErrInvalidRequest)
		return
	}

	// Try to parse as step event
	var stepEvent NavigationStepCallback
	if err := json.Unmarshal(body, &stepEvent); err == nil && stepEvent.StepNumber > 0 {
		h.handleStepCallback(ctx, w, &stepEvent)
		return
	}

	// Try to parse as complete event
	var completeEvent NavigationCompleteCallback
	if err := json.Unmarshal(body, &completeEvent); err == nil && completeEvent.Status != "" {
		h.handleCompleteCallback(ctx, w, &completeEvent)
		return
	}

	h.log.WithField("body", string(body)).Warn("vision_navigation_callback: unknown event type")
	RespondError(w, ErrInvalidRequest)
}

// handleStepCallback processes a navigation step event.
func (h *VisionNavigationHandler) handleStepCallback(ctx context.Context, w http.ResponseWriter, event *NavigationStepCallback) {
	h.log.WithFields(logrus.Fields{
		"navigation_id":  event.NavigationID,
		"step_number":    event.StepNumber,
		"action_type":    event.Action["type"],
		"goal_achieved":  event.GoalAchieved,
		"awaiting_human": event.AwaitingHuman,
	}).Debug("vision_navigation_callback: received step")

	// Update navigation session
	h.mu.Lock()
	session := h.activeNavigations[event.NavigationID]
	if session != nil {
		session.StepCount = event.StepNumber
		session.TotalTokens += event.TokensUsed.TotalTokens
		session.AwaitingHuman = event.AwaitingHuman
		session.HumanIntervention = event.HumanIntervention
		if event.AwaitingHuman {
			session.Status = "awaiting_human"
		}
	}
	h.mu.Unlock()

	// Charge credits if tracker is available
	if h.aiCreditsTracker != nil && session != nil {
		// Calculate credit cost based on tokens
		// Simple model: 1 credit per 1000 tokens (configurable)
		credits := (event.TokensUsed.TotalTokens + 999) / 1000
		if credits < 1 {
			credits = 1
		}

		err := h.aiCreditsTracker.ChargeCredits(ctx, session.UserID, credits, "vision_navigation", session.Model)
		if err != nil {
			h.log.WithError(err).Warn("vision_navigation_callback: failed to charge credits")
		}
	}

	// Broadcast via WebSocket
	if h.wsHub != nil && session != nil {
		// Always send step event
		wsEvent := map[string]interface{}{
			"type":         "ai_navigation_step",
			"navigationId": event.NavigationID,
			"sessionId":    session.SessionID,
			"stepNumber":   event.StepNumber,
			"action":       event.Action,
			"reasoning":    event.Reasoning,
			"currentUrl":   event.CurrentURL,
			"goalAchieved": event.GoalAchieved,
			"tokensUsed":   event.TokensUsed,
			"durationMs":   event.DurationMs,
			"timestamp":    time.Now().UTC().Format(time.RFC3339),
		}
		if event.Error != "" {
			wsEvent["error"] = event.Error
		}

		h.wsHub.BroadcastEnvelope(wsEvent)

		// If awaiting human intervention, send additional event
		if event.AwaitingHuman && event.HumanIntervention != nil {
			humanEvent := map[string]interface{}{
				"type":             "ai_navigation_awaiting_human",
				"navigationId":    event.NavigationID,
				"sessionId":       session.SessionID,
				"stepNumber":      event.StepNumber,
				"reason":          event.HumanIntervention.Reason,
				"interventionType": event.HumanIntervention.InterventionType,
				"trigger":         event.HumanIntervention.Trigger,
				"timestamp":       time.Now().UTC().Format(time.RFC3339),
			}
			if event.HumanIntervention.Instructions != "" {
				humanEvent["instructions"] = event.HumanIntervention.Instructions
			}

			h.wsHub.BroadcastEnvelope(humanEvent)

			h.log.WithFields(logrus.Fields{
				"navigation_id":     event.NavigationID,
				"intervention_type": event.HumanIntervention.InterventionType,
				"trigger":           event.HumanIntervention.Trigger,
			}).Info("vision_navigation_callback: awaiting human intervention")
		}
	}

	// Respond with acknowledgment
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"received": true,
		"abort":    false, // Could implement abort signaling here
	})
}

// handleCompleteCallback processes a navigation complete event.
func (h *VisionNavigationHandler) handleCompleteCallback(ctx context.Context, w http.ResponseWriter, event *NavigationCompleteCallback) {
	h.log.WithFields(logrus.Fields{
		"navigation_id": event.NavigationID,
		"status":        event.Status,
		"total_steps":   event.TotalSteps,
		"total_tokens":  event.TotalTokens,
	}).Info("vision_navigation_callback: navigation completed")

	// Update and finalize navigation session
	h.mu.Lock()
	session := h.activeNavigations[event.NavigationID]
	if session != nil {
		session.Status = event.Status
		session.StepCount = event.TotalSteps
		session.TotalTokens = event.TotalTokens
	}
	// Keep session for a while for status queries, then clean up
	h.mu.Unlock()

	// Broadcast via WebSocket
	if h.wsHub != nil && session != nil {
		wsEvent := map[string]interface{}{
			"type":            "ai_navigation_complete",
			"navigationId":    event.NavigationID,
			"sessionId":       session.SessionID,
			"status":          event.Status,
			"totalSteps":      event.TotalSteps,
			"totalTokens":     event.TotalTokens,
			"totalDurationMs": event.TotalDurationMs,
			"finalUrl":        event.FinalURL,
			"timestamp":       time.Now().UTC().Format(time.RFC3339),
		}
		if event.Error != "" {
			wsEvent["error"] = event.Error
		}
		if event.Summary != "" {
			wsEvent["summary"] = event.Summary
		}

		h.wsHub.BroadcastEnvelope(wsEvent)
	}

	// Schedule cleanup after a delay
	go func() {
		time.Sleep(5 * time.Minute)
		h.removeNavigation(event.NavigationID)
	}()

	// Respond with acknowledgment
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"received": true,
	})
}

// HandleAINavigateStatus handles GET /api/v1/ai-navigate/:id/status.
func (h *VisionNavigationHandler) HandleAINavigateStatus(w http.ResponseWriter, r *http.Request) {
	// Extract navigation ID from URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		RespondError(w, ErrInvalidRequest)
		return
	}
	navigationID := parts[len(parts)-2] // /api/v1/ai-navigate/{id}/status

	h.mu.RLock()
	session := h.activeNavigations[navigationID]
	h.mu.RUnlock()

	if session == nil {
		RespondError(w, &APIError{
			Status:  http.StatusNotFound,
			Code:    "NAVIGATION_NOT_FOUND",
			Message: "Navigation session not found",
		})
		return
	}

	response := NavigationStatusResponse{
		NavigationID: session.NavigationID,
		SessionID:    session.SessionID,
		Status:       session.Status,
		StepCount:    session.StepCount,
		TotalTokens:  session.TotalTokens,
		StartedAt:    session.StartedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleAINavigateAbort handles POST /api/v1/ai-navigate/:id/abort.
func (h *VisionNavigationHandler) HandleAINavigateAbort(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract navigation ID from URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		RespondError(w, ErrInvalidRequest)
		return
	}
	navigationID := parts[len(parts)-2] // /api/v1/ai-navigate/{id}/abort

	h.mu.RLock()
	session := h.activeNavigations[navigationID]
	h.mu.RUnlock()

	if session == nil {
		RespondError(w, &APIError{
			Status:  http.StatusNotFound,
			Code:    "NAVIGATION_NOT_FOUND",
			Message: "Navigation session not found",
		})
		return
	}

	// Forward abort to playwright-driver
	driverURL := fmt.Sprintf("%s/session/%s/ai-navigate/abort", h.driverBaseURL, session.SessionID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, driverURL, nil)
	if err != nil {
		h.log.WithError(err).Error("vision_navigation: failed to create abort request")
		RespondError(w, ErrInternalServer)
		return
	}

	resp, err := h.httpClient.Do(httpReq)
	if err != nil {
		h.log.WithError(err).Warn("vision_navigation: abort request failed")
		// Still mark as aborting locally
	} else {
		resp.Body.Close()
	}

	// Update status
	h.mu.Lock()
	if session := h.activeNavigations[navigationID]; session != nil {
		session.Status = "aborting"
	}
	h.mu.Unlock()

	h.log.WithField("navigation_id", navigationID).Info("vision_navigation: abort requested")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "aborting",
		"navigation_id": navigationID,
		"message":       "Abort signal sent. Navigation will stop after current step.",
	})
}

// HandleAINavigateResume handles POST /api/v1/ai-navigate/:id/resume.
// Resumes navigation after human intervention is complete.
func (h *VisionNavigationHandler) HandleAINavigateResume(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract navigation ID from URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		RespondError(w, ErrInvalidRequest)
		return
	}
	navigationID := parts[len(parts)-2] // /api/v1/ai-navigate/{id}/resume

	h.mu.RLock()
	session := h.activeNavigations[navigationID]
	h.mu.RUnlock()

	if session == nil {
		RespondError(w, &APIError{
			Status:  http.StatusNotFound,
			Code:    "NAVIGATION_NOT_FOUND",
			Message: "Navigation session not found",
		})
		return
	}

	// Check if session is awaiting human intervention
	if !session.AwaitingHuman {
		RespondError(w, &APIError{
			Status:  http.StatusConflict,
			Code:    "NOT_AWAITING_HUMAN",
			Message: "Navigation is not awaiting human intervention",
		})
		return
	}

	// Forward resume to playwright-driver
	driverURL := fmt.Sprintf("%s/session/%s/ai-navigate/resume", h.driverBaseURL, session.SessionID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, driverURL, nil)
	if err != nil {
		h.log.WithError(err).Error("vision_navigation: failed to create resume request")
		RespondError(w, ErrInternalServer)
		return
	}

	resp, err := h.httpClient.Do(httpReq)
	if err != nil {
		h.log.WithError(err).Warn("vision_navigation: resume request failed")
		RespondError(w, &APIError{
			Status:  http.StatusBadGateway,
			Code:    "DRIVER_UNAVAILABLE",
			Message: "Failed to connect to browser driver",
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		h.log.WithField("status", resp.StatusCode).WithField("body", string(respBody)).
			Warn("vision_navigation: driver resume returned error")
		RespondError(w, &APIError{
			Status:  resp.StatusCode,
			Code:    "DRIVER_ERROR",
			Message: "Driver returned an error during resume",
		})
		return
	}

	// Update session status
	h.mu.Lock()
	if s := h.activeNavigations[navigationID]; s != nil {
		s.Status = "navigating"
		s.AwaitingHuman = false
		s.HumanIntervention = nil
	}
	h.mu.Unlock()

	// Broadcast resume event via WebSocket
	if h.wsHub != nil {
		resumeEvent := map[string]interface{}{
			"type":         "ai_navigation_resumed",
			"navigationId": navigationID,
			"sessionId":    session.SessionID,
			"timestamp":    time.Now().UTC().Format(time.RFC3339),
		}
		h.wsHub.BroadcastEnvelope(resumeEvent)
	}

	h.log.WithField("navigation_id", navigationID).Info("vision_navigation: resumed after human intervention")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "resumed",
		"navigation_id": navigationID,
		"message":       "Navigation resumed. Will continue from where it paused.",
	})
}

// ============================
// Helper Methods
// ============================

// removeNavigation removes a navigation session from tracking.
func (h *VisionNavigationHandler) removeNavigation(navigationID string) {
	h.mu.Lock()
	delete(h.activeNavigations, navigationID)
	h.mu.Unlock()
}

// resolveCallbackURL builds the callback URL for step events.
func (h *VisionNavigationHandler) resolveCallbackURL(r *http.Request, navigationID string) string {
	// Try to build from request host
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// Check for forwarded headers
	if forwardedProto := r.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
		scheme = forwardedProto
	}

	host := r.Host
	if forwardedHost := r.Header.Get("X-Forwarded-Host"); forwardedHost != "" {
		host = forwardedHost
	}

	// Fallback to internal callback URL
	if host == "" {
		host = "127.0.0.1:8110" // Default API port
	}

	return fmt.Sprintf("%s://%s/api/v1/internal/ai-navigate/callback", scheme, host)
}
