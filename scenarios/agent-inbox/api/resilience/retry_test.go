package resilience_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-inbox/resilience"
)

func TestRetry_SuccessOnFirstAttempt(t *testing.T) {
	calls := 0
	err := resilience.Retry(context.Background(), resilience.DefaultRetryConfig(), func(ctx context.Context, attempt int) error {
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

func TestRetry_SuccessOnSecondAttempt(t *testing.T) {
	calls := 0
	cfg := resilience.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond, // Fast for testing
		MaxDelay:    10 * time.Millisecond,
	}

	err := resilience.Retry(context.Background(), cfg, func(ctx context.Context, attempt int) error {
		calls++
		if attempt < 2 {
			return errors.New("temporary failure")
		}
		return nil
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestRetry_ExhaustedAttempts(t *testing.T) {
	calls := 0
	cfg := resilience.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}

	err := resilience.Retry(context.Background(), cfg, func(ctx context.Context, attempt int) error {
		calls++
		return errors.New("persistent failure")
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetry_PermanentError(t *testing.T) {
	calls := 0
	cfg := resilience.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}

	permanentErr := resilience.Permanent(errors.New("do not retry"))
	err := resilience.Retry(context.Background(), cfg, func(ctx context.Context, attempt int) error {
		calls++
		return permanentErr
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retries for permanent error), got %d", calls)
	}
	if !resilience.IsPermanent(err) {
		t.Error("expected permanent error to be detected")
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	cfg := resilience.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}

	err := resilience.Retry(ctx, cfg, func(ctx context.Context, attempt int) error {
		return errors.New("should not reach here")
	})

	if err == nil {
		t.Error("expected error due to cancelled context")
	}
}

func TestRetryWithMetrics(t *testing.T) {
	calls := 0
	cfg := resilience.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}

	result := resilience.RetryWithMetrics(context.Background(), cfg, func(ctx context.Context, attempt int) error {
		calls++
		if attempt < 2 {
			return errors.New("temporary failure")
		}
		return nil
	})

	if result.Err != nil {
		t.Errorf("expected no error, got %v", result.Err)
	}
	if result.Attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", result.Attempts)
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}
}
