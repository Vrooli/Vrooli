package audioports

import (
	"context"
	"slices"
	"sync"
	"time"

	"connectrpc.com/connect"

	"web-console/integrations/audiotools"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
)

// RemoteSpeechTextProcessor implements SpeechTextProcessor by calling
// audio-tools' TTSService NormalizeForSpeech / SplitParagraphs RPCs.
//
// These are pure functions; the cross-scenario hop adds latency. We cache
// results in-process to keep the per-character cost bounded for the same
// input within a session. Cache is small + bounded.
type RemoteSpeechTextProcessor struct {
	Client *audiotools.Client

	mu           sync.Mutex
	normCache    map[string]cachedString
	splitCache   map[string]cachedSplit
	normFlights  map[string]*processorFlight[string]
	splitFlights map[string]*processorFlight[[]string]
}

type processorFlight[T any] struct {
	done    chan struct{}
	value   T
	waiters int
}

type cachedString struct {
	value     string
	expiresAt time.Time
}

type cachedSplit struct {
	value     []string
	expiresAt time.Time
}

var _ SpeechTextProcessor = (*RemoteSpeechTextProcessor)(nil)

const remoteProcessorCacheTTL = 5 * time.Minute
const remoteProcessorCacheMaxEntries = 256
const remoteProcessorCacheMaxBytes = 1 << 20

// storeRemoteProcessorCacheEntry bounds both count and payload bytes. The TTL
// alone cannot bound memory because expired entries are only revisited when
// their exact key is requested again.
func storeRemoteProcessorCacheEntry[T any](
	cache map[string]T,
	key string,
	value T,
	now time.Time,
	expiresAt func(T) time.Time,
	entryBytes func(string, T) int,
) bool {
	incomingBytes := entryBytes(key, value)
	if incomingBytes > remoteProcessorCacheMaxBytes {
		return false
	}

	cacheBytes := 0
	for existingKey, existingValue := range cache {
		if !now.Before(expiresAt(existingValue)) {
			delete(cache, existingKey)
			continue
		}
		cacheBytes += entryBytes(existingKey, existingValue)
	}

	for len(cache) >= remoteProcessorCacheMaxEntries || cacheBytes+incomingBytes > remoteProcessorCacheMaxBytes {
		var oldestKey string
		var oldestExpiry time.Time
		found := false
		for existingKey, existingValue := range cache {
			expiry := expiresAt(existingValue)
			if !found || expiry.Before(oldestExpiry) {
				oldestKey, oldestExpiry, found = existingKey, expiry, true
			}
		}
		if !found {
			break
		}
		cacheBytes -= entryBytes(oldestKey, cache[oldestKey])
		delete(cache, oldestKey)
	}

	cache[key] = value
	return true
}

func splitParagraphBytes(paragraphs []string) int {
	bytes := 0
	for _, paragraph := range paragraphs {
		bytes += len(paragraph)
	}
	return bytes
}

func (r *RemoteSpeechTextProcessor) NormalizeForSpeech(text string) string {
	if r == nil || r.Client == nil {
		return text
	}
	r.mu.Lock()
	if r.normCache == nil {
		r.normCache = make(map[string]cachedString)
	}
	if hit, ok := r.normCache[text]; ok && time.Now().Before(hit.expiresAt) {
		r.mu.Unlock()
		return hit.value
	}
	if r.normFlights == nil {
		r.normFlights = make(map[string]*processorFlight[string])
	}
	if flight := r.normFlights[text]; flight != nil {
		flight.waiters++
		r.mu.Unlock()
		<-flight.done
		return flight.value
	}
	flight := &processorFlight[string]{done: make(chan struct{})}
	r.normFlights[text] = flight
	r.mu.Unlock()

	out := text
	cacheable := false
	defer func() {
		r.mu.Lock()
		flight.value = out
		if cacheable {
			now := time.Now()
			entry := cachedString{value: out, expiresAt: now.Add(remoteProcessorCacheTTL)}
			storeRemoteProcessorCacheEntry(r.normCache, text, entry, now,
				func(value cachedString) time.Time { return value.expiresAt },
				func(key string, value cachedString) int { return len(key) + len(value.value) })
		}
		delete(r.normFlights, text)
		close(flight.done)
		r.mu.Unlock()
	}()

	if err := r.Client.Ensure(); err != nil {
		return out
	}
	resp, err := r.Client.Clients().TTS.NormalizeForSpeech(
		context.Background(),
		connect.NewRequest(&ttsv1.NormalizeForSpeechRequest{Text: text}),
	)
	if err != nil || resp == nil || resp.Msg == nil {
		return out
	}
	out = resp.Msg.Text
	cacheable = true
	return out
}

func (r *RemoteSpeechTextProcessor) SplitIntoParagraphs(text string) []string {
	if r == nil || r.Client == nil {
		return []string{text}
	}
	r.mu.Lock()
	if r.splitCache == nil {
		r.splitCache = make(map[string]cachedSplit)
	}
	if hit, ok := r.splitCache[text]; ok && time.Now().Before(hit.expiresAt) {
		r.mu.Unlock()
		return slices.Clone(hit.value)
	}
	if r.splitFlights == nil {
		r.splitFlights = make(map[string]*processorFlight[[]string])
	}
	if flight := r.splitFlights[text]; flight != nil {
		flight.waiters++
		r.mu.Unlock()
		<-flight.done
		return slices.Clone(flight.value)
	}
	flight := &processorFlight[[]string]{done: make(chan struct{})}
	r.splitFlights[text] = flight
	r.mu.Unlock()

	out := []string{text}
	cacheable := false
	defer func() {
		r.mu.Lock()
		flight.value = slices.Clone(out)
		if cacheable {
			now := time.Now()
			entry := cachedSplit{value: slices.Clone(out), expiresAt: now.Add(remoteProcessorCacheTTL)}
			storeRemoteProcessorCacheEntry(r.splitCache, text, entry, now,
				func(value cachedSplit) time.Time { return value.expiresAt },
				func(key string, value cachedSplit) int { return len(key) + splitParagraphBytes(value.value) })
		}
		delete(r.splitFlights, text)
		close(flight.done)
		r.mu.Unlock()
	}()

	if err := r.Client.Ensure(); err != nil {
		return out
	}
	resp, err := r.Client.Clients().TTS.SplitParagraphs(
		context.Background(),
		connect.NewRequest(&ttsv1.SplitParagraphsRequest{Text: text}),
	)
	if err != nil || resp == nil || resp.Msg == nil {
		return out
	}
	out = slices.Clone(resp.Msg.Paragraphs)
	cacheable = true
	return slices.Clone(out)
}
