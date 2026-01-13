package providers

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// Provider constants (duplicated from distribution to avoid import cycle).
const (
	ProviderS3           = "s3"
	ProviderR2           = "r2"
	ProviderS3Compatible = "s3-compatible"
)

// RetryConfig controls upload retry behavior.
type RetryConfig struct {
	MaxAttempts       int
	InitialBackoffMs  int
	MaxBackoffMs      int
	BackoffMultiplier float64
}

// DefaultRetryConfig provides sensible defaults for retry behavior.
var DefaultRetryConfig = &RetryConfig{
	MaxAttempts:       3,
	InitialBackoffMs:  1000,  // 1 second
	MaxBackoffMs:      30000, // 30 seconds
	BackoffMultiplier: 2.0,
}

// DistributionTarget contains the fields needed for validation.
// This is a subset of the full distribution.DistributionTarget to avoid import cycle.
type DistributionTarget struct {
	Name               string
	Provider           string
	Bucket             string
	Endpoint           string
	Region             string
	AccessKeyIDEnv     string
	SecretAccessKeyEnv string
}

// UploadError represents an upload-specific error with context.
type UploadError struct {
	Operation string // "upload", "validate", "list", "delete"
	Target    string
	Key       string
	Attempt   int
	Retryable bool
	Cause     error
}

func (e *UploadError) Error() string {
	if e.Key != "" {
		return fmt.Sprintf("%s failed for %s/%s (attempt %d): %v",
			e.Operation, e.Target, e.Key, e.Attempt, e.Cause)
	}
	return fmt.Sprintf("%s failed for target %s (attempt %d): %v",
		e.Operation, e.Target, e.Attempt, e.Cause)
}

func (e *UploadError) Unwrap() error {
	return e.Cause
}

// IsRetryable determines if an error is retryable.
func IsRetryable(err error) bool {
	if uploadErr, ok := err.(*UploadError); ok {
		return uploadErr.Retryable
	}

	// Check for common retryable conditions
	errStr := strings.ToLower(err.Error())
	retryablePatterns := []string{
		"connection reset",
		"connection refused",
		"timeout",
		"temporary failure",
		"service unavailable",
		"rate limit",
		"429",
		"500",
		"502",
		"503",
		"504",
		"slowdown",
		"reduce your request rate",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// RetryWithBackoff executes fn with exponential backoff retry.
func RetryWithBackoff(ctx context.Context, config *RetryConfig, fn func(attempt int) error) error {
	if config == nil {
		config = DefaultRetryConfig
	}

	var lastErr error
	backoff := time.Duration(config.InitialBackoffMs) * time.Millisecond
	maxBackoff := time.Duration(config.MaxBackoffMs) * time.Millisecond

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		lastErr = fn(attempt)
		if lastErr == nil {
			return nil
		}

		if !IsRetryable(lastErr) {
			return lastErr
		}

		if attempt < config.MaxAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				// Calculate next backoff with jitter
				backoff = time.Duration(float64(backoff) * config.BackoffMultiplier)
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				// Add jitter (up to 10%)
				jitter := time.Duration(float64(backoff) * 0.1 * (rand.Float64() - 0.5))
				backoff += jitter
			}
		}
	}

	return lastErr
}

// ContentTypeFromFilename returns the content type based on file extension.
func ContentTypeFromFilename(filename string) string {
	filename = strings.ToLower(filename)

	switch {
	case strings.HasSuffix(filename, ".msi"):
		return "application/x-msi"
	case strings.HasSuffix(filename, ".exe"):
		return "application/x-msdownload"
	case strings.HasSuffix(filename, ".pkg"):
		return "application/vnd.apple.installer+xml"
	case strings.HasSuffix(filename, ".dmg"):
		return "application/x-apple-diskimage"
	case strings.HasSuffix(filename, ".appimage"):
		return "application/x-executable"
	case strings.HasSuffix(filename, ".deb"):
		return "application/vnd.debian.binary-package"
	case strings.HasSuffix(filename, ".rpm"):
		return "application/x-rpm"
	case strings.HasSuffix(filename, ".zip"):
		return "application/zip"
	case strings.HasSuffix(filename, ".tar.gz"), strings.HasSuffix(filename, ".tgz"):
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}

// ValidationError represents a target validation error.
type ValidationError struct {
	TargetName string
	Field      string
	Message    string
	Severity   string // "error", "warning"
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("[%s] %s: %s - %s", e.Severity, e.TargetName, e.Field, e.Message)
}

// ValidateTarget performs structural validation on a target config.
func ValidateTarget(target *DistributionTarget) []ValidationError {
	var errors []ValidationError

	// Required fields
	if target.Provider == "" {
		errors = append(errors, ValidationError{
			TargetName: target.Name,
			Field:      "provider",
			Message:    "provider is required",
			Severity:   "error",
		})
	}

	if target.Bucket == "" {
		errors = append(errors, ValidationError{
			TargetName: target.Name,
			Field:      "bucket",
			Message:    "bucket is required",
			Severity:   "error",
		})
	}

	if target.AccessKeyIDEnv == "" {
		errors = append(errors, ValidationError{
			TargetName: target.Name,
			Field:      "access_key_id_env",
			Message:    "access_key_id_env is required",
			Severity:   "error",
		})
	}

	if target.SecretAccessKeyEnv == "" {
		errors = append(errors, ValidationError{
			TargetName: target.Name,
			Field:      "secret_access_key_env",
			Message:    "secret_access_key_env is required",
			Severity:   "error",
		})
	}

	// Provider-specific validation
	switch target.Provider {
	case ProviderR2, ProviderS3Compatible:
		if target.Endpoint == "" {
			errors = append(errors, ValidationError{
				TargetName: target.Name,
				Field:      "endpoint",
				Message:    fmt.Sprintf("endpoint is required for provider %s", target.Provider),
				Severity:   "error",
			})
		}
	case ProviderS3:
		if target.Region == "" {
			errors = append(errors, ValidationError{
				TargetName: target.Name,
				Field:      "region",
				Message:    "region is recommended for AWS S3",
				Severity:   "warning",
			})
		}
	}

	return errors
}
