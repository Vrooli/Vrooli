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
	"web-console/internal/wireproto"
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
	resumeCh chan<- terminalResumeRequest,
) inputDispatchResult {
	switch msg.Type {
	case wireproto.MsgTypeStdin:
		if !sess.HoldsLease(client) {
			_ = sess.AcquireLease(client, session.LeaseReasonInput)
		}
		accepted := sess.ClientAcceptedThrough(client)
		head := sess.InputHeadFor(client)
		switch {
		case msg.Offset > head:
			_ = writeTerminalJSON(conn, writeMu, TerminalMessage{
				Type:            wireproto.MsgTypeStdinAck,
				AcceptedThrough: head,
				Reason:          wireproto.StdinAckReasonOffsetGap,
				Data:            "stdin offset is ahead of the session's accepted prefix",
			})
			return inputDispatchResult{}
		case msg.Offset < head:
			// Only an exact replay of bytes already delivered by this
			// connection is idempotent. A different payload at an old offset
			// is an offset gap and must not receive a success ack.
			if msg.Offset >= accepted || !sess.HasAcceptedInput(client, msg.Offset, []byte(msg.Data)) {
				_ = writeTerminalJSON(conn, writeMu, TerminalMessage{
					Type:            wireproto.MsgTypeStdinAck,
					AcceptedThrough: accepted,
					Reason:          wireproto.StdinAckReasonOffsetGap,
					Data:            "stdin payload does not match the connection's accepted prefix",
				})
				return inputDispatchResult{}
			}
			// The frame was already accepted by this connection. Ack the
			// current prefix without writing it twice.
			_ = writeTerminalJSON(conn, writeMu, TerminalMessage{
				Type:            wireproto.MsgTypeStdinAck,
				Ok:              true,
				AcceptedThrough: accepted,
			})
			return inputDispatchResult{}
		}
		data := []byte(msg.Data)
		if !sess.ReserveInputFor(client, int64(len(data))) {
			return inputDispatchResult{}
		}
		kind := pty.KindKeystroke
		if msg.Intent == wireproto.StdinIntentBulkText {
			kind = pty.KindPaste
		}
		result, enqueueErr := sess.EnqueueInput(data, kind)
		if enqueueErr != nil {
			sess.CompleteInputFor(client, data, enqueueErr)
			s.writeStdinAck(conn, writeMu, sess, client, sessionID, enqueueErr)
			return inputDispatchResult{}
		}
		go func() {
			writeErr := <-result
			sess.CompleteInputFor(client, data, writeErr)
			s.writeStdinAck(conn, writeMu, sess, client, sessionID, writeErr)
		}()
	case wireproto.MsgTypeHello:
		if msg.WantResume {
			select {
			case resumeCh <- terminalResumeRequest{want: true, renderedThrough: msg.RenderedThrough}:
			default:
			}
		}
		accepted := sess.ClientAcceptedThrough(client)
		if msg.HaveThrough > accepted {
			_ = writeTerminalJSON(conn, writeMu, TerminalMessage{
				Type:            wireproto.MsgTypeStdinAck,
				AcceptedThrough: accepted,
				Reason:          wireproto.StdinAckReasonUnreconcilable,
				Data:            "client reliable-input prefix is ahead of the session",
			})
		}
	case wireproto.MsgTypeControl:
		// Synthetic terminal bytes are intentionally best-effort. They use the
		// same ordered writer as reliable stdin, but are not assigned an offset,
		// acknowledged, or replayed after reconnect.
		if _, err := sess.EnqueueInput([]byte(msg.Data), pty.KindControl); err != nil {
			log.Printf("ws[%s]: best-effort control write failed: %v", sessionID, err)
		}
	case wireproto.MsgTypeMouseMode:
		mode := strings.TrimSpace(strings.ToLower(msg.Data))
		if mode != "on" && mode != "off" {
			_ = writeTerminalJSON(conn, writeMu, TerminalMessage{Type: wireproto.MsgTypeMouseMode, Data: "unsupported", Reason: "mouse mode must be on or off"})
			break
		}
		enabled := mode == "on"
		if err := sess.SetMouseMode(enabled); err != nil {
			_ = writeTerminalJSON(conn, writeMu, TerminalMessage{Type: wireproto.MsgTypeMouseMode, Data: "unsupported", Reason: err.Error()})
			break
		}
		_ = writeTerminalJSON(conn, writeMu, TerminalMessage{Type: wireproto.MsgTypeMouseMode, Data: mode, Ok: true})
	case wireproto.MsgTypeResize:
		if msg.Cols > 0 && msg.Rows > 0 {
			sess.DeclareSize(client, uint16(msg.Cols), uint16(msg.Rows))
			if sess.HoldsLease(client) {
				_ = sess.Resize(client, uint16(msg.Cols), uint16(msg.Rows))
			}
			_ = writeTerminalJSON(conn, writeMu, TerminalMessage{
				Type: wireproto.MsgTypeResizeInfo,
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
	case wireproto.MsgTypeTakeLease:
		if err := sess.AcquireLease(client, session.LeaseReasonExplicit); err != nil {
			return inputDispatchResult{CloseReason: err.Error()}
		}
		// Do not make the requester wait for the independent output-forwarder
		// goroutine to drain its size notification. A mobile tap needs an
		// immediate, ordered acknowledgement that it now owns the lease; the
		// broadcast remains responsible for updating every other viewer.
		cols, rows, leader, leaderDevice, holdsLease, viewerCount := sess.SizeLeaseState(client)
		_ = writeTerminalJSON(conn, writeMu, TerminalMessage{
			Type: wireproto.MsgTypeSizeInfo, Cols: int(cols), Rows: int(rows),
			Leader: leader, LeaderDevice: leaderDevice, HoldsLease: holdsLease,
			ViewerCount: viewerCount,
		})
	case wireproto.MsgTypePing:
		_ = writeTerminalJSON(conn, writeMu, TerminalMessage{Type: wireproto.MsgTypePong})
	case wireproto.MsgTypeConversationAck:
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

// writeStdinAck is called after the ordered input worker completes. Keeping
// the acknowledgement off the WebSocket read loop means a slow tmux/PTY
// backend cannot prevent the server from accepting subsequent frames.
func (s *Server) writeStdinAck(
	conn *websocket.Conn,
	writeMu *sync.Mutex,
	sess *session.Session,
	client chan []byte,
	sessionID string,
	writeErr error,
) {
	ack := TerminalMessage{Type: wireproto.MsgTypeStdinAck, Ok: writeErr == nil, AcceptedThrough: sess.ClientAcceptedThrough(client)}
	if writeErr != nil {
		ack.Data = writeErr.Error()
		switch {
		case errors.Is(writeErr, errPTYClosed):
			ack.Reason = wireproto.StdinAckReasonPTYClosed
		case errors.Is(writeErr, session.ErrInputQueueFull):
			ack.Reason = wireproto.StdinAckReasonQueueFull
		default:
			ack.Reason = wireproto.StdinAckReasonTmuxFailed
		}
		log.Printf("ws[%s]: PTY write failed: %v", sessionID, writeErr)
	}
	_ = writeTerminalJSON(conn, writeMu, ack)
	if errors.Is(writeErr, session.ErrPTYClosed) || errors.Is(writeErr, errPTYClosed) {
		_ = conn.Close()
	}
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
