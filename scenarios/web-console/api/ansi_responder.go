package main

import (
	"bytes"
)

// ansiQueryResponder detects terminal capability queries in PTY output and
// generates the responses that TUI programs (Claude Code, vim, etc.) expect.
//
// Background: the browser-side xterm.js emulator does not reliably emit
// responses to these queries for arbitrary data it receives from the PTY.
// Remote TUIs that probe the terminal at startup (Claude Code sends
// DA1/DA3/DECRQM 2026 during init) then hang on their internal ~10 s
// response timeout before proceeding — or stall permanently. The tmux
// backend dodges this by having tmux itself answer the queries on behalf
// of the pane, so the standard backend needs the same service.
//
// This responder scans PTY output as the readLoop drains it, matches known
// query sequences, and returns the reply bytes so the caller can write
// them back to the PTY master — i.e., inject them as terminal input to the
// foreground process.
//
// Supported queries (all others pass through untouched):
//
//	\x1b[c         DA1 (primary device attributes)
//	  → \x1b[?1;2c                (VT100 with advanced video options)
//	\x1b[>0q       DA3 (tertiary device attributes)
//	  → \x1bP!|00000000\x1b\\     (empty report ID, DCS wrapped)
//	\x1b[?2026$p   DECRQM for mode 2026 (synchronized output)
//	  → \x1b[?2026;2$y            (reset / not set — xterm.js has no server-
//	                               side sync mode; reporting "set" would
//	                               lie and "unknown" causes some TUIs to
//	                               retry indefinitely)
//
// The responder is stateless; it operates on each PTY read chunk
// independently. Queries that straddle two chunks are not detected — in
// practice every query we care about is atomic in a single PTY read from
// a well-behaved TUI, so this is acceptable.
//
// DOC: docs/internal/SEAMS.md#ansi-query-responder-seam-standard-backend
func generateAnsiResponses(chunk []byte) []byte {
	if !bytes.Contains(chunk, []byte{0x1b}) {
		return nil
	}
	var out []byte
	for _, q := range ansiQueryTable {
		idx := 0
		for {
			found := bytes.Index(chunk[idx:], q.query)
			if found < 0 {
				break
			}
			out = append(out, q.response...)
			idx += found + len(q.query)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ansiQueryTable is consulted in order. Entries must be literal byte
// sequences — no CSI parameter variation — because real TUIs use exactly
// these forms and we want fast matching without a full ANSI parser.
var ansiQueryTable = []struct {
	name     string
	query    []byte
	response []byte
}{
	// DA1 — bare \x1b[c
	{"da1", []byte("\x1b[c"), []byte("\x1b[?1;2c")},
	// DA1 with explicit 0 parameter
	{"da1-zero", []byte("\x1b[0c"), []byte("\x1b[?1;2c")},
	// DA3
	{"da3", []byte("\x1b[>0q"), []byte("\x1bP!|00000000\x1b\\")},
	// DECRQM for synchronized output (mode 2026). We report "not
	// recognized" (value 0) so well-behaved TUIs skip emitting the
	// DECSET/DECRST toggles at all. The query itself is also stripped
	// from the client-bound stream by sanitizeForClient because xterm.js
	// v6.0.0 crashes parsing it — see the detailed rationale there.
	{"decrqm-2026", []byte("\x1b[?2026$p"), []byte("\x1b[?2026;0$y")},
}

// sanitizeForClient strips escape sequences related to DEC mode 2026
// (synchronized output) from PTY data before it's committed to history
// or sent to WS clients. The sequences crash or stall the browser-side
// xterm.js emulator:
//
//   - `\x1b[?2026$p` — DECRQM query ("is sync mode supported?"). xterm.js
//     v6.0.0's `requestMode` handler throws `ReferenceError: r is not
//     defined` when parsing this, killing the parser mid-stream. Every
//     subsequent byte for the tab is then silently dropped and the
//     terminal appears completely blank / unresponsive. This was the
//     root cause of the "Claude Code standard-backend hang".
//   - `\x1b[?2026h` / `\x1b[?2026l` — DECSET/DECRST for sync mode.
//     xterm.js renders them correctly on desktop, but the 1 s fallback
//     flush timer is unreliable under mobile-browser timer throttling:
//     if a sync-mode block straddles an idle period the buffered frame
//     never flushes and the banner stays invisible until the next DECRST.
//     Removing them makes rendering progressive, which is the documented
//     fallback behaviour for terminals that don't advertise 2026 support.
//
// The ANSI query responder (generateAnsiResponses) answers the DECRQM
// query on the server side so the TUI gets a "mode 2026 not recognized"
// reply without xterm.js ever being asked. Regression-covered by both
// the Go integration tests in `ansi_responder_integration_test.go` and
// `claude_startup_integration_test.go`, and the browser-level Playwright
// test in `ui/tests/e2e/claude-standard-backend.mjs`.
//
// Fast path: returns the original slice with zero allocation when the
// data contains no `\x1b[?2026` token at all (the common case for plain
// bash output and most TUIs).
func sanitizeForClient(data []byte) []byte {
	if !bytes.Contains(data, syncModeToken) {
		return data
	}
	out := data
	for _, pat := range syncModeStripPatterns {
		out = removeAll(out, pat)
	}
	return out
}

var (
	syncModeToken         = []byte("\x1b[?2026")
	syncModeStripPatterns = [][]byte{
		[]byte("\x1b[?2026h"),   // DECSET sync-mode begin
		[]byte("\x1b[?2026l"),   // DECRST sync-mode end
		[]byte("\x1b[?2026$p"),  // DECRQM sync-mode query (xterm.js v6 crashes on this)
	}
)

// removeAll returns src with all occurrences of needle elided. Allocates
// only when needle is present.
func removeAll(src, needle []byte) []byte {
	if len(needle) == 0 {
		return src
	}
	idx := bytes.Index(src, needle)
	if idx < 0 {
		return src
	}
	dst := make([]byte, 0, len(src))
	for idx >= 0 {
		dst = append(dst, src[:idx]...)
		src = src[idx+len(needle):]
		idx = bytes.Index(src, needle)
	}
	dst = append(dst, src...)
	return dst
}
