# PRD Context Brief: Stream of Consciousness Analyzer

## Overview & Value Proposition

**Product:** Stream of Consciousness Analyzer — a frictionless thought-capture app for deep thinkers whose cognition is naturally graph-structured.

**Core Problem:** Formalizing thoughts into notes disrupts the deep cognitive state that produces them. Users face a forced tradeoff: go deep and lose fidelity reconstructing later, or capture well but think shallowly.

**Solution:** Eliminate the tradeoff with a dual-view system: a freeform spatial canvas for raw information capture (zero-friction) paired with an explicit thought graph for structuring relationships (when ready to organize). LLM-powered refinement and passive suggestions reduce the cognitive load of organizing without interrupting flow.

**Target Users:** Deep thinkers, strategists, researchers, and decision-makers whose natural thinking patterns are graph-structured (decision trees, cause/effect chains). Primary usage contexts: mobile (couch, bed, walking), with desktop as secondary.

**Vrooli Compound Intelligence Value:** Every scheme becomes a reusable knowledge artifact via the export API. Other scenarios can consume structured thought graphs, turning personal thinking into system-wide intelligence. The agent chat integration means the system can actively assist in structuring and connecting ideas.

## P0 Operational Targets (Core Capabilities)

### OT-1: Scheme Management
- Create, rename, switch, and delete capture workspaces
- Auto-save on all changes
- Floating title at top of screen for quick rename/switch

### OT-2: Information Capture
- Support types: voice recordings, text, photos/videos, URLs, files, tables, todos
- Place items at center of viewport, draggable after creation
- Zero-friction: open app → start capturing immediately

### OT-3: Canvas View
- Freeform spatial layout with drag-to-reposition
- Virtualized rendering with spatial indexing (quadtree/R-tree) — must handle 200+ items on mobile
- Filter by type or associated thought (dim non-matching items, preserve spatial context)
- Touch-optimized for mobile-first use

### OT-4: Thought Graph View
- Directional edges by default (cause→effect, decision→outcome) with undirected toggle
- Tap thought for detail panel: connected thoughts, linked information, connection controls
- Cross-scheme thoughts display distinctly with navigation to linked schemes

### OT-5: Voice Capture & Transcription
- Tap microphone to start recording (unlimited duration)
- Real-time transcription via local Whisper resource (WebSocket streaming + HTTP batch fallback)
- ffmpeg transcoding to 16kHz mono WAV
- LLM refinement button: converts raw transcription to coherent thought using nearby canvas context
- Raw transcription always accessible when LLM version exists

### OT-6: Agent Chat
- Agent-manager-powered embedded chat panel
- Quick-action buttons: ask, suggest, enhance
- Full free-text multi-turn conversation
- Agent has read/write access to scheme: create thoughts, link information, restructure graph
- Context-injected with full scheme state

### OT-7: Export
- One-tap flow: detect connected scenarios → choose target → generate context prompt with CLI read command → open target app with pre-loaded context
- Initial targets: web-console, agent-manager/app-monitor

### OT-8: Storage Architecture
- Repository interface pattern (per storage-steer skill): SchemeRepository, InformationRepository, ThoughtRepository, EdgeRepository
- PostgreSQL implementation for server deployment
- All business logic uses interfaces, never concrete DB types
- Idempotent schema initialization
- Environment-driven configuration via Vrooli resource system

### OT-9: Server-Authoritative Sync
- Optimistic local writes to IndexedDB queue
- Background sync via service worker when online
- Last-write-wins conflict resolution (server authoritative)
- Visual indicators for unsynced items

## P1 Operational Targets (Important Enhancements)

### OT-10: Ghost Node Suggestions
- LLM-generated thought suggestions as dismissible ghost nodes in graph view
- Periodic generation: configurable interval (30s default), only when changes exist, max 1 pending
- Batch multiple changes into single LLM call
- Respect Ollama concurrent request limits to prevent resource starvation

### OT-11: Scheme Export API
- GET /api/v1/schemes/{id}/export — full scheme graph in standardized format
- Read-only, consumable by any scenario
- Enables compound intelligence: schemes become reusable knowledge artifacts

### OT-12: Offline PWA Support
- Service worker for offline functionality
- IndexedDB for local data persistence
- Background sync when connectivity restored

## P2 Operational Targets (Nice-to-Have Polish)

### OT-13: OpenRouter Fallback
- Alternative LLM provider when Ollama unavailable
- Configurable provider selection

### OT-14: Cross-Scheme Thought Navigation
- Distinct visual styling for cross-scheme thoughts
- One-tap navigation to linked schemes

## Tech Direction Snapshot

- **Stack:** Go API + Go CLI + React UI (standard Vrooli scenario)
- **Storage:** PostgreSQL via repository abstraction (SQLite-ready interface for future mobile)
- **Canvas:** React with viewport virtualization, spatial index (quadtree), touch-optimized
- **Voice:** Local Whisper resource (port 8090), WebSocket streaming, ffmpeg transcoding
- **LLM:** Ollama (primary) / OpenRouter (fallback) for refinement and ghost nodes
- **Sync:** IndexedDB → service worker → API server, last-write-wins
- **Agent:** Agent-manager integration for embedded chat with scheme access

## Dependencies & Launch Plan

### Resource Dependencies
| Resource | Required | Purpose |
|----------|----------|---------|
| postgres | Yes | Primary structured storage |
| redis | Yes | Caching and sync state |
| ollama | Yes | Local LLM for refinement and suggestions |
| whisper-stt | Yes | Voice transcription |

### Scenario Dependencies
| Scenario | Required | Purpose |
|----------|----------|---------|
| agent-manager | Yes | Embedded agent chat |
| web-console | No | Export target |
| app-monitor | No | Export target |

### Launch Sequence
1. Storage layer (repository interfaces + PostgreSQL implementation + schema initialization)
2. Scheme CRUD API + basic UI shell
3. Canvas view with virtualization + information item types
4. Voice capture + Whisper integration
5. Thought graph view + edge management
6. Agent chat integration
7. Export flow
8. Ghost node suggestions
9. Offline/sync layer
10. Export API for cross-scenario consumption
