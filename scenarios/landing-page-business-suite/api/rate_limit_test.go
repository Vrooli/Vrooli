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
