package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/authz"
	"scenario-to-cloud/reach"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// terminalUpgrader binds the WebSocket origin policy to the management
// boundary: an interactive session needs a browser Origin that the boundary
// admits (same origin, an allowed origin, or loopback on a loopback bind).
// An absent Origin is refused because the terminal is a browser surface and a
// missing header cannot be told apart from a stripped one.
func (s *Server) terminalUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			return origin != "" && s.authz.OriginAllowed(origin, r.Host)
		},
	}
}

// TerminalMessage represents a message sent over the terminal WebSocket.
type TerminalMessage struct {
	Type string `json:"type"` // "data", "resize", "ping", "error"
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

// handleTerminalWebSocket bridges a browser WebSocket to an interactive
// session on the deployment's target. The session is opened through reach
// on the transport the target binding selects (SSH PTY or Bridge session
// channel); cloud holds no connection plane of its own.
// WS /api/v1/deployments/{id}/terminal
func (s *Server) handleTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Re-validate the principal before any session exists. The boundary
	// already admitted the route; this is the effect-time recheck.
	if denied := s.authz.RequireEffect(r.Context(), id, authz.EffectInteractive); denied != nil {
		apierrors.Write(w, denied)
		return
	}

	dc, lerr := s.loadDeploymentContext(r.Context(), id)
	if lerr != nil {
		apierrors.Write(w, lerr)
		return
	}
	opener, ok := s.reachFor(dc).(reach.SessionOpener)
	if !ok {
		apierrors.Write(w, apierrors.New(apierrors.CodeReachProtocolUnsupported, "the bound transport has no interactive session capability"))
		return
	}

	// Upgrade to WebSocket
	conn, err := s.terminalUpgrader().Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Terminal: failed to upgrade WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// The session lives no longer than the credential that opened it.
	ctx, cancel := s.authz.SessionContext(r.Context())
	defer cancel()

	session, err := opener.OpenSession(ctx, s.targetFor(dc), reach.SessionSpec{Term: "xterm-256color", Cols: 80, Rows: 24})
	if err != nil {
		sendTerminalError(conn, "Failed to open a session on the target: "+reach.APIError(err).Message)
		return
	}
	defer session.Close()

	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			sendTerminalError(conn, "Session credential expired")
		}
		session.Close()
	}()

	var wg sync.WaitGroup

	// Read from the session and send to WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := session.Read(buf)
			if n > 0 {
				if werr := conn.WriteJSON(TerminalMessage{Type: "data", Data: string(buf[:n])}); werr != nil {
					log.Printf("Terminal: WebSocket write error: %v", werr)
					cancel()
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("Terminal: session read error: %v", err)
				}
				cancel()
				return
			}
		}
	}()

	// Read from WebSocket and write to the session
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var msg TerminalMessage
				if err := conn.ReadJSON(&msg); err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						log.Printf("Terminal: WebSocket read error: %v", err)
					}
					cancel()
					return
				}

				switch msg.Type {
				case "data":
					if _, err := session.Write([]byte(msg.Data)); err != nil {
						log.Printf("Terminal: session write error: %v", err)
						cancel()
						return
					}
				case "resize":
					if msg.Cols > 0 && msg.Rows > 0 {
						if err := session.Resize(msg.Rows, msg.Cols); err != nil {
							log.Printf("Terminal: resize error: %v", err)
						}
					}
				case "ping":
					if err := conn.WriteJSON(TerminalMessage{Type: "pong"}); err != nil {
						log.Printf("Terminal: pong write error: %v", err)
						cancel()
						return
					}
				}
			}
		}
	}()

	// Wait for the session to end
	if err := session.Wait(); err != nil {
		log.Printf("Terminal: session wait error: %v", err)
	}
	cancel()
	wg.Wait()
}

// sendTerminalError sends an error message over the terminal WebSocket.
func sendTerminalError(conn *websocket.Conn, message string) {
	msg := TerminalMessage{
		Type: "error",
		Data: message,
	}
	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("Terminal: error write failed: %v", err)
	}
}
