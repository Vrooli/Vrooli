package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"scenario-authenticator/internal/redisstate"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

func TestLimiterBlocksAfterLimit(t *testing.T) {
	store := redisstate.NewMemory()
	l := New(store, Config{Limit: 3, Window: time.Minute, PathPrefixes: []string{"/login"}}, nil)
	h := l.Middleware(okHandler())

	do := func() int {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "1.2.3.4:5555"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	for i := 0; i < 3; i++ {
		if code := do(); code != http.StatusOK {
			t.Fatalf("req %d blocked early: %d", i, code)
		}
	}
	if code := do(); code != http.StatusTooManyRequests {
		t.Fatalf("4th request not limited: %d", code)
	}
}

func TestLimiterIgnoresUnmatchedPaths(t *testing.T) {
	store := redisstate.NewMemory()
	l := New(store, Config{Limit: 1, Window: time.Minute, PathPrefixes: []string{"/login"}}, nil)
	h := l.Middleware(okHandler())
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("health throttled: %d", rec.Code)
		}
	}
}

func TestLimiterPerIPIsolation(t *testing.T) {
	store := redisstate.NewMemory()
	l := New(store, Config{Limit: 1, Window: time.Minute, PathPrefixes: []string{"/login"}}, nil)
	h := l.Middleware(okHandler())
	hit := func(ip string) int {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = ip + ":1"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	if hit("1.1.1.1") != http.StatusOK || hit("2.2.2.2") != http.StatusOK {
		t.Fatal("distinct IPs should each get their own budget")
	}
	if hit("1.1.1.1") != http.StatusTooManyRequests {
		t.Fatal("repeat IP should be limited")
	}
}

func TestNilStorePassThrough(t *testing.T) {
	l := New(nil, Config{Limit: 1, Window: time.Minute}, nil)
	h := l.Middleware(okHandler())
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("nil-store limiter should pass through, got %d", rec.Code)
		}
	}
}
