package main

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestGitConfigCache_HitAndMiss(t *testing.T) {
	t.Parallel()

	cache := NewGitConfigCache(60 * time.Second)
	cache.Set("/repo", "user.name", "Alice")

	// Hit
	val, ok := cache.Get("/repo", "user.name")
	if !ok || val != "Alice" {
		t.Fatalf("expected hit with 'Alice', got %q ok=%v", val, ok)
	}

	// Miss: different key
	_, ok = cache.Get("/repo", "user.email")
	if ok {
		t.Fatal("expected miss for uncached key")
	}

	// Miss: different repo
	_, ok = cache.Get("/other", "user.name")
	if ok {
		t.Fatal("expected miss for different repo")
	}
}

func TestGitConfigCache_Expiry(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cache := NewGitConfigCache(10 * time.Second)
	cache.now = func() time.Time { return now }

	cache.Set("/repo", "user.name", "Alice")

	// Still valid at now+9s
	cache.now = func() time.Time { return now.Add(9 * time.Second) }
	val, ok := cache.Get("/repo", "user.name")
	if !ok || val != "Alice" {
		t.Fatalf("expected hit before expiry, got %q ok=%v", val, ok)
	}

	// Expired at now+11s
	cache.now = func() time.Time { return now.Add(11 * time.Second) }
	_, ok = cache.Get("/repo", "user.name")
	if ok {
		t.Fatal("expected miss after expiry")
	}
}

func TestGitConfigCache_ConfigGetCachesResult(t *testing.T) {
	t.Parallel()

	fake := NewFakeGitRunner()
	fake.ConfigValues["user.name"] = "Bob"

	cache := NewGitConfigCache(60 * time.Second)

	// First call: cache miss -> hits the GitRunner
	val, err := cache.ConfigGet(context.Background(), fake, "/fake/repo", "user.name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "Bob" {
		t.Fatalf("expected 'Bob', got %q", val)
	}

	configGetCalls := countCalls(fake.Calls, "ConfigGet")
	if configGetCalls != 1 {
		t.Fatalf("expected 1 ConfigGet call, got %d", configGetCalls)
	}

	// Second call: cache hit -> does NOT hit the GitRunner
	val, err = cache.ConfigGet(context.Background(), fake, "/fake/repo", "user.name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "Bob" {
		t.Fatalf("expected 'Bob' from cache, got %q", val)
	}

	configGetCalls = countCalls(fake.Calls, "ConfigGet")
	if configGetCalls != 1 {
		t.Fatalf("expected still 1 ConfigGet call (served from cache), got %d", configGetCalls)
	}
}

func TestGitConfigCache_ConfigGetPropagatesError(t *testing.T) {
	t.Parallel()

	fake := NewFakeGitRunner()
	fake.ConfigError = fmt.Errorf("config read failed")

	cache := NewGitConfigCache(60 * time.Second)

	_, err := cache.ConfigGet(context.Background(), fake, "/fake/repo", "user.name")
	if err == nil {
		t.Fatal("expected error from ConfigGet")
	}

	// Failed lookups should not be cached
	_, ok := cache.Get("/fake/repo", "user.name")
	if ok {
		t.Fatal("failed lookup should not be cached")
	}
}

func countCalls(calls []FakeGitCall, method string) int {
	n := 0
	for _, c := range calls {
		if c.Method == method {
			n++
		}
	}
	return n
}
