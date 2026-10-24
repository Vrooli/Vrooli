package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/domain"
)

// GetNavigationState observes current browser capabilities under the selected page owner.
func (h *Handler) GetNavigationState(w http.ResponseWriter, r *http.Request) {
	h.getRecordingNavigation(w, r, false)
}

func (h *Handler) GetNavigationStack(w http.ResponseWriter, r *http.Request) {
	h.getRecordingNavigation(w, r, true)
}

func (h *Handler) getRecordingNavigation(w http.ResponseWriter, r *http.Request, stack bool) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "sessionId"}))
		return
	}
	owner, ok := h.recordModeService.GetSession(sessionID)
	if !ok || owner == nil {
		h.respondError(w, ErrExecutionNotFound.WithMessage("Session not found"))
		return
	}
	pageID := r.URL.Query().Get("page_id")
	expected, apiErr := recordingSelectedPage(owner, pageID)
	if apiErr != nil {
		h.respondError(w, apiErr)
		return
	}
	var result any
	var err error
	if stack {
		result, err = owner.GetNavigationStack(ctx, expected)
	} else {
		result, err = owner.GetNavigationState(ctx, expected)
	}
	if err != nil {
		h.respondError(w, recordingNavigationError(err))
		return
	}
	current, ok := h.recordModeService.GetSession(sessionID)
	if !ok || current != owner {
		h.respondError(w, ErrConflict.WithMessage("Recording session changed during history read"))
		return
	}
	selected, apiErr := recordingSelectedPage(owner, pageID)
	if apiErr != nil {
		h.respondError(w, apiErr)
		return
	}
	if selected != expected {
		h.respondError(w, ErrConflict.WithMessage("Recording tab changed during history read"))
		return
	}
	h.respondSuccess(w, http.StatusOK, result)
}

// recordingSelectedPage binds omitted page intent to the current selection.
func recordingSelectedPage(owner *session.Session, pageID string) (string, *APIError) {
	if pageID == "" {
		pages := owner.Pages()
		if pages == nil {
			return "", ErrServiceUnavailable.WithMessage("Recording page tracking is unavailable")
		}
		page := pages.GetActivePage()
		if page == nil {
			return "", ErrConflict.WithMessage("No recording tab is selected")
		}
		pageID = page.ID.String()
	}
	return navigationRequestPage(owner, pageID)
}

// navigationRequestPage translates an optional canonical page precondition.
// The driver checks the translated identity again before effects.
func navigationRequestPage(owner *session.Session, pageID string) (string, *APIError) {
	if pageID == "" {
		return "", nil
	}
	id, err := uuid.Parse(pageID)
	if err != nil {
		return "", ErrInvalidRequest.WithMessage("Invalid recording page ID")
	}
	pages := owner.Pages()
	if pages == nil {
		return "", ErrServiceUnavailable.WithMessage("Recording page tracking is unavailable")
	}
	page := pages.GetActivePage()
	if page == nil || page.ID != id || page.Status != domain.PageStatusActive {
		return "", ErrConflict.WithMessage("The selected recording tab changed before the request started")
	}
	if page.DriverPageID == "" {
		return "", ErrServiceUnavailable.WithMessage("Recording page identity is unavailable")
	}
	return page.DriverPageID, nil
}

func recordingNavigationError(err error) *APIError {
	var driverError *driver.Error
	if errors.As(err, &driverError) && driverError.Status == http.StatusConflict {
		return ErrConflict.WithMessage("The selected recording tab changed before the request started")
	}
	return ErrServiceUnavailable.WithDetails(map[string]string{"error": err.Error()})
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
	resp, err := owner.Navigate(ctx, req.URL, session.WithExpectedPage(expectedPage), session.WithWaitUntil(req.WaitUntil), session.WithNavigateTimeout(req.TimeoutMs), session.WithCapture(req.Capture))
	if err != nil {
		h.log.WithError(err).Error("Failed to navigate recording session")
		h.respondError(w, recordingNavigationError(err))
		return
	}
	pages, pageID, apiErr := h.navigationResultPage(owner, resp.DriverPageID)
	if apiErr != nil {
		h.respondError(w, apiErr)
		return
	}
	pages.UpdatePageInfo(pageID, resp.URL, resp.Title, resp.FaviconURL)
	h.wsHub.BroadcastPageEvent(sessionID, &domain.PageEvent{ID: uuid.New(), Type: domain.PageEventNavigated, PageID: pageID, URL: resp.URL, Title: resp.Title, FaviconURL: resp.FaviconURL, Timestamp: time.Now()})

	h.respondSuccess(w, http.StatusOK, NavigateRecordingResponse{URL: resp.URL, Title: resp.Title, CanGoBack: resp.CanGoBack, CanGoForward: resp.CanGoForward, StatusCode: resp.StatusCode, Screenshot: resp.Screenshot})
}
