package main

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/tts"
)

func TestTTSCache_PutAndGet(t *testing.T) {
	cache := NewTTSCache(1024 * 1024) // 1MB

	key := TTSCacheKey{EventID: "evt1", Voice: "af_heart", Speed: 1.0, Version: "active"}
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

func TestTTSCache_CacheMiss(t *testing.T) {
	cache := NewTTSCache(1024 * 1024)

	key := TTSCacheKey{EventID: "nonexistent", Voice: "af_heart", Speed: 1.0, Version: "active"}
	entry, ok := cache.Get(key)
	if ok || entry != nil {
		t.Fatal("expected cache miss")
	}
}

func TestTTSCache_VersionSeparation(t *testing.T) {
	cache := NewTTSCache(1024 * 1024)

	activeKey := TTSCacheKey{EventID: "evt1", Voice: "af_heart", Speed: 1.0, Version: "active"}
	originalKey := TTSCacheKey{EventID: "evt1", Voice: "af_heart", Speed: 1.0, Version: "original"}

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

func TestTTSCache_LRUEviction(t *testing.T) {
	// Cache that can hold exactly 20 bytes
	cache := NewTTSCache(20)

	key1 := TTSCacheKey{EventID: "evt1", Voice: "v", Speed: 1.0, Version: "active"}
	key2 := TTSCacheKey{EventID: "evt2", Voice: "v", Speed: 1.0, Version: "active"}
	key3 := TTSCacheKey{EventID: "evt3", Voice: "v", Speed: 1.0, Version: "active"}

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

func TestTTSCache_EvictByEventID(t *testing.T) {
	cache := NewTTSCache(1024 * 1024)

	// Store two versions for the same event
	activeKey := TTSCacheKey{EventID: "evt1", Voice: "af_heart", Speed: 1.0, Version: "active"}
	originalKey := TTSCacheKey{EventID: "evt1", Voice: "af_heart", Speed: 1.0, Version: "original"}
	otherKey := TTSCacheKey{EventID: "evt2", Voice: "af_heart", Speed: 1.0, Version: "active"}

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

func TestTTSCache_ConcurrentAccess(t *testing.T) {
	cache := NewTTSCache(1024 * 1024)
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := TTSCacheKey{EventID: fmt.Sprintf("evt%d", i), Voice: "v", Speed: 1.0, Version: "active"}
			cache.Put(key, []byte(fmt.Sprintf("audio-%d", i)), "audio/mpeg")
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := TTSCacheKey{EventID: fmt.Sprintf("evt%d", i), Voice: "v", Speed: 1.0, Version: "active"}
			cache.Get(key)
		}(i)
	}

	wg.Wait()
	// No panics or data races = pass (run with -race)
}

func TestTTSCache_OversizedEntry(t *testing.T) {
	cache := NewTTSCache(10) // 10 bytes max
	key := TTSCacheKey{EventID: "evt1", Voice: "v", Speed: 1.0, Version: "active"}
	cache.Put(key, []byte("this-is-more-than-10-bytes"), "audio/mpeg")

	if _, ok := cache.Get(key); ok {
		t.Error("expected oversized entry to not be cached")
	}
}

func TestTTSCache_UpdateExistingKey(t *testing.T) {
	cache := NewTTSCache(1024)
	key := TTSCacheKey{EventID: "evt1", Voice: "v", Speed: 1.0, Version: "active"}

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

func TestTTSCache_Stats(t *testing.T) {
	cache := NewTTSCache(1024)
	stats := cache.Stats()
	if stats.EntryCount != 0 || stats.TotalBytes != 0 {
		t.Error("expected empty stats for new cache")
	}
	if stats.MaxBytes != 1024 {
		t.Errorf("expected max 1024, got %d", stats.MaxBytes)
	}
}

// --- Handler tests ---

func newCacheTestServer() *Server {
	srv := newFakeTestServer()
	srv.ttsCache = NewTTSCache(1024 * 1024)
	srv.ttsConfig = DefaultTTSConfig()
	return srv
}

func TestHandleGetTTSCache_Hit(t *testing.T) {
	srv := newCacheTestServer()

	key := TTSCacheKey{EventID: "abc123", Voice: "af_heart", Speed: 1.0, Version: "active"}
	srv.ttsCache.Put(key, []byte("cached-audio"), "audio/mpeg")

	resp, err := callTTSGetCache(t, srv, &ttsv1.GetCacheRequest{
		EventId: "abc123", Voice: "af_heart", Speed: 1, Version: "active",
	})
	if err != nil {
		t.Fatalf("expected cache hit, got %v", err)
	}
	if resp.GetContentType() != "audio/mpeg" {
		t.Errorf("expected audio/mpeg, got %s", resp.GetContentType())
	}
	if string(resp.GetAudio()) != "cached-audio" {
		t.Errorf("expected 'cached-audio', got %q", string(resp.GetAudio()))
	}
}

func TestHandleGetTTSCache_Miss(t *testing.T) {
	srv := newCacheTestServer()

	_, err := callTTSGetCache(t, srv, &ttsv1.GetCacheRequest{
		EventId: "nonexistent", Voice: "af_heart", Speed: 1, Version: "active",
	})
	if connectCode(err) != connect.CodeNotFound {
		t.Fatalf("expected CodeNotFound, got %v (err=%v)", connectCode(err), err)
	}
}

func TestHandleGetTTSCache_DefaultVersion(t *testing.T) {
	srv := newCacheTestServer()

	key := TTSCacheKey{EventID: "abc123", Voice: "af_heart", Speed: 1.0, Version: "active"}
	srv.ttsCache.Put(key, []byte("default-audio"), "audio/mpeg")

	resp, err := callTTSGetCache(t, srv, &ttsv1.GetCacheRequest{
		EventId: "abc123", Voice: "af_heart", Speed: 1,
	})
	if err != nil {
		t.Fatalf("expected default version hit, got %v", err)
	}
	if string(resp.GetAudio()) != "default-audio" {
		t.Errorf("unexpected audio: %q", string(resp.GetAudio()))
	}
}

// --- Pre-synthesis tests ---

func TestPreSynthesizeTTS_HappyPath(t *testing.T) {
	srv := newFakeTestServer()
	srv.ttsCache = NewTTSCache(1024 * 1024)
	srv.ttsConfig = TTSConfig{AutoEnabled: true, Backend: "kokoro", KokoroVoice: "af_heart", KokoroSpeed: 1.0}
	srv.ttsSynthesizer = &mockSynthesizer{
		body:        "synthesized-audio",
		contentType: "audio/mpeg",
	}

	event := ConversationEvent{
		ID:               "evt-test",
		Role:             ConversationRoleAssistant,
		SpeechParagraphs: []string{"Hello world"},
	}

	srv.preSynthesizeTTS(event, "session1")

	// Give the sync call time to complete (it's actually synchronous here)
	key := TTSCacheKey{EventID: "evt-test", Voice: "af_heart", Speed: 1.0, Version: "active"}
	entry, ok := srv.ttsCache.Get(key)
	if !ok {
		t.Fatal("expected cache to be populated after pre-synthesis")
	}
	if string(entry.Audio) != "synthesized-audio" {
		t.Errorf("expected 'synthesized-audio', got %q", string(entry.Audio))
	}
}

func TestPreSynthesizeTTS_SynthesizerNil(t *testing.T) {
	srv := newFakeTestServer()
	srv.ttsCache = NewTTSCache(1024 * 1024)
	srv.ttsSynthesizer = nil // no synthesizer

	event := ConversationEvent{
		ID:               "evt-test",
		Role:             ConversationRoleAssistant,
		SpeechParagraphs: []string{"Hello world"},
	}

	// Should not panic
	srv.preSynthesizeTTS(event, "session1")
}

func TestPreSynthesizeTTS_NonAssistant(t *testing.T) {
	srv := newFakeTestServer()
	srv.ttsCache = NewTTSCache(1024 * 1024)
	srv.ttsSynthesizer = &mockSynthesizer{body: "audio", contentType: "audio/mpeg"}

	event := ConversationEvent{
		ID:               "evt-test",
		Role:             ConversationRoleUser,
		SpeechParagraphs: []string{"Hello"},
	}

	srv.preSynthesizeTTS(event, "session1")

	key := TTSCacheKey{EventID: "evt-test", Voice: "af_heart", Speed: 1.0, Version: "active"}
	if _, ok := srv.ttsCache.Get(key); ok {
		t.Error("expected no cache entry for user events")
	}
}

func TestPreSynthesizeTTS_EmptyParagraphs(t *testing.T) {
	srv := newFakeTestServer()
	srv.ttsCache = NewTTSCache(1024 * 1024)
	srv.ttsSynthesizer = &mockSynthesizer{body: "audio", contentType: "audio/mpeg"}

	event := ConversationEvent{
		ID:               "evt-test",
		Role:             ConversationRoleAssistant,
		SpeechParagraphs: []string{},
	}

	srv.preSynthesizeTTS(event, "session1")

	key := TTSCacheKey{EventID: "evt-test", Voice: "af_heart", Speed: 1.0, Version: "active"}
	if _, ok := srv.ttsCache.Get(key); ok {
		t.Error("expected no cache entry for empty paragraphs")
	}
}

// --- Integration tests ---

func TestEndToEnd_PreCacheFlow(t *testing.T) {
	srv := newFakeTestServer()
	srv.ttsCache = NewTTSCache(1024 * 1024)
	srv.ttsConfig = TTSConfig{AutoEnabled: true, Backend: "kokoro", KokoroVoice: "af_heart", KokoroSpeed: 1.0}
	srv.ttsSynthesizer = &mockSynthesizer{body: "precached-audio", contentType: "audio/mpeg"}
	srv.conversations = NewConversationStore()

	// Simulate appending an event and pre-synthesizing
	event := ConversationEvent{
		ID:               "evt-e2e",
		Role:             ConversationRoleAssistant,
		SpeechParagraphs: []string{"Test paragraph"},
		CreatedAt:        time.Now(),
	}
	srv.preSynthesizeTTS(event, "session1")

	resp, err := callTTSGetCache(t, srv, &ttsv1.GetCacheRequest{
		EventId: "evt-e2e", Voice: "af_heart", Speed: 1, Version: "active",
	})
	if err != nil {
		t.Fatalf("expected pre-cached hit, got %v", err)
	}
	if string(resp.GetAudio()) != "precached-audio" {
		t.Errorf("expected 'precached-audio', got %q", string(resp.GetAudio()))
	}
}

// --- Invalidation tests ---

func TestInvalidateTTSCacheForEvent_ClearsAllVariants(t *testing.T) {
	srv := newFakeTestServer()
	srv.ttsCache = NewTTSCache(1024 * 1024)

	variants := []TTSCacheKey{
		{EventID: "evt1", Voice: "af_heart", Speed: 1.0, Version: "active"},
		{EventID: "evt1", Voice: "af_heart", Speed: 1.0, Version: "original"},
		{EventID: "evt1", Voice: "am_adam", Speed: 1.25, Version: "active"},
	}
	for _, k := range variants {
		srv.ttsCache.Put(k, []byte("audio-"+k.Voice+"-"+k.Version), "audio/mpeg")
	}
	otherKey := TTSCacheKey{EventID: "evt-other", Voice: "af_heart", Speed: 1.0, Version: "active"}
	srv.ttsCache.Put(otherKey, []byte("other"), "audio/mpeg")

	srv.invalidateTTSCacheForEvent("evt1")

	for _, k := range variants {
		if _, ok := srv.ttsCache.Get(k); ok {
			t.Errorf("variant %+v should have been evicted", k)
		}
	}
	if _, ok := srv.ttsCache.Get(otherKey); !ok {
		t.Error("unrelated event should not be evicted")
	}
}

func TestInvalidateTTSCacheForEvent_NilCacheIsSafe(t *testing.T) {
	srv := newFakeTestServer()
	srv.ttsCache = nil
	// Must not panic.
	srv.invalidateTTSCacheForEvent("evt1")
}

func TestInvalidateTTSCacheForEvent_EmptyIDIsNoop(t *testing.T) {
	srv := newFakeTestServer()
	srv.ttsCache = NewTTSCache(1024 * 1024)
	key := TTSCacheKey{EventID: "evt1", Voice: "v", Speed: 1.0, Version: "active"}
	srv.ttsCache.Put(key, []byte("x"), "audio/mpeg")

	srv.invalidateTTSCacheForEvent("")

	if _, ok := srv.ttsCache.Get(key); !ok {
		t.Error("empty eventID should not evict anything")
	}
}

func TestEndToEnd_CacheMissFallback(t *testing.T) {
	srv := newCacheTestServer()

	_, err := callTTSGetCache(t, srv, &ttsv1.GetCacheRequest{
		EventId: "missing-event", Voice: "af_heart", Speed: 1,
	})
	if connectCode(err) != connect.CodeNotFound {
		t.Fatalf("expected CodeNotFound, got %v (err=%v)", connectCode(err), err)
	}
}
