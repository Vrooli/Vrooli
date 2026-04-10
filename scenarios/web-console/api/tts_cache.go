package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// TTSCacheKey identifies a cached TTS audio entry.
type TTSCacheKey struct {
	EventID string
	Voice   string
	Speed   float64
	Version string // "active" | "original"
}

// cacheKeyHash returns a deterministic string hash for map storage.
func cacheKeyHash(k TTSCacheKey) string {
	raw := fmt.Sprintf("%s|%s|%.4f|%s", k.EventID, k.Voice, k.Speed, k.Version)
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum[:16])
}

// TTSCacheEntry holds cached audio data.
type TTSCacheEntry struct {
	Audio       []byte
	ContentType string
	CreatedAt   time.Time
	eventID     string // for reverse lookup during eviction
}

// TTSCacheStats provides observability into cache state.
type TTSCacheStats struct {
	EntryCount int
	TotalBytes int
	MaxBytes   int
}

// TTSCache is an in-memory LRU audio cache for pre-synthesized TTS.
type TTSCache struct {
	mu       sync.RWMutex
	entries  map[string]*TTSCacheEntry
	order    []string // LRU order: oldest first
	maxSize  int
	currSize int
}

// NewTTSCache creates a cache with the given maximum size in bytes.
func NewTTSCache(maxSizeBytes int) *TTSCache {
	return &TTSCache{
		entries: make(map[string]*TTSCacheEntry),
		maxSize: maxSizeBytes,
	}
}

// Get retrieves a cached entry. Returns nil, false on miss.
func (c *TTSCache) Get(key TTSCacheKey) (*TTSCacheEntry, bool) {
	hash := cacheKeyHash(key)
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[hash]
	return entry, ok
}

// Put stores audio in the cache, evicting oldest entries if necessary.
func (c *TTSCache) Put(key TTSCacheKey, audio []byte, contentType string) {
	hash := cacheKeyHash(key)
	size := len(audio)

	c.mu.Lock()
	defer c.mu.Unlock()

	// If this single entry exceeds max size, don't cache it.
	if size > c.maxSize {
		return
	}

	// Remove existing entry with this key if present (update case).
	if existing, ok := c.entries[hash]; ok {
		c.currSize -= len(existing.Audio)
		delete(c.entries, hash)
		c.removeFromOrder(hash)
	}

	// Evict oldest entries until there's room.
	for c.currSize+size > c.maxSize && len(c.order) > 0 {
		c.evictOldest()
	}

	c.entries[hash] = &TTSCacheEntry{
		Audio:       audio,
		ContentType: contentType,
		CreatedAt:   time.Now(),
		eventID:     key.EventID,
	}
	c.order = append(c.order, hash)
	c.currSize += size
}

// Evict removes all cached entries for a given event ID (all versions/voices).
func (c *TTSCache) Evict(eventID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var toRemove []string
	for hash, entry := range c.entries {
		if entry.eventID == eventID {
			toRemove = append(toRemove, hash)
		}
	}
	for _, hash := range toRemove {
		if entry, ok := c.entries[hash]; ok {
			c.currSize -= len(entry.Audio)
			delete(c.entries, hash)
			c.removeFromOrder(hash)
		}
	}
}

// Stats returns current cache statistics.
func (c *TTSCache) Stats() TTSCacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return TTSCacheStats{
		EntryCount: len(c.entries),
		TotalBytes: c.currSize,
		MaxBytes:   c.maxSize,
	}
}

func (c *TTSCache) evictOldest() {
	if len(c.order) == 0 {
		return
	}
	oldest := c.order[0]
	c.order = c.order[1:]
	if entry, ok := c.entries[oldest]; ok {
		c.currSize -= len(entry.Audio)
		delete(c.entries, oldest)
	}
}

func (c *TTSCache) removeFromOrder(hash string) {
	for i, h := range c.order {
		if h == hash {
			c.order = append(c.order[:i], c.order[i+1:]...)
			return
		}
	}
}

// preSynthesizeTTS asynchronously synthesizes TTS audio for an assistant event
// and stores it in the cache for instant playback on tab switch.
func (s *Server) preSynthesizeTTS(event ConversationEvent, sessionID string) {
	if s.ttsSynthesizer == nil || s.ttsCache == nil {
		return
	}
	if event.Role != ConversationRoleAssistant {
		return
	}
	if len(event.SpeechParagraphs) == 0 {
		return
	}

	cfg := s.getTTSConfig()
	voice := cfg.KokoroVoice
	if voice == "" {
		voice = "af_heart"
	}
	speed := cfg.KokoroSpeed
	if speed <= 0 {
		speed = 1.0
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	audio, contentType, err := s.synthesizeParagraphs(ctx, event.SpeechParagraphs, voice, speed)
	if err != nil {
		log.Printf("tts-precache: synthesis failed for event %s: %v", event.ID, err)
		return
	}

	key := TTSCacheKey{
		EventID: event.ID,
		Voice:   voice,
		Speed:   speed,
		Version: "active",
	}
	s.ttsCache.Put(key, audio, contentType)
	log.Printf("tts-precache: cached %d bytes for event %s (voice=%s speed=%.1f)",
		len(audio), event.ID, voice, speed)
}

// synthesizeParagraphs synthesizes multiple paragraphs and concatenates the
// resulting audio into a single MP3 blob. This reuses the same Kokoro backend
// as the on-demand handleTTSSynthesize handler.
func (s *Server) synthesizeParagraphs(ctx context.Context, paragraphs []string, voice string, speed float64) ([]byte, string, error) {
	var combined []byte
	var contentType string

	for _, p := range paragraphs {
		if len(p) == 0 {
			continue
		}
		if len(p) > maxSynthesizeInputLength {
			p = p[:maxSynthesizeInputLength]
		}

		req := SynthesizeRequest{
			Input:          p,
			Voice:          voice,
			ResponseFormat: "mp3",
			Speed:          speed,
		}

		body, ct, err := s.ttsSynthesizer.Synthesize(ctx, req)
		if err != nil {
			return nil, "", fmt.Errorf("synthesize paragraph: %w", err)
		}

		data, err := io.ReadAll(body)
		body.Close()
		if err != nil {
			return nil, "", fmt.Errorf("read synthesis response: %w", err)
		}

		if len(data) > 0 {
			combined = append(combined, data...)
			if contentType == "" {
				contentType = ct
			}
		}
	}

	if len(combined) == 0 {
		return nil, "", fmt.Errorf("all paragraphs produced empty audio")
	}

	return combined, contentType, nil
}

// handleGetTTSCache serves cached TTS audio for an event.
// GET /api/v1/tts/cache/{eventId}?voice={voice}&speed={speed}&version={active|original}
func (s *Server) handleGetTTSCache(w http.ResponseWriter, r *http.Request) {
	if s.ttsCache == nil {
		http.NotFound(w, r)
		return
	}

	vars := mux.Vars(r)
	eventID := vars["eventId"]
	if eventID == "" {
		writeCatalogError(w, "tts_cache_missing_id", "eventId is required")
		return
	}

	voice := r.URL.Query().Get("voice")
	if voice == "" {
		cfg := s.getTTSConfig()
		voice = cfg.KokoroVoice
		if voice == "" {
			voice = "af_heart"
		}
	}

	speedStr := r.URL.Query().Get("speed")
	speed := 1.0
	if speedStr != "" {
		if parsed, err := strconv.ParseFloat(speedStr, 64); err == nil && parsed > 0 {
			speed = parsed
		}
	}

	version := r.URL.Query().Get("version")
	if version == "" {
		version = "active"
	}

	key := TTSCacheKey{
		EventID: eventID,
		Voice:   voice,
		Speed:   speed,
		Version: version,
	}

	entry, ok := s.ttsCache.Get(key)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"error":"not_cached","message":"No cached audio for this event"}`)
		return
	}

	w.Header().Set("Content-Type", entry.ContentType)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(entry.Audio); err != nil {
		log.Printf("tts-cache: write response: %v", err)
	}
}
