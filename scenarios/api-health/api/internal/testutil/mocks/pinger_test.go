package mocks

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestFakePinger_DefaultPingNil(t *testing.T) {
	var p FakePinger
	if err := p.PingContext(context.Background()); err != nil {
		t.Fatalf("default PingContext should be nil, got %v", err)
	}
	if got := p.Calls.Load(); got != 1 {
		t.Fatalf("Calls = %d, want 1", got)
	}
}

func TestFakePinger_PingErrSurfaces(t *testing.T) {
	want := errors.New("db down")
	p := &FakePinger{PingErr: want}
	if err := p.PingContext(context.Background()); !errors.Is(err, want) {
		t.Fatalf("PingContext = %v, want %v", err, want)
	}
}

func TestFakePinger_CallsCounted(t *testing.T) {
	var p FakePinger
	for i := 0; i < 3; i++ {
		_ = p.PingContext(context.Background())
	}
	if got := p.Calls.Load(); got != 3 {
		t.Fatalf("Calls = %d, want 3", got)
	}
}

// TestFakePinger_RaceCleanWhenSharedAcrossGoroutines is the load-bearing
// regression test for the atomic.Int64 conversion. Run with `go test
// -race`; with a plain int counter this test trips the race detector
// the moment two goroutines write the field. With atomic.Int64 the
// final count is exactly N every time.
func TestFakePinger_RaceCleanWhenSharedAcrossGoroutines(t *testing.T) {
	t.Parallel()
	const goroutines = 100
	var p FakePinger
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = p.PingContext(context.Background())
		}()
	}
	wg.Wait()
	if got := p.Calls.Load(); got != goroutines {
		t.Fatalf("Calls = %d, want %d", got, goroutines)
	}
}
