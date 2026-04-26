// scrollback.go: bounded ring buffer of decoded Lines.
//
// The Emulator pushes a Line here every time content scrolls off the top
// of the *normal* buffer. Alt-buffer content never reaches this ring, by
// design — that's the whole point of the snapshot model: scrollback
// survives alt-buffer transitions.

package terminal

// Scrollback is an append-only bounded ring of decoded Lines.
//
// Capacity is in *lines*, not bytes. Default 10_000 (≈ 1–2 MB at typical
// terminal widths) is set by the Emulator constructor.
type Scrollback struct {
	cap   int
	lines []Line
}

// newScrollback returns a ring with the given line capacity. cap < 1 is
// treated as 1.
func newScrollback(cap int) *Scrollback {
	if cap < 1 {
		cap = 1
	}
	return &Scrollback{cap: cap}
}

// push appends a Line, trimming the oldest line(s) if over capacity.
// The Line is captured by reference; callers must not mutate it after
// pushing. (newLine in screen.go always returns a freshly-allocated
// slice, and scrollUp copies references into a new slice, so this is
// safe in our usage.)
func (sb *Scrollback) push(l Line) {
	sb.lines = append(sb.lines, l)
	if len(sb.lines) > sb.cap {
		// Trim the oldest excess in one shift; cheap because cap is
		// bounded and trimming happens at most once per push.
		drop := len(sb.lines) - sb.cap
		sb.lines = append(sb.lines[:0], sb.lines[drop:]...)
	}
}

// Len reports the number of stored lines.
func (sb *Scrollback) Len() int { return len(sb.lines) }

// All returns the lines in oldest-to-newest order. The caller must NOT
// mutate the returned slice or its Lines.
func (sb *Scrollback) All() []Line { return sb.lines }

// reflow rebuilds scrollback for a new column count. Lines longer than
// the new width are truncated (we do not re-wrap; that would require
// joining adjacent lines, and our simple model doesn't track soft
// wraps). Lines shorter than the new width are left as-is — the snapshot
// emitter pads with `\r\n` at line ends, so trailing blanks don't need
// to be materialized.
func (sb *Scrollback) reflow(cols int) {
	if cols < 1 {
		cols = 1
	}
	for i, l := range sb.lines {
		if len(l) <= cols {
			continue
		}
		nl := make(Line, cols)
		copy(nl, l[:cols])
		sb.lines[i] = nl
	}
}
