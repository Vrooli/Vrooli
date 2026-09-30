package runtime

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestBackgroundCoordinatorStartsJobsOnceWithServingContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{}, 1)
	stopped := make(chan struct{}, 1)
	var calls atomic.Int32
	coordinator := NewBackgroundCoordinator(func(jobCtx context.Context) {
		calls.Add(1)
		started <- struct{}{}
		<-jobCtx.Done()
		stopped <- struct{}{}
	})
	coordinator.Start(ctx)
	coordinator.Start(ctx)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("background job did not start")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("job calls = %d, want 1", got)
	}
	cancel()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("background job did not receive serving cancellation")
	}
}
