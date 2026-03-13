package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// DOC: docs/concepts/ARCHITECTURE.md#terminal-io
// DOC: docs/internal/ERROR-SEMANTICS.md#websocket-error-protocol
// WebSocket message types for terminal I/O.
// [REQ:P0-002b] WebSocket I/O Streaming
const (
	MsgTypeStdin       = "stdin"
	MsgTypeStdout      = "stdout"
	MsgTypeResize      = "resize"
	MsgTypeResizeInfo  = "resize_info"
	MsgTypeExit        = "exit"
	MsgTypeError       = "error"
	MsgTypePing        = "ping"
	MsgTypePong        = "pong"
	MsgTypeSyncWarning = "sync_warning"
)

// TerminalMessage is the WebSocket JSON message format.
type TerminalMessage struct {
	Type          string `json:"type"`
	Data          string `json:"data,omitempty"`
	Cols          int    `json:"cols,omitempty"`
	Rows          int    `json:"rows,omitempty"`
	Code          int    `json:"code,omitempty"`
	DroppedFrames int    `json:"dropped_frames,omitempty"`
}

// handleTerminalWS upgrades to WebSocket and bridges bidirectional I/O between
// the browser and a PTY session. Two concurrent loops run for the connection:
//   - outputForwarder: reads PTY output and writes it to the WebSocket
//   - inputLoop (inline): reads WebSocket messages and dispatches stdin/resize/ping
//
// Either loop exiting (PTY death, WS close, or write error) tears down the other
// via the writerDone channel.
// [REQ:P0-002b] WebSocket I/O Streaming
func (s *Server) handleTerminalWS(w http.ResponseWriter, r *http.Request) {
	sess := s.lookupSession(w, r)
	if sess == nil {
		return
	}
	sessionID := sess.ID

	if sess.IsDead() {
		writeCatalogError(w, "session_terminated",
			"Session has terminated")
		return
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  s.sessions.cfg.WSBufferSize,
		WriteBufferSize: s.sessions.cfg.WSBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			return true // Origin validation handled by parent proxy
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws[%s]: upgrade failed: %v", sessionID, err)
		return
	}
	defer conn.Close()

	// [REQ:P1-004a] Emit connection event
	s.events.Emit(EventSessionConnected, sessionID, nil)
	s.metrics.ConnectionsTotal.Add(1)
	s.metrics.ActiveConnections.Add(1)
	defer func() {
		s.events.Emit(EventSessionDisconnected, sessionID, nil)
		s.metrics.ActiveConnections.Add(-1)
	}()

	// Subscribe to PTY output with default dimensions (first resize message updates them)
	outputCh, notifyCh := sess.Subscribe(s.sessions.cfg.DefaultCols, s.sessions.cfg.DefaultRows)
	defer sess.Unsubscribe(outputCh)

	// writeMu serializes WebSocket writes from the output forwarder goroutine
	// and the inline input loop (which also writes pong/error responses).
	var writeMu sync.Mutex
	// writerDone signals the input loop to stop when the output forwarder exits
	// (PTY process died or WebSocket write failed).
	writerDone := make(chan struct{})

	sendError := func(msg string) {
		writeMu.Lock()
		_ = conn.WriteJSON(TerminalMessage{Type: MsgTypeError, Data: msg})
		writeMu.Unlock()
	}

	// Output forwarder: PTY output + drop notifications → WebSocket client
	go func() {
		defer func() {
			// Close writerDone to signal the input loop — but only if it hasn't
			// already been closed (e.g. by the input loop returning first).
			select {
			case <-writerDone:
				// Already closed by input loop exit path
			default:
				close(writerDone)
			}
		}()
		for {
			select {
			case data, ok := <-outputCh:
				if !ok {
					// Channel closed = process exited; forward the real exit code
					writeMu.Lock()
					_ = conn.WriteJSON(TerminalMessage{Type: MsgTypeExit, Code: sess.ExitCode()})
					writeMu.Unlock()
					return
				}
				writeMu.Lock()
				err := conn.WriteJSON(TerminalMessage{
					Type: MsgTypeStdout,
					Data: string(data),
				})
				writeMu.Unlock()
				s.metrics.WSMessagesSent.Add(1)
				if err != nil {
					return
				}
			case dropped := <-notifyCh:
				writeMu.Lock()
				_ = conn.WriteJSON(TerminalMessage{
					Type:          MsgTypeSyncWarning,
					DroppedFrames: dropped,
				})
				writeMu.Unlock()
				s.metrics.WSMessagesSent.Add(1)
			}
		}
	}()

	// Input loop: WebSocket client → PTY stdin / resize / ping-pong
	for {
		select {
		case <-writerDone:
			return
		default:
		}

		_, rawMsg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var msg TerminalMessage
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			log.Printf("ws[%s]: invalid message JSON: %v", sessionID, err)
			sendError("Invalid message format")
			continue
		}

		s.metrics.WSMessagesReceived.Add(1)

		switch msg.Type {
		case MsgTypeStdin:
			if _, err := sess.Write([]byte(msg.Data)); err != nil {
				log.Printf("ws[%s]: PTY write failed: %v", sessionID, err)
				sendError("Terminal process is not accepting input")
				return
			}
		case MsgTypeResize:
			if msg.Cols > 0 && msg.Rows > 0 {
				effCols, effRows := sess.ResizeClient(outputCh, uint16(msg.Cols), uint16(msg.Rows))
				// Inform this client of the effective PTY size (may differ from requested)
				writeMu.Lock()
				_ = conn.WriteJSON(TerminalMessage{
					Type: MsgTypeResizeInfo,
					Cols: int(effCols),
					Rows: int(effRows),
				})
				writeMu.Unlock()
				// [REQ:P1-004a] Emit resize event
				s.events.Emit(EventPaneResized, sessionID, map[string]string{
					"cols": fmt.Sprintf("%d", msg.Cols),
					"rows": fmt.Sprintf("%d", msg.Rows),
				})
				s.metrics.ResizeCount.Add(1)
			}
		case MsgTypePing:
			writeMu.Lock()
			_ = conn.WriteJSON(TerminalMessage{Type: MsgTypePong})
			writeMu.Unlock()
		}
	}
}
