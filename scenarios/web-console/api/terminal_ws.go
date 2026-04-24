package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// wsPingPeriod is the keepalive ping interval; tests may override it.
var wsPingPeriod = 30 * time.Second

// probeReadyTimeout bounds how long the input loop waits for the PTY's
// attach handshake (tmux-backed sessions) to complete before giving up
// and closing the WS with a typed error. Matches the client-side 2 s ack
// budget with a 1 s safety margin so a borderline-slow tmux server surfaces
// as a ready-fail rather than a per-message ack timeout.
var probeReadyTimeout = 3 * time.Second

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
	// MsgTypeHistoryEnd signals that all buffered history chunks have been
	// sent and subsequent stdout messages are live PTY output. The client
	// uses this to batch-render history in a single write, avoiding the
	// visible "fast-forward replay" effect on page load/refresh.
	MsgTypeHistoryEnd = "history_end"
	// MsgTypeConversationEvent carries a semantic assistant event for the
	// owning session.
	MsgTypeConversationEvent = "conversation_event"
	// MsgTypeConversationAck records browser-side delivery/playback progress.
	MsgTypeConversationAck = "conversation_event_ack"
	// MsgTypeConversationEventUpdate delivers async updates (e.g. summarization)
	// for an already-delivered conversation event.
	MsgTypeConversationEventUpdate = "conversation_event_update"
	// MsgTypeConversationOutOfSync signals the client that at least one
	// conversation event was dropped server-side (per-subscriber channel
	// full). The client refetches via GET /conversation?since_sequence=N to
	// close the gap.
	MsgTypeConversationOutOfSync = "conversation_out_of_sync"
	// MsgTypeSessionReady is emitted exactly once per WS connection after the
	// PTY is confirmed to accept writes (ProbeReady). Until the client sees
	// this, stdin must stay in the pending queue.
	MsgTypeSessionReady = "session_ready"
	// MsgTypeStdinAck is echoed for every stdin message the server
	// processes. Seq matches the client-assigned sequence; Ok reports
	// whether the backend accepted the bytes. On Ok=false, Reason
	// carries a typed error code (see StdinAckReason*).
	MsgTypeStdinAck = "stdin_ack"
)

// StdinKind values discriminate keystroke input from paste payloads on
// the wire. Must stay in sync with the UI's InputKind / TerminalMessage
// kind field.
const (
	StdinKindKeystroke = "keystroke"
	StdinKindPaste     = "paste"
)

// StdinAckReason* are the typed reason codes the server emits on
// stdin_ack when Ok=false. The UI maps these to user-visible messages.
const (
	StdinAckReasonTmuxWriteFailed = "tmux_write_failed"
	StdinAckReasonPTYClosed       = "pty_closed"
	StdinAckReasonNotReady        = "not_ready"
	StdinAckReasonInvalidInput    = "invalid_input"
	// MsgTypePTYState reports a terminal-mode transition the server has
	// observed on the PTY output stream. Today it carries only the alt-
	// buffer flag (AltBuffer). Emitted once after history_end so a fresh
	// client knows the current state, and again on every transition.
	MsgTypePTYState = "pty_state"
)

// TerminalMessage is the WebSocket JSON message format.
type TerminalMessage struct {
	Type            string `json:"type"`
	Data            string `json:"data,omitempty"`
	Cols            int    `json:"cols,omitempty"`
	Rows            int    `json:"rows,omitempty"`
	Code            int    `json:"code,omitempty"`
	CoalescedFrames int    `json:"coalesced_frames,omitempty"`
	// TotalBytes is the server's monotonic output byte count. Sent with
	// history_end so the client can cache and resume from this offset.
	TotalBytes int64 `json:"total_bytes,omitempty"`
	// Resumed indicates that the client's resume offset was valid and only
	// delta data was sent (not the full history).
	Resumed                  bool     `json:"resumed,omitempty"`
	EventID                  string   `json:"eventId,omitempty"`
	Source                   string   `json:"source,omitempty"`
	Stage                    string   `json:"stage,omitempty"`
	Backend                  string   `json:"backend,omitempty"`
	Role                     string   `json:"role,omitempty"`
	CreatedAt                string   `json:"createdAt,omitempty"`
	Sequence                 int64    `json:"sequence,omitempty"`
	SpeechParagraphs         []string `json:"speechParagraphs,omitempty"`
	OriginalSpeechParagraphs []string `json:"originalSpeechParagraphs,omitempty"`
	Summarized               bool     `json:"summarized,omitempty"`
	// SummarizeError carries an auto-summarization failure message when an
	// async summarize attempt fails. Sent on conversation_event_update so the
	// UI can surface a persistent banner with retry.
	SummarizeError string `json:"summarizeError,omitempty"`
	// Seq is the client-assigned sequence number for stdin messages; the
	// server echoes it in the matching stdin_ack. Opaque to the server.
	Seq int64 `json:"seq,omitempty"`
	// Ok reports whether a server-acknowledged action succeeded (used by
	// stdin_ack).
	Ok bool `json:"ok,omitempty"`
	// AltBuffer carries the alternate-screen-buffer flag in pty_state
	// messages. Absent for all other message types.
	AltBuffer bool `json:"altBuffer,omitempty"`
	// Gen is the per-connection generation counter. The server echoes it
	// in session_ready; clients use it to decide whether a re-enqueued
	// payload belongs to the current connection (see wsGen write barrier
	// in terminal-session-refactor-implementation-plan.md §8 Phase 1).
	Gen int64 `json:"gen,omitempty"`
	// Kind discriminates keystroke vs paste on stdin frames. Empty
	// defaults to keystroke for backward-compatibility-with-nothing (the
	// UI always sends it explicitly). Values: "keystroke", "paste".
	Kind string `json:"kind,omitempty"`
	// Reason is the typed error code populated on stdin_ack frames when
	// Ok=false (and unset when Ok=true). See StdinAckReason*.
	Reason string `json:"reason,omitempty"`
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

	// Parse optional history resume offset from query string.
	// DOC: docs/concepts/ARCHITECTURE.md#terminal-history-caching
	var resumeOffset int64
	if raw := r.URL.Query().Get("history_offset"); raw != "" {
		resumeOffset, _ = strconv.ParseInt(raw, 10, 64)
	}

	// Subscribe to PTY output (the client's first resize message sets dimensions)
	sub := sess.Subscribe(resumeOffset)
	defer sess.Unsubscribe(sub.OutputCh)

	// Assign this connection a fresh generation so clients can detect
	// reconnect boundaries on their stdin-ack write barrier.
	wsGen := s.nextWSGen.Add(1)

	// Subscribe to conversation side-channel for semantic assistant events.
	conversationCh := sess.SubscribeConversation()
	defer sess.UnsubscribeConversation(conversationCh)
	conversationResyncCh := sess.ConversationResyncSignal(conversationCh)

	// writeMu serializes WebSocket writes from the output forwarder goroutine
	// and the inline input loop (which also writes pong/error responses).
	var writeMu sync.Mutex

	// Context-based goroutine lifecycle: when the input loop exits (WS
	// disconnect), cancel() fires and the output forwarder sees ctx.Done().
	// When the forwarder exits first (PTY death or WS write error), it
	// closes the connection, which unblocks the input loop's ReadMessage().
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	sendError := func(msg string) {
		writeMu.Lock()
		_ = conn.WriteJSON(TerminalMessage{Type: MsgTypeError, Data: msg})
		writeMu.Unlock()
	}

	// historyEndMsg is the history_end message for this subscription, built
	// once and reused for both the immediate (no-history) and sentinel paths.
	historyEndMsg := TerminalMessage{
		Type:       MsgTypeHistoryEnd,
		TotalBytes: sub.TotalBytes,
		Resumed:    sub.Resumed,
	}

	// Output forwarder: PTY output + coalescing notifications → WebSocket client.
	// Guaranteed to exit: either ctx.Done() fires (input loop returned) or
	// outputCh is closed (PTY exited) or WS write fails.
	go func() {
		defer conn.Close() // unblocks the input loop's ReadMessage on forwarder exit
		defer func() {
			if r := recover(); r != nil {
				log.Printf("ws[%s]: output forwarder panic (recovered): %v", sessionID, r)
			}
		}()

		// If no history was buffered, tell the client immediately so it
		// can skip waiting for the sentinel and enter live pass-through.
		if !sub.HadData {
			writeMu.Lock()
			_ = conn.WriteJSON(historyEndMsg)
			_ = conn.WriteJSON(TerminalMessage{Type: MsgTypePTYState, AltBuffer: sub.InitialAltBuffer})
			writeMu.Unlock()
		}

		// Server-side WebSocket keepalive: send a ping every 30s to prevent
		// reverse proxies (Cloudflare tunnel default idle timeout ~100s) from
		// killing the connection during periods without PTY output.
		pingTicker := time.NewTicker(wsPingPeriod)
		defer pingTicker.Stop()

		for {
			select {
			case <-pingTicker.C:
				writeMu.Lock()
				err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
				writeMu.Unlock()
				if err != nil {
					return
				}
			case data, ok := <-sub.OutputCh:
				if !ok {
					// Channel closed = process exited; forward the real exit code.
					writeMu.Lock()
					_ = conn.WriteJSON(TerminalMessage{Type: MsgTypeExit, Code: sess.ExitCode()})
					writeMu.Unlock()
					return
				}
				if data == nil {
					// Nil sentinel from Subscribe: all history chunks have
					// been forwarded. Signal the client to flush its buffer
					// and deliver the initial pty_state so local-echo and
					// paste-gating decisions can react.
					writeMu.Lock()
					_ = conn.WriteJSON(historyEndMsg)
					_ = conn.WriteJSON(TerminalMessage{Type: MsgTypePTYState, AltBuffer: sub.InitialAltBuffer})
					writeMu.Unlock()
					continue
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
				// Drain coalesced data after a successful write so the
				// broadcast loop can resume normal per-frame delivery.
				sess.FlushPending(sub.OutputCh)
			case coalesced := <-sub.NotifyCh:
				writeMu.Lock()
				_ = conn.WriteJSON(TerminalMessage{
					Type:            MsgTypeSyncWarning,
					CoalescedFrames: coalesced,
				})
				writeMu.Unlock()
				s.metrics.WSMessagesSent.Add(1)
			case altBuffer := <-sub.StateCh:
				writeMu.Lock()
				_ = conn.WriteJSON(TerminalMessage{
					Type:      MsgTypePTYState,
					AltBuffer: altBuffer,
				})
				writeMu.Unlock()
				s.metrics.WSMessagesSent.Add(1)
			case event, ok := <-conversationCh:
				if !ok {
					continue
				}
				msgType := MsgTypeConversationEvent
				if event.IsUpdate {
					msgType = MsgTypeConversationEventUpdate
				}
				writeMu.Lock()
				_ = conn.WriteJSON(TerminalMessage{
					Type:                     msgType,
					Data:                     event.Text,
					EventID:                  event.ID,
					Source:                   event.Source,
					Role:                     string(event.Role),
					CreatedAt:                event.CreatedAt.UTC().Format(time.RFC3339),
					Sequence:                 event.Sequence,
					SpeechParagraphs:         event.SpeechParagraphs,
					OriginalSpeechParagraphs: event.OriginalSpeechParagraphs,
					Summarized:               event.Summarized,
					SummarizeError:           event.SummarizeError,
				})
				writeMu.Unlock()
			case <-conversationResyncCh:
				// A prior SendConversation drop left this client out of
				// sync. Tell the client so it refetches the gap via
				// GET /conversation?since_sequence=N.
				writeMu.Lock()
				_ = conn.WriteJSON(TerminalMessage{Type: MsgTypeConversationOutOfSync})
				writeMu.Unlock()
			case <-ctx.Done():
				// Input loop exited (WS disconnect) — stop forwarding.
				return
			}
		}
	}()

	// Confirm the PTY pipeline is actually accepting writes before telling
	// the client it's safe to send stdin. For the standard backend this
	// returns immediately; for persistent (tmux) sessions this waits for
	// the attach-session handshake to complete. Without this gate, writes
	// issued during the 50–500 ms tmux attach window are silently dropped.
	probeCtx, probeCancel := context.WithTimeout(ctx, probeReadyTimeout)
	probeErr := sess.ProbeReady(probeCtx)
	probeCancel()
	if probeErr != nil {
		log.Printf("ws[%s]: ProbeReady failed (backend=%s): %v", sessionID, sess.Backend, probeErr)
		sendError("session_not_ready")
		return
	}

	// sessionReady gates stdin-loss diagnostics: any stdin received before
	// this flips true means the client skipped waiting for session_ready,
	// which should be impossible in the current protocol.
	sessionReady := false
	writeMu.Lock()
	if err := conn.WriteJSON(TerminalMessage{Type: MsgTypeSessionReady, Gen: wsGen}); err != nil {
		writeMu.Unlock()
		log.Printf("ws[%s]: failed to send session_ready: %v", sessionID, err)
		return
	}
	writeMu.Unlock()
	sessionReady = true

	// Input loop: WebSocket client → PTY stdin / resize / ping-pong.
	// When this returns, defer cancel() signals the output forwarder to
	// exit. Per-message handling lives in terminal_ws_input.go so the
	// lifecycle glue stays small here.
	for {
		_, rawMsg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		msg, decodeErr, ok := s.decodeInputMessage(sessionID, rawMsg)
		if !ok {
			sendError(decodeErr)
			continue
		}
		res := s.dispatchInputMessage(conn, &writeMu, sess, sessionID, msg, sessionReady)
		if res.CloseReason != "" {
			sendError(res.CloseReason)
		}
		if res.Close {
			return
		}
	}
}
