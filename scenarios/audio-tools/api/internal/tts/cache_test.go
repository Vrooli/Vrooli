package tts

import (
	"fmt"
	"sync"
	"testing"
)

func TestCache_PutAndGet(t *testing.T) {
	cache := NewCache(1024 * 1024) // 1MB

	key := CacheKey{EventID: "evt1", Voice: "af_heart", Speed: 1.0, Version: "active"}
	audio := []byte("fake-audio-data")

	cache.Put(key, audio, "audio/mpeg")

	entry, ok := cache.Get(key)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(entry.Audio) != "fake-audio-data" {
		t.Errorf("expected 'fake-audio-data', got %q", string(entry.Audio))
	}
	if entry.ContentType != "audio/mpeg" {
		t.Errorf("expected content type audio/mpeg, got %q", entry.ContentType)
	}
}

func TestCache_CacheMiss(t *testing.T) {
	cache := NewCache(1024 * 1024)

	key := CacheKey{EventID: "nonexistent", Voice: "af_heart", Speed: 1.0, Version: "active"}
	entry, ok := cache.Get(key)
	if ok || entry != nil {
		t.Fatal("expected cache miss")
	}
}

func TestCache_VersionSeparation(t *testing.T) {
	cache := NewCache(1024 * 1024)

	activeKey := CacheKey{EventID: "evt1", Voice: "af_heart", Speed: 1.0, Version: "active"}
	originalKey := CacheKey{EventID: "evt1", Voice: "af_heart", Speed: 1.0, Version: "original"}

	cache.Put(activeKey, []byte("summarized-audio"), "audio/mpeg")
	cache.Put(originalKey, []byte("original-audio"), "audio/mpeg")

	activeEntry, ok := cache.Get(activeKey)
	if !ok || string(activeEntry.Audio) != "summarized-audio" {
		t.Error("expected active version to be 'summarized-audio'")
	}

	originalEntry, ok := cache.Get(originalKey)
	if !ok || string(originalEntry.Audio) != "original-audio" {
		t.Error("expected original version to be 'original-audio'")
	}
}

func TestCache_ChunkIndexSeparation(t *testing.T) {
	// A spoken message is synthesized as N ordered paragraphs under one
	// event_id. Each paragraph must occupy a distinct cache slot keyed by its
	// chunk index — otherwise per-paragraph audio collides (last-write-wins)
	// and a replay serves only one paragraph.
	cache := NewCache(1024 * 1024)

	chunk0 := CacheKey{EventID: "evt1", Voice: "af_heart", Speed: 1.0, Version: "active", ChunkIndex: 0}
	chunk1 := CacheKey{EventID: "evt1", Voice: "af_heart", Speed: 1.0, Version: "active", ChunkIndex: 1}

	cache.Put(chunk0, []byte("paragraph-0-audio"), "audio/mpeg")
	cache.Put(chunk1, []byte("paragraph-1-audio"), "audio/mpeg")

	e0, ok := cache.Get(chunk0)
	if !ok || string(e0.Audio) != "paragraph-0-audio" {
		t.Errorf("chunk 0 collided or missing: ok=%v audio=%q", ok, e0.Audio)
	}
	e1, ok := cache.Get(chunk1)
	if !ok || string(e1.Audio) != "paragraph-1-audio" {
		t.Errorf("chunk 1 collided or missing: ok=%v audio=%q", ok, e1.Audio)
	}

	// Both chunks share the event id, so Evict(event) clears the whole message.
	cache.Evict("evt1")
	if _, ok := cache.Get(chunk0); ok {
		t.Error("expected chunk 0 evicted with the event")
	}
	if _, ok := cache.Get(chunk1); ok {
		t.Error("expected chunk 1 evicted with the event")
	}
}

func TestCache_LRUEviction(t *testing.T) {
	// Cache that can hold exactly 20 bytes
	cache := NewCache(20)

	key1 := CacheKey{EventID: "evt1", Voice: "v", Speed: 1.0, Version: "active"}
	key2 := CacheKey{EventID: "evt2", Voice: "v", Speed: 1.0, Version: "active"}
	key3 := CacheKey{EventID: "evt3", Voice: "v", Speed: 1.0, Version: "active"}

	cache.Put(key1, []byte("1234567890"), "audio/mpeg") // 10 bytes
	cache.Put(key2, []byte("abcdefghij"), "audio/mpeg") // 10 bytes, total 20

	// This should evict key1 (oldest) to make room
	cache.Put(key3, []byte("XXXXXXXXXX"), "audio/mpeg") // 10 bytes

	if _, ok := cache.Get(key1); ok {
		t.Error("expected key1 to be evicted")
	}
	if _, ok := cache.Get(key2); !ok {
		t.Error("expected key2 to still be cached")
	}
	if _, ok := cache.Get(key3); !ok {
		t.Error("expected key3 to be cached")
	}

	stats := cache.Stats()
	if stats.EntryCount != 2 {
		t.Errorf("expected 2 entries, got %d", stats.EntryCount)
	}
	if stats.TotalBytes != 20 {
		t.Errorf("expected 20 total bytes, got %d", stats.TotalBytes)
	}
}

func TestCache_EvictByEventID(t *testing.T) {
	cache := NewCache(1024 * 1024)

	// Store two versions for the same event
	activeKey := CacheKey{EventID: "evt1", Voice: "af_heart", Speed: 1.0, Version: "active"}
	originalKey := CacheKey{EventID: "evt1", Voice: "af_heart", Speed: 1.0, Version: "original"}
	otherKey := CacheKey{EventID: "evt2", Voice: "af_heart", Speed: 1.0, Version: "active"}

	cache.Put(activeKey, []byte("active-audio"), "audio/mpeg")
	cache.Put(originalKey, []byte("original-audio"), "audio/mpeg")
	cache.Put(otherKey, []byte("other-audio"), "audio/mpeg")

	cache.Evict("evt1")

	if _, ok := cache.Get(activeKey); ok {
		t.Error("expected active version of evt1 to be evicted")
	}
	if _, ok := cache.Get(originalKey); ok {
		t.Error("expected original version of evt1 to be evicted")
	}
	if _, ok := cache.Get(otherKey); !ok {
		t.Error("expected evt2 to still be cached")
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	cache := NewCache(1024 * 1024)
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := CacheKey{EventID: fmt.Sprintf("evt%d", i), Voice: "v", Speed: 1.0, Version: "active"}
			cache.Put(key, []byte(fmt.Sprintf("audio-%d", i)), "audio/mpeg")
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := CacheKey{EventID: fmt.Sprintf("evt%d", i), Voice: "v", Speed: 1.0, Version: "active"}
			cache.Get(key)
		}(i)
	}

	wg.Wait()
	// No panics or data races = pass (run with -race)
}

func TestCache_OversizedEntry(t *testing.T) {
	cache := NewCache(10) // 10 bytes max
	key := CacheKey{EventID: "evt1", Voice: "v", Speed: 1.0, Version: "active"}
	cache.Put(key, []byte("this-is-more-than-10-bytes"), "audio/mpeg")

	if _, ok := cache.Get(key); ok {
		t.Error("expected oversized entry to not be cached")
	}
}

func TestCache_UpdateExistingKey(t *testing.T) {
	cache := NewCache(1024)
	key := CacheKey{EventID: "evt1", Voice: "v", Speed: 1.0, Version: "active"}

	cache.Put(key, []byte("original"), "audio/mpeg")
	cache.Put(key, []byte("updated"), "audio/mpeg")

	entry, ok := cache.Get(key)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(entry.Audio) != "updated" {
		t.Errorf("expected 'updated', got %q", string(entry.Audio))
	}
	stats := cache.Stats()
	if stats.EntryCount != 1 {
		t.Errorf("expected 1 entry after update, got %d", stats.EntryCount)
	}
}

func TestCache_Stats(t *testing.T) {
	cache := NewCache(1024)
	stats := cache.Stats()
	if stats.EntryCount != 0 || stats.TotalBytes != 0 {
		t.Error("expected empty stats for new cache")
	}
	if stats.MaxBytes != 1024 {
		t.Errorf("expected max 1024, got %d", stats.MaxBytes)
	}
}
