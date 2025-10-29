package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// HandleWebSocket handles GET /ws
// Upgrades HTTP connection to WebSocket for real-time updates
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}

	// Check if client wants to subscribe to a specific execution
	var executionID *uuid.UUID
	if execIDStr := r.URL.Query().Get("execution_id"); execIDStr != "" {
		if id, err := uuid.Parse(execIDStr); err == nil {
			executionID = &id
		}
	}

	h.log.WithFields(logrus.Fields{
		"remote_addr":  r.RemoteAddr,
		"execution_id": executionID,
	}).Info("New WebSocket connection established")

	h.wsHub.ServeWS(conn, executionID)
}
