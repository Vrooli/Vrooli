package main

// terminal_ws_input.go: Client → server input dispatch.
//
// This file owns the per-message handling inside the terminal WebSocket
// input loop: intent-aware stdin dispatch (persistent-mode-safe via
// PTY.WriteInput), stdin_ack emission with typed reason codes, resize,
// ping/pong, and conversation_event_ack routing. The main loop glue
// itself stays in terminal_ws.go so the WebSocket lifecycle is easy to
// read in one place.
//
// (screaming architecture) and §8.2 (input kind/reason wire contract).

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"web-console/internal/events"
	"web-console/internal/pty"
	"web-console/session"

	"github.com/gorilla/websocket"
)

// inputDispatchResult tells the input loop whether to continue or to
// close the WebSocket. When CloseReason is non-empty the loop emits
// that reason as a WS-level error before returning.
type inputDispatchResult struct {
	Close       bool
	CloseReason string
}

// dispatchInputMessage processes a single TerminalMessage received on
// the WebSocket. Extracted from handleTerminalWS so the per-type
// switch is separately testable and the lifecycle glue stays small.
//
// Fatal backend failures (PTY write errors after a seemingly-ready
// session) are surfaced as Close=true + CloseReason — the caller then
// writes an error frame and returns.
func (s *Server) dispatchInputMessage(
	conn *websocket.Conn,
	writeMu *sync.Mutex,
	sess *session.Session,
	client chan []byte,
	sessionID string,
	msg TerminalMessage,
) inputDispatchResult {
	switch msg.Type {
	case MsgTypeStdin:
		if !sess.HoldsLease(client) {
			_ = sess.AcquireLease(client, session.LeaseReasonInput)
		}
		accepted := sess.AcceptedThrough()
		switch {
		case msg.Offset > accepted:
			_ = writeTerminalJSON(conn, writeMu, TerminalMessage{
				Type:            MsgTypeStdinAck,
				AcceptedThrough: accepted,
				Reason:          StdinAckReasonOffsetGap,
				Data:            "stdin offset is ahead of the session's accepted prefix",
			})
			return inputDispatchResult{}
		case msg.Offset < accepted:
			// The frame was already accepted before a reconnect or duplicate
			// delivery. Ack the current prefix without writing it twice.
			_ = writeTerminalJSON(conn, writeMu, TerminalMessage{
				Type:            MsgTypeStdinAck,
				Ok:              true,
				AcceptedThrough: accepted,
			})
			return inputDispatchResult{}
		}
		in := session.InputText(msg.Data).WithSource("ws")
		if msg.Intent == StdinIntentBulkText {
			in = in.AsPaste()
		}
		written, writeErr := sess.SendInputCount(in)
		if writeErr == nil {
			accepted = sess.AdvanceAcceptedThrough(int64(written))
		}
		ackMsg := TerminalMessage{
			Type:            MsgTypeStdinAck,
			Ok:              writeErr == nil,
			AcceptedThrough: accepted,
		}
		if writeErr != nil {
			ackMsg.Data = writeErr.Error()
			switch {
			case errors.Is(writeErr, errPTYClosed):
				ackMsg.Reason = StdinAckReasonPTYClosed
			default:
				// tmux send-keys / paste-buffer failure, or realPTY
				// write error (broken pipe, etc.). The full message
				// is in Data; Reason is the typed code the UI keys
				// on.
				ackMsg.Reason = StdinAckReasonTmuxWriteFailed
			}
		}
		_ = writeTerminalJSON(conn, writeMu, ackMsg)
		if writeErr != nil {
			log.Printf("ws[%s]: PTY write failed: %v", sessionID, writeErr)
			// Only a dead PTY is fatal to the connection. A backend
			// write that rejects one payload must not take the pane down;
			// the client already has the typed failure acknowledgement.
			if errors.Is(writeErr, errPTYClosed) {
				return inputDispatchResult{Close: true, CloseReason: "Terminal process is not accepting input"}
			}
		}
	case MsgTypeHello:
		accepted := sess.AcceptedThrough()
		if msg.HaveThrough > accepted {
			_ = writeTerminalJSON(conn, writeMu, TerminalMessage{
				Type:            MsgTypeStdinAck,
				AcceptedThrough: accepted,
				Reason:          StdinAckReasonUnreconcilable,
				Data:            "client reliable-input prefix is ahead of the session",
			})
		}
	case MsgTypeControl:
		// Synthetic terminal bytes are intentionally best-effort. They bypass
		// reliable-input queue and are written directly through the
		// PTY's control kind so a reconnect cannot replay stale gestures.
		if err := sess.SendInput(session.InputRaw([]byte(msg.Data)).WithSource("ws-control").WithKind(pty.KindControl)); err != nil {
			log.Printf("ws[%s]: best-effort control write failed: %v", sessionID, err)
		}
	case MsgTypeMouseMode:
		mode := strings.TrimSpace(strings.ToLower(msg.Data))
		if mode != "on" && mode != "off" {
			_ = writeTerminalJSON(conn, writeMu, TerminalMessage{Type: MsgTypeMouseMode, Data: "unsupported", Reason: "mouse mode must be on or off"})
			break
		}
		enabled := mode == "on"
		if err := sess.SetMouseMode(enabled); err != nil {
			_ = writeTerminalJSON(conn, writeMu, TerminalMessage{Type: MsgTypeMouseMode, Data: "unsupported", Reason: err.Error()})
			break
		}
		_ = writeTerminalJSON(conn, writeMu, TerminalMessage{Type: MsgTypeMouseMode, Data: mode, Ok: true})
	case MsgTypeResize:
		if msg.Cols > 0 && msg.Rows > 0 {
			sess.DeclareSize(client, uint16(msg.Cols), uint16(msg.Rows))
			if sess.HoldsLease(client) {
				_ = sess.Resize(client, uint16(msg.Cols), uint16(msg.Rows))
			}
			_ = writeTerminalJSON(conn, writeMu, TerminalMessage{
				Type: MsgTypeResizeInfo,
				Cols: msg.Cols,
				Rows: msg.Rows,
			})
			// [REQ:P1-004a] Emit resize event
			s.events.Emit(events.PaneResized, sessionID, map[string]string{
				"cols": fmt.Sprintf("%d", msg.Cols),
				"rows": fmt.Sprintf("%d", msg.Rows),
			})
			s.metrics.ResizeCount.Add(1)
		}
	case MsgTypeTakeLease:
		if err := sess.AcquireLease(client, session.LeaseReasonExplicit); err != nil {
			return inputDispatchResult{CloseReason: err.Error()}
		}
		// Do not make the requester wait for the independent output-forwarder
		// goroutine to drain its size notification. A mobile tap needs an
		// immediate, ordered acknowledgement that it now owns the lease; the
		// broadcast remains responsible for updating every other viewer.
		cols, rows, leader, leaderDevice, holdsLease, viewerCount := sess.SizeLeaseState(client)
		_ = writeTerminalJSON(conn, writeMu, TerminalMessage{
			Type: MsgTypeSizeInfo, Cols: int(cols), Rows: int(rows),
			Leader: leader, LeaderDevice: leaderDevice, HoldsLease: holdsLease,
			ViewerCount: viewerCount,
		})
	case MsgTypePing:
		_ = writeTerminalJSON(conn, writeMu, TerminalMessage{Type: MsgTypePong})
	case MsgTypeConversationAck:
		if msg.EventID == "" || msg.Source == "" || msg.Stage == "" {
			return inputDispatchResult{CloseReason: "Invalid TTS acknowledgment"}
		}
		s.recordTTSAck(TTSClientAck{
			EventID:   msg.EventID,
			Source:    msg.Source,
			SessionID: sessionID,
			Stage:     msg.Stage,
			Backend:   msg.Backend,
			Message:   msg.Data,
		})
	default:
		// Unknown message types are forward-compatible no-ops.
	}
	return inputDispatchResult{}
}

// decodeInputMessage parses the raw WebSocket frame and increments the
// messages-received counter. Returns false + a typed error string when
// decoding fails so the caller can emit a WS-level error.
func (s *Server) decodeInputMessage(sessionID string, raw []byte) (TerminalMessage, string, bool) {
	var msg TerminalMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		log.Printf("ws[%s]: invalid message JSON: %v", sessionID, err)
		return msg, "Invalid message format", false
	}
	s.metrics.WSMessagesReceived.Add(1)
	return msg, "", true
}
