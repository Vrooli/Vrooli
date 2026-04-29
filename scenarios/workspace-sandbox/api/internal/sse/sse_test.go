package sse

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestEncodeFrame_RoundTrips is the wire-format invariant: anything
// the encoder writes must parse back to the same (event, data) pair.
// This pins the multi-line encoding bug Phase 3 had to paper over.
func TestEncodeFrame_RoundTrips(t *testing.T) {
	cases := []struct {
		name  string
		event string
		data  string
	}{
		{"named-empty", "exit", ""},
		{"named-single-line", "exit", "code=0"},
		{"named-trailing-newline", "exit", "code=0\n"},
		{"named-internal-newline", "exit", "line1\nline2"},
		{"named-multi-line-trailing", "exit", "line1\nline2\n"},
		{"default-empty", "", ""},
		{"default-single-line", "", "hello, world"},
		{"default-trailing-newline", "", "chunk-A\n"},
		{"default-internal-newline", "", "chunk-A\nchunk-B"},
		{"default-multi-line-trailing", "", "a\nb\nc\n"},
		{"json-payload", "exit", `{"exitCode":1,"signal":0}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wire := encodeFrame(tc.event, []byte(tc.data))
			frames := ParseStream(bytes.NewReader(wire))
			if len(frames) != 1 {
				t.Fatalf("expected 1 frame, got %d (wire=%q)", len(frames), wire)
			}
			if frames[0].Event != tc.event {
				t.Errorf("event = %q, want %q", frames[0].Event, tc.event)
			}
			if string(frames[0].Data) != tc.data {
				t.Errorf("data = %q, want %q (wire=%q)", string(frames[0].Data), tc.data, wire)
			}
		})
	}
}

// TestEncodeFrame_EndsWithBlankLine is a sanity check that every
// emitted frame is terminated by a single blank line — the dispatcher
// boundary in the SSE spec.
func TestEncodeFrame_EndsWithBlankLine(t *testing.T) {
	wire := encodeFrame("exit", []byte("ok"))
	if !bytes.HasSuffix(wire, []byte("\n\n")) {
		t.Errorf("frame does not end with \\n\\n: %q", wire)
	}
}

// TestNewHTTPWriter_RejectsMissingFlusher is the regression gate for
// the 2026-04-28 SSE Flusher bug. A response writer without
// http.Flusher must surface ErrFlusherUnsupported instead of silently
// streaming into a buffered void.
func TestNewHTTPWriter_RejectsMissingFlusher(t *testing.T) {
	w := &nonFlushingWriter{header: http.Header{}}
	hw, err := NewHTTPWriter(w)
	if !errors.Is(err, ErrFlusherUnsupported) {
		t.Fatalf("err = %v, want ErrFlusherUnsupported", err)
	}
	if hw != nil {
		t.Errorf("HTTPWriter = %v, want nil on error", hw)
	}
	// Headers must NOT have been mutated on the failure path — the
	// caller still wants to write a non-SSE error response.
	if got := w.Header().Get("Content-Type"); got != "" {
		t.Errorf("Content-Type set on failure path: %q", got)
	}
}

// TestNewHTTPWriter_SetsSSEHeaders confirms the SSE response headers
// are stamped on construction so handlers don't have to.
func TestNewHTTPWriter_SetsSSEHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	if _, err := NewHTTPWriter(rec); err != nil {
		t.Fatalf("NewHTTPWriter: %v", err)
	}
	cases := []struct{ key, want string }{
		{"Content-Type", "text/event-stream"},
		{"Cache-Control", "no-cache"},
		{"Connection", "keep-alive"},
		{"X-Accel-Buffering", "no"},
	}
	for _, tc := range cases {
		if got := rec.Header().Get(tc.key); got != tc.want {
			t.Errorf("header %s = %q, want %q", tc.key, got, tc.want)
		}
	}
}

// TestHTTPWriter_FrameOrder asserts the Write* methods produce frames
// in the order they were called and that Close emits a single
// `event: end` frame at the tail.
func TestHTTPWriter_FrameOrder(t *testing.T) {
	rec := httptest.NewRecorder()
	hw, err := NewHTTPWriter(rec)
	if err != nil {
		t.Fatalf("NewHTTPWriter: %v", err)
	}

	if err := hw.WriteData([]byte("first\n")); err != nil {
		t.Fatalf("WriteData: %v", err)
	}
	if err := hw.WriteData([]byte("second\n")); err != nil {
		t.Fatalf("WriteData: %v", err)
	}
	if err := hw.WriteEvent("exit", []byte(`{"exitCode":0}`)); err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}
	if err := hw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	frames := ParseStream(rec.Body)
	wantEvents := []string{"", "", "exit", "end"}
	if len(frames) != len(wantEvents) {
		t.Fatalf("got %d frames, want %d (frames=%+v)", len(frames), len(wantEvents), frames)
	}
	for i, want := range wantEvents {
		if frames[i].Event != want {
			t.Errorf("frame[%d].Event = %q, want %q", i, frames[i].Event, want)
		}
	}
	if string(frames[0].Data) != "first\n" {
		t.Errorf("frame[0].Data = %q, want %q", frames[0].Data, "first\n")
	}
	if string(frames[1].Data) != "second\n" {
		t.Errorf("frame[1].Data = %q, want %q", frames[1].Data, "second\n")
	}
	if string(frames[2].Data) != `{"exitCode":0}` {
		t.Errorf("frame[2].Data = %q, want JSON", frames[2].Data)
	}
}

// TestHTTPWriter_CloseIdempotent: a second Close is a no-op and
// produces no additional `event: end` frame.
func TestHTTPWriter_CloseIdempotent(t *testing.T) {
	rec := httptest.NewRecorder()
	hw, err := NewHTTPWriter(rec)
	if err != nil {
		t.Fatalf("NewHTTPWriter: %v", err)
	}
	if err := hw.WriteData([]byte("hi")); err != nil {
		t.Fatalf("WriteData: %v", err)
	}
	if err := hw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := hw.Close(); err != nil {
		t.Fatalf("second Close should be no-op, got %v", err)
	}
	if err := hw.Close(); err != nil {
		t.Fatalf("third Close should be no-op, got %v", err)
	}
	body := rec.Body.String()
	count := strings.Count(body, "event: end")
	if count != 1 {
		t.Errorf("expected exactly one `event: end` line, got %d (body=%q)", count, body)
	}
}

// TestHTTPWriter_WritesAfterCloseRejected: the contract bans writing
// after Close. ErrAlreadyClosed must surface so handlers can detect
// programming errors rather than silently dropping frames.
func TestHTTPWriter_WritesAfterCloseRejected(t *testing.T) {
	rec := httptest.NewRecorder()
	hw, err := NewHTTPWriter(rec)
	if err != nil {
		t.Fatalf("NewHTTPWriter: %v", err)
	}
	if err := hw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := hw.WriteData([]byte("late")); !errors.Is(err, ErrAlreadyClosed) {
		t.Errorf("WriteData after Close = %v, want ErrAlreadyClosed", err)
	}
	if err := hw.WriteEvent("exit", []byte("late")); !errors.Is(err, ErrAlreadyClosed) {
		t.Errorf("WriteEvent after Close = %v, want ErrAlreadyClosed", err)
	}
	// Flush after Close is intentionally a no-op (Close already flushed).
	if err := hw.Flush(); err != nil {
		t.Errorf("Flush after Close = %v, want nil", err)
	}
}

// TestHTTPWriter_FlushForwarded: every Write* call flushes, but an
// explicit Flush after several writes must also reach the underlying
// flusher. The recording flusher counts both implicit and explicit
// flushes to prove pass-through.
func TestHTTPWriter_FlushForwarded(t *testing.T) {
	rec := &countingRecorder{ResponseRecorder: httptest.NewRecorder()}
	hw, err := NewHTTPWriter(rec)
	if err != nil {
		t.Fatalf("NewHTTPWriter: %v", err)
	}
	if err := hw.WriteData([]byte("a")); err != nil {
		t.Fatalf("WriteData: %v", err)
	}
	if err := hw.WriteEvent("ping", []byte("b")); err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}
	if err := hw.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	// Each Write* flushes once, plus the explicit Flush = 3.
	if rec.flushes < 3 {
		t.Errorf("expected ≥ 3 flushes, got %d", rec.flushes)
	}
	if err := hw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// Close flushes once more.
	if rec.flushes < 4 {
		t.Errorf("expected ≥ 4 flushes after Close, got %d", rec.flushes)
	}
}

// TestHTTPWriter_MultiLineDataParseable verifies the Phase-3 papered-
// over bug is fixed: a chunk with an embedded newline produces ONE
// frame whose parsed Data preserves the input verbatim.
func TestHTTPWriter_MultiLineDataParseable(t *testing.T) {
	rec := httptest.NewRecorder()
	hw, err := NewHTTPWriter(rec)
	if err != nil {
		t.Fatalf("NewHTTPWriter: %v", err)
	}
	chunk := []byte("line1\nline2\nline3")
	if err := hw.WriteData(chunk); err != nil {
		t.Fatalf("WriteData: %v", err)
	}
	if err := hw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	frames := ParseStream(rec.Body)
	// Two frames expected: the data chunk + `end`.
	if len(frames) != 2 {
		t.Fatalf("got %d frames, want 2 (frames=%+v)", len(frames), frames)
	}
	if string(frames[0].Data) != string(chunk) {
		t.Errorf("frame[0].Data = %q, want %q", frames[0].Data, chunk)
	}
	if frames[1].Event != "end" {
		t.Errorf("frame[1].Event = %q, want end", frames[1].Event)
	}
}

// nonFlushingWriter implements http.ResponseWriter but NOT http.Flusher.
// Used to exercise the missing-Flusher failure path.
type nonFlushingWriter struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func (n *nonFlushingWriter) Header() http.Header         { return n.header }
func (n *nonFlushingWriter) Write(b []byte) (int, error) { return n.body.Write(b) }
func (n *nonFlushingWriter) WriteHeader(s int)           { n.status = s }

// countingRecorder wraps httptest.ResponseRecorder and counts Flush
// calls. ResponseRecorder.Flush is a no-op; we override to track it.
type countingRecorder struct {
	*httptest.ResponseRecorder
	flushes int
}

func (c *countingRecorder) Flush() {
	c.flushes++
	c.ResponseRecorder.Flush()
}
