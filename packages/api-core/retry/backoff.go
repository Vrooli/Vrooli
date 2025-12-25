// Package retry provides generic retry logic with exponential backoff and jitter.
//
// The retry package is designed to prevent thundering herd problems when multiple
// service instances attempt to reconnect to a shared resource simultaneously.
// It uses exponential backoff to space out retries, and random jitter to
// desynchronize retry attempts across instances.
//
// Usage:
//
//	err := retry.Do(ctx, retry.DefaultConfig(), func(attempt int) error {
//	    return db.Ping()
//	})
//
// With custom configuration:
//
//	cfg := retry.Config{
//	    MaxAttempts: 5,
//	    BaseDelay:   time.Second,
//	    MaxDelay:    time.Minute,
//	}
//	err := retry.Do(ctx, cfg, func(attempt int) error {
//	    return connectToService()
//	})
package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Config controls retry behavior.
type Config struct {
	// MaxAttempts is the maximum number of attempts before giving up.
	// Default: 10
	MaxAttempts int

	// BaseDelay is the initial delay before the first retry.
	// Subsequent delays grow exponentially from this value.
	// Default: 500ms
	BaseDelay time.Duration

	// MaxDelay caps the delay between retries.
	// Exponential growth stops once this limit is reached.
	// Default: 30s
	MaxDelay time.Duration

	// JitterFraction adds randomness to prevent thundering herd.
	// A value of 0.25 adds up to 25% extra delay randomly.
	// Default: 0.25
	JitterFraction float64

	// OnRetry is called before each retry sleep (not before the first attempt).
	// Useful for logging retry attempts. The error is from the previous attempt.
	// Optional.
	OnRetry func(attempt int, err error, delay time.Duration)

	// Sleeper overrides time.Sleep for testing.
	// If nil, uses time.Sleep.
	Sleeper func(time.Duration)

	// Rand overrides rand.Float64 for testing.
	// Must return a value in [0.0, 1.0).
	// If nil, uses rand.Float64.
	Rand func() float64
}

// DefaultConfig returns production-ready defaults.
//
// Default values:
//   - MaxAttempts: 10
//   - BaseDelay: 500ms
//   - MaxDelay: 30s
//   - JitterFraction: 0.25 (25%)
func DefaultConfig() Config {
	return Config{
		MaxAttempts:    10,
		BaseDelay:      500 * time.Millisecond,
		MaxDelay:       30 * time.Second,
		JitterFraction: 0.25,
	}
}

// Do executes op until it succeeds or attempts are exhausted.
//
// The operation function receives the current attempt number (0-indexed).
// If the operation returns nil, Do returns nil immediately.
// If the context is cancelled, Do returns the context error.
// If all attempts fail, Do returns an error wrapping the last failure.
//
// Delay calculation:
//   - Base delay grows exponentially: baseDelay * 2^attempt
//   - Delay is capped at maxDelay
//   - Random jitter is added: delay + rand(0, delay * jitterFraction)
func Do(ctx context.Context, cfg Config, op func(attempt int) error) error {
	cfg = applyDefaults(cfg)

	var lastErr error
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		// Check context before each attempt
		if err := ctx.Err(); err != nil {
			if lastErr != nil {
				return fmt.Errorf("context cancelled after %d attempts (last error: %w): %v", attempt, lastErr, err)
			}
			return err
		}

		// Execute the operation
		if err := op(attempt); err == nil {
			return nil // Success
		} else {
			lastErr = err
		}

		// Don't sleep after the last attempt
		if attempt < cfg.MaxAttempts-1 {
			delay := computeDelay(cfg, attempt)

			// Notify callback before sleeping
			if cfg.OnRetry != nil {
				cfg.OnRetry(attempt+1, lastErr, delay)
			}

			// Sleep with context awareness
			if err := sleepWithContext(ctx, delay, cfg.Sleeper); err != nil {
				return fmt.Errorf("context cancelled during retry backoff (last error: %w): %v", lastErr, err)
			}
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

// computeDelay calculates the delay for a given attempt.
// Uses exponential backoff: baseDelay * 2^attempt, capped at maxDelay.
// Adds random jitter: delay + rand(0, delay * jitterFraction).
func computeDelay(cfg Config, attempt int) time.Duration {
	// Exponential backoff: baseDelay * 2^attempt
	delay := float64(cfg.BaseDelay) * math.Pow(2, float64(attempt))

	// Cap at maxDelay
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}

	// Add jitter: random value in [0, delay * jitterFraction)
	jitter := delay * cfg.JitterFraction * cfg.randFloat(cfg)

	return time.Duration(delay + jitter)
}

// sleepWithContext sleeps for the given duration, but returns early if context is cancelled.
func sleepWithContext(ctx context.Context, d time.Duration, sleeper func(time.Duration)) error {
	if sleeper != nil {
		// Custom sleeper (for testing) - just call it directly
		sleeper(d)
		return ctx.Err()
	}

	// Real sleep with context cancellation support
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// randFloat returns a random float using the configured Rand function or default.
func (cfg Config) randFloat(c Config) float64 {
	if c.Rand != nil {
		return c.Rand()
	}
	return rand.Float64()
}

// applyDefaults fills in zero values with defaults.
func applyDefaults(cfg Config) Config {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 10
	}
	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay = 500 * time.Millisecond
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 30 * time.Second
	}
	if cfg.JitterFraction < 0 {
		cfg.JitterFraction = 0.25
	}
	return cfg
}

// IsRetryError reports whether err is a retry exhaustion error from Do.
func IsRetryError(err error) bool {
	if err == nil {
		return false
	}
	// Check if the error message matches our format
	// This is a simple check; could be enhanced with custom error type
	msg := err.Error()
	return len(msg) > 20 && msg[:7] == "failed "
}
