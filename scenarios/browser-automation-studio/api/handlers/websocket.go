package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

// HandleWebSocket handles GET /ws
// Upgrades HTTP connection to WebSocket for real-time updates
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}

	h.log.WithFields(logrus.Fields{
		"remote_addr": r.RemoteAddr,
	}).Info("New WebSocket connection established")

	// The generic socket carries unscoped application events only. Recording
	// session subscriptions are admitted through the path-scoped socket below.
	h.wsHub.ServeWS(conn, nil)
}

// HandleRecordingWebSocket serves a path-scoped recording stream. The
// HttpOnly cookie is issued only by successful recording-session admission.
func (h *Handler) HandleRecordingWebSocket(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")
	if sessionID == "" || !h.isOriginAllowed(r) {
		http.Error(w, "recording stream unavailable", http.StatusForbidden)
		return
	}
	cookie, err := r.Cookie(wsHub.SessionStreamCookieName(sessionID))
	if err != nil || !h.wsHub.ValidateSessionStreamCookie(sessionID, cookie.Value) {
		http.Error(w, "recording stream unavailable", http.StatusForbidden)
		return
	}
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.WithError(err).Error("Failed to upgrade recording WebSocket")
		return
	}
	if !h.wsHub.ServeRecordingWS(conn, sessionID, cookie.Value) {
		_ = conn.Close()
	}
}
