// Package resilience provides patterns for graceful degradation under failure.
// These utilities help the service fail safely, observably, and recoverably.
package resilience

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

// RetryConfig controls retry behavior.
type RetryConfig struct {
	// MaxAttempts is the total number of attempts (including the first).
	MaxAttempts int
	// BaseDelay is the initial delay before the first retry.
	BaseDelay time.Duration
	// MaxDelay caps the exponential backoff.
	MaxDelay time.Duration
	// Jitter adds randomness to delays (0.0 = none, 1.0 = up to 100% of delay).
	Jitter float64
}

// DefaultRetryConfig returns sensible defaults for retry behavior.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Second,
		MaxDelay:    10 * time.Second,
		Jitter:      0.1,
	}
}

// RetryableFunc is a function that may be retried on failure.
// Return nil for success, or an error to trigger retry.
// Return a PermanentError to stop retrying immediately.
type RetryableFunc func(ctx context.Context, attempt int) error

// PermanentError wraps an error to indicate it should not be retried.
type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string {
	return e.Err.Error()
}

func (e *PermanentError) Unwrap() error {
	return e.Err
}

// Permanent wraps an error to indicate no further retries should be attempted.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &PermanentError{Err: err}
}

// IsPermanent checks if an error is marked as permanent (non-retryable).
func IsPermanent(err error) bool {
	var pe *PermanentError
	return errors.As(err, &pe)
}

// Retry executes fn with exponential backoff retry logic.
// It stops on:
//   - Success (fn returns nil)
//   - Context cancellation
//   - Permanent error
//   - MaxAttempts exceeded
//
// The returned error includes attempt information for debugging.
func Retry(ctx context.Context, cfg RetryConfig, fn RetryableFunc) error {
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// Check context before each attempt
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("retry cancelled after %d attempts: %w", attempt-1, err)
		}

		lastErr = fn(ctx, attempt)

		// Success
		if lastErr == nil {
			return nil
		}

		// Permanent error - don't retry
		if IsPermanent(lastErr) {
			return lastErr
		}

		// Last attempt - don't wait
		if attempt == cfg.MaxAttempts {
			break
		}

		// Calculate delay with exponential backoff
		delay := calculateDelay(cfg, attempt)

		// Wait for delay or context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled during backoff after %d attempts: %w", attempt, ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("retry exhausted after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

// calculateDelay computes the backoff delay for an attempt.
func calculateDelay(cfg RetryConfig, attempt int) time.Duration {
	// Exponential backoff: baseDelay * 2^(attempt-1)
	multiplier := math.Pow(2, float64(attempt-1))
	delay := time.Duration(float64(cfg.BaseDelay) * multiplier)

	// Cap at max delay
	if delay > cfg.MaxDelay {
		delay = cfg.MaxDelay
	}

	// Add jitter (simplified - no random for determinism)
	// In production, you might want: delay + time.Duration(rand.Float64()*cfg.Jitter*float64(delay))

	return delay
}

// RetryResult contains the outcome of a retry operation.
type RetryResult struct {
	Attempts int
	Duration time.Duration
	Err      error
}

// RetryWithMetrics executes retry and returns detailed metrics.
func RetryWithMetrics(ctx context.Context, cfg RetryConfig, fn RetryableFunc) RetryResult {
	start := time.Now()
	var attempts int
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		attempts = attempt

		if err := ctx.Err(); err != nil {
			return RetryResult{
				Attempts: attempts,
				Duration: time.Since(start),
				Err:      fmt.Errorf("retry cancelled: %w", err),
			}
		}

		lastErr = fn(ctx, attempt)

		if lastErr == nil {
			return RetryResult{
				Attempts: attempts,
				Duration: time.Since(start),
			}
		}

		if IsPermanent(lastErr) {
			return RetryResult{
				Attempts: attempts,
				Duration: time.Since(start),
				Err:      lastErr,
			}
		}

		if attempt < cfg.MaxAttempts {
			delay := calculateDelay(cfg, attempt)
			select {
			case <-ctx.Done():
				return RetryResult{
					Attempts: attempts,
					Duration: time.Since(start),
					Err:      fmt.Errorf("retry cancelled during backoff: %w", ctx.Err()),
				}
			case <-time.After(delay):
			}
		}
	}

	return RetryResult{
		Attempts: attempts,
		Duration: time.Since(start),
		Err:      fmt.Errorf("retry exhausted: %w", lastErr),
	}
}
