package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestHistoryStore_ByteCountingMonotonic locks in that totalOutputBytes
// advances by exactly the appended byte count, independent of trimming.
// Cache-consistency on the client side depends on this invariant.
func TestHistoryStore_ByteCountingMonotonic(t *testing.T) {
	s := &Session{offlineBufferMax: 1024}
	s.appendHistory([]byte("hello"))
	if got, want := s.totalOutputBytes, int64(5); got != want {
		t.Errorf("after first append: totalOutputBytes=%d want=%d", got, want)
	}
	s.appendHistory([]byte(" world"))
	if got, want := s.totalOutputBytes, int64(11); got != want {
		t.Errorf("after second append: totalOutputBytes=%d want=%d", got, want)
	}
	// Trim-triggering append: counter must still advance by full length.
	huge := bytes.Repeat([]byte("x"), 2000)
	s.appendHistory(huge)
	if got, want := s.totalOutputBytes, int64(11+2000); got != want {
		t.Errorf("after trimming append: totalOutputBytes=%d want=%d", got, want)
	}
}

// TestHistoryStore_HistoryStartAdvancesAfterTrim asserts that
// historyStart reflects bytes that have been trimmed from the ring, so
// Subscribe can detect when a client's resume offset has been aged out.
func TestHistoryStore_HistoryStartAdvancesAfterTrim(t *testing.T) {
	s := &Session{offlineBufferMax: 16}
	s.appendHistory([]byte("line1\nline2\nline3\nline4\n"))
	// Total 24 bytes appended; ring capped at 16. historyStart must be
	// strictly > 0 because the early bytes were trimmed.
	if got := s.historyStart(); got <= 0 {
		t.Errorf("historyStart after trim: got=%d want >0", got)
	}
	if got := s.totalOutputBytes; got != 24 {
		t.Errorf("totalOutputBytes=%d want 24", got)
	}
	// historyStart + len(outputHistory) == totalOutputBytes
	if got, want := s.historyStart()+int64(len(s.outputHistory)), s.totalOutputBytes; got != want {
		t.Errorf("history accounting violation: start+len=%d totalBytes=%d", got, want)
	}
}

// TestHistoryStore_SnapToCleanBoundary_MidCSIScapeStripped covers the
// mid-ANSI-sequence guard used when the ring trim lands inside a color
// escape. Returned bytes must not start with raw SGR parameter bytes.
func TestHistoryStore_SnapToCleanBoundary_MidCSIScapeStripped(t *testing.T) {
	// Simulate a trim inside `\x1b[31m` (red SGR): the ESC byte was cut
	// off, leaving `[31m` followed by useful data.
	in := []byte("[31mhello\nworld")
	out := snapToCleanBoundary(in)
	if bytes.HasPrefix(out, []byte("[31m")) {
		t.Errorf("snapToCleanBoundary returned mid-CSI prefix: %q", out)
	}
	// Should end up at the newline boundary or later.
	if !bytes.Contains(out, []byte("world")) {
		t.Errorf("snapToCleanBoundary discarded too much: %q", out)
	}
}

// TestHistoryStore_SnapToCleanBoundary_NormalTextUnchanged verifies
// looksLikeMidSequence doesn't false-positive on benign leading digits
// or punctuation. A line like "42 is the answer" must be returned
// mostly intact (the leading portion may move to the next newline).
func TestHistoryStore_SnapToCleanBoundary_NormalTextUnchanged(t *testing.T) {
	in := []byte("42 is\nthe answer\n")
	out := snapToCleanBoundary(in)
	if !strings.Contains(string(out), "the answer") {
		t.Errorf("snapToCleanBoundary over-trimmed normal text: %q", out)
	}
}

// TestHistoryStore_EmptyAppendNoop ensures a zero-length append does
// not change state. readLoop can deliver empty slices when PTY reads
// return 0 without error.
func TestHistoryStore_EmptyAppendNoop(t *testing.T) {
	s := &Session{offlineBufferMax: 64}
	s.appendHistory([]byte("seed"))
	before := s.totalOutputBytes
	s.appendHistory(nil)
	s.appendHistory([]byte{})
	if got := s.totalOutputBytes; got != before {
		t.Errorf("empty append advanced counter: before=%d after=%d", before, got)
	}
}

// TestHistoryStore_UnboundedBuffer covers the offlineBufferMax<=0 path
// (bufferless sessions; every byte retained).
func TestHistoryStore_UnboundedBuffer(t *testing.T) {
	s := &Session{offlineBufferMax: 0}
	big := bytes.Repeat([]byte("a"), 10000)
	s.appendHistory(big)
	if got, want := s.totalOutputBytes, int64(len(big)); got != want {
		t.Errorf("totalOutputBytes=%d want %d", got, want)
	}
	// The ring itself is not populated when offlineBufferMax <= 0; this
	// locks in the contract that replay is bounded to the configured
	// window only.
	if len(s.outputHistory) != 0 {
		t.Errorf("outputHistory=%d bytes want 0 when max<=0", len(s.outputHistory))
	}
}
