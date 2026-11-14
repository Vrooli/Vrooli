package backends

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAvailabilityCoalescesConcurrentLoadsAndDoesNotPinFailedProvisioning(t *testing.T) {
	cache := AvailabilityCache{TTL: time.Minute}
	var calls atomic.Int32
	failed := func() Availability { calls.Add(1); return Availability{Detail: "not installed"} }
	cache.Do(context.Background(), "venv", failed)
	cache.Do(context.Background(), "venv", failed)
	if calls.Load() != 2 {
		t.Fatal("failure concealed runtime repair")
	}
	calls.Store(0)
	var jobs sync.WaitGroup
	for range 20 {
		jobs.Go(func() {
			value := cache.Do(context.Background(), "venv", func() Availability { calls.Add(1); time.Sleep(time.Millisecond); return Availability{Available: true} })
			if !value.Available {
				t.Error("ready runtime unavailable")
			}
		})
	}
	jobs.Wait()
	if calls.Load() != 1 {
		t.Fatalf("20 requests constructed %d probes", calls.Load())
	}
	cache.Do(context.Background(), "replacement-venv", func() Availability { calls.Add(1); return Availability{Available: true} })
	if calls.Load() != 2 {
		t.Fatal("interpreter identity change retained old evidence")
	}
}

func TestAvailabilityWaiterCanCancelWithoutInterruptingAnotherProbe(t *testing.T) {
	cache := AvailabilityCache{TTL: time.Minute}
	entered, finish, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		cache.Do(context.Background(), "venv", func() Availability { close(entered); <-finish; return Availability{Available: true} })
	}()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	value := cache.Do(ctx, "venv", func() Availability { t.Error("canceled waiter started a probe"); return Availability{} })
	if value.Available {
		t.Fatal("canceled wait advertised availability")
	}
	close(finish)
	<-done
}
