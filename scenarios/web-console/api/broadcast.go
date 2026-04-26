package main

// broadcast.go: Output fan-out, per-client delivery, coalesce-on-slow-client,
// pending-buffer trim, and SIGWINCH-based recovery when a trim happens.
//
// This file owns:
//   - ClientInfo         (per-subscriber flow-control state)
//   - (*Session).broadcast, .deliver, .notifyIfThreshold
//   - (*Session).FlushPending
//   - (*Session).maybeSIGWINCHRecovery
//
// The file intentionally names the fan-out concern front-and-center so
// that regressions to delivery semantics land here, not in the session
// lifecycle or history-store files.
//
// See docs/plans/terminal-session-refactor-implementation-plan.md §7.1
// for the target architecture and §10.4 for the greenfield assertion
// covering SetSize invocation gating.

import (
	"fmt"
	"time"
)

// ClientInfo tracks per-client broadcast flow control for a subscribed
// WebSocket connection. When the client's output channel is full, incoming
// frames are coalesced into a pending buffer instead of being dropped.
// The WebSocket output forwarder calls FlushPending after each successful
// write to drain coalesced data back into the channel.
type ClientInfo struct {
	pending         []byte    // coalesced data awaiting consumer drain
	pendingTrimmed  bool      // set when pending buffer was trimmed; triggers SIGWINCH after drain
	CoalescedFrames int       // count of coalesced frames (observability)
	NotifyCh        chan int  // receives cumulative coalesced count when threshold crossed
	StateCh         chan bool // receives alt-buffer state on each transition (true=enter, false=exit)
}

// broadcast fans out PTY output to all connected WebSocket clients while
// preserving bounded output history for reconnect/reload replay.
// Slow clients that can't keep up have frames coalesced into a pending buffer
// instead of being dropped. The WebSocket output forwarder calls FlushPending
// after each successful write to drain coalesced data back into the channel.
func (s *Session) broadcast(data []byte) {
	// Strip escape sequences the browser xterm.js emulator mishandles
	// (DECSET/DECRST/DECRQM for mode 2026 — see sanitizeForClient for the
	// full rationale) before the data hits the history buffer or any WS
	// client. Without this, a single Claude Code startup burst kills the
	// xterm parser and the entire terminal goes blank.
	data = sanitizeForClient(data)
	if len(data) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Update alt-buffer awareness before deciding whether to SIGWINCH-
	// recover after coalesce trims. Tracker runs on the sanitized stream;
	// the alt-buffer toggles (1049/1047/47) are outside the 2026 set that
	// sanitizeForClient strips. Each transition timestamps so the
	// SIGWINCH gate can refuse to fire during brief exit windows in
	// between TUI render cycles.
	altTransitioned := s.ptyState.Observe(data)
	if altTransitioned {
		s.lastAltBufferTransition = time.Now()
	}
	s.appendHistory(data)
	bctrace("broadcast", s.ID, fmt.Sprintf("clients=%d alt=%v total=%d", len(s.clients), s.ptyState.IsAltBuffer(), s.totalOutputBytes), data)
	if len(s.clients) == 0 {
		return
	}
	// Copy to avoid data races since buf is reused by readLoop.
	cp := make([]byte, len(data))
	copy(cp, data)
	newState := s.ptyState.IsAltBuffer()
	for ch, info := range s.clients {
		s.deliver(ch, info, cp)
		if altTransitioned {
			select {
			case info.StateCh <- newState:
			default:
				// Channel is full; forwarder is behind on state drains.
				// The forwarder drains with non-blocking reads, so a
				// dropped notification is recoverable: it will converge
				// on the current value on the next transition.
			}
		}
	}
}

// DOC: docs/internal/ERROR-SEMANTICS.md#sync-warning-coalescing-notification
// deliver sends data to a client channel, coalescing into the pending buffer
// when the channel is full. Must be called with s.mu held.
func (s *Session) deliver(ch chan []byte, info *ClientInfo, data []byte) {
	if len(info.pending) > 0 {
		// Already coalescing — append to pending buffer.
		info.pending = append(info.pending, data...)
		info.CoalescedFrames++
		// Cap pending at offlineBufferMax to prevent unbounded growth.
		// Snap to a clean ANSI boundary so partial escape sequences don't
		// corrupt the terminal when the coalesced data is flushed.
		// DOC: docs/concepts/ARCHITECTURE.md#terminal-io
		if s.offlineBufferMax > 0 && len(info.pending) > s.offlineBufferMax {
			trimmed := info.pending[len(info.pending)-s.offlineBufferMax:]
			trimmedClean := snapToCleanBoundary(trimmed)
			// Prepend SGR reset so trimmed color-setting sequences don't
			// bleed stale attributes into the client's terminal.
			info.pending = make([]byte, 0, len(sgrReset)+len(trimmedClean))
			info.pending = append(info.pending, sgrReset...)
			info.pending = append(info.pending, trimmedClean...)
			info.pendingTrimmed = true
		}
		s.notifyIfThreshold(info)
		return
	}
	select {
	case ch <- data:
		// Sent immediately.
	default:
		// Channel full — start coalescing instead of dropping.
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
			// Notification already pending.
		}
	}
}

// FlushPending drains any coalesced output for the given client channel.
// The WebSocket output forwarder calls this after each successful write
// to resume normal per-frame delivery. Data is chunked at historyChunkSize
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
		end := historyChunkSize
		if end > len(info.pending) {
			end = len(info.pending)
		}
		chunk := make([]byte, end)
		copy(chunk, info.pending[:end])
		select {
		case ch <- chunk:
			info.pending = info.pending[end:]
		default:
			return // Channel full mid-flush — keep remainder for next cycle
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
// exit during which SIGWINCH recovery is still refused. Claude Code
// (and other heavy TUIs) briefly flicker out of the alt screen between
// render cycles; a SIGWINCH landing in one of those windows makes the
// TUI redraw its status row to the pane's NORMAL buffer, which tmux
// captures into scrollback. Users then see the footer repeated
// wherever they scroll. Five seconds is long enough to cover typical
// TUI refresh cycles, short enough that the legitimate coalesce-trim
// recovery still fires in calm periods.
const altBufferSettleWindow = 5 * time.Second

// maybeSIGWINCHRecovery decides whether to fire a SIGWINCH to the PTY
// after a coalesce trim. The signal causes well-behaved shells and TUIs
// to redraw, recovering from the trim. But when the foreground process
// is in — or has recently been in — the alternate screen buffer
// (Claude Code, vim, tmux TUI, etc.), a mid-render SIGWINCH races the
// TUI's own redraw and interleaves paint output with the scrollback,
// visible to the user as duplicated status lines (see
// terminal-session-refactor-implementation-plan.md §4.5).
//
// This path is only safe when ALL of the following hold:
//  1. The session's backend is the standard (non-tmux) PTY. Persistent
//     tmux panes are managed by tmux itself — a SIGWINCH sent via our
//     SetSize just triggers a `tmux resize-window`, which has no
//     recovery value for coalesce trims and actively causes the
//     footer-duplication bug when Claude Code is the pane program.
//  2. We are NOT currently in an alternate screen buffer.
//  3. We have not observed ANY alt-buffer transition within
//     altBufferSettleWindow — this covers the brief exit between
//     render cycles of heavy TUIs.
//  4. It has been at least sigwinchCooldown since the last recovery.
//
// Must be called with s.mu held.
func (s *Session) maybeSIGWINCHRecovery() {
	if s.Backend == BackendPersistent {
		return
	}
	if s.ptyState.IsAltBuffer() {
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
