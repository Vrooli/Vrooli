# Requirements Context: Stream of Consciousness Analyzer

## Technical Constraints

### Storage Portability (Critical)
- All data access MUST go through repository interfaces (SchemeRepository, InformationRepository, ThoughtRepository, EdgeRepository)
- No PostgreSQL-specific SQL in business logic — all dialect-specific queries isolated in repository implementations
- Interface design must accommodate both PostgreSQL (server) and SQLite (future mobile/desktop)
- Connection configuration via Vrooli environment variables (POSTGRES_HOST, POSTGRES_PORT, etc.)
- Schema isolation via service.json schema declaration
- Connection retry with exponential backoff and jitter
- SQLite connections limited to max 1 open connection

### Canvas Performance (Critical)
- Virtualized rendering mandatory — only render items within viewport
- Spatial index (quadtree or R-tree) required for efficient viewport queries
- Canvas positions stored in database with spatial index
- Must maintain smooth interactions at 200+ items on mobile devices
- Touch event handling optimized for mobile-first (drag, tap, pinch-zoom)

### Voice Pipeline (Critical)
- Local Whisper resource expected on port 8090
- Dual-mode: WebSocket streaming for real-time partials, HTTP batch as fallback
- Capability checking via whisper-stt probe before attempting transcription
- Audio transcoding: ffmpeg to 16kHz mono WAV before sending to Whisper
- Recording duration unlimited — streaming mode essential for long recordings

### LLM Resource Sharing (Important)
- Ollama is shared across scenarios — must not monopolize
- Ghost node generation: configurable minimum interval (30s default)
- Max 1 pending generation request at a time
- Batch multiple canvas changes into single LLM context
- Respect Ollama concurrent request limits
- OpenRouter as configurable fallback provider

### Sync Architecture (Important)
- Server is authoritative source of truth
- Local writes go to IndexedDB queue immediately (optimistic)
- Service worker handles background sync when online
- Conflict resolution: last-write-wins with server timestamp authority
- Unsynced items must be visually distinguishable in the UI
- App must be fully functional offline (read + write to local queue)

## Dependency Relationships

```
                    ┌──────────────┐
                    │   postgres   │ ← Primary storage
                    └──────┬───────┘
                           │
┌──────────┐    ┌──────────┴───────────┐    ┌───────────┐
│  redis   │────│   SoC Analyzer API   │────│  ollama   │
└──────────┘    └──────────┬───────────┘    └───────────┘
  (caching)                │                (LLM refinement,
                           │                 ghost nodes)
              ┌────────────┼────────────┐
              │            │            │
      ┌───────┴──┐  ┌──────┴─────┐  ┌──┴──────────┐
      │whisper   │  │agent-      │  │ OpenRouter   │
      │-stt      │  │manager     │  │ (fallback)   │
      └──────────┘  └────────────┘  └──────────────┘
      (voice)       (agent chat)    (LLM fallback)
```

## Validation Approach

### Unit Testing
- Repository interface enables in-memory test implementations
- Canvas spatial index: test viewport queries with known item positions
- Sync queue: test optimistic write → sync → conflict resolution flow
- Ghost node throttling: test interval enforcement and queue limits

### Integration Testing
- Testcontainers with postgres:15-alpine for repository tests (per LPBS pattern)
- Whisper integration: mock WebSocket server for streaming tests
- Agent-manager: mock agent API for chat panel tests
- Export flow: verify prompt generation and target scenario detection

### E2E Testing
- Full capture flow: open app → record voice → transcription appears → refine with LLM
- Canvas interaction: add items → drag → filter → verify spatial positions persist
- Graph manipulation: create thoughts → connect with edges → verify directional display
- Offline flow: disconnect → capture items → reconnect → verify sync
- Agent chat: open panel → send message → verify scheme modifications

### Performance Testing
- Canvas with 200+ items: measure render time, interaction latency on mobile viewport
- Voice streaming: measure time-to-first-partial-transcription
- Ghost node generation: verify throttling under rapid changes
- Sync backlog: measure time to sync 50+ queued items

## Data Model Overview

### Core Entities
- **Scheme**: id, title, created_at, updated_at (the workspace container)
- **Information**: id, scheme_id, type (enum), content (jsonb), canvas_x, canvas_y, created_at, updated_at
- **Thought**: id, title, description, created_at, updated_at (can span schemes)
- **ThoughtSchemeLink**: thought_id, scheme_id (many-to-many for cross-scheme thoughts)
- **ThoughtEdge**: id, source_thought_id, target_thought_id, directed (bool), label, created_at
- **InformationThoughtLink**: information_id, thought_id (many-to-many)

### Information Type Enum
voice_recording, text, photo, video, url, file, table, todo

### Spatial Index
- Canvas positions (canvas_x, canvas_y) on Information table
- Index for viewport queries: WHERE canvas_x BETWEEN $1 AND $2 AND canvas_y BETWEEN $3 AND $4
