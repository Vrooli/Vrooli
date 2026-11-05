package app

import (
	"context"
	"testing"
	"time"
)

func TestBackgroundWorkersQuiesceAndResume(t *testing.T) {
	started := make(chan struct{}, 2)
	stopped := make(chan struct{}, 2)
	workers := newBackgroundWorkers(func(ctx context.Context) {
		started <- struct{}{}
		<-ctx.Done()
		stopped <- struct{}{}
	})

	for cycle := 0; cycle < 2; cycle++ {
		if err := workers.Start(context.Background()); err != nil {
			t.Fatalf("Start() cycle %d: %v", cycle, err)
		}
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("worker did not start")
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := workers.Stop(ctx)
		cancel()
		if err != nil {
			t.Fatalf("Stop() cycle %d: %v", cycle, err)
		}
		select {
		case <-stopped:
		case <-time.After(time.Second):
			t.Fatal("worker did not stop")
		}
	}
}

func TestBackgroundWorkersStopHonorsDeadline(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	workers := newBackgroundWorkers(func(context.Context) {
		close(started)
		<-release
	})
	if err := workers.Start(context.Background()); err != nil {
		t.Fatalf("Start(): %v", err)
	}
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if err := workers.Stop(ctx); err == nil {
		t.Fatal("Stop() succeeded while a worker ignored cancellation")
	}
	close(release)
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := workers.Stop(ctx); err != nil {
		t.Fatalf("Stop() after worker exit: %v", err)
	}
}
