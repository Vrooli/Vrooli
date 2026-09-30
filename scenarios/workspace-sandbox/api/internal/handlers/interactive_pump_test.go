// Unit tests for the PTY output pump used by interactive sessions.
//
// Pure helper-function tests: no router, no middleware, no httptest —
// see handler_test_pattern_test.go for why that allowance exists.
//
// These tests assert the DESIRED behavior: once the PTY reports closure
// in any of the forms a Linux pty master can produce (io.EOF,
// os.ErrClosed, syscall.EIO) or degenerates into repeated zero-length
// reads, the pump must exit promptly, cancel the session context, and
// must not spin.

package handlers

import (
	"context"
	"errors"
	"io"
	"os"
	"syscall"
	"testing"
	"time"
)

// scriptedReader returns the scripted results in order and repeats the
// last one forever, counting every call.
type scriptedReader struct {
	results []readResult
	calls   int
}

type readResult struct {
	data []byte
	err  error
}

func (s *scriptedReader) Read(p []byte) (int, error) {
	idx := s.calls
	if idx >= len(s.results) {
		idx = len(s.results) - 1
	}
	s.calls++
	r := s.results[idx]
	return copy(p, r.data), r.err
}

// runPump drives pumpPTY on a goroutine and fails the test if it does
// not return within the deadline (a spinning pump never returns).
func runPump(t *testing.T, r io.Reader, send func([]byte) error) (reads int, ctxDone bool) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := make(chan int, 1)
	go func() { out <- pumpPTY(cancel, r, send) }()

	select {
	case reads = <-out:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("pumpPTY did not return within 100ms: pump is spinning")
	}

	select {
	case <-ctx.Done():
		ctxDone = true
	case <-time.After(100 * time.Millisecond):
	}
	return reads, ctxDone
}

func noSend([]byte) error { return nil }

func TestPumpPTY_ExitsOnReadError(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"io.EOF", io.EOF},
		{"os.ErrClosed", os.ErrClosed},
		{"syscall.EIO", syscall.EIO},
		{"wrapped EIO", &os.PathError{Op: "read", Path: "/dev/ptmx", Err: syscall.EIO}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &scriptedReader{results: []readResult{{err: tc.err}}}
			reads, ctxDone := runPump(t, r, noSend)
			if reads != 1 {
				t.Fatalf("expected exactly 1 read before exit on %v, got %d", tc.err, reads)
			}
			if !ctxDone {
				t.Fatalf("pump exited on %v but did not cancel the session context", tc.err)
			}
		})
	}
}

func TestPumpPTY_ExitsOnRepeatedEmptyReads(t *testing.T) {
	r := &scriptedReader{results: []readResult{{data: nil, err: nil}}}
	reads, ctxDone := runPump(t, r, noSend)
	if reads > maxConsecutiveEmptyPTYReads {
		t.Fatalf("pump kept polling after %d empty reads (bound is %d)", reads, maxConsecutiveEmptyPTYReads)
	}
	if !ctxDone {
		t.Fatal("pump exited on repeated empty reads but did not cancel the session context")
	}
}

func TestPumpPTY_DataResetsEmptyReadCounter(t *testing.T) {
	// Two empty reads, real data, two empty reads, then EOF: the data
	// read must reset the empty counter so the pump does not give up on
	// a live-but-quiet PTY.
	r := &scriptedReader{results: []readResult{
		{},
		{},
		{data: []byte("hello")},
		{},
		{},
		{err: io.EOF},
	}}
	var got []byte
	reads, ctxDone := runPump(t, r, func(b []byte) error {
		got = append(got, b...)
		return nil
	})
	if string(got) != "hello" {
		t.Fatalf("expected forwarded output %q, got %q", "hello", got)
	}
	if reads != 6 {
		t.Fatalf("expected the pump to run until EOF (6 reads), got %d", reads)
	}
	if !ctxDone {
		t.Fatal("pump did not cancel the session context on EOF")
	}
}

func TestPumpPTY_ForwardsTrailingDataWithError(t *testing.T) {
	// io.Reader permits (n > 0, err != nil); the data must be delivered
	// before the pump exits.
	r := &scriptedReader{results: []readResult{{data: []byte("bye"), err: io.EOF}}}
	var got string
	reads, ctxDone := runPump(t, r, func(b []byte) error {
		got = string(b)
		return nil
	})
	if got != "bye" {
		t.Fatalf("expected trailing data %q forwarded, got %q", "bye", got)
	}
	if reads != 1 || !ctxDone {
		t.Fatalf("expected exit after 1 read with ctx canceled, got reads=%d ctxDone=%v", reads, ctxDone)
	}
}

func TestPumpPTY_ExitsWhenSendFails(t *testing.T) {
	r := &scriptedReader{results: []readResult{{data: []byte("x")}}}
	sendErr := errors.New("websocket gone")
	reads, ctxDone := runPump(t, r, func([]byte) error { return sendErr })
	if reads != 1 {
		t.Fatalf("expected pump to stop after the first failed send, got %d reads", reads)
	}
	if !ctxDone {
		t.Fatal("pump did not cancel the session context after send failure")
	}
}

// TestPumpPTY_BlockedReadReleasedByClose covers the owner-side contract:
// a Read blocked on a real pipe is released when the owner closes it on
// cancellation, and the pump then exits and cancels.
func TestPumpPTY_BlockedReadReleasedByClose(t *testing.T) {
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer pw.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := make(chan int, 1)
	go func() { out <- pumpPTY(cancel, pr, noSend) }()

	// The pump is now parked in Read. Mirror runInteractiveSession's
	// cancel path: close the read end.
	time.Sleep(10 * time.Millisecond)
	_ = pr.Close()

	select {
	case reads := <-out:
		if reads != 1 {
			t.Fatalf("expected a single blocked read then exit, got %d", reads)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("closing the reader did not release the blocked pump")
	}
	select {
	case <-ctx.Done():
	default:
		t.Fatal("pump did not cancel the session context after close")
	}
}
