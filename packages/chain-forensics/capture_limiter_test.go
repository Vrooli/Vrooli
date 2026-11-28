package chainforensics

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEndpointRateLimiterSharesOriginAndCancelsWaiters(t *testing.T) {
	limiter := NewEndpointRateLimiter(2)
	first, err := limiter.Acquire(context.Background(), "https://rpc.example.test/eth", 0)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := limiter.Acquire(ctx, "https://rpc.example.test/solana", 0); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("same provider origin bypassed the shared slot: %v", err)
	}
	first()

	second, err := limiter.Acquire(context.Background(), "https://rpc.example.test/solana", 0)
	if err != nil {
		t.Fatalf("cancelled waiter left the provider gate occupied: %v", err)
	}
	second()
}

func TestEndpointRateLimiterBoundsProviderRegistryAndEvictsIdleGate(t *testing.T) {
	limiter := NewEndpointRateLimiter(1)
	first, err := limiter.Acquire(context.Background(), "https://first.example.test/rpc", 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := limiter.Acquire(context.Background(), "https://second.example.test/rpc", 0); !errors.Is(err, ErrCaptureLimiterCapacity) {
		t.Fatalf("expected bounded provider-registry refusal, got %v", err)
	}
	first()
	second, err := limiter.Acquire(context.Background(), "https://second.example.test/rpc", 0)
	if err != nil {
		t.Fatalf("idle endpoint gate was not evicted at capacity: %v", err)
	}
	second()
}

func TestEndpointRateLimiterRejectsMalformedEndpoint(t *testing.T) {
	limiter := NewEndpointRateLimiter(1)
	if _, err := limiter.Acquire(context.Background(), "/relative/path", 0); err == nil {
		t.Fatal("relative capture endpoint was accepted")
	}
	if _, err := limiter.Acquire(context.Background(), "https://example.test/rpc", -time.Second); err != nil {
		t.Fatalf("negative minimum interval should clamp to zero: %v", err)
	}
}

func TestEndpointRateLimiterSharesProviderRetryWindowAcrossPaths(t *testing.T) {
	limiter := NewEndpointRateLimiter(2)
	endpoint := "https://rpc.example.test/solana"
	release, err := limiter.Acquire(context.Background(), endpoint, 0)
	if err != nil {
		t.Fatal(err)
	}
	limiter.DeferUntil(endpoint, time.Now().Add(100*time.Millisecond))
	release()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := limiter.Acquire(ctx, "https://rpc.example.test/eth", 0); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("different path bypassed provider cooldown: %v", err)
	}
}

func TestEndpointRateLimiterDoesNotEvictActiveRetryWindow(t *testing.T) {
	limiter := NewEndpointRateLimiter(1)
	release, err := limiter.Acquire(context.Background(), "https://first.example.test/rpc", 0)
	if err != nil {
		t.Fatal(err)
	}
	limiter.DeferUntil("https://first.example.test/rpc", time.Now().Add(50*time.Millisecond))
	release()
	if _, err := limiter.Acquire(context.Background(), "https://second.example.test/rpc", 0); !errors.Is(err, ErrCaptureLimiterCapacity) {
		t.Fatalf("active retry window was evicted: %v", err)
	}
	time.Sleep(60 * time.Millisecond)
	second, err := limiter.Acquire(context.Background(), "https://second.example.test/rpc", 0)
	if err != nil {
		t.Fatalf("expired retry window was not eligible for eviction: %v", err)
	}
	second()
}
