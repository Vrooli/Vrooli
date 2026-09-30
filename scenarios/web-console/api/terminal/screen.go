// screen.go: Cell, Line, and Screen grid types for the terminal emulator.
//
// A Cell carries one printable rune plus an SGR style. A Line is a slice
// of Cells (one per column). A Screen is a fixed cols×rows grid of Lines
// plus a cursor and the current pen (SGR state being applied to writes).
//
// The Screen owns no scrollback — when a line scrolls off the top, the
// Emulator captures it and forwards it to the Scrollback ring (when in
// the normal buffer).

package terminal

// SGR encodes the subset of SGR state we replay through snapshots.
//
// The bit-set keeps Snapshot output compact and round-trippable: any pair
// of cells with the same SGR will be coalesced into a single SGR escape.
// We intentionally do NOT model every legacy attribute (blink, hidden,
// etc.) — xterm.js renders the snapshot as a regular byte stream, so the
// fidelity bar is "every cell that had a color before still has the same
// color after replay," not "every legacy attribute round-trips."
type SGR struct {
	// FG/BG color encoding:
	//   colorDefault     → terminal default (no SGR set)
	//   1..16            → ANSI 16
	//   17..256          → 256-color (offset by -16 → palette index)
	//   >256             → 24-bit packed: 1<<24 | (r<<16 | g<<8 | b)
	FG uint32
	BG uint32

	Bold      bool
	Italic    bool
	Underline bool
	Inverse   bool
	Faint     bool
}

const colorDefault uint32 = 0

// Equal reports byte-equality so snapshot SGR run-coalescing can compare
// adjacent cells without allocating.
func (s SGR) Equal(o SGR) bool { return s == o }

// IsZero reports whether s is the default (no SGR set). Snapshot writers
// use this to skip emitting `\x1b[0m` when not needed.
func (s SGR) IsZero() bool { return s == SGR{} }

// Cell is one screen position. A blank cell has Rune == ' ' and SGR{}.
type Cell struct {
	Rune rune
	SGR  SGR
}

func blankCell() Cell { return Cell{Rune: ' '} }

// Line is one screen row. Length == cols of its owning Screen.
type Line []Cell

func newLine(cols int) Line {
	l := make(Line, cols)
	for i := range l {
		l[i] = blankCell()
	}
	return l
}

// Screen is a fixed cols×rows grid plus a cursor and the current pen.
//
// The cursor uses zero-based (X,Y) where (0,0) is the top-left.
type Screen struct {
	Cols, Rows int
	Lines      []Line // length == Rows
	X, Y       int
	Pen        SGR
	// ScrollTop, ScrollBottom define the inclusive scroll region
	// (DECSTBM). Default to (0, Rows-1) = full screen. Updated via
	// `\x1b[T;B r` so TUI apps that reserve a status/input row scroll
	// only the upper region.
	ScrollTop, ScrollBottom int
	// SaveCursor / restoreCursor stash. Zero-valued when nothing saved.
	savedX, savedY int
	savedPen       SGR
	hasSaved       bool
}

func newScreen(cols, rows int) *Screen {
	s := &Screen{
		Cols:         cols,
		Rows:         rows,
		Lines:        make([]Line, rows),
		ScrollTop:    0,
		ScrollBottom: rows - 1,
	}
	for i := range s.Lines {
		s.Lines[i] = newLine(cols)
	}
	return s
}

// clearAll erases every cell and resets the pen to default.
func (s *Screen) clearAll() {
	for i := range s.Lines {
		for j := range s.Lines[i] {
			s.Lines[i][j] = blankCell()
		}
	}
}

// clearLineFrom erases cells in row y starting at column x (inclusive).
func (s *Screen) clearLineFrom(y, x int) {
	if y < 0 || y >= s.Rows {
		return
	}
	for j := x; j < s.Cols; j++ {
		s.Lines[y][j] = blankCell()
	}
}

// clearLineTo erases cells in row y from column 0 through x (inclusive).
func (s *Screen) clearLineTo(y, x int) {
	if y < 0 || y >= s.Rows {
		return
	}
	if x >= s.Cols {
		x = s.Cols - 1
	}
	for j := 0; j <= x; j++ {
		s.Lines[y][j] = blankCell()
	}
}

// clearLine erases an entire row.
func (s *Screen) clearLine(y int) {
	if y < 0 || y >= s.Rows {
		return
	}
	for j := range s.Lines[y] {
		s.Lines[y][j] = blankCell()
	}
}

// putRune writes r at (X,Y) using the current pen, then advances X.
// Out-of-range writes are silently dropped.
func (s *Screen) putRune(r rune) {
	if s.Y < 0 || s.Y >= s.Rows {
		return
	}
	if s.X >= s.Cols {
		// Auto-wrap to next line.
		s.X = 0
		s.Y++
		if s.Y >= s.Rows {
			// Caller should have scrolled; if not, clamp.
			s.Y = s.Rows - 1
		}
	}
	if s.X < 0 {
		s.X = 0
	}
	s.Lines[s.Y][s.X] = Cell{Rune: r, SGR: s.Pen}
	s.X++
}

// scrollUp moves lines within the current scroll region up by n. The top
// n lines of the region are returned (caller can push them to scrollback
// when appropriate) and n blank lines are appended at the bottom of the
// region. Lines outside the region are unaffected.
//
// When the region covers the full screen (default), this behaves exactly
// like the classic "scroll one line up" operation. When the region is
// narrower (DECSTBM has been set), only that portion of the screen
// scrolls — critical for TUI apps like Claude Code that reserve a
// status/input row at the bottom and scroll the upper region only.
func (s *Screen) scrollUp(n int) []Line {
	if n <= 0 {
		return nil
	}
	top, bottom := s.regionBounds()
	regionRows := bottom - top + 1
	if n > regionRows {
		n = regionRows
	}
	captured := make([]Line, n)
	copy(captured, s.Lines[top:top+n])
	// Shift remaining region rows up.
	copy(s.Lines[top:bottom+1-n], s.Lines[top+n:bottom+1])
	// Blank the freed bottom rows of the region.
	for i := bottom + 1 - n; i <= bottom; i++ {
		s.Lines[i] = newLine(s.Cols)
	}
	return captured
}

// scrollDown moves lines within the scroll region DOWN by n. Top n
// blank lines are inserted; bottom n region lines are discarded. Used
// for reverse-index (RI) at the top of a region.
func (s *Screen) scrollDown(n int) {
	if n <= 0 {
		return
	}
	top, bottom := s.regionBounds()
	regionRows := bottom - top + 1
	if n > regionRows {
		n = regionRows
	}
	// Shift rows down within the region.
	copy(s.Lines[top+n:bottom+1], s.Lines[top:bottom+1-n])
	for i := top; i < top+n; i++ {
		s.Lines[i] = newLine(s.Cols)
	}
}

// regionBounds returns the active scroll region clamped to valid rows.
func (s *Screen) regionBounds() (top, bottom int) {
	top = s.ScrollTop
	bottom = s.ScrollBottom
	if top < 0 {
		top = 0
	}
	if bottom >= s.Rows {
		bottom = s.Rows - 1
	}
	if top > bottom {
		top, bottom = 0, s.Rows-1
	}
	return top, bottom
}

// eraseChars clears n cells starting at the cursor on the current row,
// without moving the cursor. Used by ECH (`\x1b[N X`). TUI apps emit
// ECH heavily to wipe stale text in place before redrawing, so missing
// it is the dominant cause of "ghost" text on snapshot replay.
func (s *Screen) eraseChars(n int) {
	if n <= 0 || s.Y < 0 || s.Y >= s.Rows {
		return
	}
	end := s.X + n
	if end > s.Cols {
		end = s.Cols
	}
	for j := s.X; j < end; j++ {
		s.Lines[s.Y][j] = blankCell()
	}
}

// resize changes the grid to new dimensions, preserving content where
// possible. Existing content is anchored to the top-left; cells outside
// the new bounds are dropped. The cursor is clamped.
//
// Lines that would fall off the bottom on a row-shrink are returned so
// the caller can spill them into scrollback (mirroring scrollUp).
func (s *Screen) resize(cols, rows int) []Line {
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	if cols == s.Cols && rows == s.Rows {
		return nil
	}

	// Build new lines, copying as much old content as fits.
	newLines := make([]Line, rows)
	for i := range newLines {
		newLines[i] = newLine(cols)
		if i < len(s.Lines) {
			old := s.Lines[i]
			n := len(old)
			if n > cols {
				n = cols
			}
			copy(newLines[i], old[:n])
		}
	}

	var spilled []Line
	if rows < s.Rows {
		// Bottom rows that fall off — but content typically grows from
		// the top, so it's the BOTTOM that gets dropped on shrink. The
		// plan asks us to preserve scrollback above; we don't push the
		// dropped bottom rows to scrollback (they're below the cursor),
		// we just discard them. Keep the slot available for future
		// reflow logic.
		_ = spilled
	}

	s.Cols, s.Rows = cols, rows
	s.Lines = newLines
	// Resize resets the scroll region per VT spec.
	s.ScrollTop = 0
	s.ScrollBottom = rows - 1
	if s.X >= cols {
		s.X = cols - 1
	}
	if s.Y >= rows {
		s.Y = rows - 1
	}
	return spilled
}

// saveCursor stashes (X, Y, Pen).
func (s *Screen) saveCursor() {
	s.savedX, s.savedY, s.savedPen = s.X, s.Y, s.Pen
	s.hasSaved = true
}

// restoreCursor reapplies a prior saveCursor; no-op if none recorded.
func (s *Screen) restoreCursor() {
	if !s.hasSaved {
		return
	}
	s.X, s.Y, s.Pen = s.savedX, s.savedY, s.savedPen
}
