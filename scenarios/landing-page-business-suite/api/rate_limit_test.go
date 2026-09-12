package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterMiddleware_AllowsUnderLimit(t *testing.T) {
	limiter := NewRateLimiter(5, 1*time.Minute)

	// Track how many times handler is called
	callCount := 0
	handler := limiter.Middleware(IPKeyFunc())(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	})

	// Make requests under the limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status %d, got %d", i+1, http.StatusOK, w.Code)
		}
	}

	if callCount != 5 {
		t.Errorf("Expected handler to be called 5 times, got %d", callCount)
	}
}

func TestRateLimiterMiddleware_BlocksOverLimit(t *testing.T) {
	limiter := NewRateLimiter(3, 1*time.Minute)

	callCount := 0
	handler := limiter.Middleware(IPKeyFunc())(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	})

	// Make requests up to and over the limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()

		handler(w, req)

		if i < 3 {
			// First 3 should succeed
			if w.Code != http.StatusOK {
				t.Errorf("Request %d: Expected status %d, got %d", i+1, http.StatusOK, w.Code)
			}
		} else {
			// Requests 4 and 5 should be blocked
			if w.Code != http.StatusTooManyRequests {
				t.Errorf("Request %d: Expected status %d, got %d", i+1, http.StatusTooManyRequests, w.Code)
			}
		}
	}

	if callCount != 3 {
		t.Errorf("Expected handler to be called 3 times, got %d", callCount)
	}
}

func TestRateLimiterMiddleware_EmptyKey(t *testing.T) {
	limiter := NewRateLimiter(1, 1*time.Minute)

	callCount := 0
	// Key function that always returns empty string
	handler := limiter.Middleware(func(r *http.Request) string {
		return ""
	})(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	})

	// All requests should be allowed when key is empty
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status %d with empty key, got %d", i+1, http.StatusOK, w.Code)
		}
	}

	if callCount != 5 {
		t.Errorf("Expected handler to be called 5 times with empty key, got %d", callCount)
	}
}

func TestRateLimiterMiddleware_TimeWindow(t *testing.T) {
	limiter := NewRateLimiter(2, 100*time.Millisecond)

	// Use injectable time provider
	currentTime := time.Now()
	limiter.UseTimeProvider(func() time.Time {
		return currentTime
	})

	callCount := 0
	handler := limiter.Middleware(IPKeyFunc())(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	})

	// Make 2 requests (at limit)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status %d, got %d", i+1, http.StatusOK, w.Code)
		}
	}

	// Third request should be blocked
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Third request should be blocked, got status %d", w.Code)
	}

	// Advance time past the window
	currentTime = currentTime.Add(200 * time.Millisecond)

	// Now requests should be allowed again
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w = httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Request after window reset should succeed, got status %d", w.Code)
	}
}

func TestRateLimiter_UseTimeProvider_NilReset(t *testing.T) {
	limiter := NewRateLimiter(5, 1*time.Minute)

	// Set a custom time provider
	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	limiter.UseTimeProvider(func() time.Time {
		return fixedTime
	})

	// Verify it's using the fixed time
	if !limiter.now().Equal(fixedTime) {
		t.Error("Expected limiter to use fixed time")
	}

	// Reset to nil should restore time.Now
	limiter.UseTimeProvider(nil)

	// Should now return current time (approximately)
	now := limiter.now()
	if now.Before(time.Now().Add(-1*time.Second)) || now.After(time.Now().Add(1*time.Second)) {
		t.Error("Expected limiter to use real time after nil reset")
	}
}

func TestCleanupOldBuckets(t *testing.T) {
	limiter := NewRateLimiter(10, 1*time.Second)

	// Use injectable time provider
	currentTime := time.Now()
	limiter.UseTimeProvider(func() time.Time {
		return currentTime
	})

	// Make requests from different keys
	keys := []string{"key1", "key2", "key3"}
	for _, key := range keys {
		limiter.Allow(key)
	}

	// Verify buckets exist
	limiter.mu.Lock()
	if len(limiter.buckets) != 3 {
		t.Errorf("Expected 3 buckets, got %d", len(limiter.buckets))
	}
	limiter.mu.Unlock()

	// Clear all timestamps to simulate idle buckets
	limiter.mu.Lock()
	for _, bucket := range limiter.buckets {
		bucket.timestamps = nil
	}
	limiter.mu.Unlock()

	// Advance time past the max age
	currentTime = currentTime.Add(2 * time.Hour)

	// Cleanup should remove all buckets
	limiter.CleanupOldBuckets(1 * time.Hour)

	limiter.mu.Lock()
	remaining := len(limiter.buckets)
	limiter.mu.Unlock()

	if remaining != 0 {
		t.Errorf("Expected 0 buckets after cleanup, got %d", remaining)
	}
}

func TestCleanupOldBuckets_KeepsRecent(t *testing.T) {
	limiter := NewRateLimiter(10, 1*time.Second)

	// Use injectable time provider
	currentTime := time.Now()
	limiter.UseTimeProvider(func() time.Time {
		return currentTime
	})

	// Make request
	limiter.Allow("active-key")

	// Don't clear timestamps - bucket should be kept
	limiter.CleanupOldBuckets(1 * time.Hour)

	limiter.mu.Lock()
	remaining := len(limiter.buckets)
	limiter.mu.Unlock()

	if remaining != 1 {
		t.Errorf("Expected 1 bucket to remain, got %d", remaining)
	}
}

func TestIPKeyFunc_XForwardedFor(t *testing.T) {
	keyFunc := IPKeyFunc()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	req.RemoteAddr = "10.0.0.1:12345"

	key := keyFunc(req)
	if key != "1.2.3.4" {
		t.Errorf("Expected IP '1.2.3.4', got '%s'", key)
	}
}

func TestIPKeyFunc_XRealIP(t *testing.T) {
	keyFunc := IPKeyFunc()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Real-IP", "1.2.3.4")
	req.RemoteAddr = "10.0.0.1:12345"

	key := keyFunc(req)
	if key != "1.2.3.4" {
		t.Errorf("Expected IP '1.2.3.4', got '%s'", key)
	}
}

func TestIPKeyFunc_RemoteAddr(t *testing.T) {
	keyFunc := IPKeyFunc()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.100:54321"

	key := keyFunc(req)
	if key != "192.168.1.100" {
		t.Errorf("Expected IP '192.168.1.100', got '%s'", key)
	}
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	limiter := NewRateLimiter(2, 1*time.Minute)

	handler := limiter.Middleware(IPKeyFunc())(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Make 2 requests from IP 1 (at limit)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("IP1 Request %d: Expected status %d, got %d", i+1, http.StatusOK, w.Code)
		}
	}

	// Third request from IP 1 should be blocked
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Error("IP1 Third request should be blocked")
	}

	// Request from different IP should succeed
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	w = httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("IP2 Request should succeed, got status %d", w.Code)
	}
}

func TestRateLimiter_Remaining(t *testing.T) {
	limiter := NewRateLimiter(5, 1*time.Minute)

	key := "test-key"

	// Initially should have full limit
	if remaining := limiter.Remaining(key); remaining != 5 {
		t.Errorf("Expected 5 remaining initially, got %d", remaining)
	}

	// After 2 requests, should have 3 remaining
	limiter.Allow(key)
	limiter.Allow(key)

	if remaining := limiter.Remaining(key); remaining != 3 {
		t.Errorf("Expected 3 remaining after 2 requests, got %d", remaining)
	}

	// After hitting limit, should have 0 remaining
	limiter.Allow(key)
	limiter.Allow(key)
	limiter.Allow(key)

	if remaining := limiter.Remaining(key); remaining != 0 {
		t.Errorf("Expected 0 remaining after hitting limit, got %d", remaining)
	}
}

// --- BucketCount Tests ---

func TestBucketCount_Empty(t *testing.T) {
	limiter := NewRateLimiter(5, 1*time.Minute)

	count := limiter.BucketCount()
	if count != 0 {
		t.Errorf("Expected 0 buckets for new limiter, got %d", count)
	}
}

func TestBucketCount_WithBuckets(t *testing.T) {
	limiter := NewRateLimiter(5, 1*time.Minute)

	// Create some buckets
	limiter.Allow("key1")
	limiter.Allow("key2")
	limiter.Allow("key3")

	count := limiter.BucketCount()
	if count != 3 {
		t.Errorf("Expected 3 buckets, got %d", count)
	}
}

// --- StartCleanup Tests ---

func TestStartCleanup_CleansOldBuckets(t *testing.T) {
	limiter := NewRateLimiter(10, 1*time.Second)

	// Use injectable time provider
	currentTime := time.Now()
	limiter.UseTimeProvider(func() time.Time {
		return currentTime
	})

	// Create some buckets
	limiter.Allow("cleanup-key1")
	limiter.Allow("cleanup-key2")

	// Clear timestamps to simulate idle buckets
	limiter.mu.Lock()
	for _, bucket := range limiter.buckets {
		bucket.timestamps = nil
	}
	limiter.mu.Unlock()

	// Advance time
	currentTime = currentTime.Add(2 * time.Hour)

	// Start cleanup with short interval
	cancel := limiter.StartCleanup(50*time.Millisecond, 1*time.Hour)
	defer cancel()

	// Wait for cleanup to run
	time.Sleep(100 * time.Millisecond)

	count := limiter.BucketCount()
	if count != 0 {
		t.Errorf("Expected 0 buckets after cleanup, got %d", count)
	}
}

func TestStartCleanup_CancelStops(t *testing.T) {
	limiter := NewRateLimiter(10, 1*time.Second)

	// Use injectable time provider
	currentTime := time.Now()
	limiter.UseTimeProvider(func() time.Time {
		return currentTime
	})

	limiter.Allow("cancel-test")

	// Start cleanup
	cancel := limiter.StartCleanup(10*time.Millisecond, 1*time.Hour)

	// Cancel immediately
	cancel()

	// Wait a bit to ensure goroutine has time to stop
	time.Sleep(50 * time.Millisecond)

	count := limiter.BucketCount()
	if count != 1 {
		t.Fatalf("bucket count after cancellation = %d, want 1", count)
	}
}

func TestStartCleanup_MultipleIntervals(t *testing.T) {
	limiter := NewRateLimiter(10, 1*time.Second)

	// Use injectable time provider
	currentTime := time.Now()
	limiter.UseTimeProvider(func() time.Time {
		return currentTime
	})

	// Add a key
	limiter.Allow("multi-interval")

	// Clear to make it eligible for cleanup
	limiter.mu.Lock()
	limiter.buckets["multi-interval"].timestamps = nil
	limiter.mu.Unlock()

	// Advance time past max age
	currentTime = currentTime.Add(2 * time.Hour)

	// Start cleanup with very short interval
	cancel := limiter.StartCleanup(20*time.Millisecond, 1*time.Hour)
	defer cancel()

	// Wait for multiple cleanup cycles
	time.Sleep(60 * time.Millisecond)

	// Bucket should be cleaned
	count := limiter.BucketCount()
	if count != 0 {
		t.Errorf("Expected bucket to be cleaned after multiple intervals, got %d", count)
	}
}

// --- NewRateLimiterWithOptions Edge Cases ---

func TestNewRateLimiterWithOptions_ZeroMaxBuckets(t *testing.T) {
	limiter := NewRateLimiterWithOptions(5, 1*time.Minute, 0)

	// Should use default max buckets
	if limiter.maxBuckets != DefaultMaxBuckets {
		t.Errorf("Expected default max buckets %d for zero value, got %d", DefaultMaxBuckets, limiter.maxBuckets)
	}
}

func TestNewRateLimiterWithOptions_NegativeMaxBuckets(t *testing.T) {
	limiter := NewRateLimiterWithOptions(5, 1*time.Minute, -10)

	// Should use default max buckets
	if limiter.maxBuckets != DefaultMaxBuckets {
		t.Errorf("Expected default max buckets %d for negative value, got %d", DefaultMaxBuckets, limiter.maxBuckets)
	}
}

// --- LRU Eviction Tests ---

func TestRateLimiter_LRUEviction(t *testing.T) {
	// Create limiter with very small max buckets
	limiter := NewRateLimiterWithOptions(5, 1*time.Minute, 3)

	// Add 3 buckets (at capacity)
	limiter.Allow("key1")
	limiter.Allow("key2")
	limiter.Allow("key3")

	if limiter.BucketCount() != 3 {
		t.Errorf("Expected 3 buckets, got %d", limiter.BucketCount())
	}

	// Add 4th bucket - should evict oldest (key1)
	limiter.Allow("key4")

	if limiter.BucketCount() != 3 {
		t.Errorf("Expected 3 buckets after eviction, got %d", limiter.BucketCount())
	}

	// key1 should have been evicted
	limiter.mu.Lock()
	_, key1Exists := limiter.buckets["key1"]
	_, key4Exists := limiter.buckets["key4"]
	limiter.mu.Unlock()

	if key1Exists {
		t.Error("Expected key1 to be evicted")
	}
	if !key4Exists {
		t.Error("Expected key4 to exist")
	}
}

func TestRateLimiter_LRURecentlyUsedNotEvicted(t *testing.T) {
	// Create limiter with small max buckets
	limiter := NewRateLimiterWithOptions(5, 1*time.Minute, 3)

	// Add 3 buckets
	limiter.Allow("key1")
	limiter.Allow("key2")
	limiter.Allow("key3")

	// Access key1 again (moves to front of LRU)
	limiter.Allow("key1")

	// Add 4th bucket - should evict key2 (now oldest)
	limiter.Allow("key4")

	limiter.mu.Lock()
	_, key1Exists := limiter.buckets["key1"]
	_, key2Exists := limiter.buckets["key2"]
	limiter.mu.Unlock()

	if !key1Exists {
		t.Error("Expected key1 to survive (recently used)")
	}
	if key2Exists {
		t.Error("Expected key2 to be evicted (oldest after key1 was accessed)")
	}
}

// --- IPKeyFunc Edge Cases ---

func TestIPKeyFunc_NoPort(t *testing.T) {
	keyFunc := IPKeyFunc()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1" // No port

	key := keyFunc(req)
	if key != "192.168.1.1" {
		t.Errorf("Expected IP '192.168.1.1', got '%s'", key)
	}
}

func TestIPKeyFunc_SingleXFF(t *testing.T) {
	keyFunc := IPKeyFunc()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4") // Single IP, no comma
	req.RemoteAddr = "10.0.0.1:12345"

	key := keyFunc(req)
	if key != "1.2.3.4" {
		t.Errorf("Expected IP '1.2.3.4', got '%s'", key)
	}
}

func TestIPKeyFunc_IPv6RemoteAddr(t *testing.T) {
	keyFunc := IPKeyFunc()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "[2001:db8::1]:8080"

	key := keyFunc(req)
	// The function strips the port by finding the last colon.
	// For IPv6 in bracket notation [addr]:port, it returns [addr]
	// This is acceptable for rate limiting as it's still a unique key per IP.
	if key != "[2001:db8::1]" {
		t.Errorf("Expected IPv6 '[2001:db8::1]', got '%s'", key)
	}
}
