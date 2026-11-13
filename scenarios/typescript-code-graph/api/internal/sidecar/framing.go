package sidecar

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
// trailing newline, one frame at a time so concurrent request methods
// cannot interleave bytes on the child's stdin.
//
// Every write is bounded by a context. A child whose event loop is stuck in
// synchronous work stops reading stdin; once the pipe fills, an unbounded
// write blocks forever, and with it the caller's deadline and the heartbeat
// watchdog that should replace the child.
type frameWriter struct {
	// slot has capacity 1; holding it grants exclusive use of w. A channel
	// rather than a mutex so waiting for it can observe a context.
	slot chan struct{}
	w    io.Writer
}

func newFrameWriter(w io.Writer) *frameWriter {
	return &frameWriter{slot: make(chan struct{}, 1), w: w}
}

// Write marshals v as JSON, appends "\n", and writes the frame with a single
// Write call. It returns ctx's error if the frame cannot be handed over
// before ctx ends. A frame that has started writing still completes (or
// fails with the pipe) in the background, so giving up never leaves a
// partial line that would corrupt framing for later writers.
func (f *frameWriter) Write(ctx context.Context, v any) error {
	buf, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("frame marshal: %w", err)
	}
	buf = append(buf, '\n')

	select {
	case f.slot <- struct{}{}:
	case <-ctx.Done():
		return fmt.Errorf("frame write: %w", ctx.Err())
	}
	done := make(chan error, 1)
	go func() {
		_, err := f.w.Write(buf)
		<-f.slot
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("frame write: %w", err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("frame write: %w", ctx.Err())
	}
}
