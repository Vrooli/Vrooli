package ttschain

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBackendEnsurerSingleFlightsConcurrentStartsAndCachesResult(t *testing.T) {
	var starts atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	e := NewBackendEnsurerForTest(func(ctx context.Context, resource string) error {
		if resource != "kale-tts" {
			t.Errorf("resource = %q", resource)
		}
		if starts.Add(1) == 1 {
			close(entered)
		}
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}, "kale-tts")
	var wg sync.WaitGroup
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := e.EnsureRunning(context.Background(), "kale-tts"); err != nil {
				t.Errorf("ensure: %v", err)
			}
		}()
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("start did not begin")
	}
	close(release)
	wg.Wait()
	if got := starts.Load(); got != 1 {
		t.Fatalf("starts = %d, want 1", got)
	}
	if err := e.EnsureRunning(context.Background(), "kale-tts"); err != nil {
		t.Fatal(err)
	}
	if got := starts.Load(); got != 1 {
		t.Fatalf("cooldown caused %d starts", got)
	}
}

func TestBackendEnsurerRejectsUndeclaredResource(t *testing.T) {
	e := NewBackendEnsurerForTest(func(context.Context, string) error { return nil }, "kale-tts", "kokoro")
	if err := e.EnsureRunning(context.Background(), "whisper"); err == nil {
		t.Fatal("expected undeclared resource rejection")
	}
}

func TestBackendEnsurerUsesInjectedResourceAllowlist(t *testing.T) {
	var starts []string
	e := NewBackendEnsurerForTest(func(_ context.Context, resource string) error {
		starts = append(starts, resource)
		return nil
	}, "declared-new-tts")
	if err := e.EnsureRunning(context.Background(), "declared-new-tts"); err != nil {
		t.Fatalf("declared resource should start: %v", err)
	}
	if err := e.EnsureRunning(context.Background(), "unconfigured-tts"); err == nil {
		t.Fatal("unconfigured resource should be rejected")
	}
	if len(starts) != 1 || starts[0] != "declared-new-tts" {
		t.Fatalf("started resources = %v, want only declared-new-tts", starts)
	}
}

func TestBackendEnsurerBoundsStartsAcrossResources(t *testing.T) {
	entered := make(chan string, 2)
	release := make(chan struct{})
	var active, peak, starts atomic.Int32
	e := NewBackendEnsurerForTest(func(ctx context.Context, resource string) error {
		starts.Add(1)
		current := active.Add(1)
		for {
			old := peak.Load()
			if current <= old || peak.CompareAndSwap(old, current) {
				break
			}
		}
		entered <- resource
		defer active.Add(-1)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}, "kale-tts", "kokoro")

	var wg sync.WaitGroup
	for _, resource := range []string{"kale-tts", "kokoro"} {
		wg.Add(1)
		go func(resource string) {
			defer wg.Done()
			if err := e.EnsureRunning(context.Background(), resource); err != nil {
				t.Errorf("ensure %s: %v", resource, err)
			}
		}(resource)
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("first start did not begin")
	}
	// Let the second request reach the serialized start slot before releasing.
	time.Sleep(20 * time.Millisecond)
	if got := starts.Load(); got != 1 {
		t.Fatalf("concurrent starts entered = %d, want 1", got)
	}
	close(release)
	wg.Wait()
	if got := peak.Load(); got != 1 {
		t.Fatalf("peak simultaneous starts = %d, want 1", got)
	}
}

func TestBackendEnsurerCanceledWaitDoesNotCacheCooldown(t *testing.T) {
	entered := make(chan string, 2)
	releaseFirst := make(chan struct{})
	var kokoroStarts atomic.Int32
	e := NewBackendEnsurerForTest(func(ctx context.Context, resource string) error {
		if resource == "kokoro" {
			kokoroStarts.Add(1)
		}
		entered <- resource
		if resource == "kale-tts" {
			select {
			case <-releaseFirst:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}, "kale-tts", "kokoro")

	firstDone := make(chan error, 1)
	go func() { firstDone <- e.EnsureRunning(context.Background(), "kale-tts") }()
	select {
	case resource := <-entered:
		if resource != "kale-tts" {
			t.Fatalf("first start resource = %q", resource)
		}
	case <-time.After(time.Second):
		t.Fatal("Kale start did not begin")
	}

	ctx, cancel := context.WithCancel(context.Background())
	secondDone := make(chan error, 1)
	go func() { secondDone <- e.EnsureRunning(ctx, "kokoro") }()
	deadline := time.Now().Add(time.Second)
	for {
		e.mu.Lock()
		_, waiting := e.inflight["kokoro"]
		e.mu.Unlock()
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			close(releaseFirst)
			t.Fatal("Kokoro request did not enter the in-flight set")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	if err := <-secondDone; err == nil {
		t.Fatal("canceled Kokoro request succeeded")
	}
	close(releaseFirst)
	if err := <-firstDone; err != nil {
		t.Fatalf("Kale start: %v", err)
	}

	if err := e.EnsureRunning(context.Background(), "kokoro"); err != nil {
		t.Fatalf("retry after canceled wait: %v", err)
	}
	if got := kokoroStarts.Load(); got != 1 {
		t.Fatalf("Kokoro starts = %d, want immediate retry to start once", got)
	}
}

func TestBackendEnsurerCanceledLeaderDoesNotCacheCooldown(t *testing.T) {
	entered := make(chan struct{})
	var starts atomic.Int32
	e := NewBackendEnsurerForTest(func(ctx context.Context, _ string) error {
		if starts.Add(1) == 1 {
			close(entered)
			<-ctx.Done()
			return ctx.Err()
		}
		return nil
	}, "kale-tts")

	ctx, cancel := context.WithCancel(context.Background())
	firstDone := make(chan error, 1)
	go func() { firstDone <- e.EnsureRunning(ctx, "kale-tts") }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("start did not begin")
	}
	cancel()
	if err := <-firstDone; err == nil {
		t.Fatal("canceled start succeeded")
	}
	if err := e.EnsureRunning(context.Background(), "kale-tts"); err != nil {
		t.Fatalf("retry after canceled leader: %v", err)
	}
	if got := starts.Load(); got != 2 {
		t.Fatalf("start attempts = %d, want canceled attempt plus immediate retry", got)
	}
}
