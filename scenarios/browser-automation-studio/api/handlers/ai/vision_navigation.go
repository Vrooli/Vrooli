package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/vrooli/browser-automation-studio/services/vision"
)

// VisionNavigationHandler retains only the playwright-driver callback
// webhook. The user-facing vision-navigation surface (list/start/status/
// abort/resume) is served via Connect-RPC by VisionNavigationService; see
// scenarios/browser-automation-studio/api/handlers/vision_navigation/.
//
// The callback below is intentionally REST: it is the receiver of a
// fire-and-forget webhook from the out-of-process playwright driver and is
// not part of the proto-owned RPC surface. Tracked in
// docs/internal/REST_EXCEPTIONS.md as RESTReason=webhook_receiver.
type VisionNavigationHandler struct {
	log           *logrus.Logger
	playwrightNav *vision.PlaywrightVisionNavigator
}

// VisionNavigationHandlerOption configures VisionNavigationHandler.
type VisionNavigationHandlerOption func(*VisionNavigationHandler)

// WithPlaywrightNavigator sets a direct reference to the playwright
// navigator for callback dispatch.
func WithPlaywrightNavigator(nav *vision.PlaywrightVisionNavigator) VisionNavigationHandlerOption {
	return func(h *VisionNavigationHandler) {
		h.playwrightNav = nav
	}
}

// NewVisionNavigationHandler creates a callback-only handler.
func NewVisionNavigationHandler(log *logrus.Logger, opts ...VisionNavigationHandlerOption) *VisionNavigationHandler {
	h := &VisionNavigationHandler{log: log}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// HandleAINavigateCallback handles POST /api/v1/internal/ai-navigate/callback.
// Receives step and completion events from playwright-driver.
//
// RESTException: webhook_receiver. The driver protocol is not RPC-shaped;
// migrating it to Connect would require driving changes through the
// out-of-process playwright fork.
func (h *VisionNavigationHandler) HandleAINavigateCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.WithError(err).Warn("vision_navigation_callback: failed to read body")
		RespondError(w, ErrInvalidRequest)
		return
	}

	// Try to parse as step event first.
	var stepEvent vision.NavigationStep
	if err := json.Unmarshal(body, &stepEvent); err == nil && stepEvent.StepNumber > 0 {
		h.handleStepCallback(ctx, w, &stepEvent)
		return
	}

	// Then as a completion event.
	var completeEvent vision.NavigationResult
	if err := json.Unmarshal(body, &completeEvent); err == nil && completeEvent.Status != "" {
		h.handleCompleteCallback(ctx, w, &completeEvent)
		return
	}

	h.log.WithField("body", string(body)).Warn("vision_navigation_callback: unknown event type")
	RespondError(w, ErrInvalidRequest)
}

func (h *VisionNavigationHandler) handleStepCallback(ctx context.Context, w http.ResponseWriter, event *vision.NavigationStep) {
	if h.playwrightNav != nil {
		if err := h.playwrightNav.HandleStepCallback(ctx, event); err != nil {
			h.log.WithError(err).Warn("vision_navigation_callback: failed to handle step")
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"received": true,
		"abort":    false,
	}); err != nil {
		h.log.WithError(err).Warn("vision_navigation_callback: failed to encode response")
	}
}

func (h *VisionNavigationHandler) handleCompleteCallback(ctx context.Context, w http.ResponseWriter, result *vision.NavigationResult) {
	if h.playwrightNav != nil {
		if err := h.playwrightNav.HandleCompleteCallback(ctx, result); err != nil {
			h.log.WithError(err).Warn("vision_navigation_callback: failed to handle complete")
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"received": true,
	}); err != nil {
		h.log.WithError(err).Warn("vision_navigation_callback: failed to encode completion response")
	}
}
