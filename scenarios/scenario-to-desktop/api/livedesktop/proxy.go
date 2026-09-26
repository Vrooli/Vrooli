package livedesktop

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"

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

	// Bidirectional proxy. Closing either connection unblocks both readers, and
	// the upgraded request context closes them if the client disconnects.
	done := make(chan struct{}, 2)
	var closeOnce sync.Once
	closeConnections := func() {
		closeOnce.Do(func() {
			_ = browserConn.Close()
			_ = vncConn.Close()
		})
	}
	stopContextClose := context.AfterFunc(r.Context(), closeConnections)
	defer stopContextClose()

	// Browser → VNC
	go func(ctx context.Context) {
		defer func() { done <- struct{}{} }()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
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
	}(r.Context())

	// VNC → Browser
	go func(ctx context.Context) {
		defer func() { done <- struct{}{} }()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
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
	}(r.Context())

	// Wait for either direction to finish
	<-done
	closeConnections()
	<-done
}
