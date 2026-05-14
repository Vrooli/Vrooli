package sessions

import (
	"fmt"
	"testing"
	"time"
)

// [REQ:P0-003b] Idempotency cache: expired entries are cleaned up on Get
func TestIdempotencyCache_TTLExpiry(t *testing.T) {
	c := &IdempotencyCache{
		entries: make(map[string]idempotencyEntry),
		ttl:     10 * time.Millisecond,
	}

	resp := Response{ID: "s1"}
	c.Set("key1", resp)

	got, ok := c.Get("key1")
	if !ok || got.ID != "s1" {
		t.Fatal("entry should be retrievable immediately after set")
	}

	time.Sleep(15 * time.Millisecond)

	_, ok = c.Get("key1")
	if ok {
		t.Error("expired entry should not be returned by Get")
	}

	c.mu.Lock()
	_, stillThere := c.entries["key1"]
	c.mu.Unlock()
	if stillThere {
		t.Error("expired entry should be removed from cache map after Get")
	}
}

// [REQ:P0-003b] Idempotency cache: eviction scan triggered when >100 entries
func TestIdempotencyCache_EvictionScan(t *testing.T) {
	c := &IdempotencyCache{
		entries: make(map[string]idempotencyEntry),
		ttl:     time.Hour,
	}

	for i := 0; i < 100; i++ {
		c.entries[fmt.Sprintf("expired-%d", i)] = idempotencyEntry{
			expiresAt: time.Now().Add(-time.Minute),
		}
	}

	c.Set("fresh-key", Response{ID: "fresh"})

	c.mu.Lock()
	count := len(c.entries)
	c.mu.Unlock()

	if count != 1 {
		t.Errorf("expected 1 entry after eviction (fresh-key), got %d", count)
	}

	got, ok := c.Get("fresh-key")
	if !ok || got.ID != "fresh" {
		t.Error("fresh entry should survive eviction scan")
	}
}
