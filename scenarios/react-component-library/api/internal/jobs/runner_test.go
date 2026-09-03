package jobs

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestAcquireHonorsCancellationWhileBusy(t *testing.T) {
	runner := New(nil)
	if err := runner.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer runner.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := runner.Acquire(ctx); err == nil {
		t.Fatal("Acquire succeeded while the matrix slot was occupied")
	}
}

func TestRunMatrixHonorsCancellationFromWork(t *testing.T) {
	runner := New(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var calls atomic.Int32
	err := runner.RunMatrix(ctx, 4, 2, func(ctx context.Context, _ int) error {
		calls.Add(1)
		cancel()
		return ctx.Err()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunMatrix error = %v, want context.Canceled", err)
	}
	if calls.Load() == 0 {
		t.Fatal("RunMatrix did not invoke work")
	}
}

func TestRunMatrixRunsEveryIndex(t *testing.T) {
	runner := New(nil)
	var calls atomic.Int32
	if err := runner.RunMatrix(context.Background(), 5, 2, func(context.Context, int) error {
		calls.Add(1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if got := calls.Load(); got != 5 {
		t.Fatalf("RunMatrix calls = %d, want 5", got)
	}
}

func TestRunMatrixStopsProducerWhenWorkerFails(t *testing.T) {
	runner := New(nil)
	want := errors.New("gate failed")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := runner.RunMatrix(ctx, 100, 2, func(_ context.Context, index int) error {
		if index == 0 {
			return want
		}
		return nil
	})
	if !errors.Is(err, want) {
		t.Fatalf("RunMatrix error = %v, want %v", err, want)
	}
}

func TestAcquireReleaseAllowsNextMatrix(t *testing.T) {
	runner := New(nil)
	if err := runner.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	runner.Release()
	if err := runner.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	runner.Release()
}
