package session

import (
	"bytes"
	"testing"
)

func TestSanitizeForClient_PassesThroughPlainData(t *testing.T) {
	input := []byte("hello\x1b[38;2;215;119;87m world\x1b[39m")
	got := sanitizeForClient(input)
	if !bytes.Equal(got, input) {
		t.Errorf("plain data should pass through unchanged; got %q want %q", got, input)
	}
}

func TestSanitizeForClient_StripsSyncModeSet(t *testing.T) {
	input := []byte("before\x1b[?2026hafter")
	got := sanitizeForClient(input)
	want := []byte("beforeafter")
	if !bytes.Equal(got, want) {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestSanitizeForClient_StripsSyncModeReset(t *testing.T) {
	input := []byte("before\x1b[?2026lafter")
	got := sanitizeForClient(input)
	want := []byte("beforeafter")
	if !bytes.Equal(got, want) {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestSanitizeForClient_StripsBoth(t *testing.T) {
	input := []byte("\x1b[?2026h\x1b[38m content \x1b[?2026l\x1b[39m")
	got := sanitizeForClient(input)
	want := []byte("\x1b[38m content \x1b[39m")
	if !bytes.Equal(got, want) {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestSanitizeForClient_StripsMultipleOccurrences(t *testing.T) {
	// REGRESSION: multiple unbalanced \x1b[?2026h in one chunk must all
	// be stripped, not just the first.
	input := []byte("\x1b[?2026h A \x1b[?2026h B \x1b[?2026h C")
	got := sanitizeForClient(input)
	want := []byte(" A  B  C")
	if !bytes.Equal(got, want) {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestSanitizeForClient_DoesNotEatSimilarSequences(t *testing.T) {
	input := []byte("\x1b[?25h\x1b[?2004l\x1b[?1049h")
	got := sanitizeForClient(input)
	if !bytes.Equal(got, input) {
		t.Errorf("non-2026 toggles must be preserved; got %q want %q", got, input)
	}
}

func TestSanitizeForClient_StripsDecrqm2026Query(t *testing.T) {
	// REGRESSION: xterm.js v6.0.0 crashes parsing `\x1b[?2026$p`; the
	// server strips it on the client-bound stream so xterm.js never sees
	// it. The matching reply is produced separately by the ANSI
	// responder so the TUI still gets its answer.
	input := []byte("before\x1b[?2026$pafter")
	got := sanitizeForClient(input)
	want := []byte("beforeafter")
	if !bytes.Equal(got, want) {
		t.Errorf("DECRQM 2026 query not stripped: got %q want %q", got, want)
	}
}

func TestSanitizeForClient_StripsClaudeStartupSequence(t *testing.T) {
	// Claude Code 2.1.x emits DA3 + DECRQM 2026 + DA1 in one chunk, then
	// a banner wrapped in 2026 sync-mode toggles. After sanitize, the
	// DA3/DA1 queries (handled by xterm.js) + banner payload remain; the
	// DECRQM query and sync-mode wrapper are gone.
	input := []byte("\x1b[>0q\x1b[?2026$p\x1b[c\x1b[?2026h banner \x1b[?2026l")
	got := sanitizeForClient(input)
	want := []byte("\x1b[>0q\x1b[c banner ")
	if !bytes.Equal(got, want) {
		t.Errorf("Claude startup strip:\n  got  %q\n  want %q", got, want)
	}
}
