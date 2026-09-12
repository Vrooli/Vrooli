package lifecycle

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestStepSinkForwardsAllBytesToInner(t *testing.T) {
	var inner bytes.Buffer
	sink := newStepSink(&inner)

	payload := "first\nsecond\nthird partial"
	if _, err := sink.Write([]byte(payload)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := inner.String(); got != payload {
		t.Fatalf("inner = %q, want %q", got, payload)
	}
}

func TestStepSinkRingCapturesRecentLines(t *testing.T) {
	var inner bytes.Buffer
	sink := newStepSink(&inner)

	lines := []string{"alpha", "beta", "gamma"}
	for _, line := range lines {
		if _, err := sink.Write([]byte(line + "\n")); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	var out bytes.Buffer
	sink.ReplayTo(&out, "build-step", "/tmp/log.txt")
	got := out.String()
	for _, line := range lines {
		if !strings.Contains(got, line) {
			t.Errorf("replay missing %q; got %q", line, got)
		}
	}
	if !strings.Contains(got, "/tmp/log.txt") {
		t.Errorf("replay missing log pointer: %q", got)
	}
	if !strings.Contains(got, "build-step") {
		t.Errorf("replay missing step name: %q", got)
	}
}

func TestStepSinkRingTruncatesToCap(t *testing.T) {
	ring := newLineRing(3)
	for i := 0; i < 10; i++ {
		ring.push(fmt.Sprintf("line-%d", i))
	}
	snap := ring.snapshot()
	if got, want := len(snap), 3; got != want {
		t.Fatalf("len(snapshot) = %d, want %d", got, want)
	}
	want := []string{"line-7", "line-8", "line-9"}
	for i, line := range snap {
		if line != want[i] {
			t.Errorf("snap[%d] = %q, want %q", i, line, want[i])
		}
	}
}

func TestStepSinkFlushCapturesPartialLine(t *testing.T) {
	var inner bytes.Buffer
	sink := newStepSink(&inner)

	if _, err := sink.Write([]byte("no-trailing-newline")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	// Before Flush, ring is empty.
	if !sink.ring.empty() {
		t.Fatal("ring should be empty before Flush (no newline seen)")
	}
	sink.Flush()
	snap := sink.ring.snapshot()
	if len(snap) != 1 || snap[0] != "no-trailing-newline" {
		t.Fatalf("snapshot = %v, want [no-trailing-newline]", snap)
	}
}

func TestStepSinkReplayNoOpWhenEmpty(t *testing.T) {
	sink := newStepSink(&bytes.Buffer{})
	var out bytes.Buffer
	sink.ReplayTo(&out, "step", "/tmp/log")
	if out.Len() != 0 {
		t.Fatalf("replay wrote %q on empty ring", out.String())
	}
}

func TestStepSinkStitchesPartialLinesAcrossWrites(t *testing.T) {
	sink := newStepSink(&bytes.Buffer{})
	sink.Write([]byte("hello "))
	sink.Write([]byte("world\nnext line"))
	sink.Flush()
	snap := sink.ring.snapshot()
	want := []string{"hello world", "next line"}
	if len(snap) != len(want) {
		t.Fatalf("snapshot = %v, want %v", snap, want)
	}
	for i, line := range want {
		if snap[i] != line {
			t.Errorf("snap[%d] = %q, want %q", i, snap[i], line)
		}
	}
}
