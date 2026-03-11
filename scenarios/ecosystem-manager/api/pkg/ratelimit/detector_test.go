package ratelimit

import (
	"errors"
	"testing"
	"time"
)

type exitCodeErr struct {
	code int
}

func (e exitCodeErr) Error() string {
	return "exit error"
}

func (e exitCodeErr) ExitCode() int {
	return e.code
}

func TestDetectFromError_ExitCode429(t *testing.T) {
	err := exitCodeErr{code: HTTPStatusTooManyRequests}
	result := DetectFromError(err, "retry_after: 600", 10*time.Second)

	if !result.IsRateLimited {
		t.Fatalf("expected rate limited true, got false")
	}
	if result.RetryAfter != 600 {
		t.Fatalf("expected retry after 600, got %d", result.RetryAfter)
	}
	if !result.CheckWindow {
		t.Fatalf("expected CheckWindow true for 429 exit code")
	}
}

func TestDetectFromError_PatternWithinWindow(t *testing.T) {
	result := DetectFromError(errors.New("network failure"), "Rate limit exceeded, retry-after=400", 30*time.Second)

	if !result.IsRateLimited {
		t.Fatalf("expected rate limited true, got false")
	}
	if result.RetryAfter != 400 {
		t.Fatalf("expected retry after 400, got %d", result.RetryAfter)
	}
	if !result.CheckWindow {
		t.Fatalf("expected CheckWindow true when within detection window")
	}
}

func TestDetectFromError_NoMatch(t *testing.T) {
	result := DetectFromError(nil, "success output", 5*time.Second)
	if result.IsRateLimited {
		t.Fatalf("expected rate limited false, got true")
	}
}

func TestDetectFromError_HitYourLimitPattern(t *testing.T) {
	result := DetectFromError(nil, "You've hit your limit · resets 10am (America/New_York)", 5*time.Second)
	if !result.IsRateLimited {
		t.Fatalf("expected rate limited true, got false")
	}
}

func TestDetectFromError_Bare429NotRateLimit(t *testing.T) {
	// A bare "429" in output (e.g., line numbers, IDs) should NOT trigger rate limit detection
	result := DetectFromError(nil, "processing line 429 of file.go", 5*time.Second)
	if result.IsRateLimited {
		t.Fatalf("bare '429' in output should not trigger rate limit detection")
	}
}

func TestDetectFromError_BareRateLimitNotDetected(t *testing.T) {
	// Discussion about rate limits should NOT trigger detection
	result := DetectFromError(nil, "implementing rate limit handling for the API", 5*time.Second)
	if result.IsRateLimited {
		t.Fatalf("discussion about rate limits should not trigger detection")
	}
}

func TestDetectFromError_Error429Detected(t *testing.T) {
	// "error 429" should still trigger detection
	result := DetectFromError(nil, "error 429: too many requests", 5*time.Second)
	if !result.IsRateLimited {
		t.Fatalf("'error 429' should trigger rate limit detection")
	}
}

func TestExtractRetryDurationCapsAndPatterns(t *testing.T) {
	if got := ExtractRetryDuration("retry-after=50000"); got != MaxRetrySeconds {
		t.Fatalf("expected max cap %d, got %d", MaxRetrySeconds, got)
	}

	if got := ExtractRetryDuration("4 hour timeout noted"); got != 4*3600 {
		t.Fatalf("expected 4 hour duration, got %d", got)
	}

	if got := ExtractRetryDuration("critical rate limit hit"); got != CriticalRetrySeconds {
		t.Fatalf("expected critical retry %d, got %d", CriticalRetrySeconds, got)
	}
}
