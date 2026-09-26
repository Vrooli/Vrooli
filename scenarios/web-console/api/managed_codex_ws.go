package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"web-console/backends/codex"
	"web-console/internal/wireproto"
)

// handleManagedCodexWS is the browser transport for an app-server-owned
// session. It intentionally speaks only the stable terminal envelope: the
// actual output is projected through the conversation event stream, while
// line-oriented input becomes a native turn and never a second PTY writer.
func (s *Server) handleManagedCodexWS(w http.ResponseWriter, r *http.Request, sessionID string, owner *codex.ManagedOwner) {
	upgrader := websocket.Upgrader{ReadBufferSize: 64 * 1024, WriteBufferSize: 64 * 1024, CheckOrigin: func(*http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	var writeMu sync.Mutex
	write := func(message TerminalMessage) error {
		return writeTerminalJSON(conn, &writeMu, message)
	}
	if err := write(TerminalMessage{Type: wireproto.MsgTypeSessionReady, ProtocolVersion: wireproto.ProtocolVersion, Gen: time.Now().UnixNano()}); err != nil {
		return
	}
	var pending strings.Builder
	var acceptedThrough int64
	for {
		if err := conn.SetReadDeadline(time.Now().Add(2 * time.Duration(wsPingPeriod.Load()))); err != nil {
			return
		}
		var message TerminalMessage
		if err := conn.ReadJSON(&message); err != nil {
			return
		}
		switch message.Type {
		case wireproto.MsgTypePing:
			if err := write(TerminalMessage{Type: wireproto.MsgTypePong}); err != nil {
				return
			}
		case wireproto.MsgTypeHello:
			if err := write(TerminalMessage{Type: wireproto.MsgTypeHistoryEnd}); err != nil {
				return
			}
		case wireproto.MsgTypeResize:
			if err := write(TerminalMessage{Type: wireproto.MsgTypeResizeInfo, Cols: message.Cols, Rows: message.Rows}); err != nil {
				return
			}
		case wireproto.MsgTypeControl:
			if strings.EqualFold(strings.TrimSpace(message.Data), "interrupt") {
				turnID := owner.ActiveTurnID()
				if turnID == "" {
					_ = write(TerminalMessage{Type: wireproto.MsgTypeError, Reason: "no_active_turn"})
					continue
				}
				if err := owner.Interrupt(context.Background(), turnID); err != nil {
					_ = write(TerminalMessage{Type: wireproto.MsgTypeError, Reason: err.Error()})
				}
			}
		case wireproto.MsgTypeStdin:
			pending.WriteString(message.Data)
			if !strings.Contains(pending.String(), "\n") {
				acceptedThrough = message.Offset + int64(len(message.Data))
				_ = write(TerminalMessage{Type: wireproto.MsgTypeStdinAck, Ok: true, AcceptedThrough: acceptedThrough})
				continue
			}
			text := strings.TrimSpace(pending.String())
			pending.Reset()
			if text == "" {
				_ = write(TerminalMessage{Type: wireproto.MsgTypeStdinAck, Ok: true, AcceptedThrough: message.Offset + int64(len(message.Data))})
				continue
			}
			if _, err := owner.StartTurn(context.Background(), text); err != nil {
				log.Printf("managed Codex turn[%s]: %v", sessionID, err)
				_ = write(TerminalMessage{Type: wireproto.MsgTypeStdinAck, Reason: err.Error(), AcceptedThrough: acceptedThrough})
				continue
			}
			acceptedThrough = message.Offset + int64(len(message.Data))
			_ = write(TerminalMessage{Type: wireproto.MsgTypeStdinAck, Ok: true, AcceptedThrough: acceptedThrough})
		}
	}
}
