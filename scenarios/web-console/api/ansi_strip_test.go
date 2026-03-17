package main

import (
	"bytes"
	"testing"
)

func TestStripANSI_PlainText(t *testing.T) {
	input := []byte("hello world")
	got := stripANSI(input)
	if !bytes.Equal(got, input) {
		t.Errorf("expected %q, got %q", input, got)
	}
}

func TestStripANSI_CSIColorCodes(t *testing.T) {
	// \x1b[31m = red, \x1b[1;32m = bold green
	input := []byte("\x1b[31mred\x1b[1;32mgreen")
	want := []byte("redgreen")
	got := stripANSI(input)
	if !bytes.Equal(got, want) {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestStripANSI_OSCTitleSequence(t *testing.T) {
	// OSC set title terminated by BEL
	input := []byte("\x1b]0;my title\x07visible")
	want := []byte("visible")
	got := stripANSI(input)
	if !bytes.Equal(got, want) {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestStripANSI_OSCWithSTTerminator(t *testing.T) {
	input := []byte("\x1b]0;title\x1b\\visible")
	want := []byte("visible")
	got := stripANSI(input)
	if !bytes.Equal(got, want) {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestStripANSI_MixedContent(t *testing.T) {
	input := []byte("before\x1b[31m middle \x1b[0mafter")
	want := []byte("before middle after")
	got := stripANSI(input)
	if !bytes.Equal(got, want) {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestStripANSI_UTF8Preservation(t *testing.T) {
	input := []byte("hello \xe4\xb8\x96\xe7\x95\x8c\x1b[31m color \x1b[0m\xf0\x9f\x98\x80")
	want := []byte("hello \xe4\xb8\x96\xe7\x95\x8c color \xf0\x9f\x98\x80")
	got := stripANSI(input)
	if !bytes.Equal(got, want) {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestStripANSI_EmptyInput(t *testing.T) {
	got := stripANSI([]byte{})
	if len(got) != 0 {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestStripANSI_NilInput(t *testing.T) {
	got := stripANSI(nil)
	if got != nil {
		t.Errorf("expected nil, got %q", got)
	}
}

func TestStripANSI_SGRReset(t *testing.T) {
	input := []byte("\x1b[0m")
	got := stripANSI(input)
	if len(got) != 0 {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestStripANSI_SimpleEscape(t *testing.T) {
	// Two-byte escape like \x1bM (reverse line feed)
	input := []byte("a\x1bMb")
	want := []byte("ab")
	got := stripANSI(input)
	if !bytes.Equal(got, want) {
		t.Errorf("expected %q, got %q", want, got)
	}
}
