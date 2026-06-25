package collectors

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestProcessCollector_NoSteadyForks asserts that one Collect cycle does not
// fork for process-table metrics. The steady path reads /proc directly; the
// commandOutput seam remains only for exceptional future commands.
func TestProcessCollector_NoSteadyForks(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("process collector reads /proc only on linux")
	}

	var mu sync.Mutex
	forks := 0

	orig := commandOutput
	defer func() { commandOutput = orig }()
	commandOutput = func(_ context.Context, _ time.Duration, name string, args ...string) ([]byte, error) {
		mu.Lock()
		defer mu.Unlock()
		forks++
		return []byte(""), nil
	}

	c := NewProcessCollector()
	if _, err := c.Collect(context.Background()); err != nil {
		t.Fatalf("collect: %v", err)
	}

	if forks != 0 {
		t.Errorf("process collection forked %d times, want 0", forks)
	}
}
