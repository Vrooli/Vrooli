package scheduler

import (
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

// WSNotifier implements ScheduleNotifier using WebSocket broadcasts.
// It sends schedule events to all connected WebSocket clients.
type WSNotifier struct {
	hub wsHub.HubInterface
	log *logrus.Logger
}

// NewWSNotifier creates a new WebSocket-based schedule notifier.
func NewWSNotifier(hub wsHub.HubInterface, log *logrus.Logger) ScheduleNotifier {
	return &WSNotifier{
		hub: hub,
		log: log,
	}
}

// BroadcastScheduleEvent sends a schedule event to all connected WebSocket clients.
func (n *WSNotifier) BroadcastScheduleEvent(event ScheduleEvent) {
	if n.hub == nil {
		n.log.Warn("Cannot broadcast schedule event: WebSocket hub not initialized")
		return
	}

	// Convert event to WebSocket message format
	data := map[string]any{
		"schedule_id":   event.ScheduleID.String(),
		"workflow_id":   event.WorkflowID.String(),
		"schedule_name": event.ScheduleName,
		"timestamp":     event.Timestamp,
	}

	// Add optional fields if present
	if event.ExecutionID != uuid.Nil {
		data["execution_id"] = event.ExecutionID.String()
	}
	if event.NextRunAt != "" {
		data["next_run_at"] = event.NextRunAt
	}
	if event.Error != "" {
		data["error"] = event.Error
	}

	msg := map[string]any{
		"type": event.Type,
		"data": data,
	}

	n.hub.BroadcastEnvelope(msg)

	n.log.WithFields(logrus.Fields{
		"event_type":    event.Type,
		"schedule_id":   event.ScheduleID,
		"schedule_name": event.ScheduleName,
	}).Debug("Broadcast schedule event")
}
