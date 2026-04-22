package main

// PTYStateTracker observes PTY output bytes and maintains a view of
// terminal modes that the server must know about for scheduling decisions.
// Today it tracks the alternate screen buffer (DEC private modes 47, 1047,
// and 1049). The tracker is a pure state machine: Observe returns true
// iff the alt-buffer flag flipped as a result of the current bytes.
//
// The tracker is resilient to escape sequences split across multiple
// reads — parser state persists between Observe calls.
type PTYStateTracker struct {
	altBuffer bool

	state   parserState
	private bool
	// params stores the raw parameter bytes of the current CSI sequence
	// (digits and ';' separators). Cleared on every final byte or reset.
	params []byte
}

type parserState uint8

const (
	psGround parserState = iota
	psEsc
	psCSI
)

// IsAltBuffer reports whether the PTY is currently in the alternate
// screen buffer, according to the sequences seen so far.
func (t *PTYStateTracker) IsAltBuffer() bool {
	return t.altBuffer
}

// Observe feeds bytes through the parser and returns true iff the
// IsAltBuffer value changed as a result of the call.
func (t *PTYStateTracker) Observe(data []byte) bool {
	before := t.altBuffer
	for _, b := range data {
		t.step(b)
	}
	return before != t.altBuffer
}

func (t *PTYStateTracker) step(b byte) {
	switch t.state {
	case psGround:
		if b == 0x1b {
			t.state = psEsc
		}
	case psEsc:
		if b == '[' {
			t.state = psCSI
			t.params = t.params[:0]
			t.private = false
			return
		}
		// Not a CSI — back to ground.
		t.state = psGround
	case psCSI:
		// Private marker (`?`) only valid before any param byte.
		if b == '?' && !t.private && len(t.params) == 0 {
			t.private = true
			return
		}
		// Parameter bytes: digits and ';'.
		if (b >= '0' && b <= '9') || b == ';' {
			t.params = append(t.params, b)
			// Cap param buffer size to prevent unbounded growth from a
			// stream of digits; real mode numbers fit comfortably.
			if len(t.params) > 64 {
				t.params = t.params[:0]
				t.private = false
				t.state = psGround
			}
			return
		}
		// Intermediate bytes (0x20–0x2F) are ignored between params and final.
		if b >= 0x20 && b <= 0x2F {
			return
		}
		// Final byte (0x40–0x7E) terminates the sequence.
		if b >= 0x40 && b <= 0x7E {
			if t.private && (b == 'h' || b == 'l') {
				if anyAltBufferMode(t.params) {
					t.altBuffer = (b == 'h')
				}
			}
			t.state = psGround
			t.params = t.params[:0]
			t.private = false
			return
		}
		// Anything else (control byte etc.) aborts the sequence.
		t.state = psGround
		t.params = t.params[:0]
		t.private = false
	}
}

// anyAltBufferMode returns true if any semicolon-separated parameter in
// `params` matches a known alt-buffer mode number (47, 1047, 1049).
// Mode 1048 (cursor save/restore alone) is deliberately not tracked here.
func anyAltBufferMode(params []byte) bool {
	if len(params) == 0 {
		return false
	}
	start := 0
	for i := 0; i <= len(params); i++ {
		if i == len(params) || params[i] == ';' {
			if i > start {
				n := parseSmallInt(params[start:i])
				if n == 47 || n == 1047 || n == 1049 {
					return true
				}
			}
			start = i + 1
		}
	}
	return false
}

// parseSmallInt parses a short digit run into an int. Returns -1 on empty
// or non-digit input. No overflow handling — callers pass small params.
func parseSmallInt(b []byte) int {
	if len(b) == 0 {
		return -1
	}
	n := 0
	for _, c := range b {
		if c < '0' || c > '9' {
			return -1
		}
		n = n*10 + int(c-'0')
		if n > 1_000_000 {
			return -1
		}
	}
	return n
}
