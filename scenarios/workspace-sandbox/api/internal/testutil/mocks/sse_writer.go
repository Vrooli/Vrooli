package mocks

import (
	"sync"

	"workspace-sandbox/internal/sse"
)

// FakeSSEWriter is the canonical sse.Writer fake. Tests that take a
// handler/service through an injected writer use this instead of an
// httptest.ResponseRecorder so they can assert on the structured
// frame sequence rather than re-parsing the wire bytes.
//
// The fake enforces the same Close-once contract as HTTPWriter:
// writes after Close return sse.ErrAlreadyClosed; Close is
// idempotent. That keeps fake-driven tests honest about the same
// invariants the production writer enforces.
type FakeSSEWriter struct {
	mu     sync.Mutex
	frames []FakeSSEFrame
	closed bool

	// Per-method error overrides — tests set these to drive the
	// failure paths in the consumer (e.g., the connection drops mid
	// stream, or Flush surfaces a typed error). When non-nil the
	// matching method returns the error after recording the call.
	WriteDataErr  error
	WriteEventErr error
	FlushErr      error
	CloseErr      error
}

// FakeSSEFrame mirrors the wire-level frame shape the writer produces,
// but stays in-process so tests can assert directly on it.
type FakeSSEFrame struct {
	// Event is empty for default-event (`message`) frames, "exit",
	// "error", "end", etc. otherwise.
	Event string
	// Data is a copy of the bytes the caller passed to WriteData /
	// WriteEvent — the fake never mutates the caller's slice.
	Data []byte
	// Flushed is true when the fake observed the Flush that the
	// writer guarantees after every successful write. Exposes the
	// invariant that the production writer does not buffer.
	Flushed bool
}

// NewFakeSSEWriter returns an empty fake. Tests typically set error
// fields after construction.
func NewFakeSSEWriter() *FakeSSEWriter {
	return &FakeSSEWriter{}
}

// WriteData implements sse.Writer.
func (f *FakeSSEWriter) WriteData(data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return sse.ErrAlreadyClosed
	}
	if f.WriteDataErr != nil {
		return f.WriteDataErr
	}
	f.frames = append(f.frames, FakeSSEFrame{
		Event:   "",
		Data:    append([]byte(nil), data...),
		Flushed: true,
	})
	return nil
}

// WriteEvent implements sse.Writer.
func (f *FakeSSEWriter) WriteEvent(name string, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return sse.ErrAlreadyClosed
	}
	if f.WriteEventErr != nil {
		return f.WriteEventErr
	}
	f.frames = append(f.frames, FakeSSEFrame{
		Event:   name,
		Data:    append([]byte(nil), data...),
		Flushed: true,
	})
	return nil
}

// Flush implements sse.Writer.
func (f *FakeSSEWriter) Flush() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return nil
	}
	if f.FlushErr != nil {
		return f.FlushErr
	}
	return nil
}

// Close implements sse.Writer. Idempotent: a second Close is a no-op
// even if CloseErr is set.
func (f *FakeSSEWriter) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return nil
	}
	f.closed = true
	if f.CloseErr != nil {
		return f.CloseErr
	}
	f.frames = append(f.frames, FakeSSEFrame{
		Event:   "end",
		Data:    []byte("stream closed"),
		Flushed: true,
	})
	return nil
}

// Frames returns a copy of every frame recorded (in order). Safe to
// call concurrently with writers — the returned slice is a snapshot.
func (f *FakeSSEWriter) Frames() []FakeSSEFrame {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]FakeSSEFrame, len(f.frames))
	copy(out, f.frames)
	return out
}

// Closed reports whether Close has been called.
func (f *FakeSSEWriter) Closed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

// Compile-time check.
var _ sse.Writer = (*FakeSSEWriter)(nil)
