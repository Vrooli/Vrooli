// Package runner provides runner adapter implementations.
//
// This file provides shared text sanitization utilities used by all runners
// to strip ANSI escape sequences from CLI subprocess output before emitting
// events. This prevents raw terminal formatting from polluting the event
// stream, database, and UI.
package runner

import "strings"

// stripANSI removes ANSI escape sequences from a string.
// Handles CSI sequences (ESC[...<letter>), OSC sequences (ESC]...ST),
// and simple two-byte escapes (ESC<letter>).
func stripANSI(s string) string {
	// Fast path: no escape characters at all
	if !strings.Contains(s, "\x1b") {
		return s
	}

	result := strings.Builder{}
	result.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\x1b' {
			result.WriteByte(s[i])
			continue
		}
		// ESC found — determine the sequence type
		if i+1 >= len(s) {
			continue // trailing ESC, skip
		}
		switch s[i+1] {
		case '[': // CSI: ESC[ <params> <letter>
			i += 2 // skip ESC and [
			for i < len(s) {
				c := s[i]
				if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
					break // terminating letter
				}
				i++
			}
		case ']': // OSC: ESC] ... (ST = ESC\ or BEL)
			i += 2
			for i < len(s) {
				if s[i] == '\x07' { // BEL terminator
					break
				}
				if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '\\' {
					i++ // skip the backslash of ST
					break
				}
				i++
			}
		default:
			i++ // simple two-byte escape (ESC + letter), skip both
		}
	}
	return result.String()
}

// isOnlyANSI returns true when s consists entirely of ANSI escape sequences
// and whitespace — i.e., stripping ANSI leaves nothing meaningful.
// Use this to skip emitting events for pure terminal-formatting lines.
func isOnlyANSI(s string) bool {
	if !strings.Contains(s, "\x1b") {
		return false
	}
	return strings.TrimSpace(stripANSI(s)) == ""
}
