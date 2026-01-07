package health

import (
	"context"
	"time"
)

// CheckerFunc adapts a function to the Checker interface.
type CheckerFunc func(ctx context.Context) CheckResult

// Check runs the checker function.
func (f CheckerFunc) Check(ctx context.Context) CheckResult {
	return f(ctx)
}

// Func creates a Checker from a simple function that returns an error.
// Latency is measured automatically and included in the CheckResult.
func Func(name string, fn func(ctx context.Context) error) Checker {
	return CheckerFunc(func(ctx context.Context) CheckResult {
		if fn == nil {
			return CheckResult{
				Name:      name,
				Connected: false,
				Error:     errNotConfigured,
			}
		}

		start := time.Now()
		err := fn(ctx)
		latency := time.Since(start)

		if err != nil {
			return CheckResult{
				Name:      name,
				Connected: false,
				Latency:   latency,
				Error:     err,
			}
		}

		return CheckResult{
			Name:      name,
			Connected: true,
			Latency:   latency,
		}
	})
}

// NewErrorDetail creates a structured error detail for dependency failures.
func NewErrorDetail(code, message, category string, retryable bool) *ErrorDetail {
	return &ErrorDetail{
		Code:      code,
		Message:   message,
		Category:  category,
		Retryable: retryable,
	}
}
