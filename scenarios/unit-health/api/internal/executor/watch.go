package executor

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

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
	if w.overflow {
		return "...[truncated]...\n" + string(w.buf)
	}
	return string(w.buf)
}

// watchNoOutput cancels via cancel() if none of the writers produce output for
// the given duration. It returns a flag set true when it fired.
func watchNoOutput(ctx context.Context, cancel context.CancelFunc, writers []*tailWriter, timeout time.Duration) *atomic.Bool {
	fired := &atomic.Bool{}
	ticker := time.NewTicker(time.Second)
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
