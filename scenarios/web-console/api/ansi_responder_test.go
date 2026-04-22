package main

import (
	"bytes"
	"testing"
)

func TestAnsiResponder_DA1(t *testing.T) {
	got := generateAnsiResponses([]byte("\x1b[c"))
	want := []byte("\x1b[?1;2c")
	if !bytes.Equal(got, want) {
		t.Errorf("DA1 reply: got %q want %q", got, want)
	}
}

func TestAnsiResponder_DA1ExplicitZero(t *testing.T) {
	got := generateAnsiResponses([]byte("\x1b[0c"))
	want := []byte("\x1b[?1;2c")
	if !bytes.Equal(got, want) {
		t.Errorf("DA1 (0) reply: got %q want %q", got, want)
	}
}

func TestAnsiResponder_DA3(t *testing.T) {
	got := generateAnsiResponses([]byte("\x1b[>0q"))
	want := []byte("\x1bP!|00000000\x1b\\")
	if !bytes.Equal(got, want) {
		t.Errorf("DA3 reply: got %q want %q", got, want)
	}
}

func TestAnsiResponder_DECRQM2026(t *testing.T) {
	got := generateAnsiResponses([]byte("\x1b[?2026$p"))
	want := []byte("\x1b[?2026;0$y")
	if !bytes.Equal(got, want) {
		t.Errorf("DECRQM 2026 reply: got %q want %q", got, want)
	}
}

func TestAnsiResponder_ClaudeStartupSequence(t *testing.T) {
	// REGRESSION: Claude Code 2.1.x emits these three queries back-to-back
	// during init. All three must produce replies in one scan so the caller
	// can write a single response chunk to the PTY.
	chunk := []byte("\x1b[?25l\x1b[?2004h\x1b[?1004h\x1b[?2031h\x1b[>0q\x1b[?2026$p\x1b[c")
	got := generateAnsiResponses(chunk)
	if !bytes.Contains(got, []byte("\x1bP!|00000000\x1b\\")) {
		t.Errorf("missing DA3 reply in %q", got)
	}
	if !bytes.Contains(got, []byte("\x1b[?2026;0$y")) {
		t.Errorf("missing DECRQM 2026 reply in %q", got)
	}
	if !bytes.Contains(got, []byte("\x1b[?1;2c")) {
		t.Errorf("missing DA1 reply in %q", got)
	}
}

func TestAnsiResponder_NoEscape(t *testing.T) {
	if got := generateAnsiResponses([]byte("hello world")); got != nil {
		t.Errorf("plain text should yield no reply; got %q", got)
	}
}

func TestAnsiResponder_UnknownEscape(t *testing.T) {
	// Set-cursor-position is not a query; must not produce a reply.
	if got := generateAnsiResponses([]byte("\x1b[1;1H")); got != nil {
		t.Errorf("CUP should yield no reply; got %q", got)
	}
}

func TestAnsiResponder_MultipleSameQuery(t *testing.T) {
	// If a chunk (e.g. history replay) contains the same query twice,
	// reply once per occurrence so ordering stays correct for the TUI.
	got := generateAnsiResponses([]byte("\x1b[c\x1b[c"))
	want := []byte("\x1b[?1;2c\x1b[?1;2c")
	if !bytes.Equal(got, want) {
		t.Errorf("duplicate DA1: got %q want %q", got, want)
	}
}

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
	// REGRESSION: Claude Code sometimes emits multiple unbalanced
	// \x1b[?2026h without a closing \x1b[?2026l in the same render
	// chunk. All of them must be stripped, not just the first.
	input := []byte("\x1b[?2026h A \x1b[?2026h B \x1b[?2026h C")
	got := sanitizeForClient(input)
	want := []byte(" A  B  C")
	if !bytes.Equal(got, want) {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestSanitizeForClient_DoesNotEatSimilarSequences(t *testing.T) {
	// Only exact mode-2026 toggles are stripped; other DECSET/DECRST
	// sequences must survive.
	input := []byte("\x1b[?25h\x1b[?2004l\x1b[?1049h")
	got := sanitizeForClient(input)
	if !bytes.Equal(got, input) {
		t.Errorf("non-2026 toggles must be preserved; got %q want %q", got, input)
	}
}

func TestSanitizeForClient_StripsDecrqm2026Query(t *testing.T) {
	// REGRESSION: xterm.js v6.0.0 crashes with `ReferenceError: r is not
	// defined` inside its `requestMode` handler when parsing
	// `\x1b[?2026$p` (DECRQM for synchronized output). That throws during
	// the parser's action phase and kills every subsequent render for the
	// tab — so the whole Claude UI never appears, matching the user's
	// reported "session is completely unresponsive" symptom. We strip the
	// query so xterm.js never sees it. The server-side responder already
	// answers the query so the TUI doesn't lose its reply.
	input := []byte("before\x1b[?2026$pafter")
	got := sanitizeForClient(input)
	want := []byte("beforeafter")
	if !bytes.Equal(got, want) {
		t.Errorf("DECRQM 2026 query not stripped: got %q want %q", got, want)
	}
}

func TestSanitizeForClient_StripsClaudeStartupSequence(t *testing.T) {
	// REGRESSION: Claude Code 2.1.x emits DA3 + DECRQM 2026 + DA1 in a
	// single chunk, then a banner block wrapped in `\x1b[?2026h`. After
	// sanitize, only the DA3 and DA1 queries (which xterm.js handles
	// correctly) plus the banner payload should remain; the DECRQM
	// query AND the sync-mode wrapper must both be gone.
	input := []byte("\x1b[>0q\x1b[?2026$p\x1b[c\x1b[?2026h banner \x1b[?2026l")
	got := sanitizeForClient(input)
	want := []byte("\x1b[>0q\x1b[c banner ")
	if !bytes.Equal(got, want) {
		t.Errorf("Claude startup strip:\n  got  %q\n  want %q", got, want)
	}
}
