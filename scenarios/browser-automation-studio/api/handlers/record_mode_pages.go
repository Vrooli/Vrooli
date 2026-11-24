package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/domain"
)

// CreateRecordingPageRequest is the request body for creating a new page.
type CreateRecordingPageRequest struct {
	URL string `json:"url"`
}

// CreateRecordingPage handles POST /api/v1/recordings/live/{sessionId}/pages
// Creates a new page (tab) in the recording session.
func (h *Handler) CreateRecordingPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	var req CreateRecordingPageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	// Default to about:blank if no URL provided
	pageURL := req.URL
	if pageURL == "" {
		pageURL = "about:blank"
	}

	// Call the service to create a new page
	result, err := h.recordModeService.CreatePage(ctx, sessionID, pageURL)
	if err != nil {
		h.log.WithError(err).Error("Failed to create new page")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// The canonical page receipt is usable before recording callbacks are attached.
	h.log.WithFields(map[string]interface{}{
		"session_id":     sessionID,
		"driver_page_id": result.DriverPageID,
		"url":            result.URL,
	}).Info("New page created by user request")

	h.respondSuccess(w, http.StatusCreated, map[string]any{
		"driverPageId": result.DriverPageID,
		"url":          result.URL,
		"page":         result,
		"activePageId": result.ID.String(),
	})
}

// GetRecordingPages handles GET /api/v1/recordings/live/{sessionId}/pages
// Returns all pages in the recording session.
func (h *Handler) GetRecordingPages(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	result, err := h.recordModeService.GetPages(sessionID)
	if err != nil {
		h.log.WithError(err).Error("Failed to get pages")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusOK, domain.PagesResponse{
		Pages:        result.Pages,
		ActivePageID: result.ActivePageID,
	})
}

// ActivateRecordingPage handles POST /api/v1/recordings/live/{sessionId}/pages/{pageId}/activate
// Switches the active page for frame streaming and input forwarding.
func (h *Handler) ActivateRecordingPage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	pageIDStr := chi.URLParam(r, "pageId")

	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}
	if pageIDStr == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "pageId",
		}))
		return
	}

	pageID, err := uuid.Parse(pageIDStr)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "invalid page ID format",
		}))
		return
	}

	if err := h.recordModeService.ActivatePage(r.Context(), sessionID, pageID); err != nil {
		h.log.WithError(err).Error("Failed to activate page")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Broadcast page switch to all clients
	h.wsHub.BroadcastPageSwitch(sessionID, pageIDStr)

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"activePageId": pageIDStr,
	})
}

// CloseRecordingPage handles POST /api/v1/recordings/live/{sessionId}/pages/{pageId}/close
// Closes a page in the recording session (user-initiated close).
func (h *Handler) CloseRecordingPage(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	pageIDStr := chi.URLParam(r, "pageId")

	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}
	if pageIDStr == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "pageId",
		}))
		return
	}

	pageID, err := uuid.Parse(pageIDStr)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "invalid page ID format",
		}))
		return
	}

	result, err := h.recordModeService.ClosePage(r.Context(), sessionID, pageID)
	if err != nil {
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{"error": err.Error()}))
		return
	}
	pageEvent := result.Event

	// Broadcast page close via WebSocket
	h.wsHub.BroadcastPageEvent(sessionID, pageEvent)

	h.wsHub.BroadcastPageSwitch(sessionID, result.ActivePageID)

	h.log.WithFields(map[string]interface{}{
		"session_id": sessionID,
		"page_id":    pageIDStr,
	}).Info("Page closed by user")

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"closedPageId": pageIDStr,
		"activePageId": result.ActivePageID,
	})
}

// ReceivePageEvent handles POST /api/v1/recordings/live/{sessionId}/page-event
// Called by Playwright-driver when page lifecycle events occur.
func (h *Handler) ReceivePageEvent(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	var event domain.DriverPageEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}
	if strings.TrimSpace(event.DriverPageID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "driverPageId"}))
		return
	}
	if event.SessionID != "" && event.SessionID != sessionID {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "driver callback session does not match route"}))
		return
	}
	occurredAt := driverPageEventTime(event.Timestamp)

	sess, ok := h.recordModeService.GetSession(sessionID)
	if !ok {
		h.respondError(w, ErrExecutionNotFound.WithMessage("Session not found"))
		return
	}

	pages := sess.Pages()
	if pages == nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Page tracking not initialized for session",
		}))
		return
	}

	if sess.ID() != "" && sess.ID() != sessionID {
		h.respondError(w, ErrExecutionNotFound.WithMessage("Session owner does not match callback route"))
		return
	}
	var appendErr error
	pageEvent, err := pages.ApplyDriverEvent(sessionID, event, occurredAt, func(candidate *domain.PageEvent) error {
		appendErr = h.recordModeService.AddTimelinePageEvent(r.Context(), sessionID, candidate)
		return appendErr
	})
	if err != nil {
		if appendErr != nil {
			h.respondError(w, ErrServiceUnavailable.WithMessage("Recording journal append failed; page registry unchanged").WithDetails(map[string]string{"error": err.Error()}))
			return
		}
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": err.Error()}))
		return
	}
	// Initial callbacks seed the durable timeline and driver mapping; they are
	// not a second UI page creation because the initial page is already loaded
	// from the session's page snapshot.
	if pageEvent != nil && event.EventType != "initial" {
		// Publish only after both the durable append and tracker update succeed.
		h.wsHub.BroadcastPageEvent(sessionID, pageEvent)
	}

	w.WriteHeader(http.StatusOK)
}

func driverPageEventTime(value string) time.Time {
	if parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value)); err == nil {
		return parsed
	}
	return time.Now()
}

// GetRecordingTimeline handles GET /api/v1/recordings/live/{sessionId}/timeline
// Returns the unified timeline of actions and page events for the recording session.
func (h *Handler) GetRecordingTimeline(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	// Parse query parameters
	var pageID *uuid.UUID
	if pageIDStr := r.URL.Query().Get("pageId"); pageIDStr != "" {
		parsedID, err := uuid.Parse(pageIDStr)
		if err != nil {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
				"error": "invalid pageId format",
			}))
			return
		}
		pageID = &parsedID
	}

	// Parse limit with default
	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		var err error
		limit, err = parseInt(limitStr, 1, 1000)
		if err != nil {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
				"error": "invalid limit: " + err.Error(),
			}))
			return
		}
	}

	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			h.respondError(w, ErrInvalidRequest.WithMessage("offset must be a non-negative integer"))
			return
		}
		offset = parsed
	}
	// Get timeline from service
	timeline, err := h.recordModeService.GetTimeline(r.Context(), sessionID, pageID, limit, offset)
	if err != nil {
		h.log.WithError(err).Error("Failed to get timeline")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusOK, timeline)
}

// parseInt parses an integer from a string with bounds checking.
func parseInt(s string, min, max int) (int, error) {
	var val int
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil {
		return 0, fmt.Errorf("not a valid integer")
	}
	if val < min {
		return min, nil
	}
	if val > max {
		return max, nil
	}
	return val, nil
}
