package livedesktop

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// handleVNCProxy upgrades the browser connection and proxies binary frames
// to/from the websockify instance for the given session.
func (h *Handler) handleVNCProxy(w http.ResponseWriter, r *http.Request) {
	sessionID := extractSessionID(r)
	session, err := h.service.GetSession(sessionID)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	if session.State != StateRunning {
		http.Error(w, "session is not running", http.StatusConflict)
		return
	}

	// Upgrade browser connection
	browserConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}
	defer browserConn.Close()

	// Connect to websockify
	wsURL := fmt.Sprintf("ws://localhost:%d", session.WSPort)
	vncConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		slog.Error("failed to connect to websockify", "error", err, "url", wsURL)
		_ = browserConn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "VNC backend unavailable"))
		return
	}
	defer vncConn.Close()

	session.Touch()

	// Bidirectional proxy
	done := make(chan struct{}, 2)

	// Browser → VNC
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			msgType, data, err := browserConn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) &&
					err != io.ErrUnexpectedEOF {
					slog.Debug("browser read error", "error", err)
				}
				return
			}
			if err := vncConn.WriteMessage(msgType, data); err != nil {
				return
			}
			session.Touch()
		}
	}()

	// VNC → Browser
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			msgType, data, err := vncConn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) &&
					err != io.ErrUnexpectedEOF {
					slog.Debug("vnc read error", "error", err)
				}
				return
			}
			if err := browserConn.WriteMessage(msgType, data); err != nil {
				return
			}
		}
	}()

	// Wait for either direction to finish
	<-done

	// Set a deadline so the other goroutine doesn't hang
	_ = browserConn.SetReadDeadline(time.Now())
	_ = vncConn.SetReadDeadline(time.Now())
	<-done
}
