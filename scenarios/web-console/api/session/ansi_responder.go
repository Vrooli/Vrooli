// ansi_responder.go: server-side replies to terminal capability queries.
//
// TUI programs (Claude Code, vim, modern shells) probe the terminal at
// startup with DA1/DA3/XTVERSION/DECRQM 2026 queries. The browser-side
// xterm.js emulator does not reliably answer these, so the program then
// stalls on its internal response timeout (~10s for Claude Code) or
// hangs permanently. The tmux backend dodges this because tmux answers
// queries for its own panes; the standard backend needs this responder.
//
// Pre-Phase-3 design: a stateless byte-scanner ran inline in readLoop,
// matched literal query bytes, and wrote reply bytes back to the PTY.
//
// Phase 3 design: the emulator's parser emits a parsed ControlEvent for
// every recognized CSI query (see terminal/events.go). A long-lived
// goroutine here consumes that channel, builds the right reply, and
// delivers it through the canonical SendInput path with InputSource
// "ansi-responder". The byte-scanner is gone. The seam is the
// ControlEvent channel, not a global function pointer.
//
// DOC: docs/internal/SEAMS.md#ansi-strip-seam-api

package session

import (
	"log"

	"web-console/internal/backend"
	"web-console/internal/pty"
	"web-console/terminal"
)

// ansiResponderSource tags SendInput calls so debug logs and metrics can
// distinguish server-origin replies from client-origin keystrokes.
const ansiResponderSource = "ansi-responder"

// startAnsiResponder wires the server-side query responder to the
// session's emulator. It returns immediately; a single background
// goroutine consumes ControlEvents until the session exits.
//
// Persistent (tmux) sessions skip this — tmux answers terminal queries
// for its own panes, and a server-side reply would race with tmux's.
//
// Must be called BEFORE the emulator starts receiving bytes (i.e.
// before readLoop is spawned) so the lazily-allocated ControlEvent
// channel is in place when the first query arrives.
func (s *Session) startAnsiResponder() {
	if s.Backend == backend.Persistent {
		return
	}
	events := s.emu.ControlEvents()
	go s.runAnsiResponder(events)
}

// runAnsiResponder is the responder goroutine body. Exits when the
// session's exitCh is closed.
func (s *Session) runAnsiResponder(events <-chan terminal.ControlEvent) {
	for {
		select {
		case <-s.exitCh:
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			reply := ansiReplyFor(ev)
			if len(reply) == 0 {
				continue
			}
			if _, err := s.EnqueueInput(reply, pty.KindControl); err != nil {
				log.Printf("session %s: ansi-responder enqueue failed: %v", s.ID, err)
			}
		}
	}
}

// ansiReplyFor returns the reply bytes for a parsed control event, or
// nil if the event needs no server-side answer.
//
// Reply choices (must match the legacy literal-byte responder so client
// behaviour is byte-for-byte identical):
//
//	DA1     →  \x1b[?1;2c                  (VT100 with advanced video)
//	DA3     →  \x1bP!|00000000\x1b\\       (empty report ID, DCS-wrapped)
//	XTVER   →  \x1bP!|00000000\x1b\\       (same as DA3)
//	DECRQM  →  no reply; xterm.js owns synchronized-output capability state.
func ansiReplyFor(ev terminal.ControlEvent) []byte {
	if ev.Kind != terminal.EventCSIQuery {
		return nil
	}
	switch ev.Final {
	case 'c':
		if ev.Private {
			// CSI = c — classic DA3. The '=' is the private flag.
			return []byte("\x1bP!|00000000\x1b\\")
		}
		// CSI c or CSI 0 c — DA1.
		if len(ev.Params) == 0 || (len(ev.Params) == 1 && ev.Params[0] == 0) {
			return []byte("\x1b[?1;2c")
		}
		return nil
	case 'q':
		// CSI > 0 q — XTVERSION. We share the empty-report reply with
		// the legacy DA3 path because the previous responder did.
		if ev.Private && len(ev.Params) == 1 && ev.Params[0] == 0 {
			return []byte("\x1bP!|00000000\x1b\\")
		}
		return nil
	}
	return nil
}
