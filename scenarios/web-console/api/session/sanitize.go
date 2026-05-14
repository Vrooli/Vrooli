// sanitize.go: client-bound stream sanitizer for DEC mode 2026.
//
// xterm.js v6.0.0 crashes parsing `\x1b[?2026$p` (DECRQM for synchronized
// output): the requestMode handler throws `ReferenceError: r is not
// defined` and silently kills the parser for that terminal. Every byte
// after the offending sequence is then dropped and the tab looks blank.
//
// The xterm-side DECSET/DECRST pair (`\x1b[?2026h` / `\x1b[?2026l`)
// renders correctly on desktop but is unreliable under mobile-browser
// timer throttling: if a sync-mode block straddles an idle period, the
// 1s fallback flush timer can miss and the framed output stays invisible
// until the next DECRST. Stripping the pair makes rendering progressive,
// the documented fallback behaviour for terminals that don't advertise
// 2026 support.
//
// The matching ANSI responder in ansi_responder.go answers DECRQM 2026
// with "mode not recognized" so well-behaved TUIs skip emitting the
// DECSET/DECRST toggles entirely. The strip path here is defence-in-depth
// for TUIs that emit the toggles unconditionally.

package session

import "bytes"

var (
	syncModeToken         = []byte("\x1b[?2026")
	syncModeStripPatterns = [][]byte{
		[]byte("\x1b[?2026h"),
		[]byte("\x1b[?2026l"),
		[]byte("\x1b[?2026$p"),
	}
)

// sanitizeForClient returns data with DEC mode 2026 sequences removed.
// Hot-path: the input slice is returned unchanged when the sync-mode
// prefix is absent (the common case for plain bash output and most TUIs).
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
