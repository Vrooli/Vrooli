package findings_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"web-search/internal/findings"

	"github.com/stretchr/testify/require"
)

// capturingWriter records the batches the recorder flushes.
type capturingWriter struct {
	mu      sync.Mutex
	batches [][]string
}

func (w *capturingWriter) RecordSurfaced(_ context.Context, ids []string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.batches = append(w.batches, ids)
	return nil
}

func (w *capturingWriter) total() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := 0
	for _, b := range w.batches {
		n += len(b)
	}
	return n
}

// TestUsageRecorderDrainsAsync proves Surfaced enqueues without blocking and the
// background Run drains every batch to the writer.
func TestUsageRecorderDrainsAsync(t *testing.T) {
	w := &capturingWriter{}
	rec := findings.NewUsageRecorder(w, 64, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go rec.Run(ctx)

	rec.Surfaced([]string{"a", "b"})
	rec.Surfaced([]string{"c"})

	require.Eventually(t, func() bool { return w.total() == 3 }, 2*time.Second, 5*time.Millisecond)
}

// TestUsageRecorderSurfacedIsNonBlocking proves a full buffer drops events
// rather than blocking the caller (the search hot path must never stall).
func TestUsageRecorderSurfacedIsNonBlocking(t *testing.T) {
	// No Run goroutine: the buffer fills and excess is dropped, but Surfaced
	// never blocks. With buffer 2, the 3rd+ enqueues are dropped silently.
	rec := findings.NewUsageRecorder(&capturingWriter{}, 2, nil)
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			rec.Surfaced([]string{"x"})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Surfaced blocked on a full buffer")
	}
}

// TestUsageRecorderNilAndEmptyAreNoOps guards the defensive paths.
func TestUsageRecorderNilAndEmptyAreNoOps(t *testing.T) {
	var rec *findings.UsageRecorder
	require.NotPanics(t, func() { rec.Surfaced([]string{"a"}) })

	w := &capturingWriter{}
	rec = findings.NewUsageRecorder(w, 8, nil)
	require.NotPanics(t, func() { rec.Surfaced(nil) })
	require.Equal(t, 0, w.total())
}
