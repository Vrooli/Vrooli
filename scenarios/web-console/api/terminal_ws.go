package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"web-console/internal/config"
	"web-console/internal/events"
	"web-console/internal/wireproto"
	"web-console/session"
	"web-console/terminal"

	"github.com/gorilla/websocket"
)

// wsPingPeriod is the keepalive ping interval in nanoseconds. It is atomic so
// tests can shorten it without racing an already-running websocket handler.
var wsPingPeriod atomic.Int64

func init() {
	wsPingPeriod.Store(int64(30 * time.Second))
}

// probeReadyTimeout bounds how long the input loop waits for the PTY's
// attach handshake (tmux-backed sessions) to complete before giving up
// and closing the WS with a typed error. Matches the client-side 2 s ack
// budget with a 1 s safety margin so a borderline-slow tmux server surfaces
// as a ready-fail rather than a per-message ack timeout.
var probeReadyTimeout = 3 * time.Second

// wsWriteTimeout bounds every terminal JSON write, including snapshot replay.
const wsWriteTimeout = 10 * time.Second

type terminalResumeRequest struct {
	want            bool
	renderedThrough int64
}

// sizeInfoMessage is the single construction of the size_info payload. Three
// call sites previously each destructured a tuple and rebuilt this literal, so
// a new presentational field meant three edits and three chances to miss one.
func sizeInfoMessage(snapshot session.SizeLeaseSnapshot) TerminalMessage {
	return TerminalMessage{
		Type:         wireproto.MsgTypeSizeInfo,
		Cols:         int(snapshot.Cols),
		Rows:         int(snapshot.Rows),
		Leader:       snapshot.Leader,
		LeaderDevice: snapshot.LeaderDevice,
		DeviceClass:  snapshot.LeaderClass,
		KbOpen:       snapshot.LeaderKbOpen,
		HoldsLease:   snapshot.HoldsLease,
		ViewerCount:  snapshot.ViewerCount,
	}
}

// presenceMessage is the single construction of the presence payload. It
// carries the same leader-presentation fields as size_info, without the grid.
func presenceMessage(state session.PresenceState) TerminalMessage {
	return TerminalMessage{
		Type:         wireproto.MsgTypePresence,
		Leader:       state.Leader,
		LeaderDevice: state.LeaderDevice,
		DeviceClass:  state.LeaderClass,
		KbOpen:       state.LeaderKbOpen,
		HoldsLease:   state.HoldsLease,
		ViewerCount:  state.ViewerCount,
	}
}

func writeTerminalJSON(conn *websocket.Conn, writeMu *sync.Mutex, msg TerminalMessage) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	if err := conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout)); err != nil {
		return err
	}
	return conn.WriteJSON(msg)
}

type TerminalMessage = wireproto.TerminalMessage

// boundSnapshot keeps the newest complete line region and resets the renderer
// before it receives the suffix. The notice is a separate frame so it cannot
// become terminal content or corrupt a full-screen application.
func boundSnapshot(snapshot []byte, maxBytes int) ([]byte, int, bool) {
	if maxBytes <= 0 || len(snapshot) <= maxBytes {
		return snapshot, 0, false
	}
	reset := []byte(terminal.SnapshotPrologue)
	if maxBytes <= len(reset) {
		return append([]byte(nil), reset...), bytes.Count(snapshot, []byte{'\n'}), true
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
		return conn.SetReadDeadline(time.Now().Add(2 * time.Duration(wsPingPeriod.Load())))
	}
	if err := resetReadDeadline(); err != nil {
		log.Printf("ws[%s]: failed to set read deadline: %v", sessionID, err)
		return
	}
	conn.SetPongHandler(func(string) error { return resetReadDeadline() })

	s.metrics.ConnectionsTotal.Add(1)
	s.metrics.ActiveConnections.Add(1)

	// Subscribe to PTY output. Subscribe atomically captures the current
	// emulator snapshot before registering the live channel; live frames
	// are applied on top of the snapshot on the receiver.
	deviceID := r.URL.Query().Get("deviceId")
	deviceLabel := r.URL.Query().Get("deviceLabel")
	deviceClass := r.URL.Query().Get("deviceClass")
	sub := sess.Subscribe(deviceID, deviceLabel, deviceClass)
	defer sess.Unsubscribe(sub.OutputCh)
	deviceDetails := map[string]string{
		"deviceId": deviceID, "deviceLabel": deviceLabel, "deviceClass": deviceClass, "connId": sub.ConnID,
	}
	// [REQ:P1-004a] Emit connection events with the same display-only identity
	// the roster receives from the live session projection.
	s.events.Emit(events.SessionConnected, sessionID, deviceDetails)
	defer func() {
		s.events.Emit(events.SessionDisconnected, sessionID, deviceDetails)
		s.metrics.ActiveConnections.Add(-1)
	}()

	// Assign this connection a fresh generation so clients can detect
	// reconnect boundaries on their stdin-ack write barrier.
	wsGen := s.nextWSGen.Add(1)

	// writeMu serializes WebSocket writes from the output forwarder goroutine
	// and the inline input loop (which also writes pong/error responses).
	var writeMu sync.Mutex
	sess.SetClientProbe(sub.OutputCh, func() {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
	})
	conn.SetPongHandler(func(string) error {
		sess.MarkClientPong(sub.OutputCh)
		return resetReadDeadline()
	})
	if sub.ReclaimClient != nil {
		prior := sub.ReclaimClient
		go func() {
			if !sess.ProbeClient(prior) {
				sess.Supersede(prior)
			}
		}()
	}
	readyCh := make(chan struct{})
	resumeCh := make(chan terminalResumeRequest, 1)

	// Context-based goroutine lifecycle: when the input loop exits (WS
	// disconnect), cancel() fires and the output forwarder sees ctx.Done().
	// When the forwarder exits first (PTY death or WS write error), it
	// closes the connection, which unblocks the input loop's ReadMessage().
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	sendError := func(msg string) {
		_ = writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: wireproto.MsgTypeError, Data: msg})
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

		// A reconnecting client identifies the output cursor it has already
		// rendered. Wait only briefly so legacy clients still receive the
		// initial snapshot promptly; the browser sends its hello immediately
		// after opening the socket.
		resume := terminalResumeRequest{}
		select {
		case resume = <-resumeCh:
		case <-time.After(50 * time.Millisecond):
		}
		replayed := false
		if resume.want {
			frames, cursor, ok := sess.ReplayFrom(resume.renderedThrough)
			if ok {
				for _, frame := range frames {
					if err := writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: wireproto.MsgTypeStdout, Data: string(frame.Data), OutputCursor: frame.EndCursor}); err != nil {
						return
					}
					s.metrics.WSMessagesSent.Add(1)
				}
				if err := writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: wireproto.MsgTypeHistoryEnd, OutputCursor: cursor}); err != nil {
					return
				}
				replayed = true
			} else if err := writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: wireproto.MsgTypeResync}); err != nil {
				return
			}
		}

		// Stream the self-contained ANSI snapshot first so the client
		// reproduces the current (screen, alt-buffer, scrollback) triple
		// before any live frame arrives. Chunked at session.HistoryChunkSize so
		// no single JSON message stalls the renderer.
		if !replayed {
			snapshot, droppedLines, truncated := boundSnapshot(sub.Snapshot, config.Load().MaxSnapshotBytes)
			if truncated {
				if err := writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: wireproto.MsgTypeSnapshotNotice, Data: fmt.Sprintf("%d scrollback lines omitted", droppedLines)}); err != nil {
					return
				}
			}
			for off := 0; off < len(snapshot); off += session.HistoryChunkSize {
				end := off + session.HistoryChunkSize
				if end > len(snapshot) {
					end = len(snapshot)
				}
				if err := writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: wireproto.MsgTypeStdout, Data: string(snapshot[off:end])}); err != nil {
					return
				}
				s.metrics.WSMessagesSent.Add(1)
			}
			if err := writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: wireproto.MsgTypeHistoryEnd, OutputCursor: sess.OutputCursor()}); err != nil {
				return
			}
		}
		sess.RefreshEchoState(false)
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
				Type: wireproto.MsgTypeEchoState, EchoKnown: state.Known, EchoEnabled: state.EchoEnabled,
				InAltBuffer: state.InAltBuffer, CursorAtLineEnd: state.CursorAtLineEnd,
			}) == nil
		}
		if !emitEchoState() {
			return
		}
		if err := writeTerminalJSON(conn, &writeMu, sizeInfoMessage(sess.SizeLeaseState(sub.OutputCh))); err != nil {
			return
		}

		// Server-side WebSocket keepalive: send a ping every 30s to prevent
		// reverse proxies (Cloudflare tunnel default idle timeout ~100s) from
		// killing the connection during periods without PTY output.
		pingTicker := time.NewTicker(time.Duration(wsPingPeriod.Load()))
		defer pingTicker.Stop()
		// Idle sessions still refresh echo at the bounded maximum; active
		// output/input paths use the shared 250 ms sampling floor.
		echoTicker := time.NewTicker(5 * time.Second)
		defer echoTicker.Stop()
		// Presence may already contain the subscription's initial state. Hold
		// that channel behind the session_ready barrier so handshake messages
		// retain their deterministic order on the wire.
		presenceCh := (<-chan session.PresenceState)(nil)

		for {
			select {
			case <-sub.SupersedeCh:
				return
			case <-pingTicker.C:
				writeMu.Lock()
				err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
				writeMu.Unlock()
				if err != nil {
					return
				}
			case <-echoTicker.C:
				sess.RefreshEchoState(false)
				if !emitEchoState() {
					return
				}
			case <-readyCh:
				presenceCh = sub.PresenceCh
				// SizeLeaseState above is the authoritative handshake snapshot.
				// Consume the subscription bootstrap notification here so it
				// cannot appear after history_end and reorder the established
				// terminal stream. Future presence changes remain on the channel.
				select {
				case <-presenceCh:
				default:
				}
				readyCh = nil
			case frame, ok := <-sub.FrameCh:
				if !ok {
					_ = writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: wireproto.MsgTypeExit, Code: sess.ExitCode()})
					return
				}
				err := writeTerminalJSON(conn, &writeMu, TerminalMessage{
					Type: wireproto.MsgTypeStdout,
					Data: string(frame.Data), OutputCursor: frame.EndCursor,
				})
				s.metrics.WSMessagesSent.Add(1)
				if err != nil {
					return
				}
				sess.RefreshEchoState(false)
				if !emitEchoState() {
					return
				}
				if sess.FlushPendingFrame(sub.FrameCh) {
					snapshot, generation, ok := sess.Resync(sub.OutputCh)
					if !ok {
						continue
					}
					snapshot, droppedLines, truncated := boundSnapshot(snapshot, config.Load().MaxSnapshotBytes)
					if err := writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: wireproto.MsgTypeResync}); err != nil {
						return
					}
					if truncated {
						if err := writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: wireproto.MsgTypeSnapshotNotice, Data: fmt.Sprintf("%d scrollback lines omitted", droppedLines)}); err != nil {
							return
						}
					}
					for off := 0; off < len(snapshot); off += session.HistoryChunkSize {
						end := off + session.HistoryChunkSize
						if end > len(snapshot) {
							end = len(snapshot)
						}
						if err := writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: wireproto.MsgTypeStdout, Data: string(snapshot[off:end])}); err != nil {
							return
						}
					}
					if err := writeTerminalJSON(conn, &writeMu, TerminalMessage{Type: wireproto.MsgTypeHistoryEnd, OutputCursor: sess.OutputCursor()}); err != nil {
						return
					}
					sess.CompleteResync(sub.OutputCh, generation)
				}
			case coalesced := <-sub.NotifyCh:
				_ = writeTerminalJSON(conn, &writeMu, TerminalMessage{
					Type:            wireproto.MsgTypeSyncWarning,
					CoalescedFrames: coalesced,
				})
				s.metrics.WSMessagesSent.Add(1)
			case size, ok := <-sub.SizeCh:
				if !ok {
					return
				}
				snapshot := sess.SizeLeaseState(sub.OutputCh)
				if snapshot.Cols != size[0] || snapshot.Rows != size[1] {
					continue
				}
				err := writeTerminalJSON(conn, &writeMu, sizeInfoMessage(snapshot))
				if err != nil {
					return
				}
				s.metrics.WSMessagesSent.Add(1)
			case presence, ok := <-presenceCh:
				if !ok {
					return
				}
				err := writeTerminalJSON(conn, &writeMu, presenceMessage(presence))
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
		Type:            wireproto.MsgTypeSessionReady,
		Gen:             wsGen,
		AcceptedThrough: sess.AcceptedThroughFor(sub.OutputCh),
		ProtocolVersion: wireproto.ProtocolVersion,
		MouseMode:       mouseMode,
		MouseModeKnown:  mouseModeErr == nil,
	}); err != nil {
		log.Printf("ws[%s]: failed to send session_ready: %v", sessionID, err)
		return
	}
	// The output forwarder uses this barrier to begin consuming presence
	// notifications only after session_ready has been written.
	close(readyCh)
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
		res := s.dispatchInputMessage(conn, &writeMu, sess, sub.OutputCh, sessionID, msg, resumeCh)
		if res.CloseReason != "" {
			sendError(res.CloseReason)
		}
		if res.Close {
			return
		}
	}
}
