package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/telemetry"
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

	var pageID uuid.UUID
	persisted := false
	if session, ok := h.recordModeService.GetSession(sessionID); ok && session.Pages() != nil {
		pages := session.Pages()
		if action.DriverPageID != "" {
			if resolved := pages.GetPageIDByDriverID(action.DriverPageID); resolved != nil {
				action.PageID, pageID = resolved.String(), *resolved
			}
		}
		if action.PageID == "" {
			pageID, action.PageID = pages.GetActivePageID(), pages.GetActivePageID().String()
		}
		h.recordModeService.AddTimelineAction(sessionID, &action, pageID)
		persisted = true
	}

	broadcast := h.wsHub.BroadcastTimelineEntry(sessionID, h.createTimelineEntry(&action))
	h.log.WithFields(map[string]interface{}{
		"correlation_id": correlationID, "session_id": sessionID, "action_type": action.ActionType,
		"action_id": action.ID, "sequence_num": action.SequenceNum, "persisted": persisted,
		"broadcast_sent": broadcast.SentCount > 0, "subscriber_count": broadcast.SubscriberCount,
		"sent_count": broadcast.SentCount, "dropped_count": broadcast.DroppedCount,
	}).Debug("Action recorded")
	h.respondSuccess(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) createTimelineEntry(action *driver.RecordedAction) *bastimeline.TimelineEntry {
	return telemetry.TelemetryToTimelineEntry(telemetry.RecordedActionToTelemetry(action))
}
