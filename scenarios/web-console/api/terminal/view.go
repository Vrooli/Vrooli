// view.go: Structured screen-read API.
//
// Snapshot() returns an ANSI byte stream meant for replay into another
// terminal emulator (e.g. xterm.js). View()/Cursor()/Cells() return the
// decoded grid as plain Go values so callers — Connect-RPC handlers, CLI
// commands, prompt detectors — can inspect the current screen without
// parsing ANSI.
//
// All readers return deep copies of the active screen state; callers may
// hold or mutate the result freely. The Emulator itself is not safe for
// concurrent use; the owning Session serializes access.

package terminal

// CursorPos is the (0,0)-origin cursor position on the active screen.
type CursorPos struct {
	X, Y int
}

// ScreenView is a deep copy of the active screen's visible grid plus the
// cursor and the alt-buffer flag. Suitable for serialization across a
// process boundary (Connect-RPC, JSON output, agent prompt context).
type ScreenView struct {
	Cols, Rows      int
	Cells           [][]Cell
	Cursor          CursorPos
	InAltBuffer     bool
	ScrollbackLines int
}

// Cursor returns the (column, row) cursor position of the active screen.
// Zero-based; (0,0) is the top-left.
func (e *Emulator) Cursor() CursorPos {
	return CursorPos{X: e.cur.X, Y: e.cur.Y}
}

// Cells returns a deep copy of the active screen's visible cells. The
// outer slice has Rows entries; each inner slice has Cols entries. The
// returned grid is independent of the emulator: callers may retain or
// mutate it without affecting subsequent emulator state.
//
// Honors the alt-buffer flag: when an alt-buffer-using TUI is active,
// the returned grid is the alt buffer, not the normal buffer.
func (e *Emulator) Cells() [][]Cell {
	rows := e.cur.Rows
	cols := e.cur.Cols
	out := make([][]Cell, rows)
	for y := 0; y < rows; y++ {
		row := make([]Cell, cols)
		if y < len(e.cur.Lines) {
			src := e.cur.Lines[y]
			n := cols
			if len(src) < n {
				n = len(src)
			}
			copy(row[:n], src[:n])
		}
		out[y] = row
	}
	return out
}

// View returns a single deep-copy snapshot of the active screen: cells,
// cursor, dimensions, alt-buffer flag, and scrollback line count.
func (e *Emulator) View() ScreenView {
	return ScreenView{
		Cols:            e.cur.Cols,
		Rows:            e.cur.Rows,
		Cells:           e.Cells(),
		Cursor:          e.Cursor(),
		InAltBuffer:     e.inAlt,
		ScrollbackLines: e.scrollback.Len(),
	}
}

// PlainText returns the active screen rendered as plain UTF-8 text,
// one row per line separated by "\n". Trailing blank cells on each row
// are stripped. SGR / ANSI escapes are NOT included — this is the read
// API for tools that want to assert on visible content (e.g. "did the
// shell finish running echo hello world").
//
// When includeScrollback is true, scrollback lines (oldest first) are
// prepended to the visible rows, separated by "\n".
func (e *Emulator) PlainText(includeScrollback bool) string {
	rowToString := func(row Line) string {
		end := len(row)
		for end > 0 && row[end-1].Rune == ' ' {
			end--
		}
		if end == 0 {
			return ""
		}
		runes := make([]rune, end)
		for i := 0; i < end; i++ {
			runes[i] = row[i].Rune
		}
		return string(runes)
	}

	var out []string
	if includeScrollback {
		for _, l := range e.scrollback.All() {
			out = append(out, rowToString(l))
		}
	}
	for _, l := range e.cur.Lines {
		out = append(out, rowToString(l))
	}
	// Join with \n, no trailing newline.
	if len(out) == 0 {
		return ""
	}
	total := len(out) - 1
	for _, s := range out {
		total += len(s)
	}
	buf := make([]byte, 0, total)
	for i, s := range out {
		if i > 0 {
			buf = append(buf, '\n')
		}
		buf = append(buf, s...)
	}
	return string(buf)
}
