package asyncpool

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestPoolRunsQueuedJobsAndShutsDown(t *testing.T) {
	pool := New[int](2, 2)
	completed := make(chan struct{}, 2)
	var sum atomic.Int64
	pool.Bind(context.Background(), func(_ context.Context, job int) {
		sum.Add(int64(job))
		completed <- struct{}{}
	})
	pool.Submit(2)
	pool.Submit(3)
	for range 2 {
		select {
		case <-completed:
		case <-time.After(time.Second):
			t.Fatal("queued job did not complete")
		}
	}
	if err := pool.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := sum.Load(); got != 5 {
		t.Fatalf("sum = %d, want 5", got)
	}
}

func TestPoolShutdownHonorsCallerDeadline(t *testing.T) {
	pool := New[int](1, 1)
	started := make(chan struct{})
	release := make(chan struct{})
	pool.Bind(context.Background(), func(ctx context.Context, _ int) {
		_ = ctx
		close(started)
		<-release
	})
	pool.Submit(1)
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if err := pool.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown error = %v, want deadline exceeded", err)
	}
	close(release)
	if err := pool.Shutdown(context.Background()); err != nil {
		t.Fatalf("final shutdown = %v", err)
	}
}
