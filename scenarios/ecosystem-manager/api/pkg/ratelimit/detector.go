package ratelimit

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// Rate limiting constants
const (
	// DefaultRetrySeconds is the default pause duration when rate limited (30 minutes)
	DefaultRetrySeconds = 1800

	// MinRetrySeconds is the minimum rate limit pause duration (5 minutes)
	MinRetrySeconds = 300

	// MaxRetrySeconds is the maximum rate limit pause duration (4 hours)
	MaxRetrySeconds = 14400

	// CriticalRetrySeconds is the pause for critical rate limits (30 minutes)
	CriticalRetrySeconds = 1800

	// DetectionWindow is the time window for detecting early rate limit failures
	DetectionWindow = 1 * time.Minute

	// HTTPStatusTooManyRequests indicates a rate limit error (HTTP 429)
	HTTPStatusTooManyRequests = 429
)

// Detection represents the result of rate limit detection
type Detection struct {
	IsRateLimited bool // Whether rate limiting was detected
	RetryAfter    int  // Seconds to wait before retry
	CheckWindow   bool // Whether this detection should respect the detection window
}

// DetectFromError performs consolidated rate limit detection from error and output
// Returns (isRateLimited, retryAfterSeconds, shouldCheckWindow)
func DetectFromError(err error, output string, elapsed time.Duration) Detection {
	// Check exit code first - HTTP 429 is a definitive rate limit
	exitCode, hasExit := extractExitCode(err)
	if hasExit && exitCode == HTTPStatusTooManyRequests {
		retryAfter := ExtractRetryDuration(output)
		return Detection{
			IsRateLimited: true,
			RetryAfter:    retryAfter,
			CheckWindow:   true, // Exit code 429 should respect detection window
		}
	}

	// Check for rate limit patterns in output
	if !isRateLimitPattern(output) {
		return Detection{IsRateLimited: false}
	}

	// Pattern matched - extract retry duration
	retryAfter := ExtractRetryDuration(output)

	// Early failures (< 1 minute) are more likely to be real rate limits
	// Later failures might just mention rate limits in documentation/logs
	shouldCheckWindow := elapsed <= DetectionWindow

	return Detection{
		IsRateLimited: true,
		RetryAfter:    retryAfter,
		CheckWindow:   shouldCheckWindow,
	}
}

// isRateLimitPattern detects if the output contains rate limit error patterns
func isRateLimitPattern(output string) bool {
	patterns := []string{
		"ai usage limit reached",
		"rate/usage limit reached",
		"claude ai usage limit reached",
		"you've reached your claude usage limit",
		"usage limit reached",
		"rate limit reached",
		"rate limit exceeded",
		"too many requests",
		"quota exceeded",
		"rate limits are critical",
		"error 429",
		"usage limit",
		"rate limit",
		"429",
	}

	lower := strings.ToLower(output)
	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// ExtractRetryDuration attempts to extract retry-after duration from rate limit error messages
func ExtractRetryDuration(output string) int {
	// Default to 30 minutes if we can't parse
	defaultRetry := DefaultRetrySeconds
	lowerOutput := strings.ToLower(output)

	// Common time duration patterns and their backoff values
	// Using 30 minutes default for 1 hour pattern to be less disruptive
	hourPatterns := map[string]int{
		"5 hour":        5 * 3600,
		"5-hour":        5 * 3600,
		"every 5 hours": 5 * 3600,
		"4 hour":        4 * 3600,
		"4-hour":        4 * 3600,
		"1 hour":        DefaultRetrySeconds,
		"1-hour":        DefaultRetrySeconds,
	}

	// Check for common time patterns
	for pattern, duration := range hourPatterns {
		if strings.Contains(lowerOutput, pattern) {
			return duration
		}
	}

	// Look for "retry_after" or "retry-after" patterns
	if strings.Contains(lowerOutput, "retry") && (strings.Contains(lowerOutput, "after") || strings.Contains(lowerOutput, "_after")) {
		// Try to extract number after retry_after or retry-after
		parts := strings.FieldsFunc(output, func(r rune) bool {
			return r == ':' || r == '=' || r == ' ' || r == '\t' || r == '\n'
		})

		for i, part := range parts {
			if strings.Contains(strings.ToLower(part), "retry") && i+1 < len(parts) {
				if seconds, err := strconv.Atoi(strings.Trim(parts[i+1], "\"'")); err == nil && seconds > 0 {
					// Cap at 4 hours maximum
					if seconds > MaxRetrySeconds {
						return MaxRetrySeconds
					}
					// Minimum 5 minutes
					if seconds < MinRetrySeconds {
						return MinRetrySeconds
					}
					return seconds
				}
			}
		}
	}

	// If we see "critical" rate limits, use a longer default
	if strings.Contains(lowerOutput, "critical") {
		return CriticalRetrySeconds
	}

	return defaultRetry
}

// extractExitCode extracts the exit code from an error
func extractExitCode(err error) (int, bool) {
	if err == nil {
		return 0, false
	}

	// Try ExitCode() interface
	var exitCoder interface {
		ExitCode() int
	}
	if errors.As(err, &exitCoder) {
		return exitCoder.ExitCode(), true
	}

	// Try ExitStatus() interface
	var statusErr interface {
		ExitStatus() int
	}
	if errors.As(err, &statusErr) {
		return statusErr.ExitStatus(), true
	}

	return 0, false
}
