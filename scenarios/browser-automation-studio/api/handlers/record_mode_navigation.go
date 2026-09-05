package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/domain"
)

// GetNavigationState returns the current driver navigation capabilities. This
// read-side route is kept separate from navigation commands and recording
// session lifecycle.
func (h *Handler) GetNavigationState(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "sessionId"}))
		return
	}
	state, err := h.recordModeService.DriverClient().GetNavigationState(ctx, sessionID)
	if err != nil {
		h.log.WithError(err).Error("Failed to get navigation state")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{"error": err.Error()}))
		return
	}
	h.respondSuccess(w, http.StatusOK, NavigationStateResponse{SessionID: sessionID, URL: state.URL, Title: state.Title, CanGoBack: state.CanGoBack, CanGoForward: state.CanGoForward})
}

func (h *Handler) GetNavigationStack(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "sessionId"}))
		return
	}
	stack, err := h.recordModeService.DriverClient().GetNavigationStack(ctx, sessionID)
	if err != nil {
		h.log.WithError(err).Error("Failed to get navigation stack")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{"error": err.Error()}))
		return
	}
	h.respondSuccess(w, http.StatusOK, stack)
}

func (h *Handler) NavigateRecordingSession(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "sessionId"}))
		return
	}
	var req NavigateRecordingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "Invalid JSON body: " + err.Error()}))
		return
	}
	if strings.TrimSpace(req.URL) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "url"}))
		return
	}
	resp, err := h.recordModeService.DriverClient().Navigate(ctx, sessionID, &driver.NavigateRequest{URL: req.URL, WaitUntil: req.WaitUntil, TimeoutMs: req.TimeoutMs, Capture: req.Capture})
	if err != nil {
		h.log.WithError(err).Error("Failed to navigate recording session")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{"error": err.Error()}))
		return
	}
	if session, ok := h.recordModeService.GetSession(sessionID); ok && session.Pages() != nil {
		pageID := session.Pages().GetActivePageID()
		session.Pages().UpdatePageInfo(pageID, resp.URL, resp.Title)
		h.wsHub.BroadcastPageEvent(sessionID, &domain.PageEvent{ID: uuid.New(), Type: domain.PageEventNavigated, PageID: pageID, URL: resp.URL, Title: resp.Title, Timestamp: time.Now()})
	}
	h.respondSuccess(w, http.StatusOK, NavigateRecordingResponse{URL: resp.URL, Title: resp.Title, CanGoBack: resp.CanGoBack, CanGoForward: resp.CanGoForward, StatusCode: resp.StatusCode, Screenshot: resp.Screenshot})
}
