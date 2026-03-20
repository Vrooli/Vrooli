# Enhanced Plan: Stream of Consciousness Analyzer

## Overview
A frictionless thought-capture app for deep thinkers whose cognition is naturally graph-structured. It eliminates the tradeoff between thinking deeply and capturing accurately by providing a spatial canvas for raw information capture paired with an explicit thought graph for structuring relationships. The app uses a server-authoritative storage model with optimistic local writes for offline support, a portable storage abstraction (PostgreSQL on server, SQLite for future mobile), and mirrors web-console patterns for voice transcription and deployment.

## Clarifications Applied

| Question | Answer | Impact |
|----------|--------|--------|
| Offline sync architecture | Server-first (server authoritative, local caches) | Server is the source of truth. Local writes are optimistic with background sync. No CRDT complexity. Unsynced items are visually marked. |
| User model | Single-user only | No auth/sharing layer needed. Simplifies data model and API — no user IDs, no permissions, no collaboration conflict resolution. |
| Whisper integration | Same approach as web-console scenario | Use local Whisper resource on port 8090 with WebSocket streaming for real-time partials + HTTP batch fallback. Reuse web-console VoiceStreamProvider/WhisperProvider patterns. |
| MVP scope | All five toolbar features in v1 | Mic, plus, graph toggle, agent chat, and export are all in scope for v1. No phased rollout. |
| Canvas scaling | Virtualized canvas (only render visible items) | Spatial indexing (quadtree/R-tree) from day one. Viewport culling for rendering. Canvas positions stored and indexed in the database. |
| Ghost node generation trigger | Periodic if there are changes and previous generation is not still going | Configurable minimum interval (default 30s), max 1 pending generation, batch multiple changes into single LLM call, respect Ollama concurrent request limits. |
| Delivery format | Same approach as web-console; avoid non-portable databases | Use storage repository abstraction pattern from storage-steer skill. PostgreSQL on Vrooli server, SQLite adapter for future mobile/desktop portability. No direct Postgres dependency in business logic. |

## Suggestions Integrated

### Accepted
| Suggestion | Integration |
|------------|-------------|
| S1: Storage repository abstraction with SQLite portability | Repository interface pattern (TaskRepository-style) from storage-steer skill. PostgresSchemeRepository + future SQLiteSchemeRepository. All business logic uses interfaces, never concrete DB types. Must be baked in from day one. |
| S2: Mirror web-console Whisper integration | Local Whisper resource, dual-mode transcription (WebSocket streaming + HTTP batch fallback), whisper-stt capability probe, ffmpeg 16kHz mono WAV transcoding. Reuse VoiceStreamProvider/WhisperProvider on frontend, handleVoiceStreamWS on backend. |
| S3: Server-authoritative sync with optimistic local writes | Write to local queue (IndexedDB), sync to server when online, last-write-wins conflict resolution with server as authority. Service worker handles background sync. Visual indicators for unsynced items. |
| S4: Canvas virtualization with spatial indexing from day one | Quadtree or R-tree spatial index for viewport queries. Store canvas positions in DB with spatial index. Prevents performance cliff at 100+ items. |
| S5: Throttled periodic ghost-node generation | Configurable min interval (30s default), queue depth limit (max 1 pending), batch context for multiple changes, respect Ollama concurrency limits. Prevents resource starvation of other scenarios sharing Ollama. |
| S6: Expose scheme data as reusable scenario capability | Read-only API endpoint (GET /api/v1/schemes/{id}/export) returning full scheme graph in standardized format. Any scenario can consume scheme data. Turns schemes into reusable knowledge artifacts per Vrooli compound intelligence vision. |

### Not Accepted
| Suggestion | Reason |
|------------|--------|
| (none) | All suggestions were accepted. |

## Refined Scope

### Included (Must Have)
- **Scheme CRUD** — create, rename, switch, delete capture workspaces with auto-save
- **Information items** — voice recordings, text, photos/videos, URLs, files, tables, todos on a freeform spatial canvas
- **Thought graph** — directional edges (with undirected toggle), thought detail panel, cross-scheme thought linking
- **Canvas view** — freeform spatial layout with drag-to-reposition, type/thought filtering (dim non-matching), virtualized rendering with spatial indexing
- **Thought graph view** — explicit relationship visualization, tap-to-inspect, add/edit/remove connections
- **Voice capture** — tap-to-record, Whisper transcription (WebSocket streaming + HTTP batch), LLM refinement of raw transcription using nearby canvas context
- **Plus menu** — add text, photo/video, URL, file, table, todo to canvas
- **Agent chat** — agent-manager-powered embedded chat panel with quick-action buttons (ask/suggest/enhance) and free-text conversation; agent has read/write access to scheme state
- **Export** — one-tap flow to detect connected scenarios, generate context prompt with CLI read command, open target app with pre-loaded context
- **Server-authoritative sync** — optimistic local writes, background sync, last-write-wins, visual unsynced indicators
- **Storage abstraction** — repository interface pattern, PostgreSQL implementation for server
- **Scheme export API** — GET /api/v1/schemes/{id}/export for cross-scenario consumption
- **Ghost node suggestions** — periodic LLM-generated thought suggestions as dismissible ghost nodes in graph view, throttled with batching

### Included (Should Have)
- **Offline PWA support** — service worker, IndexedDB local queue, background sync
- **OpenRouter fallback** — alternative to Ollama when local LLM unavailable
- **Cross-scheme thought visual distinction** — different styling for thoughts spanning multiple schemes with navigation to linked schemes

### Excluded (Out of Scope)
- Multi-user / collaboration — single-user only per clarification
- Semantic embeddings / vector search — agent + graph structure replaces this need
- Native mobile app (React Native) — web/PWA approach per web-console pattern
- SQLite implementation — interface designed for it, but only PostgreSQL impl in v1
- End-to-end encryption — single-user local server, not needed

### Deferred (Future)
- SQLite storage backend — for mobile/desktop portability (v2, interface ready from v1)
- Real-time collaboration — if multi-user is ever added
- Advanced graph algorithms — clustering, path finding, community detection on thought graph
- Plugin system for custom information types — extensible type system is designed in but no plugin API yet
- Scheme templates — pre-built starting layouts for common thinking patterns

## Implementation Notes

### Technical Approach
- **Scenario structure**: Standard Vrooli scenario — API (Go), CLI (Go), UI (React)
- **Storage pattern**: Repository interface per storage-steer skill. Interfaces: SchemeRepository, InformationRepository, ThoughtRepository, EdgeRepository. PostgreSQL implementation using environment variables from Vrooli resource system. Schema isolation via service.json.
- **Canvas rendering**: React with viewport virtualization. Spatial index (quadtree) on the data layer for efficient viewport queries. Touch-optimized interactions for mobile-first UX. Consider react-flow or similar library for canvas foundation.
- **Voice pipeline**: Local Whisper resource (port 8090) → WebSocket streaming for real-time partials → ffmpeg transcoding (16kHz mono WAV) → transcription result placed on canvas → optional LLM refinement button.
- **LLM integration**: Ollama (primary) / OpenRouter (fallback) for voice refinement and ghost node generation. Throttled periodic generation with batching.
- **Sync architecture**: IndexedDB as local write queue → service worker background sync → API server (PostgreSQL) as authority → last-write-wins conflict resolution.
- **Schema initialization**: Idempotent SQL in initialization/storage/postgres/schema.sql per storage-steer patterns.

### Integration Points
- **Ollama**: LLM for voice transcription refinement and passive thought suggestions
- **Whisper resource**: Voice-to-text transcription (local, port 8090)
- **Redis**: Caching layer and sync state tracking
- **Agent-manager**: Embedded chat panel with scheme read/write access
- **Web-console / app-monitor**: Export targets for scheme-to-action flow
- **OpenRouter**: Fallback LLM provider when Ollama unavailable

### Dependencies
- **postgres**: Primary structured storage (schemes, thoughts, information, edges, canvas positions)
- **redis**: Caching, sync state, real-time updates
- **ollama**: Local LLM for refinement and ghost node generation
- **whisper-stt**: Voice transcription resource
- **agent-manager**: Embedded agent chat capability
- **api-core/storage**: Filesystem runtime state utilities

### Risks & Mitigations
| Risk | Mitigation |
|------|------------|
| Canvas performance on mobile with many items | Virtualized rendering + spatial indexing from day one (S4) |
| Ollama resource contention with other scenarios | Throttled generation with configurable intervals and queue limits (S5) |
| Offline sync conflicts | Server-authoritative last-write-wins — simple, predictable, no CRDT complexity (S3) |
| Voice transcription latency | WebSocket streaming provides real-time feedback; batch mode as fallback (S2) |
| Storage portability blocked by Postgres-specific SQL | Repository abstraction enforced from day one; all SQL isolated in repository implementations (S1) |
| Agent chat complexity | Leverage existing agent-manager integration patterns; chat is an embedded consumer, not a new agent framework |

## Success Criteria
- [ ] User can open app and immediately start capturing (voice, text, photo) with no setup
- [ ] Canvas supports 200+ items without noticeable performance degradation on mobile
- [ ] Voice recordings are transcribed in real-time with streaming feedback
- [ ] Thought graph accurately represents directional relationships between thoughts
- [ ] Cross-scheme thoughts are visually distinct and navigable
- [ ] Ghost node suggestions appear periodically without interrupting user flow
- [ ] Agent chat can read scheme state and create/modify thoughts and connections
- [ ] Export flow successfully pre-loads scheme context into target scenario
- [ ] App works offline with visual sync status indicators
- [ ] Scheme data is accessible via export API for cross-scenario consumption
- [ ] All storage access goes through repository interfaces (no direct SQL in business logic)

## Readiness Gate
- [x] All critical questions answered
- [x] Scope clearly defined
- [x] Technical approach validated
- [x] Dependencies available (postgres, redis, ollama, whisper-stt, agent-manager all exist as Vrooli resources/scenarios)
- [x] Success criteria measurable
- [x] Archive materials incorporated into staging artifacts (no archive materials present)

**Ready for processing:** Yes

## Staging Artifacts Produced
- `enhance/prd-context.md` — Full PRD context brief covering value proposition, target users, P0/P1/P2 operational targets, tech direction, and dependencies
- `enhance/requirements-context.md` — Technical constraints, validation approach, storage portability requirements, and dependency relationships
- `enhance/doc-outlines.md` — README structure, RESEARCH topics, PROBLEMS entries, and PROGRESS initial entry
