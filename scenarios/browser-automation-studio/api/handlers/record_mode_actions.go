package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/telemetry"
	"github.com/vrooli/browser-automation-studio/domain"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
)

// ReceiveRecordingAction is the recording-action ingress. It owns decoding,
// page attribution, timeline persistence, and the typed WebSocket projection.
func (h *Handler) ReceiveRecordingAction(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "sessionId"}))
		return
	}

	correlationID := h.generateCorrelationID(sessionID)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "Failed to read request body: " + err.Error()}))
		return
	}

	var action driver.RecordedAction
	if err := json.Unmarshal(body, &action); err != nil || action.ActionType == "" {
		var entry bastimeline.TimelineEntry
		if err := protojson.Unmarshal(body, &entry); err != nil {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "Invalid JSON body: " + err.Error()}))
			return
		}
		action = driver.RecordedActionFromTimelineEntry(&entry)
	}

	if err := h.commitRecordingAction(r.Context(), sessionID, &action); err != nil {
		h.respondError(w, err)
		return
	}

	broadcast := h.wsHub.BroadcastTimelineEntry(sessionID, h.createTimelineEntry(&action))
	h.log.WithFields(map[string]interface{}{
		"correlation_id": correlationID, "session_id": sessionID, "action_type": action.ActionType,
		"action_id": action.ID, "sequence_num": action.SequenceNum, "persisted": true,
		"broadcast_sent": broadcast.SentCount > 0, "subscriber_count": broadcast.SubscriberCount,
		"sent_count": broadcast.SentCount, "dropped_count": broadcast.DroppedCount,
	}).Debug("Action recorded")
	h.respondSuccess(w, http.StatusOK, map[string]string{"status": "ok", "entry_id": action.ID})
}

func (h *Handler) commitRecordingAction(ctx context.Context, sessionID string, action *driver.RecordedAction) *APIError {
	session, ok := h.recordModeService.GetSession(sessionID)
	if !ok || session == nil {
		return ErrExecutionNotFound.WithMessage("Session not found")
	}
	pages := session.Pages()
	if pages == nil {
		return ErrServiceUnavailable.WithMessage("Recording page tracking unavailable")
	}
	var pageID uuid.UUID
	if action.DriverPageID != "" {
		resolved := pages.GetPageIDByDriverID(action.DriverPageID)
		if resolved == nil {
			return ErrInvalidRequest.WithMessage("Recording page is not registered")
		}
		pageID = *resolved
	} else if action.PageID != "" {
		parsed, err := uuid.Parse(action.PageID)
		if err != nil {
			return ErrInvalidRequest.WithMessage("Invalid recording page identity")
		}
		pageID = parsed
	} else {
		pageID = pages.GetActivePageID()
	}
	if _, ok := pages.GetPage(pageID); !ok {
		return ErrInvalidRequest.WithMessage("Recording page is not registered")
	}
	action.PageID = pageID.String()
	if err := h.recordModeService.AddTimelineAction(ctx, sessionID, action, pageID); err != nil {
		return ErrServiceUnavailable.WithMessage("Recording action was not committed").WithDetails(map[string]string{"error": err.Error()})
	}
	return nil
}

func (h *Handler) createTimelineEntry(action *driver.RecordedAction) *bastimeline.TimelineEntry {
	return telemetry.TelemetryToTimelineEntry(telemetry.RecordedActionToTelemetry(action))
}

// recordCompletedNavigation commits the common outcome of reload/back/forward.
// Page metadata reflects the browser effect even if its journal commit fails.
func (h *Handler) recordCompletedNavigation(ctx context.Context, sessionID, actionType, url, title string) *APIError {
	sess, ok := h.recordModeService.GetSession(sessionID)
	if !ok || sess == nil || sess.Pages() == nil {
		return ErrServiceUnavailable.WithMessage("Browser change occurred but recording session is unavailable")
	}
	pages := sess.Pages()
	pageID := pages.GetActivePageID()
	now := time.Now()
	pages.UpdatePageInfo(pageID, url, title)
	action := &driver.RecordedAction{
		ID: uuid.NewString(), SessionID: sessionID,
		Timestamp: now.Format(time.RFC3339Nano), ActionType: actionType,
		Confidence: 1, URL: url, PageID: pageID.String(), PageTitle: title,
	}
	if err := h.recordModeService.AddTimelineAction(ctx, sessionID, action, pageID); err != nil {
		return ErrServiceUnavailable.WithMessage("Browser change occurred but recording was not committed").WithDetails(map[string]string{"error": err.Error()})
	}
	broadcast := h.wsHub.BroadcastTimelineEntry(sessionID, h.createTimelineEntry(action))
	if actionType != "reload" {
		h.wsHub.BroadcastPageEvent(sessionID, &domain.PageEvent{
			ID: uuid.New(), Type: domain.PageEventNavigated,
			PageID: pageID, URL: url, Title: title, Timestamp: now,
		})
	}
	h.log.WithFields(map[string]interface{}{
		"correlation_id": h.generateCorrelationID(sessionID),
		"session_id":     sessionID, "action_type": actionType, "action_id": action.ID,
		"url": url, "persisted": true,
		"broadcast_sent": broadcast.SentCount > 0, "subscriber_count": broadcast.SubscriberCount,
	}).Debug("Navigation action recorded")
	return nil
}
