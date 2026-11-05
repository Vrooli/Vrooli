package services

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestServiceLoopStartStopRestartLifecycle(t *testing.T) {
	var loop serviceLoop
	var starts atomic.Int32
	started := make(chan struct{}, 2)
	releaseFirst := make(chan struct{})

	run := func(ctx context.Context) {
		start := starts.Add(1)
		started <- struct{}{}
		if start == 1 {
			<-releaseFirst
			return
		}
		<-ctx.Done()
	}

	if !loop.Start(run) {
		t.Fatal("first Start() = false, want true")
	}
	<-started
	if loop.Start(run) {
		t.Fatal("duplicate Start() = true, want false")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := loop.Stop(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Stop(canceled context) error = %v, want context.Canceled", err)
	}
	if loop.Start(run) {
		t.Fatal("Start() during stopping generation = true, want false")
	}
	close(releaseFirst)
	if err := loop.Stop(context.Background()); err != nil {
		t.Fatalf("Stop(join generation) error = %v", err)
	}

	if !loop.Start(run) {
		t.Fatal("Start() after joined generation = false, want true")
	}
	<-started
	if err := loop.Stop(context.Background()); err != nil {
		t.Fatalf("Stop(restarted generation) error = %v", err)
	}
	if got := starts.Load(); got != 2 {
		t.Fatalf("run starts = %d, want 2", got)
	}
}
