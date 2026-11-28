package chainforensics

import (
	"testing"
	"time"
)

func TestCaptureCacheClonesExpiresAndEvictsLeastRecentlyUsed(t *testing.T) {
	now := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	clone := func(value []byte) []byte { return append([]byte(nil), value...) }
	cache := NewCaptureCache(2, clone)
	expires := now.Add(time.Minute)

	first := []byte("first")
	cache.Put("first", first, expires, now)
	first[0] = 'X'
	got, ok := cache.Get("first", now)
	if !ok || string(got) != "first" {
		t.Fatalf("cache did not retain a defensive write copy: got=%q ok=%t", got, ok)
	}
	got[0] = 'Y'
	got, ok = cache.Get("first", now)
	if !ok || string(got) != "first" {
		t.Fatalf("cache read exposed retained bytes: got=%q ok=%t", got, ok)
	}

	cache.Put("second", []byte("second"), expires, now)
	if _, ok := cache.Get("first", now); !ok {
		t.Fatal("expected first entry to remain live before eviction")
	}
	cache.Put("third", []byte("third"), expires, now)
	if _, ok := cache.Get("second", now); ok {
		t.Fatal("least recently used entry was not evicted")
	}
	if _, ok := cache.Get("first", now); !ok {
		t.Fatal("recently used entry was evicted")
	}

	if _, ok := cache.Get("first", expires); ok {
		t.Fatal("entry live at its exact expiration instant")
	}
	if got := cache.Len(expires); got != 0 {
		t.Fatalf("expired entries remain retained: %d", got)
	}
}

func TestCaptureCacheRejectsEmptyKeysAndNonPositiveLifetime(t *testing.T) {
	now := time.Now()
	cache := NewCaptureCache[string](1, nil)
	cache.Put("", "empty", now.Add(time.Minute), now)
	cache.Put("expired", "expired", now, now)
	if got := cache.Len(now); got != 0 {
		t.Fatalf("invalid captures entered the cache: %d", got)
	}
	if _, ok := cache.Get("", now); ok {
		t.Fatal("empty key returned a cache value")
	}
}

func TestCaptureCacheBoundsEstimatedBytesAndRetainsMostRecentlyUsed(t *testing.T) {
	now := time.Now()
	cache := NewWeightedCaptureCache(4, 5, func(value []byte) []byte { return append([]byte(nil), value...) }, func(value []byte) int64 { return int64(len(value)) })
	expires := now.Add(time.Minute)
	cache.Put("large", []byte("123456"), expires, now)
	if got := cache.Len(now); got != 0 {
		t.Fatalf("oversized capture was retained: %d", got)
	}
	cache.Put("one", []byte("123"), expires, now)
	cache.Put("two", []byte("45"), expires, now)
	if got := cache.Bytes(now); got != 5 {
		t.Fatalf("unexpected cache weight: %d", got)
	}
	if _, ok := cache.Get("one", now); !ok {
		t.Fatal("expected first entry before weighted eviction")
	}
	cache.Put("three", []byte("67"), expires, now)
	if got := cache.Bytes(now); got != 5 {
		t.Fatalf("weighted cache exceeded its byte bound: %d", got)
	}
	if _, ok := cache.Get("two", now); ok {
		t.Fatal("least recently used entry was not evicted to satisfy the byte bound")
	}
	if _, ok := cache.Get("one", now); !ok {
		t.Fatal("recently used entry was evicted unnecessarily")
	}
	if value, ok := cache.Get("three", now); !ok || string(value) != "67" {
		t.Fatalf("new capture missing after weighted eviction: %q ok=%t", value, ok)
	}
}
