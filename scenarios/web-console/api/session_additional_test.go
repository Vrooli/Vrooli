package main

import (
	"bufio"
	"bytes"
	"os"
	"testing"
)

func TestSessionRespondCursorQuery(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	defer r.Close()
	defer w.Close()

	s := &session{
		ptyFile:  w,
		termRows: 0,
		termCols: 0,
	}

	s.respondCursorQuery()

	buf := make([]byte, 16)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("failed to read cursor response: %v", err)
	}
	got := string(buf[:n])
	want := "\x1b[1;80R"
	if got != want {
		t.Fatalf("unexpected cursor report: got %q want %q", got, want)
	}
}

func TestSessionHandleOutputBufferManagement(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	defer r.Close()
	defer w.Close()

	metrics := newMetricsRegistry()
	transcriptBuf := &bytes.Buffer{}

	s := &session{
		ptyFile:          w,
		metrics:          metrics,
		termRows:         24,
		termCols:         100,
		outputBuffer:     make([]outputPayload, 0, 2),
		maxBufferSize:    1,
		maxBufferBytes:   8,
		transcriptWriter: bufio.NewWriter(transcriptBuf),
		clients:          make(map[*wsClient]struct{}),
		readBuffer:       make([]byte, 0),
		lastInputSeq:     make(map[string]uint64),
		idleReset:        make(chan struct{}, 1),
		done:             make(chan struct{}),
	}

	s.handleOutput([]byte("out1\x1b[6n"))

	cursorBuf := make([]byte, 16)
	n, err := r.Read(cursorBuf)
	if err != nil {
		t.Fatalf("failed to read cursor response: %v", err)
	}
	if got := string(cursorBuf[:n]); got != "\x1b[24;100R" {
		t.Fatalf("unexpected cursor report: %q", got)
	}

	// Second chunk forces buffer trimming due to maxBufferSize=1
	s.handleOutput([]byte("second"))

	if err := s.transcriptWriter.Flush(); err != nil {
		t.Fatalf("flush transcript: %v", err)
	}

	s.outputBufferMu.RLock()
	defer s.outputBufferMu.RUnlock()
	if len(s.outputBuffer) != 1 {
		t.Fatalf("expected buffer length 1, got %d", len(s.outputBuffer))
	}
	if !s.outputBufferDropped {
		t.Fatal("expected output buffer to mark dropped chunks")
	}
}

func TestSessionIncrementReasonMetric(t *testing.T) {
	metrics := newMetricsRegistry()
	s := &session{metrics: metrics}

	s.incrementReasonMetric(reasonPanicStop)
	s.incrementReasonMetric(reasonIdleTimeout)
	s.incrementReasonMetric(reasonTTLExpired)
	s.incrementReasonMetric(reasonClientRequested)

	if metrics.panicStops.Load() != 1 {
		t.Fatalf("expected panicStops=1, got %d", metrics.panicStops.Load())
	}
	if metrics.idleTimeOuts.Load() != 1 {
		t.Fatalf("expected idleTimeOuts=1, got %d", metrics.idleTimeOuts.Load())
	}
	if metrics.ttlExpirations.Load() != 1 {
		t.Fatalf("expected ttlExpirations=1, got %d", metrics.ttlExpirations.Load())
	}
}
