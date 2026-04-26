// emulator.go: The Emulator owns the authoritative decoded state of one
// PTY's output. It glues the parser to a Screen + Scrollback, tracks the
// alt-buffer flag, and exposes Snapshot() for replay-on-reconnect.
//
// Concurrency: Emulator is NOT safe for concurrent use. The caller (the
// owning Session) holds an existing mutex around Feed/Snapshot/Resize.

package terminal

// Options configures a new Emulator.
type Options struct {
	Cols, Rows      int
	ScrollbackLines int // soft cap on retained normal-buffer lines; default 10_000
}

// Default constants surface in config docs and tests.
const (
	DefaultCols            = 80
	DefaultRows            = 24
	DefaultScrollbackLines = 10_000
	MinScrollbackLines     = 100
	MaxScrollbackLines     = 100_000
)

// Emulator is the package's central type. Construct via New.
type Emulator struct {
	cols, rows int

	normal *Screen
	alt    *Screen
	cur    *Screen // points at one of normal/alt

	scrollback *Scrollback
	inAlt      bool

	parser *parser
}

// New returns an Emulator with the provided options. Zero values fall
// back to package defaults.
func New(opts Options) *Emulator {
	if opts.Cols < 1 {
		opts.Cols = DefaultCols
	}
	if opts.Rows < 1 {
		opts.Rows = DefaultRows
	}
	if opts.ScrollbackLines < MinScrollbackLines {
		opts.ScrollbackLines = DefaultScrollbackLines
	}
	if opts.ScrollbackLines > MaxScrollbackLines {
		opts.ScrollbackLines = MaxScrollbackLines
	}
	e := &Emulator{
		cols:       opts.Cols,
		rows:       opts.Rows,
		normal:     newScreen(opts.Cols, opts.Rows),
		alt:        newScreen(opts.Cols, opts.Rows),
		scrollback: newScrollback(opts.ScrollbackLines),
	}
	e.cur = e.normal
	e.parser = newParser(e)
	return e
}

// Feed consumes the byte stream and updates state. It is total: it never
// errors and always consumes every byte, so the io.Writer signature
// always returns (len(p), nil). Errors only exist on the signature for
// callers that want to chain through io.MultiWriter.
func (e *Emulator) Feed(p []byte) (int, error) {
	e.parser.feed(p)
	return len(p), nil
}

// Resize changes screen dimensions for both buffers. Scrollback is
// reflowed (truncated to the new column count).
func (e *Emulator) Resize(cols, rows int) {
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	if cols == e.cols && rows == e.rows {
		return
	}
	e.normal.resize(cols, rows)
	e.alt.resize(cols, rows)
	e.scrollback.reflow(cols)
	e.cols, e.rows = cols, rows
}

// InAltBuffer reports whether the emulator is currently rendering into
// the alternate screen.
func (e *Emulator) InAltBuffer() bool { return e.inAlt }

// ScrollbackLineCount returns the number of normal-buffer lines retained
// in scrollback. Useful for diagnostics + the regression test.
func (e *Emulator) ScrollbackLineCount() int { return e.scrollback.Len() }

// Cols / Rows expose current grid dimensions.
func (e *Emulator) Cols() int { return e.cols }
func (e *Emulator) Rows() int { return e.rows }

// --- handler implementation ---

func (e *Emulator) onRune(r rune) {
	if e.cur.X >= e.cur.Cols {
		// Wrap.
		e.lineFeed()
		e.cur.X = 0
	}
	e.cur.putRune(r)
}

func (e *Emulator) onC0(b byte) {
	switch b {
	case 0x07: // BEL — ignore
	case 0x08: // BS
		if e.cur.X > 0 {
			e.cur.X--
		}
	case 0x09: // HT — advance to next tab stop (8-col)
		next := (e.cur.X/8 + 1) * 8
		if next >= e.cur.Cols {
			next = e.cur.Cols - 1
		}
		e.cur.X = next
	case 0x0a: // LF
		e.lineFeed()
	case 0x0d: // CR
		e.cur.X = 0
	}
}

// lineFeed advances Y by one. Behavior depends on whether the cursor is
// at the bottom of the active scroll region:
//   - Below the region's bottom: cursor advances unconditionally (no
//     scroll).
//   - Above the region's bottom: cursor advances unconditionally.
//   - At the region's bottom: scroll the region up by 1; cursor stays.
//
// Lines scrolled out of the region are captured into scrollback only
// when the region covers the whole screen AND we're in the normal
// buffer. Region-bounded scrolls (TUIs reserving a status/input row)
// don't pollute scrollback — that matches xterm semantics and is what
// Claude Code, vim, etc. depend on.
func (e *Emulator) lineFeed() {
	top, bottom := e.cur.regionBounds()
	if e.cur.Y == bottom {
		captured := e.cur.scrollUp(1)
		fullScreenRegion := top == 0 && bottom == e.cur.Rows-1
		if !e.inAlt && fullScreenRegion {
			for _, l := range captured {
				e.scrollback.push(l)
			}
		}
		return
	}
	if e.cur.Y < e.cur.Rows-1 {
		e.cur.Y++
	}
}

// reverseIndex (RI) moves the cursor up one line. If at the top of the
// scroll region, scroll the region down by 1 instead.
func (e *Emulator) reverseIndex() {
	top, _ := e.cur.regionBounds()
	if e.cur.Y == top {
		e.cur.scrollDown(1)
		return
	}
	if e.cur.Y > 0 {
		e.cur.Y--
	}
}

func (e *Emulator) onESC(final byte) {
	switch final {
	case 'c': // RIS — full reset
		e.fullReset()
	case '7':
		e.cur.saveCursor()
	case '8':
		e.cur.restoreCursor()
	case 'D': // IND — index
		e.lineFeed()
	case 'E': // NEL — newline
		e.cur.X = 0
		e.lineFeed()
	case 'M': // RI — reverse index
		e.reverseIndex()
	}
}

func (e *Emulator) fullReset() {
	e.normal = newScreen(e.cols, e.rows)
	e.alt = newScreen(e.cols, e.rows)
	e.cur = e.normal
	e.inAlt = false
	e.scrollback = newScrollback(e.scrollback.cap)
}

func (e *Emulator) onCSI(private bool, params []int, final byte) {
	getParam := func(i, def int) int {
		if i < len(params) && params[i] > 0 {
			return params[i]
		}
		return def
	}
	if private {
		switch final {
		case 'h':
			for _, a := range params {
				e.setMode(a, true)
			}
		case 'l':
			for _, a := range params {
				e.setMode(a, false)
			}
		}
		return
	}
	switch final {
	case '@': // ICH — insert blanks
		// minimal: no-op shift, just consume
	case 'A':
		e.cur.Y -= getParam(0, 1)
		if e.cur.Y < 0 {
			e.cur.Y = 0
		}
	case 'B', 'e':
		e.cur.Y += getParam(0, 1)
		if e.cur.Y >= e.cur.Rows {
			e.cur.Y = e.cur.Rows - 1
		}
	case 'C', 'a':
		e.cur.X += getParam(0, 1)
		if e.cur.X >= e.cur.Cols {
			e.cur.X = e.cur.Cols - 1
		}
	case 'D':
		e.cur.X -= getParam(0, 1)
		if e.cur.X < 0 {
			e.cur.X = 0
		}
	case 'E':
		e.cur.X = 0
		e.cur.Y += getParam(0, 1)
		if e.cur.Y >= e.cur.Rows {
			e.cur.Y = e.cur.Rows - 1
		}
	case 'F':
		e.cur.X = 0
		e.cur.Y -= getParam(0, 1)
		if e.cur.Y < 0 {
			e.cur.Y = 0
		}
	case 'G', '`':
		e.cur.X = getParam(0, 1) - 1
		if e.cur.X < 0 {
			e.cur.X = 0
		}
		if e.cur.X >= e.cur.Cols {
			e.cur.X = e.cur.Cols - 1
		}
	case 'H', 'f': // CUP
		row := getParam(0, 1) - 1
		col := getParam(1, 1) - 1
		if row < 0 {
			row = 0
		}
		if col < 0 {
			col = 0
		}
		if row >= e.cur.Rows {
			row = e.cur.Rows - 1
		}
		if col >= e.cur.Cols {
			col = e.cur.Cols - 1
		}
		e.cur.Y, e.cur.X = row, col
	case 'J': // ED
		mode := 0
		if len(params) > 0 {
			mode = params[0]
		}
		switch mode {
		case 0:
			e.cur.clearLineFrom(e.cur.Y, e.cur.X)
			for y := e.cur.Y + 1; y < e.cur.Rows; y++ {
				e.cur.clearLine(y)
			}
		case 1:
			for y := 0; y < e.cur.Y; y++ {
				e.cur.clearLine(y)
			}
			e.cur.clearLineTo(e.cur.Y, e.cur.X)
		case 2, 3:
			e.cur.clearAll()
		}
	case 'K': // EL
		mode := 0
		if len(params) > 0 {
			mode = params[0]
		}
		switch mode {
		case 0:
			e.cur.clearLineFrom(e.cur.Y, e.cur.X)
		case 1:
			e.cur.clearLineTo(e.cur.Y, e.cur.X)
		case 2:
			e.cur.clearLine(e.cur.Y)
		}
	case 'd': // VPA
		e.cur.Y = getParam(0, 1) - 1
		if e.cur.Y < 0 {
			e.cur.Y = 0
		}
		if e.cur.Y >= e.cur.Rows {
			e.cur.Y = e.cur.Rows - 1
		}
	case 'm': // SGR
		e.applySGR(params)
	case 's':
		e.cur.saveCursor()
	case 'u':
		e.cur.restoreCursor()
	case 'S': // SU — scroll up
		n := getParam(0, 1)
		top, bottom := e.cur.regionBounds()
		captured := e.cur.scrollUp(n)
		fullScreenRegion := top == 0 && bottom == e.cur.Rows-1
		if !e.inAlt && fullScreenRegion {
			for _, l := range captured {
				e.scrollback.push(l)
			}
		}
	case 'T': // SD — scroll down
		e.cur.scrollDown(getParam(0, 1))
	case 'X': // ECH — erase characters in place (cursor unchanged)
		e.cur.eraseChars(getParam(0, 1))
	case 'r': // DECSTBM — set top/bottom margins (scroll region)
		top := getParam(0, 1) - 1
		bottom := getParam(1, e.cur.Rows) - 1
		// Per VT spec: must have at least 2 rows; otherwise reset.
		if top < 0 {
			top = 0
		}
		if bottom >= e.cur.Rows {
			bottom = e.cur.Rows - 1
		}
		if top >= bottom {
			top = 0
			bottom = e.cur.Rows - 1
		}
		e.cur.ScrollTop = top
		e.cur.ScrollBottom = bottom
		// DECSTBM also homes the cursor.
		e.cur.X = 0
		e.cur.Y = 0
	}
}

// setMode dispatches DEC private modes we care about. Unknown modes are
// silently ignored.
func (e *Emulator) setMode(mode int, set bool) {
	switch mode {
	case 47, 1047, 1049:
		e.swapBuffer(set, mode)
	}
}

func (e *Emulator) swapBuffer(toAlt bool, mode int) {
	if toAlt {
		if e.inAlt {
			return
		}
		// Mode 1049 also clears the alt buffer and saves the cursor.
		if mode == 1049 {
			e.normal.saveCursor()
			e.alt = newScreen(e.cols, e.rows)
		}
		e.inAlt = true
		e.cur = e.alt
		return
	}
	if !e.inAlt {
		return
	}
	if mode == 1049 {
		e.alt = newScreen(e.cols, e.rows)
		e.cur = e.normal
		e.inAlt = false
		e.normal.restoreCursor()
		return
	}
	if mode == 1047 {
		e.alt = newScreen(e.cols, e.rows)
	}
	e.cur = e.normal
	e.inAlt = false
}

// applySGR walks an SGR parameter list and updates the current pen.
// Supports the common subset: reset, bold/italic/underline/inverse/faint
// on/off, FG/BG ANSI 16, FG/BG default, 256-color (38;5;n / 48;5;n), and
// 24-bit (38;2;r;g;b / 48;2;r;g;b).
func (e *Emulator) applySGR(params []int) {
	if len(params) == 0 {
		e.cur.Pen = SGR{}
		return
	}
	for i := 0; i < len(params); i++ {
		p := params[i]
		switch {
		case p == 0:
			e.cur.Pen = SGR{}
		case p == 1:
			e.cur.Pen.Bold = true
		case p == 2:
			e.cur.Pen.Faint = true
		case p == 3:
			e.cur.Pen.Italic = true
		case p == 4:
			e.cur.Pen.Underline = true
		case p == 7:
			e.cur.Pen.Inverse = true
		case p == 22:
			e.cur.Pen.Bold = false
			e.cur.Pen.Faint = false
		case p == 23:
			e.cur.Pen.Italic = false
		case p == 24:
			e.cur.Pen.Underline = false
		case p == 27:
			e.cur.Pen.Inverse = false
		case p >= 30 && p <= 37:
			e.cur.Pen.FG = uint32(p - 30 + 1) // 1..8
		case p == 39:
			e.cur.Pen.FG = colorDefault
		case p >= 40 && p <= 47:
			e.cur.Pen.BG = uint32(p - 40 + 1)
		case p == 49:
			e.cur.Pen.BG = colorDefault
		case p >= 90 && p <= 97:
			e.cur.Pen.FG = uint32(p - 90 + 9) // 9..16
		case p >= 100 && p <= 107:
			e.cur.Pen.BG = uint32(p - 100 + 9)
		case p == 38 || p == 48:
			isFG := p == 38
			if i+1 >= len(params) {
				return
			}
			kind := params[i+1]
			if kind == 5 && i+2 < len(params) {
				idx := params[i+2] // 0..255
				color := uint32(idx) + 17
				if isFG {
					e.cur.Pen.FG = color
				} else {
					e.cur.Pen.BG = color
				}
				i += 2
			} else if kind == 2 && i+4 < len(params) {
				r, g, b := params[i+2]&0xff, params[i+3]&0xff, params[i+4]&0xff
				color := uint32(1<<24) | uint32(r<<16|g<<8|b)
				if isFG {
					e.cur.Pen.FG = color
				} else {
					e.cur.Pen.BG = color
				}
				i += 4
			}
		}
	}
}

// activeScreen returns the currently-rendered screen (normal or alt).
// Snapshot uses this to encode current visible state.
func (e *Emulator) activeScreen() *Screen { return e.cur }
