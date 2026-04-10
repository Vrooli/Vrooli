# TTS Audio Pre-Cache Implementation Plan

## 1. Purpose

Eliminate the perceived latency when switching to a tab with unread assistant messages by pre-synthesizing TTS audio on the backend as soon as messages arrive, rather than synthesizing on-demand when the user switches tabs.

Currently, auto-TTS audio is only synthesized when the user activates a pane with pending messages. For long messages or when Kokoro is under load, this creates a 1-5+ second delay before playback starts. By moving synthesis earlier in the pipeline — triggered immediately after message arrival and summarization — the audio will be ready for instant playback when the user switches tabs.

## 2. Required Reading

```bash
prompt-manager skill read implementation-plan-authoring cli-steer api-steer utils-unification seam-discovery-and-enforcement test boundary-of-responsibility-enforcement documentation-health
```

**Key files to read before implementing:**

| File | Why |
|------|-----|
| `scenarios/web-console/api/conversation_router.go` | Entry point where events are appended and summarized |
| `scenarios/web-console/api/conversation_store.go` | ConversationEvent struct, storage, UpdateSpeechParagraphs |
| `scenarios/web-console/api/tts_synthesize.go` | Existing synthesis handler and TTSSynthesizer interface |
| `scenarios/web-console/api/main.go:390-401` | Route registration |
| `scenarios/web-console/api/terminal_ws.go:205-222` | WebSocket broadcast of conversation events |
| `scenarios/web-console/ui/src/hooks/tts/KokoroProvider.ts` | Frontend synthesis flow, blob management |
| `scenarios/web-console/ui/src/hooks/useTextToSpeech.ts` | Public TTS API, backend selection, speakParagraphs |
| `scenarios/web-console/ui/src/components/TerminalPane.tsx:199-282` | Auto-TTS triggers (immediate + catch-up) |
| `scenarios/web-console/ui/src/components/MessagesPane.tsx:227-237` | Summarized vs. original version selection |

## 3. Problem Statement

When a user switches to a web-console tab containing an unread assistant message with auto-TTS enabled:

1. The catch-up effect in `TerminalPane.tsx:257-282` detects pending unlistened events
2. It calls `speakParagraphs()` which invokes `KokoroProvider.speakSequence()`
3. `speakSequence()` makes a **blocking HTTP POST** to `/api/v1/tts/synthesize` for each paragraph
4. Each synthesis call takes 1-3+ seconds (Kokoro cold-start or load)
5. For multi-paragraph messages, total latency = sum of all paragraph synthesis times

The user experiences a dead pause between switching tabs and hearing the message. The audio could have been synthesized during the time the message was sitting unread.

**Additional complexity:** Messages may have both an **original** (unabridged) and **summarized** (abridged) version. The system must pre-cache the correct version based on the user's current summarization configuration, and handle the case where the user toggles to the alternate version.

## 4. Scope

### In Scope

- Backend audio cache service (in-memory, per-session)
- Async pre-synthesis goroutine triggered after event append + summarization
- New cache retrieval endpoint
- Frontend "try cache first" logic before falling back to on-demand synthesis
- Cache invalidation on session cleanup
- Version-aware caching (summarized vs. original)
- WebSocket hint field so frontend knows cache is available
- Tests for the new backend cache and endpoint

### Out of Scope

- Persistent cache across server restarts (cold cache is acceptable)
- Redis or external cache backend (in-memory is sufficient for current scale)
- Changes to the summarization flow itself
- Changes to browser TTS fallback path (no network call needed)
- CLI endpoint for cache (this is a UI-only concern)
- Streaming/chunked playback (play first paragraph while synthesizing rest) — valuable but separate effort

## 5. Current Technical Context

### Key Components

**Backend (Go):**

| Component | File | Purpose |
|-----------|------|---------|
| `ConversationEvent` struct | `conversation_store.go:47-61` | Stores text, speechParagraphs, originalSpeechParagraphs, summarized flag |
| `appendConversationEvent()` | `conversation_router.go:34-55` | Entry point: append → maybe summarize → broadcast |
| `maybeSummarizeSpeechParagraphs()` | `conversation_router.go:76-134` | Synchronous summarization via Ollama before broadcast |
| `handleTTSSynthesize()` | `tts_synthesize.go:103-167` | `POST /api/v1/tts/synthesize` — on-demand synthesis |
| `TTSSynthesizer` interface | `tts_synthesize.go` | `Synthesize(ctx, req) (io.ReadCloser, string, error)` |
| WebSocket broadcast | `terminal_ws.go:205-222` | Sends TerminalMessage with speechParagraphs to client |

**Frontend (TypeScript/React):**

| Component | File | Purpose |
|-----------|------|---------|
| `KokoroProvider.speakSequence()` | `KokoroProvider.ts:84-131` | Synthesizes paragraphs → concatenates blobs → plays |
| `useTextToSpeech.speakParagraphs()` | `useTextToSpeech.ts:437-483` | Public entry: delegates to active provider |
| Auto-TTS immediate trigger | `TerminalPane.tsx:199-247` | Plays on active pane when event arrives |
| Auto-TTS catch-up trigger | `TerminalPane.tsx:257-282` | Plays pending events on tab switch |
| Version selection | `MessagesPane.tsx:227-237` | Chooses summarized vs. original paragraphs |

### Data Flow (Current)

```
Assistant response arrives
    → appendConversationEvent() [conversation_router.go:34]
    → maybeSummarizeSpeechParagraphs() [conversation_router.go:50]
        (synchronous, blocks until Ollama returns or times out)
    → event now has final speechParagraphs (summarized or original)
    → sess.SendConversation(event) [conversation_router.go:51]
    → WebSocket delivers to browser [terminal_ws.go:205-222]
    → handleConversationEvent [TerminalPane.tsx:199]
        IF active pane + auto-TTS:
            → speakParagraphs() → KokoroProvider → POST /api/v1/tts/synthesize → play
        ELSE:
            → stored as pending, cursor tracks lastListenedSequence
    → (later) user switches tab → catch-up effect fires
        → speakParagraphs() → POST /api/v1/tts/synthesize → play  ← THE DELAY
```

### Version Selection Logic

The `MessagesPane.tsx:231-235` logic determines which paragraphs to use:

```typescript
const useSummarized = playbackModes[event.id] ?? event.summarized;
const paragraphs = useSummarized
  ? event.speechParagraphs           // summarized version (active)
  : (event.originalSpeechParagraphs ?? event.speechParagraphs);  // original
```

- `event.summarized` is the default (set by backend auto-summarization)
- `playbackModes[event.id]` is a user override (toggle in audio popover)
- The "active" version (`speechParagraphs`) is what auto-TTS plays in TerminalPane

## 6. Target End State

```
Assistant response arrives
    → appendConversationEvent()
    → maybeSummarizeSpeechParagraphs() (unchanged)
    → event has final speechParagraphs
    → [NEW] async goroutine: pre-synthesize audio for speechParagraphs, store in cache
    → sess.SendConversation(event) — includes ttsPreCached hint
    → WebSocket delivers to browser
    → handleConversationEvent
        IF active pane + auto-TTS:
            → [NEW] try GET /api/v1/tts/cache/{eventId} first
            → cache hit: instant blob → play immediately
            → cache miss: fall back to POST /api/v1/tts/synthesize (existing flow)
        ELSE:
            → stored as pending
    → (later) user switches tab → catch-up effect fires
        → [NEW] try cache endpoint first → likely hit → instant playback
        → cache miss: fall back to existing synthesis

User toggles to alternate version (summarized ↔ original):
    → [NEW] try cache with version param
    → cache miss: synthesize on-demand, backend caches result for future
```

## 7. Implementation Strategy

### Phase 1: Backend Audio Cache Service

**New file:** `scenarios/web-console/api/tts_cache.go`

Create an in-memory TTS audio cache:

```go
type TTSCacheKey struct {
    EventID string
    Voice   string
    Speed   float64
    Version string // "active" | "original"
}

type TTSCacheEntry struct {
    Audio       []byte
    ContentType string
    CreatedAt   time.Time
    Size        int
}

type TTSCache struct {
    mu       sync.RWMutex
    entries  map[string]*TTSCacheEntry  // key = hash of TTSCacheKey
    order    []string                    // LRU tracking
    maxSize  int                         // max total bytes (e.g., 100MB)
    currSize int
}
```

**Methods:**
- `NewTTSCache(maxSizeBytes int) *TTSCache`
- `Get(key TTSCacheKey) (*TTSCacheEntry, bool)`
- `Put(key TTSCacheKey, audio []byte, contentType string)`
- `Evict(eventID string)` — remove all entries for an event
- `EvictSession(sessionID string)` — bulk cleanup
- `Stats() TTSCacheStats` — for observability

**LRU eviction:** When `currSize` exceeds `maxSize`, evict oldest entries until under limit.

**Wire into Server struct** in `main.go`:
```go
ttsCache *TTSCache  // add to Server struct
```

Initialize in server setup with configurable max size (default 100MB).

### Phase 2: Async Pre-Synthesis After Event Append

**Modify:** `scenarios/web-console/api/conversation_router.go`

After `maybeSummarizeSpeechParagraphs()` and before/alongside `SendConversation()`, launch an async goroutine:

```go
// In appendConversationEvent(), after line 50:
go s.preSynthesizeTTS(event, sessionID)
```

**New method** on Server (can live in `tts_cache.go` or a new `tts_precache.go`):

```go
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

    // Use the TTS config to get current voice and speed
    cfg := s.getTTSConfig()

    // Synthesize each paragraph, concatenate into single MP3
    // Use a reasonable timeout (e.g., 30s total)
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    audio, contentType, err := s.synthesizeParagraphs(ctx, event.SpeechParagraphs, cfg.Voice, cfg.Speed)
    if err != nil {
        // Log warning, don't fail — cache miss is handled gracefully
        return
    }

    key := TTSCacheKey{
        EventID: event.ID,
        Voice:   cfg.Voice,
        Speed:   cfg.Speed,
        Version: "active",
    }
    s.ttsCache.Put(key, audio, contentType)
}
```

**Important:** The `synthesizeParagraphs` helper should reuse the same Kokoro synthesis logic as `handleTTSSynthesize`, but for multiple paragraphs concatenated. Extract a shared internal method if one doesn't already exist.

### Phase 3: Cache Retrieval Endpoint

**New handler** in `tts_cache.go` (or `tts_cache_handler.go`):

```
GET /api/v1/tts/cache/{eventId}?voice={voice}&speed={speed}&version={active|original}
```

- Returns cached MP3 blob if found (200 with audio content-type)
- Returns 404 if not cached
- `version` defaults to `"active"` (the current speechParagraphs, which may be summarized)
- `version=original` returns cache for originalSpeechParagraphs (if cached)

**Register route** in `main.go` alongside other TTS routes:
```go
s.router.HandleFunc("/api/v1/tts/cache/{eventId}", s.handleGetTTSCache).Methods("GET")
```

### Phase 4: WebSocket Hint Field

**Modify:** `terminal_ws.go` — add `TTSPreCached bool` to the TerminalMessage struct:

```go
TTSPreCached bool `json:"ttsPreCached,omitempty"`
```

This is set to `true` when the pre-synthesis goroutine completes before broadcast, or when the cache is populated. Since pre-synthesis is async and the broadcast happens immediately, this field will typically be `false` on initial broadcast. However, it's still useful for:
- Future optimization where we delay broadcast slightly to wait for cache
- The catch-up path where events are re-sent on reconnect

**Alternative approach (simpler):** Skip this field entirely and always have the frontend check the cache endpoint first. The 404 response is fast and the overhead is minimal. This avoids coupling the WebSocket message format to cache state. **Recommend starting with this simpler approach.**

### Phase 5: Frontend Cache-First Logic

**Modify:** `scenarios/web-console/ui/src/lib/api.ts`

Add a new API function:

```typescript
export async function fetchCachedTTS(
    eventId: string,
    voice: string,
    speed: number,
    version: "active" | "original" = "active",
    signal?: AbortSignal,
): Promise<Blob | null> {
    const params = new URLSearchParams({ voice, speed: String(speed), version });
    const resp = await fetch(`${apiBase()}/api/v1/tts/cache/${eventId}?${params}`, { signal });
    if (resp.status === 404) return null;
    if (!resp.ok) return null;
    return resp.blob();
}
```

**Modify:** `scenarios/web-console/ui/src/hooks/tts/KokoroProvider.ts`

Add a `speakFromBlob(blob: Blob)` method (or modify `speak`/`speakSequence` to accept an optional pre-fetched blob):

```typescript
async speakFromBlob(blob: Blob, opts?: TTSSpeakOptions): Promise<void> {
    this.stop();
    this.revokeBlobUrl();
    this.blobUrl = URL.createObjectURL(blob);
    this.audio.src = this.blobUrl;
    // ... same playback setup as existing speak()
}
```

**Modify:** `scenarios/web-console/ui/src/hooks/useTextToSpeech.ts`

Add or modify `speakParagraphs` to accept an optional `eventId` parameter:

```typescript
speakParagraphs: async (
    paragraphs: string[],
    opts?: { eventId?: string; version?: "active" | "original" }
) => Promise<TTSBackend>
```

When `eventId` is provided and backend is Kokoro:
1. Try `fetchCachedTTS(eventId, voice, speed, version)`
2. If blob returned → `provider.speakFromBlob(blob)`
3. If null (cache miss) → fall through to existing `speakSequence(paragraphs)` flow

**Modify:** `scenarios/web-console/ui/src/components/TerminalPane.tsx`

In both auto-TTS triggers, pass the event ID:

- `handleConversationEvent` (line 236): `speakParagraphs(paragraphs, { eventId: event.id })`
- Catch-up effect (line 271): `speakParagraphs(paragraphs, { eventId: pending.id })`

**Modify:** `scenarios/web-console/ui/src/components/MessagesPane.tsx`

In `handleSpeakOne` (line 236), pass the event ID and version:

```typescript
onSpeakOne(event.id, event.text, paragraphs, {
    eventId: event.id,
    version: useSummarized ? "active" : "original"
});
```

### Phase 6: On-Demand Cache Population for Alternate Version

When the user toggles to the alternate version (original ↔ summarized) and it's a cache miss:

1. Frontend falls back to existing `POST /api/v1/tts/synthesize` flow
2. **Backend opportunity:** After synthesis completes for the on-demand request, optionally populate the cache so subsequent plays are cached. This can be done by having the synthesis handler check if the request includes an `eventId` query param, and if so, cache the result.

**Modify:** `handleTTSSynthesize` in `tts_synthesize.go` to accept an optional `eventId` query param. When present, after streaming the response, also store in cache. This is a minor enhancement but avoids separate "populate cache" calls.

## 8. Contract Decisions

### Cache Endpoint Contract

```
GET /api/v1/tts/cache/{eventId}?voice={voice}&speed={speed}&version={active|original}

200 OK
Content-Type: audio/mpeg (or audio/wav, etc.)
Body: raw audio bytes

404 Not Found
Content-Type: application/json
Body: {"error": "not_cached", "message": "No cached audio for this event"}
```

### Cache Key Semantics

The cache key is `(eventId, voice, speed, version)`. This means:
- Changing voice in settings invalidates cache implicitly (different key, old entries expire via LRU)
- Changing speed invalidates similarly
- `version=active` caches whatever is in `speechParagraphs` at synthesis time
- `version=original` caches the `originalSpeechParagraphs`

### Pre-Synthesis Behavior

- Pre-synthesis only caches the **active version** (`speechParagraphs`). This is the version auto-TTS plays.
- The original version is cached on-demand only when the user explicitly toggles to it.
- Pre-synthesis uses the current TTS config (voice, speed) at the time of event arrival.
- If synthesis fails (Kokoro down, timeout), no cache entry is created. The frontend falls back to the existing on-demand path, which will attempt synthesis again or fall back to browser TTS.

### WebSocket Format Change

No change to the WebSocket message format in the initial implementation. The frontend always checks the cache endpoint first. This keeps the change minimal and avoids coupling.

## 9. Testing Plan

### Backend Unit Tests

**New file:** `scenarios/web-console/api/tts_cache_test.go`

| Test | What It Validates |
|------|-------------------|
| `TestTTSCache_PutAndGet` | Basic cache storage and retrieval |
| `TestTTSCache_LRUEviction` | Entries evicted when max size exceeded; oldest first |
| `TestTTSCache_EvictByEventID` | All versions for an event are removed |
| `TestTTSCache_CacheMiss` | Returns nil/false for unknown keys |
| `TestTTSCache_ConcurrentAccess` | Safe under concurrent reads/writes |
| `TestTTSCache_VersionSeparation` | Active and original versions stored independently |

**New tests in or alongside existing test files:**

| Test | What It Validates |
|------|-------------------|
| `TestHandleGetTTSCache_Hit` | Returns 200 with audio bytes when cached |
| `TestHandleGetTTSCache_Miss` | Returns 404 with structured error |
| `TestHandleGetTTSCache_DefaultVersion` | Omitted version defaults to "active" |
| `TestPreSynthesizeTTS_HappyPath` | Goroutine completes, cache populated |
| `TestPreSynthesizeTTS_SynthesizerNil` | Gracefully no-ops when no synthesizer |
| `TestPreSynthesizeTTS_NonAssistant` | Skips user events |

### Integration Tests

| Test | What It Validates |
|------|-------------------|
| `TestEndToEnd_PreCacheFlow` | Append event → verify cache populated → GET returns audio |
| `TestEndToEnd_CacheMissFallback` | Cache empty → GET returns 404 → client can still POST synthesize |

### Frontend Tests

| Test | What It Validates |
|------|-------------------|
| `fetchCachedTTS returns blob on 200` | API client handles success |
| `fetchCachedTTS returns null on 404` | API client handles cache miss |
| `speakParagraphs tries cache first` | With eventId, attempts cache before synthesis |
| `speakParagraphs falls back on miss` | Cache miss triggers existing synthesis path |
| `version param passed correctly` | Active vs. original version flows through |

### Manual Validation Scenarios

1. **Happy path (active tab):** Send assistant message while on active tab → should play immediately (no change from current behavior, but now also caches)
2. **Tab switch (the main fix):** Send assistant message while on different tab → switch to tab → audio should start within ~100ms instead of 1-3s
3. **Summarized version:** Enable summarization → send long message → verify auto-TTS plays summarized version from cache
4. **Toggle to original:** After summarized playback, toggle to "Original" in audio popover → first play may have delay (cache miss for original), subsequent plays instant
5. **Kokoro down:** Stop Kokoro → send message → verify no error, cache just empty, falls back to browser TTS on tab switch
6. **Cache eviction:** Generate enough messages to exceed cache size → verify old entries evicted, new ones cached

## 10. Rollout / Validation Checklist

- [ ] `TTSCache` struct created with Put/Get/Evict/Stats methods
- [ ] `TTSCache` wired into Server struct and initialized in main.go
- [ ] `preSynthesizeTTS()` goroutine fires after `maybeSummarizeSpeechParagraphs()`
- [ ] `GET /api/v1/tts/cache/{eventId}` endpoint registered and functional
- [ ] `fetchCachedTTS()` added to frontend api.ts
- [ ] `KokoroProvider.speakFromBlob()` implemented
- [ ] `speakParagraphs()` accepts eventId, tries cache first
- [ ] `TerminalPane` auto-TTS triggers pass eventId
- [ ] `MessagesPane` handleSpeakOne passes eventId + version
- [ ] All new Go tests pass: `cd scenarios/web-console/api && go test ./... -timeout 300s`
- [ ] All new frontend tests pass
- [ ] Manual test: tab-switch latency reduced from seconds to <200ms
- [ ] Manual test: summarized vs. original toggle works correctly
- [ ] Manual test: Kokoro-down fallback still works
- [ ] Go code formatted: `gofumpt -w scenarios/web-console/api/`
- [ ] Go lint clean: `cd scenarios/web-console/api && golangci-lint run`
- [ ] No regressions in existing TTS tests

## 11. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Pre-synthesis goroutine leaks if Kokoro hangs | Goroutine accumulation, memory pressure | 30s context timeout; bounded concurrency (semaphore or worker pool) |
| Memory usage from cached audio | OOM on long sessions | LRU eviction with configurable max size (default 100MB); evict on session cleanup |
| Race between pre-synthesis and user requesting play | User might get cache miss even though synthesis is in-flight | Acceptable — falls back to on-demand synthesis. Could add "in-progress" state later. |
| Voice/speed settings change between cache and playback | User hears audio with old voice settings | Cache key includes voice+speed; different settings = different key = cache miss → fresh synthesis |
| Summarization config changes after cache populated | Cached audio doesn't match new summarization setting | Active version cached matches speechParagraphs at event time; config changes only affect future events. Existing events' paragraphs are immutable once set. |
| Kokoro synthesis is slow, pre-synthesis doesn't finish before tab switch | User still gets delay on first tab switch | Acceptable degradation — no worse than current behavior. Cache will be ready for subsequent plays. |

## 12. Non-Goals / Prohibited Patterns

- **DO NOT** add persistent/on-disk cache — in-memory with LRU is sufficient for current scale
- **DO NOT** delay the WebSocket broadcast to wait for pre-synthesis — broadcast must remain immediate
- **DO NOT** change the summarization flow or timing — it remains synchronous before broadcast
- **DO NOT** pre-cache the alternate version (original when summarized is active, or vice versa) — only cache on-demand to avoid doubling Kokoro load
- **DO NOT** add cache logic to the browser TTS path — browser TTS has no network latency
- **DO NOT** introduce external dependencies (Redis, disk cache libraries) for this feature
- **DO NOT** change the existing `POST /api/v1/tts/synthesize` contract — the cache endpoint is additive

## 13. Definition of Done

1. **Latency:** Tab-switch-to-playback time for cached messages is <200ms (down from 1-5s)
2. **Correctness:** Auto-TTS plays the correct version (summarized or original) based on user config
3. **Fallback:** Cache misses degrade gracefully to existing on-demand synthesis with no user-visible errors
4. **Memory:** Cache respects configurable size limit with LRU eviction
5. **Tests:** All new backend and frontend tests pass; no regressions in existing tests
6. **Safety:** Pre-synthesis goroutines are bounded and time-limited; no leaks
7. **Zero breaking changes:** Existing TTS flow works identically when cache is empty or disabled
