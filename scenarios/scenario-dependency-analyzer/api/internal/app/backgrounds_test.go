package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/graphsweeper"
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

type blockedSweepLister struct{ entered, release chan struct{} }

func (l blockedSweepLister) ListScenarios(string) ([]graphsweeper.Scenario, error) {
	close(l.entered)
	<-l.release
	return nil, nil
}

func TestBackgroundQuiescenceWaitsForGraphSweepToExit(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	sweeper := graphsweeper.New(graphsweeper.SweepConfig{Enabled: true}, nil, nil, graphsweeper.WithLister(blockedSweepLister{entered: entered, release: release}))
	service := &graphIngestService{sweeper: sweeper}
	workers := newBackgroundWorkers(service.RunSweeper)
	if err := workers.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("graph sweep never started")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := workers.Stop(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("reported quiescence before owned sweep exited: %v", err)
	}
	close(release)
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	if err := workers.Stop(ctx2); err != nil {
		t.Fatalf("sweep did not finish after release: %v", err)
	}
}
