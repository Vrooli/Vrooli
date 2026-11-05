package executor

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// boundedCapture preserves exact bytes or declares the entire capture
// incomplete. It does not interpret framework output or interrupt a process
// merely because its optional evidence exceeds the collection budget.
type boundedCapture struct {
	mu       sync.Mutex
	buf      []byte
	max      int
	overflow bool
}

func (w *boundedCapture) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.overflow {
		if len(p) > w.max-len(w.buf) {
			w.overflow = true
			w.buf = nil
		} else {
			w.buf = append(w.buf, p...)
		}
	}
	return len(p), nil
}
func (w *boundedCapture) snapshot() ([]byte, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.overflow {
		return nil, false
	}
	return append([]byte{}, w.buf...), true
}

// tailWriter keeps the last n bytes written to it and records the time of the
// most recent write so the watchdog can detect a stall.
type tailWriter struct {
	mu       sync.Mutex
	buf      []byte
	max      int
	lastWro  time.Time
	started  time.Time
	overflow bool
}

func newTailWriter(max int) *tailWriter {
	now := time.Now()
	return &tailWriter{max: max, lastWro: now, started: now}
}

func (w *tailWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lastWro = time.Now()
	w.buf = append(w.buf, p...)
	if len(w.buf) > w.max {
		w.overflow = true
		w.buf = w.buf[len(w.buf)-w.max:]
	}
	return len(p), nil
}

func (w *tailWriter) lastWrite() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastWro
}

func (w *tailWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	// Output may contain arbitrary bytes, and tail truncation can split a UTF-8
	// rune. Keep protobuf evidence serializable without changing captured bytes.
	text := strings.ToValidUTF8(string(w.buf), "\uFFFD")
	if w.overflow {
		return "...[truncated]...\n" + text
	}
	return text
}

// watchNoOutput cancels via cancel() if none of the writers produce output for
// the given duration. It returns a flag set true when it fired.
func watchNoOutput(ctx context.Context, cancel context.CancelFunc, writers []*tailWriter, timeout time.Duration) *atomic.Bool {
	fired := &atomic.Bool{}
	interval := timeout / 4
	if interval < 10*time.Millisecond {
		interval = 10 * time.Millisecond
	}
	if interval > time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				last := time.Time{}
				for _, w := range writers {
					if t := w.lastWrite(); t.After(last) {
						last = t
					}
				}
				if time.Since(last) >= timeout {
					fired.Store(true)
					cancel()
					return
				}
			}
		}
	}()
	return fired
}
