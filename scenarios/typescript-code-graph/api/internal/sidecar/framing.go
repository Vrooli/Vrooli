package sidecar

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// maxLineBytes caps the per-message size on read. The wire protocol is
// line-delimited JSON; ts-morph can emit very large extracted graphs,
// so the cap is generous. 64 MiB is the practical ceiling for a
// single project extract. The cap is deliberately above the largest known
// real-world graph: the supervisor must report a typed transport failure when
// this boundary is crossed rather than silently abandoning its stdout reader.
const maxLineBytes = 256 * 1024 * 1024

// newFrameScanner constructs a bufio.Scanner with the per-message size
// raised. The default 64 KiB buffer is far too small for an extracted
// graph payload.
func newFrameScanner(r io.Reader) *bufio.Scanner {
	s := bufio.NewScanner(r)
	// Start small, grow to maxLineBytes on demand.
	s.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
	return s
}

// frameWriter serializes JSON-encodable values onto an io.Writer with a
// trailing newline. Writes are mutex-guarded so concurrent request
// methods cannot interleave bytes on the child's stdin.
type frameWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func newFrameWriter(w io.Writer) *frameWriter {
	return &frameWriter{w: w}
}

// Write marshals v as JSON, appends "\n", and writes atomically.
func (f *frameWriter) Write(v any) error {
	buf, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("frame marshal: %w", err)
	}
	// Append newline once so the underlying writer sees one Write call.
	out := bytes.NewBuffer(buf)
	if err := out.WriteByte('\n'); err != nil {
		return fmt.Errorf("frame buffer: %w", err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, err := f.w.Write(out.Bytes()); err != nil {
		return fmt.Errorf("frame write: %w", err)
	}
	return nil
}
