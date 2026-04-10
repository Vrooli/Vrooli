package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/internal/httpjson"
	"github.com/vrooli/browser-automation-studio/services/credits"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	"github.com/vrooli/browser-automation-studio/services/vision"
)

// ============================
// Vision Navigation Handler
// ============================
//
// This handler orchestrates AI-driven browser navigation using pluggable navigators:
// 1. Selects appropriate navigator based on client source and request
// 2. Validates entitlements based on navigator's credit policy
// 3. Delegates navigation to the selected navigator
// 4. Receives step callbacks and forwards to the navigator

// VisionNavigationHandler handles AI-powered vision navigation.
type VisionNavigationHandler struct {
	log            *logrus.Logger
	creditService  credits.CreditService
	registry       *vision.NavigatorRegistry
	playwrightNav  *vision.PlaywrightVisionNavigator // Direct reference for callback handling
}

// VisionNavigationHandlerOption configures VisionNavigationHandler.
type VisionNavigationHandlerOption func(*VisionNavigationHandler)

// WithVisionNavigationCreditService sets the unified credit service.
func WithVisionNavigationCreditService(svc credits.CreditService) VisionNavigationHandlerOption {
	return func(h *VisionNavigationHandler) {
		h.creditService = svc
	}
}

// WithVisionNavigationRegistry sets the navigator registry.
func WithVisionNavigationRegistry(registry *vision.NavigatorRegistry) VisionNavigationHandlerOption {
	return func(h *VisionNavigationHandler) {
		h.registry = registry
	}
}

// WithPlaywrightNavigator sets a direct reference to the playwright navigator for callback handling.
func WithPlaywrightNavigator(nav *vision.PlaywrightVisionNavigator) VisionNavigationHandlerOption {
	return func(h *VisionNavigationHandler) {
		h.playwrightNav = nav
	}
}

// NewVisionNavigationHandler creates a new vision navigation handler.
func NewVisionNavigationHandler(log *logrus.Logger, opts ...VisionNavigationHandlerOption) *VisionNavigationHandler {
	h := &VisionNavigationHandler{
		log: log,
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// ============================
// Request/Response Types
// ============================

// AINavigateRequest is the request body for POST /api/v1/ai-navigate.
type AINavigateRequest struct {
	SessionID     string `json:"session_id"`
	Prompt        string `json:"prompt"`
	Model         string `json:"model"`
	MaxSteps      int    `json:"max_steps,omitempty"`
	APIKey        string `json:"api_key,omitempty"`           // Optional: use BYOK
	NavigatorType string `json:"navigator_type,omitempty"`    // Optional: "playwright" | "claude_code"
}

// AINavigateResponse is returned when navigation starts.
type AINavigateResponse struct {
	NavigationID  string `json:"navigation_id"`
	Status        string `json:"status"`
	Model         string `json:"model"`
	MaxSteps      int    `json:"max_steps"`
	NavigatorType string `json:"navigator_type"`
}

// NavigationStatusResponse is returned by GET /api/v1/ai-navigate/:id/status.
type NavigationStatusResponse struct {
	NavigationID  string `json:"navigation_id"`
	SessionID     string `json:"session_id"`
	Status        string `json:"status"`
	StepCount     int    `json:"step_count"`
	TotalTokens   int    `json:"total_tokens"`
	StartedAt     string `json:"started_at"`
	NavigatorType string `json:"navigator_type,omitempty"`
}

// ============================
// HTTP Handlers
// ============================

// HandleListNavigators handles GET /api/v1/ai-navigate/navigators.
// Lists available vision navigators for the current client.
func (h *VisionNavigationHandler) HandleListNavigators(w http.ResponseWriter, r *http.Request) {
	if h.registry == nil {
		RespondError(w, &APIError{
			Status:  http.StatusServiceUnavailable,
			Code:    "REGISTRY_NOT_CONFIGURED",
			Message: "Navigator registry is not configured",
		})
		return
	}

	// Get client source from header
	clientSource := vision.ClientSourceFromHeader(r.Header.Get("X-Client-Source"))

	// List all navigators with availability info for this client
	navigators := h.registry.ListNavigators(r.Context(), clientSource)

	response := vision.NavigatorsResponse{
		Navigators: navigators,
		Default:    h.registry.GetDefault(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.WithError(err).Warn("vision_navigation: failed to encode navigators response")
	}
}

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

	// Get client source from header
	clientSource := vision.ClientSourceFromHeader(r.Header.Get("X-Client-Source"))

	// Select navigator
	if h.registry == nil {
		RespondError(w, &APIError{
			Status:  http.StatusServiceUnavailable,
			Code:    "REGISTRY_NOT_CONFIGURED",
			Message: "Navigator registry is not configured",
		})
		return
	}

	preferredType := vision.NavigatorType(strings.TrimSpace(req.NavigatorType))
	navigator, err := h.registry.SelectNavigator(ctx, clientSource, preferredType)
	if err != nil {
		switch err {
		case vision.ErrNavigatorNotFound:
			RespondError(w, &APIError{
				Status:  http.StatusBadRequest,
				Code:    "NAVIGATOR_NOT_FOUND",
				Message: fmt.Sprintf("Navigator type '%s' is not registered", preferredType),
			})
		case vision.ErrNavigatorNotAvailable:
			RespondError(w, &APIError{
				Status:  http.StatusServiceUnavailable,
				Code:    "NAVIGATOR_UNAVAILABLE",
				Message: fmt.Sprintf("Navigator '%s' is not currently available", preferredType),
			})
		case vision.ErrNavigatorNotAllowed:
			RespondError(w, &APIError{
				Status:  http.StatusForbidden,
				Code:    "NAVIGATOR_NOT_ALLOWED",
				Message: fmt.Sprintf("Navigator '%s' is not allowed for client source '%s'", preferredType, clientSource),
			})
		case vision.ErrNoNavigatorsAvailable:
			RespondError(w, &APIError{
				Status:  http.StatusServiceUnavailable,
				Code:    "NO_NAVIGATORS_AVAILABLE",
				Message: "No navigators are currently available",
			})
		default:
			h.log.WithError(err).Error("vision_navigation: failed to select navigator")
			RespondError(w, ErrInternalServer)
		}
		return
	}

	// Check for BYOK key
	hasBYOK := strings.TrimSpace(req.APIKey) != ""

	// Check credit policy
	policy := navigator.CreditPolicy()
	if policy.ShouldChargeCredits(hasBYOK, false, false) && h.creditService != nil {
		userID := entitlement.UserIdentityFromContext(ctx)
		if userID == "" {
			userID = "anonymous"
		}

		canProceed, errCode, errMsg, remaining, err := h.creditService.CanPerformAIOperation(ctx, userID, policy.OperationType, hasBYOK)
		if err != nil {
			h.log.WithError(err).Warn("vision_navigation: failed to check AI operation permission")
		} else if !canProceed {
			status := http.StatusForbidden
			if errCode == "INSUFFICIENT_CREDITS" {
				status = http.StatusPaymentRequired
			}
			RespondError(w, &APIError{
				Status:  status,
				Code:    errCode,
				Message: errMsg,
				Details: map[string]string{"remaining": fmt.Sprintf("%d", remaining)},
			})
			return
		}
	}

	// Get user ID for tracking
	userID := entitlement.UserIdentityFromContext(ctx)
	if userID == "" {
		userID = "anonymous"
	}

	// Build callback URL
	callbackURL := h.resolveCallbackURL(r)

	// Create navigation request
	navReq := vision.NavigationRequest{
		SessionID:     req.SessionID,
		Prompt:        req.Prompt,
		Model:         req.Model,
		MaxSteps:      maxSteps,
		APIKey:        req.APIKey,
		NavigatorType: navigator.Type(),
		UserID:        userID,
		CallbackURL:   callbackURL,
	}

	// Start navigation
	handle, err := navigator.Navigate(ctx, navReq)
	if err != nil {
		h.log.WithError(err).Error("vision_navigation: failed to start navigation")
		RespondError(w, &APIError{
			Status:  http.StatusBadGateway,
			Code:    "NAVIGATION_FAILED",
			Message: fmt.Sprintf("Failed to start navigation: %v", err),
		})
		return
	}

	h.log.WithFields(logrus.Fields{
		"navigation_id": handle.ID(),
		"session_id":    req.SessionID,
		"model":         req.Model,
		"max_steps":     maxSteps,
		"navigator":     string(navigator.Type()),
	}).Info("vision_navigation: started")

	// Return response
	response := AINavigateResponse{
		NavigationID:  handle.ID(),
		Status:        "started",
		Model:         req.Model,
		MaxSteps:      maxSteps,
		NavigatorType: string(navigator.Type()),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.WithError(err).Warn("vision_navigation: failed to encode response")
	}
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
	var stepEvent vision.NavigationStep
	if err := json.Unmarshal(body, &stepEvent); err == nil && stepEvent.StepNumber > 0 {
		h.handleStepCallback(ctx, w, &stepEvent)
		return
	}

	// Try to parse as complete event
	var completeEvent vision.NavigationResult
	if err := json.Unmarshal(body, &completeEvent); err == nil && completeEvent.Status != "" {
		h.handleCompleteCallback(ctx, w, &completeEvent)
		return
	}

	h.log.WithField("body", string(body)).Warn("vision_navigation_callback: unknown event type")
	RespondError(w, ErrInvalidRequest)
}

// handleStepCallback processes a navigation step event.
func (h *VisionNavigationHandler) handleStepCallback(ctx context.Context, w http.ResponseWriter, event *vision.NavigationStep) {
	if h.playwrightNav != nil {
		if err := h.playwrightNav.HandleStepCallback(ctx, event); err != nil {
			h.log.WithError(err).Warn("vision_navigation_callback: failed to handle step")
		}
	}

	// Respond with acknowledgment
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"received": true,
		"abort":    false,
	}); err != nil {
		h.log.WithError(err).Warn("vision_navigation_callback: failed to encode response")
	}
}

// handleCompleteCallback processes a navigation complete event.
func (h *VisionNavigationHandler) handleCompleteCallback(ctx context.Context, w http.ResponseWriter, result *vision.NavigationResult) {
	if h.playwrightNav != nil {
		if err := h.playwrightNav.HandleCompleteCallback(ctx, result); err != nil {
			h.log.WithError(err).Warn("vision_navigation_callback: failed to handle complete")
		}
	}

	// Respond with acknowledgment
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"received": true,
	}); err != nil {
		h.log.WithError(err).Warn("vision_navigation_callback: failed to encode completion response")
	}
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

	// Currently only playwright navigator tracks sessions
	if h.playwrightNav == nil {
		RespondError(w, &APIError{
			Status:  http.StatusNotFound,
			Code:    "NAVIGATION_NOT_FOUND",
			Message: "Navigation session not found",
		})
		return
	}

	session, exists := h.playwrightNav.GetSession(navigationID)
	if !exists {
		RespondError(w, &APIError{
			Status:  http.StatusNotFound,
			Code:    "NAVIGATION_NOT_FOUND",
			Message: "Navigation session not found",
		})
		return
	}

	response := NavigationStatusResponse{
		NavigationID:  session.NavigationID,
		SessionID:     session.SessionID,
		Status:        string(session.Status),
		StepCount:     session.StepCount,
		TotalTokens:   session.TotalTokens,
		StartedAt:     session.StartedAt.Format(time.RFC3339),
		NavigatorType: string(session.NavigatorType),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.WithError(err).Warn("vision_navigation_status: failed to encode response")
	}
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

	// Currently only playwright navigator tracks sessions
	if h.playwrightNav == nil {
		RespondError(w, &APIError{
			Status:  http.StatusNotFound,
			Code:    "NAVIGATION_NOT_FOUND",
			Message: "Navigation session not found",
		})
		return
	}

	if err := h.playwrightNav.AbortNavigation(ctx, navigationID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			RespondError(w, &APIError{
				Status:  http.StatusNotFound,
				Code:    "NAVIGATION_NOT_FOUND",
				Message: "Navigation session not found",
			})
			return
		}
		h.log.WithError(err).Error("vision_navigation: failed to abort navigation")
		RespondError(w, ErrInternalServer)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "aborting",
		"navigation_id": navigationID,
		"message":       "Abort signal sent. Navigation will stop after current step.",
	}); err != nil {
		h.log.WithError(err).Warn("vision_navigation: failed to encode abort response")
	}
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

	// Currently only playwright navigator tracks sessions
	if h.playwrightNav == nil {
		RespondError(w, &APIError{
			Status:  http.StatusNotFound,
			Code:    "NAVIGATION_NOT_FOUND",
			Message: "Navigation session not found",
		})
		return
	}

	if err := h.playwrightNav.ResumeNavigation(ctx, navigationID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			RespondError(w, &APIError{
				Status:  http.StatusNotFound,
				Code:    "NAVIGATION_NOT_FOUND",
				Message: "Navigation session not found",
			})
			return
		}
		if strings.Contains(err.Error(), "not awaiting human") {
			RespondError(w, &APIError{
				Status:  http.StatusConflict,
				Code:    "NOT_AWAITING_HUMAN",
				Message: "Navigation is not awaiting human intervention",
			})
			return
		}
		h.log.WithError(err).Error("vision_navigation: failed to resume navigation")
		RespondError(w, ErrInternalServer)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "resumed",
		"navigation_id": navigationID,
		"message":       "Navigation resumed. Will continue from where it paused.",
	}); err != nil {
		h.log.WithError(err).Warn("vision_navigation: failed to encode resume response")
	}
}

// ============================
// Helper Methods
// ============================

// resolveCallbackURL builds the callback URL for step events.
func (h *VisionNavigationHandler) resolveCallbackURL(r *http.Request) string {
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
