package lifecycle

import (
	"bytes"
	"fmt"
	"io"
)

// defaultStepRingCap caps the ring buffer to the last N output lines of a
// foreground step, drained to stderr on failure so users can diagnose build
// errors without the raw console flood.
const defaultStepRingCap = 100

// stepSink wraps the writers a foreground step streams into. Every byte
// reaches inner (typically a MultiWriter that already contains the scenario
// lifecycle log file + the console at verbose mode). The sink additionally
// keeps the last N newline-terminated lines in memory so the caller can
// replay them to stderr when the step exits non-zero.
//
// The sink is not safe for concurrent use; a single cmd.Run() writes
// stdout/stderr into it. Steps run sequentially in ExecutePhaseDetailed, so
// that suffices.
type stepSink struct {
	inner  io.Writer
	ring   *lineRing
	leftov []byte
}

func newStepSink(inner io.Writer) *stepSink {
	return &stepSink{inner: inner, ring: newLineRing(defaultStepRingCap)}
}

// Write forwards bytes to the inner writer and also appends each completed
// line to the ring buffer. Partial lines (no trailing newline) are buffered
// and stitched onto the next write; Flush emits any remaining partial line
// to the ring on close.
func (s *stepSink) Write(p []byte) (int, error) {
	if s == nil || len(p) == 0 {
		if s != nil && s.inner != nil {
			return s.inner.Write(p)
		}
		return len(p), nil
	}
	n, err := s.inner.Write(p)
	// Even when Write returns a short count, feed the accepted prefix into
	// the ring so the replay reflects what was actually forwarded.
	s.consumeForRing(p[:n])
	return n, err
}

func (s *stepSink) consumeForRing(p []byte) {
	if len(p) == 0 {
		return
	}
	buf := p
	if len(s.leftov) > 0 {
		buf = append(s.leftov, p...)
		s.leftov = nil
	}
	for {
		i := bytes.IndexByte(buf, '\n')
		if i < 0 {
			// Hold partial line for next Write.
			if len(buf) > 0 {
				s.leftov = append(s.leftov[:0], buf...)
			}
			return
		}
		line := buf[:i]
		s.ring.push(string(line))
		buf = buf[i+1:]
	}
}

// Flush emits any buffered partial line into the ring. Call on step
// completion (success or failure) to avoid losing a final line that lacked
// a trailing newline (common with crashing processes).
func (s *stepSink) Flush() {
	if s == nil {
		return
	}
	if len(s.leftov) > 0 {
		s.ring.push(string(s.leftov))
		s.leftov = nil
	}
}

// ReplayTo writes the captured tail of the step's output to w, framed by a
// header naming the step and a footer pointing at the full log path. If the
// ring is empty, ReplayTo is a no-op so we don't add noise when a step fails
// before emitting any output.
func (s *stepSink) ReplayTo(w io.Writer, stepName, logPath string) {
	if s == nil || w == nil || s.ring.empty() {
		return
	}
	lines := s.ring.snapshot()
	fmt.Fprintf(w, "--- last %d lines of %s ---\n", len(lines), stepName)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	if logPath != "" {
		fmt.Fprintf(w, "--- full log: %s ---\n", logPath)
	}
}

// lineRing is a bounded FIFO of strings. Its zero value is unusable; call
// newLineRing to construct. It favors simplicity over byte-level cap: 100
// lines × ~256 bytes ≈ 25 KB is trivially small.
type lineRing struct {
	buf   []string
	head  int
	size  int
	limit int
}

func newLineRing(limit int) *lineRing {
	if limit <= 0 {
		limit = defaultStepRingCap
	}
	return &lineRing{buf: make([]string, limit), limit: limit}
}

func (r *lineRing) push(line string) {
	if r.size < r.limit {
		r.buf[(r.head+r.size)%r.limit] = line
		r.size++
		return
	}
	r.buf[r.head] = line
	r.head = (r.head + 1) % r.limit
}

func (r *lineRing) empty() bool { return r == nil || r.size == 0 }

func (r *lineRing) snapshot() []string {
	if r == nil || r.size == 0 {
		return nil
	}
	out := make([]string, r.size)
	for i := 0; i < r.size; i++ {
		out[i] = r.buf[(r.head+i)%r.limit]
	}
	return out
}
