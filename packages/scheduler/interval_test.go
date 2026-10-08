package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunnerWaitsFromEndAndReportsOverrun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var active int32
	var overlap int32
	done := make(chan Cycle, 2)
	r := New(10*time.Millisecond, func(context.Context) error {
		if atomic.AddInt32(&active, 1) != 1 {
			atomic.StoreInt32(&overlap, 1)
		}
		time.Sleep(25 * time.Millisecond)
		atomic.AddInt32(&active, -1)
		return nil
	}).Observe(func(c Cycle) {
		done <- c
		if len(done) == 2 {
			cancel()
		}
	})
	r.Start(ctx)
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("first cycle did not run")
	}
	first := <-done
	if !first.Overran || atomic.LoadInt32(&overlap) != 0 {
		t.Fatalf("runner overlap=%d first=%+v", overlap, first)
	}
}
