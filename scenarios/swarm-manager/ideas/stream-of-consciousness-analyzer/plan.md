# Implementation Plan: Stream of Consciousness Analyzer

## Purpose
A frictionless thought-capture app for deep thinkers whose cognition is naturally graph-structured. Eliminates the tradeoff between thinking deeply and capturing accurately by providing a spatial canvas for raw information capture paired with an explicit thought graph for structuring relationships.

## Problem Statement
Formalizing thoughts into notes disrupts the deep cognitive state that produces them. Users either go deep and lose fidelity reconstructing later, or capture well but think shallowly. This app provides zero-friction capture (voice, text, photos) on a spatial canvas with an explicit thought graph, so users can capture at the speed of thought and organize later — or let an AI agent organize for them.

## Scope
### In Scope
- **Scheme CRUD** — create, rename, switch, delete capture workspaces with auto-save
- **Information items** — voice recordings, text, photos/videos, URLs, files, tables, todos on a freeform spatial canvas
- **Thought graph** — directional edges (with undirected toggle), thought detail panel, cross-scheme thought linking
- **Canvas view** — freeform spatial layout with drag-to-reposition, type/thought filtering (dim non-matching), virtualized rendering with spatial indexing
- **Thought graph view** — explicit relationship visualization, tap-to-inspect, add/edit/remove connections
- **Voice capture** — tap-to-record, Whisper transcription (WebSocket streaming + HTTP batch), LLM refinement using nearby canvas context
- **Plus menu** — add text, photo/video, URL, file, table, todo to canvas
- **Agent chat** — agent-manager-powered embedded chat panel with quick-action buttons and free-text conversation; agent has read/write scheme access
- **Export** — one-tap flow to detect connected scenarios, generate context prompt, open target app with pre-loaded context
- **Server-authoritative sync** — optimistic local writes, background sync, last-write-wins, visual unsynced indicators
- **Storage abstraction** — repository interface pattern, PostgreSQL implementation for server
- **Scheme export API** — GET /api/v1/schemes/{id}/export for cross-scenario consumption
- **Ghost node suggestions** — periodic LLM-generated thought suggestions as dismissible ghost nodes, throttled with batching

### Out of Scope
- Multi-user / collaboration — single-user only
- Semantic embeddings / vector search — agent + graph structure replaces this
- Native mobile app (React Native) — web/PWA approach
- SQLite implementation — interface designed for it, PostgreSQL only in v1
- End-to-end encryption — single-user local server

## Current Technical Context

### What's Been Built (99/100 completeness score, production_ready)
The foundational CRUD layer is complete and thoroughly tested:

**API (Go):**
- Store interfaces: SchemeStore, InformationStore, ThoughtStore, ExportStore, SuggestionProvider
- Full service implementations with PostgreSQL (SchemeService, InformationService, ThoughtService, ExportService, SuggestionService)
- Handler factory pattern with gorilla/mux routing
- Structured error handling (5 categories: validation, not_found, conflict, dependency, internal)
- LLM provider infrastructure (Ollama primary, OpenRouter fallback) with health checking
- Scan helpers, generic collectRows, deleteByID utilities
- 97% Go test coverage (200+ tests), including sqlmock service-layer tests
- 30s request timeout middleware, status-aware logging

**CLI (Go):**
- 20 commands covering all API operations (schemes, thoughts, edges, information, providers, suggestions)
- cli-core patterns (ParseInterspersed, JSONFlag, Request method)
- Extracted helpers (cmdFlags, requireArg, doRequest, etc.)
- 19 CLI tests

**UI (React + TypeScript):**
- Components: CanvasView, GraphView, SchemeList, ThoughtNode, TextCapture, ExportButton, SuggestionList, ConnectionStatus, ProviderStatus, ErrorBoundary, PanelErrorBoundary, ErrorFallback, ErrorBanner, KeyboardShortcutHelp (29 components total)
- Canvas with pan/zoom/drag, keyboard navigation (arrows, +/-, ?)
- Graph view with link mode, edge creation/deletion, thought detail
- tanstack-query for data fetching with mutation error handling
- useMutationErrors hook, useWindowDrag hook
- Accessibility: aria-labels, aria-live regions, role="application"
- 100% component statement coverage, 197 UI tests
- Lighthouse: perf 100%, a11y 95%, BP 96%, SEO 91%

**Infrastructure:**
- 28 BATS integration tests for all API endpoints
- Comprehensive requirements hierarchy (60 requirements across 6 modules, depth 3.0)
- Idempotent schema initialization (ensureSchema)
- SEAMS.md with 6 change axes and 8 decision points

### What's NOT Yet Implemented (6 core features)
1. **Voice capture / Whisper integration** — no audio recording, no Whisper WebSocket/HTTP, no transcription display, no LLM refinement button
2. **Canvas virtualization** — current CanvasView uses basic div rendering with manual drag handlers, no spatial index, no viewport culling
3. **Ghost node generation** — SuggestionService has provider infrastructure but GenerateSuggestions returns stub data, no actual LLM calls, no ghost node UI rendering
4. **Offline-first / sync** — no IndexedDB, no service worker, no mutation queue, no sync status indicators, all data goes directly to API
5. **Agent chat** — no agent-manager integration, no chat panel UI, no scheme read/write agent access
6. **Export-to-scenario flow** — ExportButton exists but only does JSON download, no scenario detection, no prompt generation, no target app launch

## Target End State
A mobile-first PWA where users can:
1. Open the app and immediately start voice-capturing thoughts with real-time transcription
2. Arrange captures on a spatial canvas that performs smoothly with 200+ items
3. Structure thinking in an explicit thought graph with directional edges
4. Get passive AI-suggested thought connections as ghost nodes
5. Chat with an AI agent that can read/write their scheme
6. Export scheme context to other Vrooli scenarios with one tap
7. Work fully offline with transparent sync when connectivity returns

## Implementation Strategy

### Pending Decisions (Round 3)
The following decisions from workshop round 3 must be resolved before detailed phase planning:

1. **d1 — Feature phasing**: Order of implementation for the 6 remaining features
2. **d2 — Canvas library**: React Flow vs custom quadtree vs Konva.js vs Excalidraw for virtualized canvas
3. **d3 — Voice recording**: Browser MediaRecorder API vs RecordRTC library
4. **d4 — Offline sync timing**: Before new features vs after vs thin queue now
5. **d5 — Agent chat integration**: Embedded iframe vs direct API calls vs shared component library

### Architectural Decisions (Settled)
| Decision | Choice | Source |
|----------|--------|--------|
| Sync model | Server-authoritative with optimistic local writes | Round 1, d1 → B |
| User model | Single-user only | Round 1, d2 → A |
| Whisper approach | Mirror web-console (local resource, WebSocket streaming + HTTP batch) | Round 1, d3 → Other |
| MVP scope | All 5 toolbar features in v1 | Round 1, d4 → A |
| Canvas scaling | Virtualized with spatial indexing | Round 1, d5 → A |
| Ghost node trigger | Periodic when changes exist, max 1 pending | Round 1, d6 → Other |
| Delivery format | Web-console pattern, storage abstraction for portability | Round 1, d7 → Other |
| Storage abstraction | Repository interface (PostgreSQL now, SQLite-ready) | Round 2, s1 → A |
| Whisper pattern | web-console VoiceStreamProvider/WhisperProvider reuse | Round 2, s2 → A |
| Offline pattern | IndexedDB queue, service worker sync, last-write-wins | Round 2, s3 → A |
| Canvas virtualization | Quadtree/R-tree spatial index from day one | Round 2, s4 → A |
| Ghost node throttling | 30s min interval, max 1 pending, batch changes | Round 2, s5 → A |
| Scheme export API | GET /api/v1/schemes/{id}/export | Round 2, s6 → A |

## Contract Decisions
- **API endpoints**: Full REST API already implemented for schemes, information, thoughts, edges, export, providers, suggestions
- **Export format**: JSON with scheme, information, thoughts, edges, and export_format fields
- **Error responses**: Structured JSON with category (validation, not_found, conflict, dependency, internal), message, and retryable flag
- **LLM providers**: Two-tier priority system — Active+!Fallback (primary), Active+Fallback (fallback), !Active (unavailable)

## Testing Plan
### Existing Coverage
- **Go API**: 97% statement coverage via sqlmock unit tests + handler tests (200+ tests)
- **BATS integration**: 28 tests exercising all API endpoints against real PostgreSQL
- **UI components**: 100% statement coverage across all 29 components (197 tests)
- **CLI**: 19 tests covering argument validation and command groups
- **Requirements**: 60 requirements across 6 modules with test validation refs

### Testing Gaps for Remaining Features
- Voice capture: audio recording mock, Whisper WebSocket streaming tests, transcription placement
- Canvas virtualization: viewport culling correctness, spatial index query performance, 200+ item stress test
- Ghost node generation: LLM prompt construction, suggestion rendering, dismiss/accept flows
- Offline sync: IndexedDB read/write, service worker lifecycle, conflict resolution, sync status UI
- Agent chat: message send/receive, scheme mutation via agent, error states
- Export flow: scenario detection, prompt generation, target app handoff

## Rollout / Validation Checklist
- [ ] Voice: record audio → see real-time transcription → place on canvas → refine with LLM
- [ ] Canvas: add 200+ items → pan/zoom remains smooth → viewport culling verified
- [ ] Ghost nodes: make changes → wait 30s → see ghost suggestions → accept/dismiss
- [ ] Offline: disconnect → make changes → reconnect → verify sync completes
- [ ] Agent chat: open panel → ask question about scheme → agent creates thought
- [ ] Export: tap export → select target scenario → verify context loads in target

## Risks + Mitigations
| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Canvas performance on mobile with 200+ items | Medium | High | Virtualized rendering + spatial indexing from day one; benchmark on real mobile device |
| Ollama resource contention with other scenarios | Medium | Medium | Throttled generation (30s interval, max 1 pending), respect Ollama concurrent limits |
| Offline sync conflicts on reconnect | Low | Medium | Last-write-wins with server authority; visual indicators for unsynced items |
| Voice transcription latency | Medium | Medium | WebSocket streaming for real-time feedback; batch mode fallback |
| Storage portability blocked by Postgres-specific SQL | Low | High | Repository abstraction enforced; SQL isolated in implementations |
| Agent chat coupling with agent-manager | Medium | Medium | Pending d5 decision — iframe approach minimizes coupling |
| Canvas library lock-in | Medium | Medium | Pending d2 decision — evaluate before committing |
| Service worker complexity for offline | Medium | High | Consider thin queue first (d4 option C) if full offline is too invasive |

## Non-goals / Prohibited Patterns
- No multi-user support — single-user only, no auth/permissions layer
- No CRDT-based sync — server-authoritative, last-write-wins
- No semantic embeddings / vector search — graph + agent replaces this
- No native mobile app — PWA approach only
- No direct SQL in business logic — all access through repository interfaces
- No plugin system for custom information types in v1

## Definition of Done
1. All 5 toolbar features functional: microphone, plus, graph toggle, agent chat, export
2. Canvas handles 200+ items without noticeable performance degradation on mobile
3. Voice recordings transcribed in real-time with streaming feedback
4. Ghost node suggestions appear periodically without interrupting flow
5. Agent chat can read scheme state and create/modify thoughts and connections
6. Export flow pre-loads scheme context into target scenario
7. App works offline with visual sync status indicators
8. All storage access through repository interfaces
9. Scheme data accessible via export API for cross-scenario consumption
10. Test coverage maintained above 90% for new code
