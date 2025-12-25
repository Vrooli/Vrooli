package resilience

import (
	"context"
	"log"
)

// FallbackFunc is a function that provides a fallback result.
type FallbackFunc[T any] func(ctx context.Context, originalErr error) (T, error)

// WithFallback executes fn and falls back to fallbackFn on error.
// This implements the graceful degradation pattern where a failure
// returns a degraded but useful result instead of a complete failure.
//
// Example:
//
//	name, err := resilience.WithFallback(ctx,
//	    func(ctx context.Context) (string, error) {
//	        return ollama.GenerateName(ctx, conversation)
//	    },
//	    func(ctx context.Context, err error) (string, error) {
//	        log.Printf("name generation failed, using fallback: %v", err)
//	        return "New Conversation", nil
//	    },
//	)
func WithFallback[T any](ctx context.Context, fn func(context.Context) (T, error), fallbackFn FallbackFunc[T]) (T, error) {
	result, err := fn(ctx)
	if err == nil {
		return result, nil
	}

	return fallbackFn(ctx, err)
}

// WithDefaultFallback executes fn and returns defaultValue on error.
// Errors are logged but not returned.
//
// Example:
//
//	name := resilience.WithDefaultFallback(ctx,
//	    func(ctx context.Context) (string, error) {
//	        return ollama.GenerateName(ctx, conversation)
//	    },
//	    "New Conversation",
//	    "chat name generation",
//	)
func WithDefaultFallback[T any](ctx context.Context, fn func(context.Context) (T, error), defaultValue T, operation string) T {
	result, err := fn(ctx)
	if err == nil {
		return result
	}

	log.Printf("%s failed, using default | error=%v", operation, err)
	return defaultValue
}

// WithCachedFallback executes fn and falls back to a cached value on error.
// The cache function should return the most recently known good value.
// This is useful for read-heavy operations where stale data is acceptable.
func WithCachedFallback[T any](ctx context.Context, fn func(context.Context) (T, error), cacheFn func() (T, bool)) (T, bool, error) {
	result, err := fn(ctx)
	if err == nil {
		return result, false, nil // false = not from cache
	}

	cachedResult, hasCached := cacheFn()
	if hasCached {
		log.Printf("primary source failed, using cached value | error=%v", err)
		return cachedResult, true, nil // true = from cache
	}

	var zero T
	return zero, false, err
}

// PartialResult represents a result that may be incomplete due to failures.
type PartialResult[T any] struct {
	Value      T
	IsComplete bool
	Errors     []error
}

// WithPartialResult executes multiple operations and returns partial results on failure.
// Use this when it's better to return some data than no data.
func WithPartialResult[T any](ctx context.Context, fns ...func(context.Context) (T, error)) PartialResult[[]T] {
	results := make([]T, 0, len(fns))
	var errors []error

	for _, fn := range fns {
		result, err := fn(ctx)
		if err != nil {
			errors = append(errors, err)
		} else {
			results = append(results, result)
		}
	}

	return PartialResult[[]T]{
		Value:      results,
		IsComplete: len(errors) == 0,
		Errors:     errors,
	}
}

// GracefulOperation combines retry, circuit breaker, and fallback patterns.
type GracefulOperation[T any] struct {
	// Primary is the main function to execute.
	Primary func(context.Context) (T, error)
	// Fallback is called when all attempts fail (optional).
	Fallback FallbackFunc[T]
	// RetryConfig controls retry behavior (optional, uses defaults if nil).
	RetryConfig *RetryConfig
	// CircuitBreaker protects the operation (optional).
	CircuitBreaker *CircuitBreaker
}

// Execute runs the operation with all configured resilience patterns.
func (op *GracefulOperation[T]) Execute(ctx context.Context) (T, error) {
	var zero T

	// Wrap primary in circuit breaker if configured
	wrapped := func(ctx context.Context) (T, error) {
		if op.CircuitBreaker != nil {
			var result T
			err := op.CircuitBreaker.Execute(ctx, func(ctx context.Context) error {
				var execErr error
				result, execErr = op.Primary(ctx)
				return execErr
			})
			return result, err
		}
		return op.Primary(ctx)
	}

	// Apply retry if configured
	if op.RetryConfig != nil {
		var result T
		err := Retry(ctx, *op.RetryConfig, func(ctx context.Context, attempt int) error {
			var retryErr error
			result, retryErr = wrapped(ctx)
			return retryErr
		})
		if err == nil {
			return result, nil
		}
		if op.Fallback != nil {
			return op.Fallback(ctx, err)
		}
		return zero, err
	}

	// No retry - just execute
	result, err := wrapped(ctx)
	if err != nil && op.Fallback != nil {
		return op.Fallback(ctx, err)
	}
	return result, err
}
