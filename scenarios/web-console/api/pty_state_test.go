package main

import (
	"testing"
)

func TestPTYStateTracker_Ground(t *testing.T) {
	var tr PTYStateTracker
	if tr.IsAltBuffer() {
		t.Fatal("initial state should not be alt-buffer")
	}
	if tr.Observe([]byte("hello world\n")) {
		t.Fatal("plain text should not transition alt-buffer")
	}
}

func TestPTYStateTracker_EnterExit1049(t *testing.T) {
	var tr PTYStateTracker
	if !tr.Observe([]byte("\x1b[?1049h")) {
		t.Fatal("expected transition to alt-buffer on ?1049h")
	}
	if !tr.IsAltBuffer() {
		t.Fatal("alt-buffer should be true")
	}
	if !tr.Observe([]byte("\x1b[?1049l")) {
		t.Fatal("expected transition to main-buffer on ?1049l")
	}
	if tr.IsAltBuffer() {
		t.Fatal("alt-buffer should be false")
	}
}

func TestPTYStateTracker_EnterExit47(t *testing.T) {
	var tr PTYStateTracker
	tr.Observe([]byte("\x1b[?47h"))
	if !tr.IsAltBuffer() {
		t.Fatal("mode 47 entry not tracked")
	}
	tr.Observe([]byte("\x1b[?47l"))
	if tr.IsAltBuffer() {
		t.Fatal("mode 47 exit not tracked")
	}
}

func TestPTYStateTracker_EnterExit1047(t *testing.T) {
	var tr PTYStateTracker
	tr.Observe([]byte("\x1b[?1047h"))
	if !tr.IsAltBuffer() {
		t.Fatal("mode 1047 entry not tracked")
	}
	tr.Observe([]byte("\x1b[?1047l"))
	if tr.IsAltBuffer() {
		t.Fatal("mode 1047 exit not tracked")
	}
}

func TestPTYStateTracker_Mode1048Ignored(t *testing.T) {
	// 1048 is cursor save/restore alone, not alt-screen.
	var tr PTYStateTracker
	if tr.Observe([]byte("\x1b[?1048h")) {
		t.Fatal("mode 1048 should not flip alt-buffer")
	}
	if tr.IsAltBuffer() {
		t.Fatal("mode 1048 incorrectly set alt-buffer")
	}
}

func TestPTYStateTracker_SplitAcrossReads(t *testing.T) {
	cases := [][]byte{
		[]byte("\x1b"),
		[]byte("["),
		[]byte("?"),
		[]byte("1"),
		[]byte("0"),
		[]byte("4"),
		[]byte("9"),
		[]byte("h"),
	}
	var tr PTYStateTracker
	for i, chunk := range cases {
		transitioned := tr.Observe(chunk)
		if i < len(cases)-1 && transitioned {
			t.Fatalf("unexpected transition at chunk %d (%q)", i, chunk)
		}
		if i == len(cases)-1 && !transitioned {
			t.Fatalf("final chunk %q should have transitioned", chunk)
		}
	}
	if !tr.IsAltBuffer() {
		t.Fatal("alt-buffer not set after split sequence")
	}
}

func TestPTYStateTracker_MultiParamSequence(t *testing.T) {
	// Some emitters combine modes — ensure we still detect 1049 in a list.
	var tr PTYStateTracker
	if !tr.Observe([]byte("\x1b[?25;1049h")) {
		t.Fatal("multi-param entry not tracked")
	}
	if !tr.IsAltBuffer() {
		t.Fatal("alt-buffer should be true after multi-param entry")
	}
}

func TestPTYStateTracker_RedundantEnterIsNotATransition(t *testing.T) {
	var tr PTYStateTracker
	if !tr.Observe([]byte("\x1b[?1049h")) {
		t.Fatal("first entry should transition")
	}
	if tr.Observe([]byte("\x1b[?1049h")) {
		t.Fatal("redundant entry should not count as transition")
	}
	if !tr.IsAltBuffer() {
		t.Fatal("alt-buffer should still be true")
	}
}

func TestPTYStateTracker_UnrelatedCSIIgnored(t *testing.T) {
	var tr PTYStateTracker
	// Cursor position, color, other DEC private modes not in our set.
	tr.Observe([]byte("\x1b[10;20H\x1b[1;31m\x1b[?25h\x1b[?2004h"))
	if tr.IsAltBuffer() {
		t.Fatal("unrelated sequences flipped alt-buffer")
	}
}

func TestPTYStateTracker_MalformedDoesNotDeadlock(t *testing.T) {
	var tr PTYStateTracker
	// ESC with no follow-up, then CSI with garbage, then recovers.
	tr.Observe([]byte("\x1b"))
	tr.Observe([]byte("garbage"))
	tr.Observe([]byte("\x1b[?1049h"))
	if !tr.IsAltBuffer() {
		t.Fatal("tracker failed to recover after malformed input")
	}
}

func TestPTYStateTracker_LongParamsReset(t *testing.T) {
	// A pathological stream of digits should not grow params forever.
	var tr PTYStateTracker
	tr.Observe([]byte("\x1b[?"))
	big := make([]byte, 200)
	for i := range big {
		big[i] = '9'
	}
	tr.Observe(big)
	tr.Observe([]byte("\x1b[?1049h"))
	if !tr.IsAltBuffer() {
		t.Fatal("tracker did not recover after param overflow")
	}
}
