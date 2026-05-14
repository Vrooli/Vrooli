package session

// broadcast.go: Output fan-out, per-client delivery, coalesce-on-slow-client,
// pending-buffer trim, and SIGWINCH-based recovery when a trim happens.
//
// This file owns:
//   - ClientInfo         (per-subscriber flow-control state)
//   - (*Session).broadcast, .deliver, .notifyIfThreshold
//   - (*Session).FlushPending
//   - (*Session).maybeSIGWINCHRecovery
//
// PTY bytes flow through the per-session terminal.Emulator (the durable
// state) and then to subscribed clients as live frames. The emulator is
// the source of truth for replay; per-client coalesce buffering only
// absorbs short bursts when a client falls behind.

import (
	"fmt"
	"time"

	"web-console/internal/backend"
)

// pendingBufferMax is the maximum bytes of coalesced output retained
// per slow client. When exceeded, oldest bytes are truncated; the next
// snapshot replay restores correct state.
const pendingBufferMax = 1 << 20 // 1 MiB

// HistoryChunkSize is the maximum bytes per WebSocket frame when
// streaming the snapshot or draining the pending buffer. Smaller chunks
// prevent browser UI freezes on large initial replays.
const HistoryChunkSize = 64 * 1024

// ClientInfo tracks per-client broadcast flow control for a subscribed
// WebSocket connection. When the client's output channel is full, incoming
// frames are coalesced into a pending buffer instead of being dropped.
// The WebSocket output forwarder calls FlushPending after each successful
// write to drain coalesced data back into the channel.
type ClientInfo struct {
	pending         []byte   // coalesced data awaiting consumer drain
	pendingTrimmed  bool     // set when pending buffer was trimmed; triggers SIGWINCH after drain
	CoalescedFrames int      // count of coalesced frames (observability)
	NotifyCh        chan int // receives cumulative coalesced count when threshold crossed
}

// broadcast feeds PTY output into the durable emulator and fans out the
// frame to all connected WebSocket clients. Slow clients have frames
// coalesced into a pending buffer instead of being dropped.
func (s *Session) broadcast(data []byte) {
	if len(data) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Feed the durable emulator with RAW PTY bytes so its parser sees
	// every CSI query — including DECRQM 2026 ($p), which the
	// client-bound sanitizer strips. The ANSI responder observes the
	// emulator's ControlEvent stream and answers queries server-side;
	// if the emulator never saw the query, the reply never fires.
	_, _ = s.emu.Feed(data)
	prevAlt := s.inAltBuffer
	s.inAltBuffer = s.emu.InAltBuffer()
	if prevAlt != s.inAltBuffer {
		s.lastAltBufferTransition = time.Now()
	}
	bctrace("broadcast", s.ID, fmt.Sprintf("clients=%d alt=%v", len(s.clients), s.inAltBuffer), data)
	s.markFrame()
	if len(s.clients) == 0 {
		return
	}
	// Sanitize only the client copy. The emulator already saw the raw
	// bytes; xterm.js gets the cleaned stream (DEC mode 2026 sequences
	// stripped — see sanitize.go for the xterm.js v6 crash rationale).
	clientData := sanitizeForClient(data)
	if len(clientData) == 0 {
		return
	}
	cp := make([]byte, len(clientData))
	copy(cp, clientData)
	for ch, info := range s.clients {
		s.deliver(ch, info, cp)
	}
}

// DOC: docs/internal/ERROR_SEMANTICS.md#sync-warning-coalescing-notification
// deliver sends data to a client channel, coalescing into the pending buffer
// when the channel is full. Must be called with s.mu held.
func (s *Session) deliver(ch chan []byte, info *ClientInfo, data []byte) {
	if len(info.pending) > 0 {
		info.pending = append(info.pending, data...)
		info.CoalescedFrames++
		if len(info.pending) > pendingBufferMax {
			info.pending = info.pending[len(info.pending)-pendingBufferMax:]
			info.pendingTrimmed = true
		}
		s.notifyIfThreshold(info)
		return
	}
	select {
	case ch <- data:
	default:
		info.pending = append([]byte(nil), data...)
		info.CoalescedFrames++
		s.notifyIfThreshold(info)
	}
}

// notifyIfThreshold sends a coalescing notification when the cumulative
// count crosses the configured threshold. Must be called with s.mu held.
func (s *Session) notifyIfThreshold(info *ClientInfo) {
	if s.coalesceNotifyThreshold > 0 && info.CoalescedFrames%s.coalesceNotifyThreshold == 0 {
		select {
		case info.NotifyCh <- info.CoalescedFrames:
		default:
		}
	}
}

// FlushPending drains any coalesced output for the given client channel.
// The WebSocket output forwarder calls this after each successful write
// to resume normal per-frame delivery. Data is chunked at HistoryChunkSize
// to prevent browser UI freezes from single large WebSocket messages.
// DOC: docs/internal/SEAMS.md#3-domain--session-lifecycle
func (s *Session) FlushPending(ch chan []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, ok := s.clients[ch]
	if !ok || len(info.pending) == 0 {
		return
	}
	for len(info.pending) > 0 {
		end := HistoryChunkSize
		if end > len(info.pending) {
			end = len(info.pending)
		}
		chunk := make([]byte, end)
		copy(chunk, info.pending[:end])
		select {
		case ch <- chunk:
			info.pending = info.pending[end:]
		default:
			return
		}
	}
	info.pending = nil
	info.CoalescedFrames = 0
	if info.pendingTrimmed {
		info.pendingTrimmed = false
		s.maybeSIGWINCHRecovery()
	}
}

// altBufferSettleWindow is the duration after an alt-buffer enter or
// exit during which SIGWINCH recovery is still refused. Heavy TUIs
// (Claude Code, vim) briefly flicker out of the alt screen between
// render cycles; a SIGWINCH landing in one of those windows makes the
// TUI redraw its status row to the pane's NORMAL buffer, which tmux
// captures into scrollback.
const altBufferSettleWindow = 5 * time.Second

// maybeSIGWINCHRecovery fires SIGWINCH after a coalesce trim so
// well-behaved shells redraw, recovering from the trim. Suppressed when
// the foreground process is — or has recently been — in the alt
// buffer, on the persistent (tmux) backend, or within the cooldown
// window. Must be called with s.mu held.
func (s *Session) maybeSIGWINCHRecovery() {
	if s.Backend == backend.Persistent {
		return
	}
	if s.inAltBuffer {
		return
	}
	now := time.Now()
	if !s.lastAltBufferTransition.IsZero() && now.Sub(s.lastAltBufferTransition) < altBufferSettleWindow {
		return
	}
	if s.sigwinchCooldown > 0 && now.Sub(s.lastSIGWINCHRecovery) < s.sigwinchCooldown {
		return
	}
	s.lastSIGWINCHRecovery = now
	_ = s.pty.SetSize(s.Cols, s.Rows)
}
