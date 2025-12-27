package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
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

	// The driver will send a page_created event via the callback endpoint,
	// which will trigger WebSocket broadcast. We just return the driver page ID here.
	h.log.WithFields(map[string]interface{}{
		"session_id":     sessionID,
		"driver_page_id": result.DriverPageID,
		"url":            result.URL,
	}).Info("New page created by user request")

	h.respondSuccess(w, http.StatusCreated, map[string]string{
		"driverPageId": result.DriverPageID,
		"url":          result.URL,
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

	// Don't allow closing the initial page
	if pageID == pages.GetInitialPageID() {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Cannot close the initial page",
		}))
		return
	}

	// Close the page
	if err := pages.ClosePage(pageID); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Create page event for timeline
	pageEvent := &domain.PageEvent{
		ID:        uuid.New(),
		Type:      domain.PageEventClosed,
		PageID:    pageID,
		Timestamp: time.Now(),
	}

	// Store in timeline
	h.recordModeService.AddTimelinePageEvent(sessionID, pageEvent)

	// Broadcast page close via WebSocket
	h.wsHub.BroadcastPageEvent(sessionID, pageEvent)

	// Broadcast new active page if it changed
	newActivePageID := pages.GetActivePageID()
	if newActivePageID.String() != pageIDStr {
		h.wsHub.BroadcastPageSwitch(sessionID, newActivePageID.String())
	}

	h.log.WithFields(map[string]interface{}{
		"session_id": sessionID,
		"page_id":    pageIDStr,
	}).Info("Page closed by user")

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"closedPageId":  pageIDStr,
		"activePageId":  newActivePageID.String(),
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

	var pageEvent *domain.PageEvent

	switch event.EventType {
	case "created":
		pageID := uuid.New()
		var openerID *uuid.UUID
		if event.OpenerDriverPageID != "" {
			openerID = pages.GetPageIDByDriverID(event.OpenerDriverPageID)
		}

		page := &domain.Page{
			ID:           pageID,
			SessionID:    sessionID,
			URL:          event.URL,
			Title:        event.Title,
			OpenerID:     openerID,
			CreatedAt:    time.Now(),
			IsInitial:    false,
			Status:       domain.PageStatusActive,
			DriverPageID: event.DriverPageID,
		}
		pages.AddPage(page)
		pages.MapDriverPageID(event.DriverPageID, pageID)

		pageEvent = &domain.PageEvent{
			ID:        uuid.New(),
			Type:      domain.PageEventCreated,
			PageID:    pageID,
			URL:       event.URL,
			Title:     event.Title,
			OpenerID:  openerID,
			Timestamp: time.Now(),
		}

		h.log.WithFields(map[string]interface{}{
			"session_id":     sessionID,
			"page_id":        pageID.String(),
			"driver_page_id": event.DriverPageID,
			"url":            event.URL,
			"opener_page_id": openerID,
		}).Info("New page created in recording session")

	case "navigated":
		vrooliPageID := pages.GetPageIDByDriverID(event.DriverPageID)
		if vrooliPageID != nil {
			pages.UpdatePageInfo(*vrooliPageID, event.URL, event.Title)

			pageEvent = &domain.PageEvent{
				ID:        uuid.New(),
				Type:      domain.PageEventNavigated,
				PageID:    *vrooliPageID,
				URL:       event.URL,
				Title:     event.Title,
				Timestamp: time.Now(),
			}

			h.log.WithFields(map[string]interface{}{
				"session_id": sessionID,
				"page_id":    vrooliPageID.String(),
				"url":        event.URL,
			}).Debug("Page navigated")
		}

	case "closed":
		vrooliPageID := pages.GetPageIDByDriverID(event.DriverPageID)
		if vrooliPageID != nil {
			if err := pages.ClosePage(*vrooliPageID); err != nil {
				h.log.WithError(err).Warn("Failed to close page")
			}

			pageEvent = &domain.PageEvent{
				ID:        uuid.New(),
				Type:      domain.PageEventClosed,
				PageID:    *vrooliPageID,
				Timestamp: time.Now(),
			}

			h.log.WithFields(map[string]interface{}{
				"session_id": sessionID,
				"page_id":    vrooliPageID.String(),
			}).Info("Page closed in recording session")
		}

	case "initial":
		// Driver is reporting the initial page's driver ID
		pages.SetInitialPageDriverID(event.DriverPageID)
		if event.URL != "" || event.Title != "" {
			pages.UpdatePageInfo(pages.GetInitialPageID(), event.URL, event.Title)
		}

		h.log.WithFields(map[string]interface{}{
			"session_id":     sessionID,
			"driver_page_id": event.DriverPageID,
			"url":            event.URL,
		}).Debug("Initial page registered")
	}

	if pageEvent != nil {
		// Store in timeline
		h.recordModeService.AddTimelinePageEvent(sessionID, pageEvent)

		// Broadcast via WebSocket
		h.wsHub.BroadcastPageEvent(sessionID, pageEvent)
	}

	w.WriteHeader(http.StatusOK)
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

	// Get timeline from service
	timeline, err := h.recordModeService.GetTimeline(sessionID, pageID, limit)
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
