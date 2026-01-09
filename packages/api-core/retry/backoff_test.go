package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDo_SucceedsOnFirstAttempt(t *testing.T) {
	t.Parallel()

	attempts := 0
	err := Do(context.Background(), DefaultConfig(), func(attempt int) error {
		attempts++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestDo_RetriesUntilSuccess(t *testing.T) {
	t.Parallel()

	failUntil := 3
	attempts := 0
	var sleeps []time.Duration

	cfg := Config{
		MaxAttempts: 5,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    10 * time.Second,
		Sleeper:     func(d time.Duration) { sleeps = append(sleeps, d) },
		Rand:        func() float64 { return 0 }, // No jitter for predictable testing
	}

	err := Do(context.Background(), cfg, func(attempt int) error {
		attempts++
		if attempts < failUntil {
			return errors.New("not ready yet")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != failUntil {
		t.Fatalf("expected %d attempts, got %d", failUntil, attempts)
	}
	// Should have slept failUntil-1 times (before each retry)
	if len(sleeps) != failUntil-1 {
		t.Fatalf("expected %d sleeps, got %d", failUntil-1, len(sleeps))
	}
}

func TestDo_ExhaustsAllAttempts(t *testing.T) {
	t.Parallel()

	attempts := 0
	cfg := Config{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		Sleeper:     func(d time.Duration) {},
	}

	err := Do(context.Background(), cfg, func(attempt int) error {
		attempts++
		return errors.New("always fails")
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	if !errors.Is(err, errors.New("always fails")) {
		// Check error message contains the wrapped error
		if err.Error() != "failed after 3 attempts: always fails" {
			t.Fatalf("unexpected error message: %v", err)
		}
	}
}

func TestDo_ExponentialBackoff(t *testing.T) {
	t.Parallel()

	var sleeps []time.Duration
	cfg := Config{
		MaxAttempts:    5,
		BaseDelay:      100 * time.Millisecond,
		MaxDelay:       10 * time.Second,
		JitterFraction: 0, // Disable jitter for exact values
		Sleeper:        func(d time.Duration) { sleeps = append(sleeps, d) },
	}

	Do(context.Background(), cfg, func(attempt int) error {
		return errors.New("fail")
	})

	expected := []time.Duration{
		100 * time.Millisecond, // 100ms * 2^0
		200 * time.Millisecond, // 100ms * 2^1
		400 * time.Millisecond, // 100ms * 2^2
		800 * time.Millisecond, // 100ms * 2^3
		// No sleep after last attempt
	}

	if len(sleeps) != len(expected) {
		t.Fatalf("expected %d sleeps, got %d: %v", len(expected), len(sleeps), sleeps)
	}

	for i, exp := range expected {
		if sleeps[i] != exp {
			t.Errorf("sleep[%d]: expected %v, got %v", i, exp, sleeps[i])
		}
	}
}

func TestDo_CapsAtMaxDelay(t *testing.T) {
	t.Parallel()

	var sleeps []time.Duration
	cfg := Config{
		MaxAttempts:    6,
		BaseDelay:      100 * time.Millisecond,
		MaxDelay:       300 * time.Millisecond, // Cap kicks in at attempt 2
		JitterFraction: 0,
		Sleeper:        func(d time.Duration) { sleeps = append(sleeps, d) },
	}

	Do(context.Background(), cfg, func(attempt int) error {
		return errors.New("fail")
	})

	// Attempt 0: 100ms, Attempt 1: 200ms, Attempt 2+: capped at 300ms
	expected := []time.Duration{
		100 * time.Millisecond, // 100 * 2^0 = 100
		200 * time.Millisecond, // 100 * 2^1 = 200
		300 * time.Millisecond, // 100 * 2^2 = 400, capped to 300
		300 * time.Millisecond, // 100 * 2^3 = 800, capped to 300
		300 * time.Millisecond, // 100 * 2^4 = 1600, capped to 300
	}

	if len(sleeps) != len(expected) {
		t.Fatalf("expected %d sleeps, got %d: %v", len(expected), len(sleeps), sleeps)
	}

	for i, exp := range expected {
		if sleeps[i] != exp {
			t.Errorf("sleep[%d]: expected %v, got %v", i, exp, sleeps[i])
		}
	}
}

func TestDo_JitterAddsRandomness(t *testing.T) {
	t.Parallel()

	var sleeps []time.Duration
	randValue := 0.5

	cfg := Config{
		MaxAttempts:    2,
		BaseDelay:      100 * time.Millisecond,
		MaxDelay:       10 * time.Second,
		JitterFraction: 0.25,
		Sleeper:        func(d time.Duration) { sleeps = append(sleeps, d) },
		Rand:           func() float64 { return randValue },
	}

	Do(context.Background(), cfg, func(attempt int) error {
		return errors.New("fail")
	})

	// Base delay: 100ms
	// Jitter: 100ms * 0.25 * 0.5 = 12.5ms
	// Total: 112.5ms
	expected := 112500 * time.Microsecond

	if len(sleeps) != 1 {
		t.Fatalf("expected 1 sleep, got %d", len(sleeps))
	}
	if sleeps[0] != expected {
		t.Fatalf("expected %v, got %v", expected, sleeps[0])
	}
}

func TestDo_JitterMaxValue(t *testing.T) {
	t.Parallel()

	var sleeps []time.Duration

	cfg := Config{
		MaxAttempts:    2,
		BaseDelay:      100 * time.Millisecond,
		MaxDelay:       10 * time.Second,
		JitterFraction: 0.25,
		Sleeper:        func(d time.Duration) { sleeps = append(sleeps, d) },
		Rand:           func() float64 { return 0.9999 }, // Near max
	}

	Do(context.Background(), cfg, func(attempt int) error {
		return errors.New("fail")
	})

	// Base delay: 100ms
	// Jitter: 100ms * 0.25 * 0.9999 = ~25ms
	// Total: ~125ms
	if sleeps[0] < 124*time.Millisecond || sleeps[0] > 126*time.Millisecond {
		t.Fatalf("expected ~125ms, got %v", sleeps[0])
	}
}

func TestDo_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	cfg := Config{
		MaxAttempts: 10,
		BaseDelay:   time.Millisecond,
		Sleeper:     func(d time.Duration) {},
	}

	err := Do(ctx, cfg, func(attempt int) error {
		attempts++
		if attempts >= 2 {
			cancel() // Cancel after second attempt
		}
		return errors.New("fail")
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		// Should contain context.Canceled in the chain
		if ctx.Err() != context.Canceled {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	}
}

func TestDo_ContextAlreadyCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel before starting

	attempts := 0
	err := Do(ctx, DefaultConfig(), func(attempt int) error {
		attempts++
		return nil
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 0 {
		t.Fatalf("expected 0 attempts with cancelled context, got %d", attempts)
	}
}

func TestDo_OnRetryCallback(t *testing.T) {
	t.Parallel()

	var callbacks []struct {
		attempt int
		delay   time.Duration
	}

	cfg := Config{
		MaxAttempts:    4,
		BaseDelay:      100 * time.Millisecond,
		MaxDelay:       10 * time.Second,
		JitterFraction: 0,
		Sleeper:        func(d time.Duration) {},
		OnRetry: func(attempt int, err error, delay time.Duration) {
			callbacks = append(callbacks, struct {
				attempt int
				delay   time.Duration
			}{attempt, delay})
		},
	}

	Do(context.Background(), cfg, func(attempt int) error {
		return errors.New("fail")
	})

	// OnRetry is called before each retry (not before first attempt)
	// So for 4 attempts, we get 3 callbacks
	if len(callbacks) != 3 {
		t.Fatalf("expected 3 callbacks, got %d", len(callbacks))
	}

	// Check attempt numbers (1-indexed for "retry 1", "retry 2", etc.)
	expectedAttempts := []int{1, 2, 3}
	for i, exp := range expectedAttempts {
		if callbacks[i].attempt != exp {
			t.Errorf("callback[%d].attempt: expected %d, got %d", i, exp, callbacks[i].attempt)
		}
	}
}

func TestDo_PassesAttemptNumber(t *testing.T) {
	t.Parallel()

	var attempts []int
	cfg := Config{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		Sleeper:     func(d time.Duration) {},
	}

	Do(context.Background(), cfg, func(attempt int) error {
		attempts = append(attempts, attempt)
		return errors.New("fail")
	})

	expected := []int{0, 1, 2}
	if len(attempts) != len(expected) {
		t.Fatalf("expected %d attempts, got %d", len(expected), len(attempts))
	}
	for i, exp := range expected {
		if attempts[i] != exp {
			t.Errorf("attempt[%d]: expected %d, got %d", i, exp, attempts[i])
		}
	}
}

func TestDo_WrapsLastError(t *testing.T) {
	t.Parallel()

	errSequence := []error{
		errors.New("error 1"),
		errors.New("error 2"),
		errors.New("error 3"),
	}

	callNum := 0
	cfg := Config{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		Sleeper:     func(d time.Duration) {},
	}

	err := Do(context.Background(), cfg, func(attempt int) error {
		e := errSequence[callNum]
		callNum++
		return e
	})

	// Should wrap the last error
	if !errors.Is(err, errSequence[2]) {
		t.Fatalf("expected wrapped error 3, got %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	if cfg.MaxAttempts != 10 {
		t.Errorf("MaxAttempts: expected 10, got %d", cfg.MaxAttempts)
	}
	if cfg.BaseDelay != 500*time.Millisecond {
		t.Errorf("BaseDelay: expected 500ms, got %v", cfg.BaseDelay)
	}
	if cfg.MaxDelay != 30*time.Second {
		t.Errorf("MaxDelay: expected 30s, got %v", cfg.MaxDelay)
	}
	if cfg.JitterFraction != 0.25 {
		t.Errorf("JitterFraction: expected 0.25, got %f", cfg.JitterFraction)
	}
}

func TestDo_AppliesDefaults(t *testing.T) {
	t.Parallel()

	// Empty config should use defaults
	cfg := Config{
		Sleeper: func(d time.Duration) {},
	}

	attempts := 0
	Do(context.Background(), cfg, func(attempt int) error {
		attempts++
		if attempts >= 3 {
			return nil // Succeed on third attempt
		}
		return errors.New("fail")
	})

	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestDo_NoSleepAfterLastAttempt(t *testing.T) {
	t.Parallel()

	sleepCount := 0
	cfg := Config{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		Sleeper:     func(d time.Duration) { sleepCount++ },
	}

	Do(context.Background(), cfg, func(attempt int) error {
		return errors.New("fail")
	})

	// 3 attempts = 2 sleeps (no sleep after last)
	if sleepCount != 2 {
		t.Fatalf("expected 2 sleeps, got %d", sleepCount)
	}
}

func TestDo_NoSleepOnSuccess(t *testing.T) {
	t.Parallel()

	sleepCount := 0
	cfg := Config{
		MaxAttempts: 10,
		BaseDelay:   time.Millisecond,
		Sleeper:     func(d time.Duration) { sleepCount++ },
	}

	err := Do(context.Background(), cfg, func(attempt int) error {
		return nil // Immediate success
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sleepCount != 0 {
		t.Fatalf("expected 0 sleeps on immediate success, got %d", sleepCount)
	}
}

// Benchmark to verify minimal overhead
func BenchmarkDo_ImmediateSuccess(b *testing.B) {
	cfg := DefaultConfig()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Do(ctx, cfg, func(attempt int) error {
			return nil
		})
	}
}
