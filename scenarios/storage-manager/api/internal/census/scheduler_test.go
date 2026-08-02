package census

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerRunOnceDelegatesWithoutStartingAHostScan(t *testing.T) {
	var calls atomic.Int32
	scheduler := NewScheduler(time.Hour, func(context.Context) error {
		calls.Add(1)
		return nil
	})
	if err := scheduler.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("calls = %d, want 1", got)
	}
}

func TestSchedulerWaitsForItsIntervalAndStopsWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	called := make(chan struct{}, 1)
	scheduler := NewScheduler(10*time.Millisecond, func(context.Context) error {
		called <- struct{}{}
		return nil
	})
	scheduler.Start(ctx)
	select {
	case <-called:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("scheduler did not run after its interval")
	}
	cancel()
}
