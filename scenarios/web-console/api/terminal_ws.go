package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"web-console/internal/events"
	"web-console/session"

	"github.com/gorilla/mux"
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
// DOC: docs/internal/ERROR_SEMANTICS.md#websocket-error-protocol
// WebSocket message types for terminal I/O.
// [REQ:P0-002b] WebSocket I/O Streaming
const (
	MsgTypeStdin       = "stdin"
	MsgTypeStdout      = "stdout"
	MsgTypeResize      = "resize"
	MsgTypeResizeInfo  = "resize_info"
	MsgTypeSizeInfo    = "size_info"
	MsgTypeTakeLease   = "take_lease"
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
	// MsgTypeConversationAck records browser-side delivery/playback progress.
	// This is a client→server INPUT message (playback telemetry); the terminal
	// WS keeps it even though conversation events themselves now stream over
	// the global SSE channel (/api/v1/events/stream), not this socket.
	MsgTypeConversationAck = "conversation_event_ack"
	// MsgTypeSessionReady is emitted exactly once per WS connection after the
	// PTY is confirmed to accept writes (ProbeReady). Until the client sees
	// this, stdin must stay in the pending queue.
	MsgTypeSessionReady = "session_ready"
	// MsgTypeStdinAck is echoed for every stdin message the server
	// processes. Seq matches the client-assigned sequence; Ok reports
	// whether the backend accepted the bytes. On Ok=false, Reason
	// carries a typed error code (see StdinAckReason*).
	MsgTypeStdinAck = "stdin_ack"
	// MsgTypeControl carries synthetic terminal bytes such as mouse input.
	// It deliberately has no sequence number, acknowledgement, or replay.
	MsgTypeControl = "control"
)

// StdinIntent values carry the semantic intent chosen by the UI source.
const (
	StdinIntentTyping   = "typing"
	StdinIntentBulkText = "bulk_text"
	StdinIntentNamedKey = "named_key"
)

// StdinAckReason* are the typed reason codes the server emits on
// stdin_ack when Ok=false. The UI maps these to user-visible messages.
const (
	StdinAckReasonTmuxWriteFailed = "tmux_write_failed"
	StdinAckReasonPTYClosed       = "pty_closed"
	StdinAckReasonNotReady        = "not_ready"
)

// TerminalMessage is the WebSocket JSON message format.
type TerminalMessage struct {
	Type            string `json:"type"`
	Data            string `json:"data,omitempty"`
	Cols            int    `json:"cols,omitempty"`
	Rows            int    `json:"rows,omitempty"`
	Code            int    `json:"code,omitempty"`
	CoalescedFrames int    `json:"coalesced_frames,omitempty"`
	EventID         string `json:"eventId,omitempty"`
	Source          string `json:"source,omitempty"`
	Stage           string `json:"stage,omitempty"`
	Backend         string `json:"backend,omitempty"`
	// Seq is the client-assigned sequence number for stdin messages; the
	// server echoes it in the matching stdin_ack. Opaque to the server.
	Seq int64 `json:"seq,omitempty"`
	// Ok reports whether a server-acknowledged action succeeded (used by
	// stdin_ack).
	Ok bool `json:"ok,omitempty"`
	// Gen is the per-connection generation counter. The server echoes it
	// in session_ready; clients use it to decide whether a re-enqueued
	// payload belongs to the current connection (see wsGen write barrier
	Gen int64 `json:"gen,omitempty"`
	// Intent carries the semantic stdin intent. Empty or unknown values
	// default to typing for forward compatibility.
	Intent string `json:"intent,omitempty"`
	// Reason is the typed error code populated on stdin_ack frames when
	// Ok=false (and unset when Ok=true). See StdinAckReason*.
	Reason       string `json:"reason,omitempty"`
	Leader       string `json:"leader,omitempty"`
	LeaderDevice string `json:"leaderDevice,omitempty"`
	HoldsLease   bool   `json:"holdsLease"`
	ViewerCount  int    `json:"viewerCount,omitempty"`
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
	if remote, ok := s.remoteForRequest(r); ok {
		s.handleRemoteTerminalWS(w, r, remote)
		return
	}
	if strings.HasPrefix(mux.Vars(r)["id"], "remote:") {
		writeCatalogError(w, "session_not_found", "Remote session not found")
		return
	}
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
		ReadBufferSize:  s.sessions.GetConfig().WSBufferSize,
		WriteBufferSize: s.sessions.GetConfig().WSBufferSize,
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
	// Bound half-open connections. The server sends protocol-level pings from
	// the output forwarder; a client pong proves the peer is still reachable
	// and extends the read deadline. Application-level JSON pings are not a
	// substitute because they can be queued behind a stalled browser.
	resetReadDeadline := func() error {
		return conn.SetReadDeadline(time.Now().Add(2 * wsPingPeriod))
	}
	if err := resetReadDeadline(); err != nil {
		log.Printf("ws[%s]: failed to set read deadline: %v", sessionID, err)
		return
	}
	conn.SetPongHandler(func(string) error { return resetReadDeadline() })

	// [REQ:P1-004a] Emit connection event
	s.events.Emit(events.SessionConnected, sessionID, nil)
	s.metrics.ConnectionsTotal.Add(1)
	s.metrics.ActiveConnections.Add(1)
	defer func() {
		s.events.Emit(events.SessionDisconnected, sessionID, nil)
		s.metrics.ActiveConnections.Add(-1)
	}()

	// Subscribe to PTY output. Subscribe atomically captures the current
	// emulator snapshot before registering the live channel; live frames
	// are applied on top of the snapshot on the receiver.
	sub := sess.Subscribe()
	defer sess.Unsubscribe(sub.OutputCh)
	sess.SetClientDevice(sub.OutputCh, r.URL.Query().Get("deviceId"), r.URL.Query().Get("deviceLabel"))

	// Assign this connection a fresh generation so clients can detect
	// reconnect boundaries on their stdin-ack write barrier.
	wsGen := s.nextWSGen.Add(1)

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

	// Output forwarder: snapshot → PTY output → WebSocket client.
	// Guaranteed to exit: either ctx.Done() fires (input loop returned) or
	// outputCh is closed (PTY exited) or WS write fails.
	go func() {
		defer conn.Close() // unblocks the input loop's ReadMessage on forwarder exit
		defer func() {
			if r := recover(); r != nil {
				log.Printf("ws[%s]: output forwarder panic (recovered): %v", sessionID, r)
			}
		}()

		// Stream the self-contained ANSI snapshot first so the client
		// reproduces the current (screen, alt-buffer, scrollback) triple
		// before any live frame arrives. Chunked at session.HistoryChunkSize so
		// no single JSON message stalls the renderer.
		writeMu.Lock()
		for off := 0; off < len(sub.Snapshot); off += session.HistoryChunkSize {
			end := off + session.HistoryChunkSize
			if end > len(sub.Snapshot) {
				end = len(sub.Snapshot)
			}
			if err := conn.WriteJSON(TerminalMessage{Type: MsgTypeStdout, Data: string(sub.Snapshot[off:end])}); err != nil {
				writeMu.Unlock()
				return
			}
			s.metrics.WSMessagesSent.Add(1)
		}
		_ = conn.WriteJSON(TerminalMessage{Type: MsgTypeHistoryEnd})
		cols, rows, leader, leaderDevice, holdsLease, viewerCount := sess.SizeLeaseState(sub.OutputCh)
		if err := conn.WriteJSON(TerminalMessage{Type: MsgTypeSizeInfo, Cols: int(cols), Rows: int(rows), Leader: leader, LeaderDevice: leaderDevice, HoldsLease: holdsLease, ViewerCount: viewerCount}); err != nil {
			writeMu.Unlock()
			return
		}
		writeMu.Unlock()

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
				sess.FlushPending(sub.OutputCh)
			case coalesced := <-sub.NotifyCh:
				writeMu.Lock()
				_ = conn.WriteJSON(TerminalMessage{
					Type:            MsgTypeSyncWarning,
					CoalescedFrames: coalesced,
				})
				writeMu.Unlock()
				s.metrics.WSMessagesSent.Add(1)
			case size, ok := <-sub.SizeCh:
				if !ok {
					return
				}
				cols, rows, leader, leaderDevice, holdsLease, viewerCount := sess.SizeLeaseState(sub.OutputCh)
				if cols != size[0] || rows != size[1] {
					continue
				}
				writeMu.Lock()
				err := conn.WriteJSON(TerminalMessage{Type: MsgTypeSizeInfo, Cols: int(cols), Rows: int(rows), Leader: leader, LeaderDevice: leaderDevice, HoldsLease: holdsLease, ViewerCount: viewerCount})
				writeMu.Unlock()
				if err != nil {
					return
				}
				s.metrics.WSMessagesSent.Add(1)
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

	writeMu.Lock()
	if err := conn.WriteJSON(TerminalMessage{Type: MsgTypeSessionReady, Gen: wsGen}); err != nil {
		writeMu.Unlock()
		log.Printf("ws[%s]: failed to send session_ready: %v", sessionID, err)
		return
	}
	writeMu.Unlock()

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
		res := s.dispatchInputMessage(conn, &writeMu, sess, sub.OutputCh, sessionID, msg)
		if res.CloseReason != "" {
			sendError(res.CloseReason)
		}
		if res.Close {
			return
		}
	}
}
