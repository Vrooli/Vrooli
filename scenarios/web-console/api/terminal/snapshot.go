// snapshot.go: ANSI snapshot serialization.
//
// Snapshot() emits a self-contained byte stream that, when fed back into
// a fresh Emulator (or written to xterm.js), reproduces an equivalent
// (screen, alt-buffer, scrollback) triple.
//
// Strategy:
//
// We DO NOT use cursor-positioning escapes (CUP) to paint the screen,
// because CUP overwrites without scrolling, and our scrollback ring is
// populated by natural scroll. Instead we stream scrollback + screen as
// a sequence of `<line>\r\n` writes; the receiver's emulator naturally
// scrolls older lines into its own scrollback and ends with the visible
// rows in place.
//
// Wire shape (no alt buffer active):
//
//   \x1bc                                     ; full reset
//   <sb[0]>\r\n <sb[1]>\r\n ... <sb[N-1]>\r\n ; scrollback, oldest first
//   <normal[0]>\r\n ... <normal[R-2]>\r\n     ; first R-1 visible rows
//   <normal[R-1]>                             ; last visible row, no \r\n
//   <CUP to normal cursor>                    ; positions cursor exactly
//   <SGR for current pen>                     ; so live writes inherit it
//
// Wire shape (alt buffer active):
//
//   <as above through normal[R-1]>            ; receiver's normal buffer
//   \x1b[?1049h                               ; switch to alt
//   <alt[0]>\r\n ... <alt[R-2]>\r\n
//   <alt[R-1]>
//   <CUP to alt cursor>
//   <SGR for current alt pen>

package terminal

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// SnapshotPrologue is the complete reset sequence shared by full and bounded
// replays. Keeping it in one constant prevents a truncated replay from
// leaving the receiver in an unknown alternate-buffer, scrollback, or SGR
// state.
const SnapshotPrologue = "\x1b[?1049l\x1bc\x1b[3J\x1b[H\x1b[0m"

// Snapshot returns a self-contained ANSI byte stream. Idempotent for a
// given emulator state.
//
// Prologue invariants (defensive, so the receiver lands in a known state
// regardless of what state it was in before):
//
//   - `\x1b[?1049l` exits alt-buffer if the receiver was somehow stuck
//     in it from a prior connection (xterm.js v6 has a known FIXME
//     where 1049 SET does not clear alt content; pairing reset+clear
//     guarantees a fresh canvas).
//   - `\x1bc` triggers RIS in xterm.js v6, which fully recreates both
//     buffers and wipes scrollback.
//   - `\x1b[3J` explicitly clears scrollback in any client where RIS
//     does not (older xterm derivatives).
//   - `\x1b[H\x1b[0m` parks the cursor at (0,0) with default SGR before
//     scrollback streaming so the first scrolled line lands cleanly.
func (e *Emulator) Snapshot() []byte {
	var buf bytes.Buffer
	buf.WriteString(SnapshotPrologue)

	// Scrollback (oldest → newest), each followed by CRLF.
	for _, line := range e.scrollback.All() {
		writeLineRaw(&buf, line)
		buf.WriteString("\r\n")
	}

	// Normal screen rows. All but the last get CRLF; last gets none so
	// the cursor doesn't trigger a final scroll on the receiver.
	streamRows(&buf, e.normal.Lines)

	if e.inAlt {
		// Park cursor at (0,0) BEFORE entering alt so xterm's
		// activateAltBuffer copies a known cursor position (0,0) into
		// alt — otherwise alt's cursor would inherit wherever the
		// last normal-screen row ended, which misaligns the first
		// alt row's content (it would print starting at that column
		// instead of column 0). This was a latent bug exposed by
		// sessions where the normal screen had non-empty content.
		writeCUP(&buf, 0, 0)
		buf.WriteString("\x1b[?1049h")
		writeCUP(&buf, 0, 0)
		streamRows(&buf, e.alt.Lines)
		writeCUP(&buf, e.alt.Y, e.alt.X)
		writeSGRReset(&buf)
		writeSGRApply(&buf, e.alt.Pen)
	} else {
		writeCUP(&buf, e.normal.Y, e.normal.X)
		writeSGRReset(&buf)
		writeSGRApply(&buf, e.normal.Pen)
	}
	return buf.Bytes()
}

// streamRows writes rows[0..len-1] as line+\r\n entries except the last,
// which has no trailing \r\n. Empty rows are still streamed (as "\r\n")
// so screen-row alignment is preserved on the receiver.
func streamRows(buf *bytes.Buffer, rows []Line) {
	if len(rows) == 0 {
		return
	}
	for i, line := range rows {
		writeLineRaw(buf, line)
		if i < len(rows)-1 {
			buf.WriteString("\r\n")
		}
	}
}

// writeLineRaw emits cells with SGR-coalesced runs and trailing blanks
// stripped. Always resets SGR at end of line.
func writeLineRaw(buf *bytes.Buffer, line Line) {
	end := len(line)
	for end > 0 {
		c := line[end-1]
		if c.Rune != ' ' || !c.SGR.IsZero() {
			break
		}
		end--
	}
	if end == 0 {
		return
	}
	cur := SGR{}
	for i := 0; i < end; i++ {
		c := line[i]
		if !c.SGR.Equal(cur) {
			writeSGRReset(buf)
			writeSGRApply(buf, c.SGR)
			cur = c.SGR
		}
		buf.WriteRune(c.Rune)
	}
	if !cur.IsZero() {
		writeSGRReset(buf)
	}
}

// writeCUP emits a CUP escape; inputs are zero-based.
func writeCUP(buf *bytes.Buffer, y, x int) {
	if y < 0 {
		y = 0
	}
	if x < 0 {
		x = 0
	}
	buf.WriteString("\x1b[")
	buf.WriteString(strconv.Itoa(y + 1))
	buf.WriteByte(';')
	buf.WriteString(strconv.Itoa(x + 1))
	buf.WriteByte('H')
}

// writeSGRReset writes \x1b[0m.
func writeSGRReset(buf *bytes.Buffer) { buf.WriteString("\x1b[0m") }

// writeSGRApply emits the parameter list to apply pen; no-op if pen is
// zero (caller already issued reset).
func writeSGRApply(buf *bytes.Buffer, pen SGR) {
	if pen.IsZero() {
		return
	}
	parts := make([]string, 0, 6)
	if pen.Bold {
		parts = append(parts, "1")
	}
	if pen.Faint {
		parts = append(parts, "2")
	}
	if pen.Italic {
		parts = append(parts, "3")
	}
	if pen.Underline {
		parts = append(parts, "4")
	}
	if pen.Inverse {
		parts = append(parts, "7")
	}
	if pen.FG != colorDefault {
		parts = append(parts, encodeColor(pen.FG, true))
	}
	if pen.BG != colorDefault {
		parts = append(parts, encodeColor(pen.BG, false))
	}
	if len(parts) == 0 {
		return
	}
	buf.WriteString("\x1b[")
	buf.WriteString(strings.Join(parts, ";"))
	buf.WriteByte('m')
}

// encodeColor returns the SGR sub-parameters for a color value. fg=true
// emits the 30/38/90 series; fg=false emits 40/48/100.
//
// Encoding (matches comment in screen.go):
//
//	1..8    → ANSI 8 (black..white)
//	9..16   → ANSI bright (brightblack..brightwhite)
//	17..272 → 256-color (palette index = c-17)
//	1<<24|… → 24-bit truecolor
func encodeColor(c uint32, fg bool) string {
	switch {
	case c == colorDefault:
		if fg {
			return "39"
		}
		return "49"
	case c >= 1 && c <= 8:
		base := 30
		if !fg {
			base = 40
		}
		return strconv.Itoa(base + int(c) - 1)
	case c >= 9 && c <= 16:
		base := 90
		if !fg {
			base = 100
		}
		return strconv.Itoa(base + int(c) - 9)
	case c >= 17 && c <= 17+255:
		base := 38
		if !fg {
			base = 48
		}
		return fmt.Sprintf("%d;5;%d", base, int(c)-17)
	case c&(1<<24) != 0:
		base := 38
		if !fg {
			base = 48
		}
		r := (c >> 16) & 0xff
		g := (c >> 8) & 0xff
		b := c & 0xff
		return fmt.Sprintf("%d;2;%d;%d;%d", base, r, g, b)
	}
	if fg {
		return "39"
	}
	return "49"
}
