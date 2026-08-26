package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"web-console/internal/config"
	"web-console/internal/events"
	"web-console/internal/wireproto"
	"web-console/session"

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

// wsWriteTimeout bounds every terminal JSON write, including snapshot replay.
const wsWriteTimeout = 10 * time.Second

func writeTerminalJSON(conn *websocket.Conn, writeMu *sync.Mutex, msg TerminalMessage) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	if err := conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout)); err != nil {
		return err
	}
	return conn.WriteJSON(msg)
}

// Keep the historical package-main names as aliases while the wire contract
// itself lives in the transport-independent protocol package.
type TerminalMessage = wireproto.TerminalMessage

// Legacy package-main variables keep older tests and integrations source
// compatible. The protocol values themselves are declared only in wireproto.
var (
	MsgTypeStdin                  = wireproto.MsgTypeStdin
	MsgTypeStdout                 = wireproto.MsgTypeStdout
	MsgTypeResize                 = wireproto.MsgTypeResize
	MsgTypeResizeInfo             = wireproto.MsgTypeResizeInfo
	MsgTypeSizeInfo               = wireproto.MsgTypeSizeInfo
	MsgTypeTakeLease              = wireproto.MsgTypeTakeLease
	MsgTypeExit                   = wireproto.MsgTypeExit
	MsgTypeError                  = wireproto.MsgTypeError
	MsgTypePing                   = wireproto.MsgTypePing
	MsgTypePong                   = wireproto.MsgTypePong
	MsgTypeSyncWarning            = wireproto.MsgTypeSyncWarning
	MsgTypeHistoryEnd             = wireproto.MsgTypeHistoryEnd
	MsgTypeConversationAck        = wireproto.MsgTypeConversationAck
	MsgTypeSessionReady           = wireproto.MsgTypeSessionReady
	MsgTypeStdinAck               = wireproto.MsgTypeStdinAck
	MsgTypeControl                = wireproto.MsgTypeControl
	MsgTypeHello                  = wireproto.MsgTypeHello
	MsgTypeResync                 = wireproto.MsgTypeResync
	MsgTypeSnapshotNotice         = wireproto.MsgTypeSnapshotNotice
	MsgTypeEchoState              = wireproto.MsgTypeEchoState
	MsgTypeMouseMode              = wireproto.MsgTypeMouseMode
	StdinIntentTyping             = wireproto.StdinIntentTyping
	StdinIntentBulkText           = wireproto.StdinIntentBulkText
	StdinIntentNamedKey           = wireproto.StdinIntentNamedKey
	StdinAckReasonTmuxWriteFailed = wireproto.StdinAckReasonTmuxFailed
	StdinAckReasonPTYClosed       = wireproto.StdinAckReasonPTYClosed
	StdinAckReasonOffsetGap       = wireproto.StdinAckReasonOffsetGap
	StdinAckReasonUnreconcilable  = wireproto.StdinAckReasonUnreconcilable
	ProtocolVersion               = wireproto.ProtocolVersion
)

// boundSnapshot keeps the newest complete line region and resets the renderer
// before it receives the suffix. The notice is a separate frame so it cannot
// become terminal content or corrupt a full-screen application.
func boundSnapshot(snapshot []byte, maxBytes int) ([]byte, int, bool) {
	if maxBytes <= 0 || len(snapshot) <= maxBytes {
		return snapshot, 0, false
	}
	reset := []byte("\x1bc")
	if maxBytes <= len(reset) {
		return append([]byte(nil), reset[:maxBytes]...), bytes.Count(snapshot, []byte{'\n'}), true
	}
	cut := len(snapshot)
	payloadMax := maxBytes - len(reset)
	// Choose the oldest complete line whose suffix fits. This avoids both
	// partial UTF-8/escape data and arbitrary mid-line cuts.
	for boundary := 0; boundary < len(snapshot); {
		relative := bytes.IndexByte(snapshot[boundary:], '\n')
		if relative < 0 {
			break
		}
		boundary += relative + 1
		if len(snapshot)-boundary <= payloadMax {
			cut = boundary
			break
		}
	}
	trimmed := snapshot[cut:]
	droppedLines := bytes.Count(snapshot[:cut], []byte{'\n'})
	bounded := make([]byte, 0, len(reset)+len(trimmed))
	bounded = append(bounded, reset...)
	bounded = append(bounded, trimmed...)
	return bounded, droppedLines, true
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
		_ = writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: MsgTypeError, Data: msg})
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
		snapshot, droppedLines, truncated := boundSnapshot(sub.Snapshot, config.Load().MaxSnapshotBytes)
		if truncated {
			if err := writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: MsgTypeSnapshotNotice, Data: fmt.Sprintf("%d scrollback lines omitted", droppedLines)}); err != nil {
				return
			}
		}
		for off := 0; off < len(snapshot); off += session.HistoryChunkSize {
			end := off + session.HistoryChunkSize
			if end > len(snapshot) {
				end = len(snapshot)
			}
			if err := writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: MsgTypeStdout, Data: string(snapshot[off:end])}); err != nil {
				return
			}
			s.metrics.WSMessagesSent.Add(1)
		}
		if err := writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: MsgTypeHistoryEnd}); err != nil {
			return
		}
		var lastEchoState session.EchoState
		haveEchoState := false
		emitEchoState := func() bool {
			state, stateErr := sess.EchoState()
			if stateErr != nil {
				state = session.EchoState{}
			}
			if haveEchoState && state == lastEchoState {
				return true
			}
			lastEchoState, haveEchoState = state, true
			return writeTerminalJSON(conn, &writeMu, TerminalMessage{
				Type: MsgTypeEchoState, EchoKnown: state.Known, EchoEnabled: state.EchoEnabled,
				InAltBuffer: state.InAltBuffer, CursorAtLineEnd: state.CursorAtLineEnd,
			}) == nil
		}
		if !emitEchoState() {
			return
		}
		cols, rows, leader, leaderDevice, holdsLease, viewerCount := sess.SizeLeaseState(sub.OutputCh)
		if err := writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: MsgTypeSizeInfo, Cols: int(cols), Rows: int(rows), Leader: leader, LeaderDevice: leaderDevice, HoldsLease: holdsLease, ViewerCount: viewerCount}); err != nil {
			return
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
					_ = writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: MsgTypeExit, Code: sess.ExitCode()})
					return
				}
				err := writeTerminalJSON(conn, &writeMu, TerminalMessage{
					Type: MsgTypeStdout,
					Data: string(data),
				})
				s.metrics.WSMessagesSent.Add(1)
				if err != nil {
					return
				}
				if !emitEchoState() {
					return
				}
				if sess.FlushPending(sub.OutputCh) {
					snapshot, generation, ok := sess.Resync(sub.OutputCh)
					if !ok {
						continue
					}
					snapshot, droppedLines, truncated := boundSnapshot(snapshot, config.Load().MaxSnapshotBytes)
					if err := writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: MsgTypeResync}); err != nil {
						return
					}
					if truncated {
						if err := writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: MsgTypeSnapshotNotice, Data: fmt.Sprintf("%d scrollback lines omitted", droppedLines)}); err != nil {
							return
						}
					}
					for off := 0; off < len(snapshot); off += session.HistoryChunkSize {
						end := off + session.HistoryChunkSize
						if end > len(snapshot) {
							end = len(snapshot)
						}
						if err := writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: MsgTypeStdout, Data: string(snapshot[off:end])}); err != nil {
							return
						}
					}
					if err := writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: MsgTypeHistoryEnd}); err != nil {
						return
					}
					if !emitEchoState() {
						return
					}
					sess.CompleteResync(sub.OutputCh, generation)
				}
			case coalesced := <-sub.NotifyCh:
				_ = writeTerminalJSON(conn, &writeMu, TerminalMessage{
					Type:            MsgTypeSyncWarning,
					CoalescedFrames: coalesced,
				})
				s.metrics.WSMessagesSent.Add(1)
			case size, ok := <-sub.SizeCh:
				if !ok {
					return
				}
				cols, rows, leader, leaderDevice, holdsLease, viewerCount := sess.SizeLeaseState(sub.OutputCh)
				if cols != size[0] || rows != size[1] {
					continue
				}
				err := writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: MsgTypeSizeInfo, Cols: int(cols), Rows: int(rows), Leader: leader, LeaderDevice: leaderDevice, HoldsLease: holdsLease, ViewerCount: viewerCount})
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

	mouseMode, mouseModeErr := sess.MouseMode()
	if err := writeTerminalJSON(conn, &writeMu, TerminalMessage{
		Type:            MsgTypeSessionReady,
		Gen:             wsGen,
		AcceptedThrough: sess.AcceptedThrough(),
		ProtocolVersion: ProtocolVersion,
		MouseMode:       mouseMode,
		MouseModeKnown:  mouseModeErr == nil,
	}); err != nil {
		log.Printf("ws[%s]: failed to send session_ready: %v", sessionID, err)
		return
	}
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
