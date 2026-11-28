package terminal

import (
	"strings"
	"testing"
)

func TestEmulator_View(t *testing.T) {
	e := New(Options{Cols: 10, Rows: 3})
	_, _ = e.Feed([]byte("hi\r\nworld"))
	v := e.View()
	if v.Cols != 10 || v.Rows != 3 {
		t.Fatalf("dims: got %dx%d, want 10x3", v.Cols, v.Rows)
	}
	if len(v.Cells) != 3 {
		t.Fatalf("Cells: got %d rows, want 3", len(v.Cells))
	}
	if len(v.Cells[0]) != 10 {
		t.Fatalf("row 0 cols: got %d, want 10", len(v.Cells[0]))
	}
	// First row "hi" + 8 blanks
	if v.Cells[0][0].Rune != 'h' || v.Cells[0][1].Rune != 'i' {
		t.Errorf("row 0 first two cells: got %v %v, want 'h' 'i'", v.Cells[0][0].Rune, v.Cells[0][1].Rune)
	}
	// Cursor after "world" is at column 5 of row 1.
	if v.Cursor.X != 5 || v.Cursor.Y != 1 {
		t.Errorf("cursor: got (%d,%d), want (5,1)", v.Cursor.X, v.Cursor.Y)
	}
	if v.InAltBuffer {
		t.Error("InAltBuffer: should be false")
	}
}

func TestEmulator_Cells_IsDeepCopy(t *testing.T) {
	e := New(Options{Cols: 4, Rows: 1})
	_, _ = e.Feed([]byte("abcd"))
	c1 := e.Cells()
	c1[0][0].Rune = 'X'
	c2 := e.Cells()
	if c2[0][0].Rune != 'a' {
		t.Errorf("Cells did not deep-copy: caller mutation leaked into emulator (got %q, want %q)", c2[0][0].Rune, 'a')
	}
}

func TestEmulator_PlainText(t *testing.T) {
	e := New(Options{Cols: 6, Rows: 3})
	_, _ = e.Feed([]byte("hi\r\nworld"))
	got := e.PlainText(false)
	want := "hi\nworld\n"
	// PlainText is rows joined by '\n' with trailing blanks stripped;
	// last row is blank → empty string.
	want = strings.TrimRight(want, "\n") + "\n"
	if got != want {
		t.Errorf("PlainText:\ngot  %q\nwant %q", got, want)
	}
}

func TestEmulator_Cursor(t *testing.T) {
	e := New(Options{Cols: 10, Rows: 2})
	_, _ = e.Feed([]byte("abc"))
	c := e.Cursor()
	if c.X != 3 || c.Y != 0 {
		t.Errorf("cursor: got (%d,%d), want (3,0)", c.X, c.Y)
	}
}

func TestEmulator_ControlEvents_AltBufferTransitions(t *testing.T) {
	e := New(Options{Cols: 20, Rows: 5})
	ch := e.ControlEvents()
	// Enter alt buffer (mode 1049).
	_, _ = e.Feed([]byte("\x1b[?1049h"))
	select {
	case ev := <-ch:
		if ev.Kind != EventAltBufferEnter {
			t.Errorf("first event kind: got %v, want EventAltBufferEnter", ev.Kind)
		}
	default:
		t.Fatal("no event emitted on alt-buffer enter")
	}
	// Exit alt buffer.
	_, _ = e.Feed([]byte("\x1b[?1049l"))
	select {
	case ev := <-ch:
		if ev.Kind != EventAltBufferExit {
			t.Errorf("second event kind: got %v, want EventAltBufferExit", ev.Kind)
		}
	default:
		t.Fatal("no event emitted on alt-buffer exit")
	}
}

func TestEmulator_ControlEvents_CSIQuery(t *testing.T) {
	e := New(Options{Cols: 20, Rows: 5})
	ch := e.ControlEvents()
	// DA1 (CSI c).
	_, _ = e.Feed([]byte("\x1b[c"))
	select {
	case ev := <-ch:
		if ev.Kind != EventCSIQuery || ev.Final != 'c' {
			t.Errorf("DA1: got kind=%v final=%q, want EventCSIQuery final 'c'", ev.Kind, ev.Final)
		}
	default:
		t.Fatal("no event for DA1")
	}
	// DECRQM 2026 (CSI ? 2026 $ p).
	_, _ = e.Feed([]byte("\x1b[?2026$p"))
	select {
	case ev := <-ch:
		if ev.Kind != EventCSIQuery || ev.Final != 'p' || !ev.Private {
			t.Errorf("DECRQM: got %+v, want private CSIQuery final 'p'", ev)
		}
	default:
		t.Fatal("no event for DECRQM")
	}
}
