package resilience_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-inbox/resilience"
)

func TestCircuitBreaker_StartsInClosedState(t *testing.T) {
	cb := resilience.NewCircuitBreaker(resilience.DefaultCircuitBreakerConfig())

	if cb.State() != resilience.StateClosed {
		t.Errorf("expected StateClosed, got %v", cb.State())
	}
}

func TestCircuitBreaker_AllowsRequestsWhenClosed(t *testing.T) {
	cb := resilience.NewCircuitBreaker(resilience.DefaultCircuitBreakerConfig())

	calls := 0
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		calls++
		return nil
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cfg := resilience.CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Cooldown:         1 * time.Hour, // Long cooldown to prevent auto-recovery
	}
	cb := resilience.NewCircuitBreaker(cfg)

	// Cause failures to open the circuit
	for i := 0; i < 3; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) error {
			return errors.New("failure")
		})
	}

	if cb.State() != resilience.StateOpen {
		t.Errorf("expected StateOpen after %d failures, got %v", 3, cb.State())
	}

	// Verify requests are rejected
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	if !errors.Is(err, resilience.ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_TransitionsToHalfOpenAfterCooldown(t *testing.T) {
	cfg := resilience.CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Cooldown:         10 * time.Millisecond, // Short cooldown for testing
	}
	cb := resilience.NewCircuitBreaker(cfg)

	// Open the circuit
	cb.Execute(context.Background(), func(ctx context.Context) error {
		return errors.New("failure")
	})

	if cb.State() != resilience.StateOpen {
		t.Fatalf("expected StateOpen, got %v", cb.State())
	}

	// Wait for cooldown
	time.Sleep(15 * time.Millisecond)

	// Next request should trigger half-open state
	calls := 0
	cb.Execute(context.Background(), func(ctx context.Context) error {
		calls++
		return nil
	})

	// Should have allowed the request (half-open allows one through)
	if calls != 1 {
		t.Errorf("expected 1 call in half-open state, got %d", calls)
	}

	// Should now be closed after success
	if cb.State() != resilience.StateClosed {
		t.Errorf("expected StateClosed after success in half-open, got %v", cb.State())
	}
}

func TestCircuitBreaker_ReopensOnFailureInHalfOpen(t *testing.T) {
	cfg := resilience.CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Cooldown:         10 * time.Millisecond,
	}
	cb := resilience.NewCircuitBreaker(cfg)

	// Open the circuit
	cb.Execute(context.Background(), func(ctx context.Context) error {
		return errors.New("failure")
	})

	// Wait for cooldown
	time.Sleep(15 * time.Millisecond)

	// Fail in half-open state
	cb.Execute(context.Background(), func(ctx context.Context) error {
		return errors.New("failure in half-open")
	})

	// Should be open again
	if cb.State() != resilience.StateOpen {
		t.Errorf("expected StateOpen after failure in half-open, got %v", cb.State())
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	cfg := resilience.CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Cooldown:         1 * time.Hour,
	}
	cb := resilience.NewCircuitBreaker(cfg)

	// Execute some successes and failures
	for i := 0; i < 3; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})
	}
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) error {
			return errors.New("failure")
		})
	}

	stats := cb.Stats()

	if stats.TotalRequests != 5 {
		t.Errorf("expected 5 total requests, got %d", stats.TotalRequests)
	}
	if stats.TotalFailures != 2 {
		t.Errorf("expected 2 total failures, got %d", stats.TotalFailures)
	}
	if stats.State != resilience.StateClosed {
		t.Errorf("expected StateClosed, got %v", stats.State)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cfg := resilience.CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Cooldown:         1 * time.Hour,
	}
	cb := resilience.NewCircuitBreaker(cfg)

	// Open the circuit
	cb.Execute(context.Background(), func(ctx context.Context) error {
		return errors.New("failure")
	})

	if cb.State() != resilience.StateOpen {
		t.Fatalf("expected StateOpen, got %v", cb.State())
	}

	// Reset should close it
	cb.Reset()

	if cb.State() != resilience.StateClosed {
		t.Errorf("expected StateClosed after reset, got %v", cb.State())
	}
}
