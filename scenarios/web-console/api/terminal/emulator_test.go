// emulator_test.go: Unit tests for Emulator invariants.

package terminal

import (
	"bytes"
	"strings"
	"testing"
)

func feed(t *testing.T, e *Emulator, s string) {
	t.Helper()
	if _, err := e.Feed([]byte(s)); err != nil {
		t.Fatalf("Feed returned error (must be total): %v", err)
	}
}

func screenText(s *Screen) string {
	var lines []string
	for _, l := range s.Lines {
		end := len(l)
		for end > 0 && l[end-1].Rune == ' ' && l[end-1].SGR.IsZero() {
			end--
		}
		row := make([]rune, 0, end)
		for i := 0; i < end; i++ {
			row = append(row, l[i].Rune)
		}
		lines = append(lines, string(row))
	}
	return strings.Join(lines, "\n")
}

func TestPlainAsciiPopulatesScreen(t *testing.T) {
	e := New(Options{Cols: 20, Rows: 5})
	feed(t, e, "hello\r\nworld")
	got := screenText(e.activeScreen())
	want := "hello\nworld\n\n\n"
	if got != want {
		t.Fatalf("screen mismatch:\n got=%q\nwant=%q", got, want)
	}
}

func TestLFCausesScrollAppendsScrollback(t *testing.T) {
	e := New(Options{Cols: 10, Rows: 3, ScrollbackLines: 100})
	for i := 0; i < 6; i++ {
		feed(t, e, "line\r\n")
	}
	if got := e.ScrollbackLineCount(); got < 3 {
		t.Fatalf("expected scrollback to grow on overflow; got %d", got)
	}
}

func TestEraseDisplayKeepsScrollback(t *testing.T) {
	e := New(Options{Cols: 10, Rows: 3, ScrollbackLines: 100})
	feed(t, e, "a\r\nb\r\nc\r\nd\r\n") // scroll a/b out
	before := e.ScrollbackLineCount()
	feed(t, e, "\x1b[2J")
	if got := e.ScrollbackLineCount(); got != before {
		t.Fatalf("erase display must not touch scrollback: before=%d after=%d", before, got)
	}
}

func TestAltBufferEnterFreezesScrollback(t *testing.T) {
	e := New(Options{Cols: 10, Rows: 3, ScrollbackLines: 100})
	feed(t, e, "a\r\nb\r\nc\r\nd\r\n")
	prev := e.ScrollbackLineCount()
	feed(t, e, "\x1b[?1049h")
	if !e.InAltBuffer() {
		t.Fatalf("expected alt buffer active after \\x1b[?1049h")
	}
	feed(t, e, "alt-line-1\r\nalt-line-2\r\nalt-line-3\r\nalt-line-4\r\n")
	if got := e.ScrollbackLineCount(); got != prev {
		t.Fatalf("alt-buffer writes must not extend scrollback: before=%d after=%d", prev, got)
	}
	feed(t, e, "\x1b[?1049l")
	if e.InAltBuffer() {
		t.Fatalf("expected alt buffer inactive after \\x1b[?1049l")
	}
}

func TestUnmatchedAltBufferEnterPreservesScrollback(t *testing.T) {
	// This is the regression for the production bug: a captured stream
	// contains \x1b[?1049h with no matching \x1b[?1049l. After replay
	// through Snapshot + a fresh emulator, scrollback must still be
	// present.
	e := New(Options{Cols: 10, Rows: 3, ScrollbackLines: 100})
	feed(t, e, "history-1\r\nhistory-2\r\nhistory-3\r\nhistory-4\r\n")
	want := e.ScrollbackLineCount()
	if want == 0 {
		t.Fatalf("test setup: expected non-empty scrollback before alt-enter")
	}
	feed(t, e, "\x1b[?1049h")
	feed(t, e, "TUI-frame-A")
	snap := e.Snapshot()
	if !bytes.Contains(snap, []byte("history-1")) {
		t.Fatalf("snapshot must contain pre-alt scrollback line; bytes=%q", snap)
	}

	e2 := New(Options{Cols: 10, Rows: 3, ScrollbackLines: 100})
	if _, err := e2.Feed(snap); err != nil {
		t.Fatalf("feed(snapshot) error: %v", err)
	}
	if got := e2.ScrollbackLineCount(); got != want {
		t.Fatalf("scrollback lost across snapshot: want=%d got=%d", want, got)
	}
	if !e2.InAltBuffer() {
		t.Fatalf("snapshot must restore alt-buffer state; was lost")
	}
}

func TestUTF8SplitAcrossFeeds(t *testing.T) {
	e := New(Options{Cols: 10, Rows: 2})
	feed(t, e, "\xe2\x9c") // first 2 of 3 bytes for U+2713 ✓
	feed(t, e, "\x93rest")
	got := screenText(e.activeScreen())
	if !strings.HasPrefix(got, "✓rest") {
		t.Fatalf("UTF-8 not stitched across feeds; got=%q", got)
	}
}

func TestSGRPenAppliedToCells(t *testing.T) {
	e := New(Options{Cols: 10, Rows: 2})
	feed(t, e, "\x1b[31mred\x1b[0mok")
	row := e.activeScreen().Lines[0]
	if row[0].SGR.FG != 2 { // ANSI 31 (red) → encoding slot 2 (1=black,2=red,…)
		t.Fatalf("first cell FG: want=2 (red) got=%d", row[0].SGR.FG)
	}
	if !row[3].SGR.IsZero() {
		t.Fatalf("post-reset cell must be plain; got=%+v", row[3].SGR)
	}
}

func TestFeedIsTotalAndNeverErrors(t *testing.T) {
	e := New(Options{})
	bad := []byte{0x1b, '[', 0xff, 0xff, '?', 'h'} // garbage CSI
	n, err := e.Feed(bad)
	if err != nil {
		t.Fatalf("Feed must never error; got %v", err)
	}
	if n != len(bad) {
		t.Fatalf("Feed must consume all bytes; got %d of %d", n, len(bad))
	}
}

func TestSnapshotIdempotent(t *testing.T) {
	e := New(Options{Cols: 20, Rows: 4})
	feed(t, e, "hello\r\n\x1b[1mworld\x1b[0m\r\n")
	a := e.Snapshot()
	b := e.Snapshot()
	if !bytes.Equal(a, b) {
		t.Fatalf("Snapshot must be idempotent under no input; differ:\n a=%q\n b=%q", a, b)
	}
}

func TestSnapshotRoundTripPlainText(t *testing.T) {
	e := New(Options{Cols: 20, Rows: 4, ScrollbackLines: 100})
	feed(t, e, "alpha\r\nbeta\r\ngamma\r\ndelta\r\nepsilon")
	wantSB := e.ScrollbackLineCount()

	snap := e.Snapshot()
	e2 := New(Options{Cols: 20, Rows: 4, ScrollbackLines: 100})
	if _, err := e2.Feed(snap); err != nil {
		t.Fatalf("feed snap: %v", err)
	}
	if e.InAltBuffer() != e2.InAltBuffer() {
		t.Fatalf("alt-buffer mismatch after round-trip")
	}
	if got := e2.ScrollbackLineCount(); got != wantSB {
		t.Fatalf("scrollback count differs: want=%d got=%d", wantSB, got)
	}
}

func TestResizePreservesScrollback(t *testing.T) {
	e := New(Options{Cols: 10, Rows: 3, ScrollbackLines: 100})
	feed(t, e, "a\r\nb\r\nc\r\nd\r\ne\r\n")
	before := e.ScrollbackLineCount()
	e.Resize(20, 6)
	if got := e.ScrollbackLineCount(); got != before {
		t.Fatalf("Resize must preserve scrollback line count: before=%d after=%d", before, got)
	}
	if e.Cols() != 20 || e.Rows() != 6 {
		t.Fatalf("Resize did not apply: %dx%d", e.Cols(), e.Rows())
	}
}

func TestScrollbackBoundedByCap(t *testing.T) {
	e := New(Options{Cols: 4, Rows: 2, ScrollbackLines: MinScrollbackLines})
	for i := 0; i < MinScrollbackLines+50; i++ {
		feed(t, e, "x\r\n")
	}
	if got := e.ScrollbackLineCount(); got > MinScrollbackLines {
		t.Fatalf("scrollback exceeded cap: cap=%d got=%d", MinScrollbackLines, got)
	}
}

func TestInvalidScrollbackOptionsClamped(t *testing.T) {
	e := New(Options{ScrollbackLines: 5}) // below MinScrollbackLines → bumped to default
	if e.scrollback.cap < MinScrollbackLines {
		t.Fatalf("scrollback cap below minimum: got=%d", e.scrollback.cap)
	}
	e2 := New(Options{ScrollbackLines: MaxScrollbackLines * 10})
	if e2.scrollback.cap > MaxScrollbackLines {
		t.Fatalf("scrollback cap above maximum: got=%d", e2.scrollback.cap)
	}
}

func TestControlSequencesCoverCursorErasureAndScrollDirections(t *testing.T) {
	e := New(Options{Cols: 12, Rows: 4, ScrollbackLines: 20})
	feed(t, e, "abcdef")
	feed(t, e, "\x1b[1;3H\x1b[2K") // erase the whole current row
	if strings.Contains(screenText(e.activeScreen()), "abcdef") {
		t.Fatal("EL did not erase the current row")
	}
	feed(t, e, "line1\r\nline2\r\nline3")
	feed(t, e, "\x1b[2;3H\x1b[1K")         // erase to cursor
	feed(t, e, "\x1b[2;1H\x1b[K")          // erase from cursor
	feed(t, e, "\x1b[2;1H\x1b[2P")         // delete characters
	feed(t, e, "\x1b[2;1H\x1b[2@")         // insert characters
	feed(t, e, "\x1b7x\x1b8")              // save and restore cursor
	feed(t, e, "\x1b[2;3r\x1b[3;1H\x1bM")  // reverse index at top of region
	feed(t, e, "\x1b[2;3r\x1b[2;1H\x1b[T") // scroll region down
	feed(t, e, "\x1b]0;title\x07")         // OSC title is intentionally ignored
	if _, err := e.Feed([]byte("\x1b[?25l\x1b[?25h")); err != nil {
		t.Fatalf("mode controls returned error: %v", err)
	}
}

// TestECHErasesCharsInPlace covers the ECH (`\x1b[N X`) sequence used
// heavily by TUI coding agents (Claude Code, vim, etc.) to wipe stale
// cells before redrawing. Without this, the snapshot retains ghost
// content that the live redraw deltas don't fully overwrite, producing
// the "duplication when scrolling through chat" symptom.
func TestECHErasesCharsInPlace(t *testing.T) {
	e := New(Options{Cols: 20, Rows: 5})
	feed(t, e, "hello world")
	// Cursor is now at column 11. Move back to column 6.
	feed(t, e, "\x1b[1;7H")
	// Erase 5 chars in place ("world").
	feed(t, e, "\x1b[5X")
	got := screenText(e.activeScreen())
	// screenText trims trailing blanks, so "hello " plus 5 erased cells
	// shows as "hello" — what matters is that "world" is gone.
	if !strings.HasPrefix(got, "hello\n") {
		t.Fatalf("ECH did not erase chars in place; got=%q", got)
	}
	// Cursor must NOT have moved (per ECH spec).
	if e.activeScreen().X != 6 {
		t.Errorf("ECH must preserve cursor column; got X=%d want=6", e.activeScreen().X)
	}
}

// TestDECSTBMScrollsOnlyRegion locks in scroll-region semantics: when
// margins are set to (1, 3) and the cursor at the bottom of the region
// emits LF, only rows in the region scroll; rows below the region stay
// untouched. This is what TUI apps use to keep a status/input row
// pinned while their chat area scrolls.
func TestDECSTBMScrollsOnlyRegion(t *testing.T) {
	e := New(Options{Cols: 10, Rows: 5})
	// Fill rows 0..4 with markers.
	feed(t, e, "row0\r\nrow1\r\nrow2\r\nrow3\r\nrow4")
	// Set scroll region to rows 1..3 (1-indexed: 2..4).
	feed(t, e, "\x1b[2;4r")
	// DECSTBM homes the cursor; move back into the region's bottom.
	feed(t, e, "\x1b[4;1H")
	// Newline at region bottom must scroll the region (1..3) up,
	// leaving row 4 ("row4") untouched below.
	feed(t, e, "\n")
	lines := []string{}
	for _, l := range e.activeScreen().Lines {
		end := len(l)
		for end > 0 && l[end-1].Rune == ' ' && l[end-1].SGR.IsZero() {
			end--
		}
		row := make([]rune, 0, end)
		for i := 0; i < end; i++ {
			row = append(row, l[i].Rune)
		}
		lines = append(lines, string(row))
	}
	// Expected: row 0 unchanged ("row0"); rows 1..3 scrolled (row 1 was
	// "row1" → now contains old row2, etc.); row 4 unchanged ("row4").
	if lines[0] != "row0" {
		t.Errorf("row 0 should be untouched (above region); got=%q", lines[0])
	}
	if lines[4] != "row4" {
		t.Errorf("row 4 should be untouched (below region); got=%q", lines[4])
	}
	if lines[1] != "row2" {
		t.Errorf("region row 1 should have scrolled to old row 2; got=%q", lines[1])
	}
	if lines[2] != "row3" {
		t.Errorf("region row 2 should have scrolled to old row 3; got=%q", lines[2])
	}
}
