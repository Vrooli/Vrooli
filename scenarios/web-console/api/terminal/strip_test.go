package terminal

import (
	"bytes"
	"testing"
)

func TestStripEscapes_PlainText(t *testing.T) {
	input := []byte("hello world")
	if got := StripEscapes(input); !bytes.Equal(got, input) {
		t.Errorf("plain text: got %q want %q", got, input)
	}
}

func TestStripEscapes_CSI(t *testing.T) {
	input := []byte("hi \x1b[31mred\x1b[0m bye")
	want := []byte("hi red bye")
	if got := StripEscapes(input); !bytes.Equal(got, want) {
		t.Errorf("CSI: got %q want %q", got, want)
	}
}

func TestStripEscapes_CSIWithIntermediates(t *testing.T) {
	// DECRQM 2026 with $ intermediate.
	input := []byte("a\x1b[?2026$pb")
	want := []byte("ab")
	if got := StripEscapes(input); !bytes.Equal(got, want) {
		t.Errorf("CSI w/ intermediate: got %q want %q", got, want)
	}
}

func TestStripEscapes_OSC_BEL(t *testing.T) {
	input := []byte("a\x1b]0;title\x07b")
	want := []byte("ab")
	if got := StripEscapes(input); !bytes.Equal(got, want) {
		t.Errorf("OSC BEL: got %q want %q", got, want)
	}
}

func TestStripEscapes_OSC_ST(t *testing.T) {
	input := []byte("a\x1b]0;title\x1b\\b")
	want := []byte("ab")
	if got := StripEscapes(input); !bytes.Equal(got, want) {
		t.Errorf("OSC ST: got %q want %q", got, want)
	}
}

func TestStripEscapes_TwoByteEscape(t *testing.T) {
	input := []byte("a\x1bcb")
	want := []byte("ab")
	if got := StripEscapes(input); !bytes.Equal(got, want) {
		t.Errorf("ESC c: got %q want %q", got, want)
	}
}

func TestStripEscapes_Empty(t *testing.T) {
	if got := StripEscapes([]byte{}); len(got) != 0 {
		t.Errorf("empty: got %q want empty", got)
	}
}

func TestStripEscapes_Nil(t *testing.T) {
	if got := StripEscapes(nil); got != nil {
		t.Errorf("nil: got %q want nil", got)
	}
}

func TestStripEscapes_UTF8Preserved(t *testing.T) {
	input := []byte("\x1b[33mhåló — ✓\x1b[0m")
	want := []byte("håló — ✓")
	if got := StripEscapes(input); !bytes.Equal(got, want) {
		t.Errorf("UTF-8: got %q want %q", got, want)
	}
}

func TestStripEscapes_TrailingEsc(t *testing.T) {
	input := []byte("abc\x1b")
	want := []byte("abc")
	if got := StripEscapes(input); !bytes.Equal(got, want) {
		t.Errorf("trailing ESC: got %q want %q", got, want)
	}
}
