package main

// terminal_ws_input.go: Client → server input dispatch.
//
// This file owns the per-message handling inside the terminal WebSocket
// input loop: kind-aware stdin dispatch (persistent-mode-safe via
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
	"sync"
	"web-console/internal/events"
	"web-console/internal/pty"

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
	sess *Session,
	sessionID string,
	msg TerminalMessage,
	sessionReady bool,
) inputDispatchResult {
	switch msg.Type {
	case MsgTypeStdin:
		if !sessionReady {
			// Should never happen — the client is required to wait
			// for session_ready before sending stdin. Log and count
			// so a future regression is immediately visible on
			// /metrics.
			s.metrics.StdinBeforeReadyTotal.Add(1)
			log.Printf("ws[%s] stdin before session_ready — backend=%s", sessionID, sess.Backend)
		}
		kind := pty.KindKeystroke
		if msg.Kind == StdinKindPaste {
			kind = pty.KindPaste
		}
		writeErr := sess.WriteInput([]byte(msg.Data), kind)
		ackMsg := TerminalMessage{Type: MsgTypeStdinAck, Seq: msg.Seq, Ok: writeErr == nil}
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
		writeMu.Lock()
		_ = conn.WriteJSON(ackMsg)
		writeMu.Unlock()
		if writeErr != nil {
			log.Printf("ws[%s]: PTY write failed: %v", sessionID, writeErr)
			return inputDispatchResult{Close: true, CloseReason: "Terminal process is not accepting input"}
		}
	case MsgTypeResize:
		if msg.Cols > 0 && msg.Rows > 0 {
			sess.Resize(uint16(msg.Cols), uint16(msg.Rows))
			writeMu.Lock()
			_ = conn.WriteJSON(TerminalMessage{
				Type: MsgTypeResizeInfo,
				Cols: msg.Cols,
				Rows: msg.Rows,
			})
			writeMu.Unlock()
			// [REQ:P1-004a] Emit resize event
			s.events.Emit(events.PaneResized, sessionID, map[string]string{
				"cols": fmt.Sprintf("%d", msg.Cols),
				"rows": fmt.Sprintf("%d", msg.Rows),
			})
			s.metrics.ResizeCount.Add(1)
		}
	case MsgTypePing:
		writeMu.Lock()
		_ = conn.WriteJSON(TerminalMessage{Type: MsgTypePong})
		writeMu.Unlock()
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
