// Package sse owns the wire-format contract for Server-Sent-Events
// streams emitted by the HTTP handlers.
//
// Why this package exists (Round 4 Phase 5):
//
// Before this seam, internal/handlers/process_logs.go inlined the SSE
// frame construction with raw fmt.Fprintf calls. Two latent bugs
// piggybacked on that:
//
//  1. Multi-line data chunks. The handler emitted `data: %s\n\n` for
//     each chunk; chunks containing embedded `\n` produced wire output
//     that strict SSE parsers (including the agent-manager consumer)
//     could not reassemble — every line after the first was silently
//     dropped because the parser saw an unknown field.
//  2. http.Flusher pass-through. The Flusher assertion was duplicated
//     in every handler that streamed; a missing Flusher in the
//     middleware (the 2026-04-28 incident) only surfaced after the
//     production middleware shipped, because httptest.ResponseRecorder
//     natively implements Flusher and masked the gap.
//
// Centralizing both concerns here lets the handler express intent
// ("write this chunk as one data frame", "write this exit event")
// without re-deriving the wire format. The same package also forbids
// double-Close and writes-after-Close, so the `event: end` invariant
// (always last, exactly once) is structural rather than convention.
//
// Spec reference: https://html.spec.whatwg.org/multipage/server-sent-events.html
package sse

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"
)

// ErrFlusherUnsupported is returned by NewHTTPWriter when the wrapped
// http.ResponseWriter does not implement http.Flusher. The handler
// should respond with a 500 instead of attempting to stream — without
// a flusher every chunk would buffer until the response closed,
// defeating the streaming contract.
//
// This is the error class that escaped detection until 2026-04-28:
// the production middleware wrapper dropped Flusher, but every
// handler test used httptest.ResponseRecorder which natively
// implements Flusher. With NewHTTPWriter as the single Flusher
// assertion site, the missing pass-through becomes a single
// regression-testable failure mode.
var ErrFlusherUnsupported = errors.New("sse: response writer does not support http.Flusher")

// ErrAlreadyClosed is returned by Write* methods after Close has been
// called. The handler contract is that Close emits the trailing
// `event: end` frame; writing more frames after that would either
// produce out-of-order traffic on the wire or be silently dropped.
// Returning a typed error makes the contract structural.
var ErrAlreadyClosed = errors.New("sse: writer already closed")

// Writer is the single SSE emission seam used by the handlers.
//
// Contract:
//   - WriteData and WriteEvent are independently callable any number
//     of times before Close, in any interleaving.
//   - Close emits a single `event: end` frame and is idempotent —
//     subsequent calls are no-ops.
//   - All Write* calls after Close return ErrAlreadyClosed.
//   - Flush forces the underlying transport to push buffered bytes
//     to the wire. After Close, Flush is a no-op (Close itself
//     flushes on the way out).
type Writer interface {
	// WriteData writes a default-event (`message`) frame whose payload
	// is `data`. Multi-line payloads are encoded per SSE spec — the
	// parser round-trips the original bytes (preserving embedded
	// newlines).
	WriteData(data []byte) error

	// WriteEvent writes a named-event frame. `name` becomes the value
	// of the `event:` field; `data` is encoded the same way as
	// WriteData. An empty `name` is treated as the default event,
	// matching the spec.
	WriteEvent(name string, data []byte) error

	// Flush forces buffered bytes onto the wire. Safe to call any
	// number of times; the Write* methods do not flush automatically.
	Flush() error

	// Close emits the final `event: end\ndata: stream closed\n\n`
	// frame, flushes, and marks the writer closed. Idempotent.
	Close() error
}

// HTTPWriter is the production Writer implementation. It owns the
// SSE response headers, the Flusher assertion, and the write
// deadline reset for the lifetime of the stream.
//
// One HTTPWriter per HTTP request — instances are not safe for
// concurrent use across goroutines. Concurrent writers should
// serialize through a channel before calling into the writer.
type HTTPWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher

	mu     sync.Mutex
	closed bool
}

// NewHTTPWriter constructs an HTTPWriter that streams to `w`.
//
// On success the SSE response headers (`Content-Type`, `Cache-Control`,
// `Connection`, `X-Accel-Buffering`) are set on `w` and the
// per-response write deadline is cleared so long-running streams
// outlive http.Server.WriteTimeout. On failure (`w` does not
// implement http.Flusher) ErrFlusherUnsupported is returned without
// mutating `w` — callers can still write a non-SSE error response.
func NewHTTPWriter(w http.ResponseWriter) (*HTTPWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, ErrFlusherUnsupported
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// X-Accel-Buffering: no disables nginx response buffering — without
	// this, agents behind nginx receive the entire stream in one shot
	// at end-of-process, breaking the streaming contract.
	w.Header().Set("X-Accel-Buffering", "no")

	// Clear the per-response write deadline. http.Server's default
	// WriteTimeout (30s in main.go) would otherwise terminate any
	// stream that runs longer than that — surfacing as
	// SANDBOX_NO_EXIT_INFO on the agent-manager side because the
	// `event: exit` frame never made it.
	if rc := http.NewResponseController(w); rc != nil {
		_ = rc.SetWriteDeadline(time.Time{})
	}

	return &HTTPWriter{w: w, flusher: flusher}, nil
}

// WriteData implements Writer.
func (h *HTTPWriter) WriteData(data []byte) error {
	return h.writeFrame("", data)
}

// WriteEvent implements Writer.
func (h *HTTPWriter) WriteEvent(name string, data []byte) error {
	return h.writeFrame(name, data)
}

// Flush implements Writer. After Close, Flush is a no-op (Close
// itself flushes on the way out).
func (h *HTTPWriter) Flush() error {
	h.mu.Lock()
	closed := h.closed
	h.mu.Unlock()
	if closed {
		return nil
	}
	h.flusher.Flush()
	return nil
}

// Close implements Writer. The trailing `event: end` frame carries
// `data: stream closed` for backwards compatibility with the
// pre-Phase-5 wire shape; agent-manager and the in-tree parser ignore
// the data of an `end` frame.
func (h *HTTPWriter) Close() error {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil
	}
	h.closed = true
	h.mu.Unlock()

	if _, err := io.WriteString(h.w, "event: end\ndata: stream closed\n\n"); err != nil {
		return err
	}
	h.flusher.Flush()
	return nil
}

// writeFrame is the common path for WriteData and WriteEvent.
func (h *HTTPWriter) writeFrame(event string, data []byte) error {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return ErrAlreadyClosed
	}
	h.mu.Unlock()

	frame := encodeFrame(event, data)
	if _, err := h.w.Write(frame); err != nil {
		return err
	}
	h.flusher.Flush()
	return nil
}

// encodeFrame builds the wire bytes for one SSE frame. Exposed as a
// package-private helper so tests can compare encoded output without
// going through an http.ResponseWriter.
//
// Encoding rules (matching the spec):
//
//   - If event is non-empty, prepend `event: <name>\n`.
//   - The data payload is split on `\n`; each segment becomes its own
//     `data: <segment>\n` line. Empty payload still emits `data: \n`
//     so the frame dispatches.
//   - A trailing blank `\n` terminates the frame.
//
// Round-trip property (verified in sse_test.go):
//
//	ParseStream(encodeFrame("e", d)) == [{Event: "e", Data: d}]
//
// for every byte slice `d`, including ones with embedded newlines.
func encodeFrame(event string, data []byte) []byte {
	var buf bytes.Buffer
	if event != "" {
		buf.WriteString("event: ")
		buf.WriteString(event)
		buf.WriteByte('\n')
	}
	if len(data) == 0 {
		buf.WriteString("data: \n")
	} else {
		buf.WriteString("data: ")
		for _, b := range data {
			if b == '\n' {
				buf.WriteString("\ndata: ")
			} else {
				buf.WriteByte(b)
			}
		}
		buf.WriteByte('\n')
	}
	buf.WriteByte('\n')
	return buf.Bytes()
}

// Compile-time check: HTTPWriter satisfies Writer.
var _ Writer = (*HTTPWriter)(nil)
