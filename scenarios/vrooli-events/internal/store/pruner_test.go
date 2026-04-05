package store

import (
	"context"
	"testing"
	"time"
)

func TestStartPrunerCancellation(t *testing.T) {
	s := newTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		StartPruner(ctx, PrunerConfig{
			Interval: 50 * time.Millisecond,
			Store:    s,
		})
		close(done)
	}()

	// Let it run a couple cycles
	time.Sleep(150 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// OK - pruner stopped
	case <-time.After(2 * time.Second):
		t.Fatal("pruner did not stop after context cancellation")
	}
}
