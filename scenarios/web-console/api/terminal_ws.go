package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
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
	// MsgTypeHistoryEnd signals that all buffered history chunks have been
	// sent and subsequent stdout messages are live PTY output. The client
	// uses this to batch-render history in a single write, avoiding the
	// visible "fast-forward replay" effect on page load/refresh.
	MsgTypeHistoryEnd = "history_end"
	// MsgTypeTTSCandidate carries a candidate response that the client must
	// correlate against the visible terminal before playback.
	MsgTypeTTSCandidate = "tts_candidate"
	// MsgTypeTTSAck records browser-side correlation/playback progress.
	MsgTypeTTSAck = "tts_ack"
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
	Resumed bool   `json:"resumed,omitempty"`
	EventID string `json:"eventId,omitempty"`
	Source  string `json:"source,omitempty"`
	Stage   string `json:"stage,omitempty"`
	Backend string `json:"backend,omitempty"`
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

	// Subscribe to TTS side-channel for text-to-speech delivery.
	ttsCh := sess.SubscribeTTS()
	defer sess.UnsubscribeTTS(ttsCh)

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

		// If no history was buffered, tell the client immediately so it
		// can skip waiting for the sentinel and enter live pass-through.
		if !sub.HadData {
			writeMu.Lock()
			_ = conn.WriteJSON(historyEndMsg)
			writeMu.Unlock()
		}

		for {
			select {
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
					// been forwarded. Signal the client to flush its buffer.
					writeMu.Lock()
					_ = conn.WriteJSON(historyEndMsg)
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
			case candidate, ok := <-ttsCh:
				if !ok {
					continue
				}
				writeMu.Lock()
				_ = conn.WriteJSON(TerminalMessage{
					Type:    MsgTypeTTSCandidate,
					Data:    candidate.Text,
					EventID: candidate.EventID,
					Source:  candidate.Source,
				})
				writeMu.Unlock()
			case <-ctx.Done():
				// Input loop exited (WS disconnect) — stop forwarding.
				return
			}
		}
	}()

	// Input loop: WebSocket client → PTY stdin / resize / ping-pong.
	// When this returns, defer cancel() signals the output forwarder to exit.
	for {
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
				sess.Resize(uint16(msg.Cols), uint16(msg.Rows))
				writeMu.Lock()
				_ = conn.WriteJSON(TerminalMessage{
					Type: MsgTypeResizeInfo,
					Cols: msg.Cols,
					Rows: msg.Rows,
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
		case MsgTypeTTSAck:
			if msg.EventID == "" || msg.Source == "" || msg.Stage == "" {
				sendError("Invalid TTS acknowledgment")
				continue
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
	}
}
