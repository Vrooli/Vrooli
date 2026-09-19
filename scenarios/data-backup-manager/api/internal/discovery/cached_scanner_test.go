package discovery_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"data-backup-manager/internal/discovery"
)

type countingScanner struct {
	mu    sync.Mutex
	calls int
	ready chan struct{}
}

func (s *countingScanner) Scan(context.Context) ([]discovery.TargetCandidate, error) {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
	if s.ready != nil {
		<-s.ready
	}
	return []discovery.TargetCandidate{{Owner: "vrooli", Name: "plans", Locator: "/plans"}}, nil
}

func (s *countingScanner) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestCachedTargetSourceScannerCoalescesConcurrentScans(t *testing.T) {
	source := &countingScanner{ready: make(chan struct{})}
	cached := discovery.NewCachedTargetSourceScanner(source, time.Minute)

	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := cached.Scan(context.Background())
			if err != nil || len(got) != 1 {
				t.Errorf("Scan() = %v, %v; want one candidate", got, err)
			}
		}()
	}
	for source.callCount() != 1 {
		time.Sleep(time.Millisecond)
	}
	close(source.ready)
	wg.Wait()

	if got := source.callCount(); got != 1 {
		t.Fatalf("source calls = %d, want one", got)
	}
}

func TestCachedTargetSourceScannerRefreshesAfterTTL(t *testing.T) {
	source := &countingScanner{}
	cached := discovery.NewCachedTargetSourceScanner(source, time.Millisecond)

	if _, err := cached.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	if _, err := cached.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := source.callCount(); got != 2 {
		t.Fatalf("source calls = %d, want two after TTL", got)
	}
}
