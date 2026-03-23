# Documentation Outlines: Stream of Consciousness Analyzer

## README Sections

### Structure
1. **Overview** — What it is, the core problem it solves, key insight (dual canvas/graph approach)
2. **Quick Start** — `make start`, open in browser, record first voice note
3. **Features**
   - Canvas view (spatial capture, virtualized rendering)
   - Thought graph view (directional edges, cross-scheme linking)
   - Voice capture (Whisper streaming, LLM refinement)
   - Agent chat (ask/suggest/enhance, free-text)
   - Export (one-tap to connected scenarios)
   - Ghost node suggestions (passive LLM-powered)
4. **Architecture** — Data model diagram, sync flow, storage abstraction
5. **Configuration** — Environment variables, Ollama/OpenRouter settings, ghost node interval
6. **API Reference** — Scheme CRUD, export endpoint, WebSocket voice streaming
7. **Development** — Build, test, lint commands; contribution guidelines

## RESEARCH.md Topics

### Canvas Virtualization
- Evaluation of react-flow vs custom solution for touch-optimized spatial canvas
- Quadtree vs R-tree for spatial indexing performance characteristics
- Mobile touch event handling patterns (drag, pinch-zoom, tap distinction)

### Offline Sync Patterns
- IndexedDB + service worker background sync reliability across browsers
- Last-write-wins timestamp precision and clock skew considerations
- Sync queue ordering and batch optimization

### Voice Streaming
- WebSocket streaming latency characteristics with local Whisper
- Audio format considerations (WAV vs opus for streaming)
- Long recording memory management

## PROBLEMS.md Entries

### P1: Clock Skew in Last-Write-Wins Sync
- **Risk:** Client and server clocks may diverge, causing incorrect conflict resolution
- **Mitigation:** Use server-assigned timestamps for all conflict resolution; client timestamps only for display
- **Status:** Design decision made, needs implementation verification

### P2: Ollama Contention Under Load
- **Risk:** Multiple scenarios + ghost node generation + voice refinement could saturate Ollama
- **Mitigation:** Throttled generation (30s interval, max 1 pending), configurable limits
- **Status:** Architectural mitigation designed, needs load testing

### P3: IndexedDB Storage Limits
- **Risk:** Browsers may impose storage limits on IndexedDB, especially with large voice recordings
- **Mitigation:** Store voice recordings as blobs with size tracking; warn user approaching limits; prioritize sync of large items
- **Status:** Needs browser-specific limit research

## PROGRESS.md Initial Entry

```markdown
# Progress

## v1.0 - Initial Release
- [ ] Storage layer (repository interfaces + PostgreSQL implementation)
- [ ] Scheme CRUD API + UI shell
- [ ] Canvas view with virtualization
- [ ] Information item types (all 7)
- [ ] Voice capture + Whisper integration
- [ ] Thought graph view + edge management
- [ ] Agent chat integration
- [ ] Export flow
- [ ] Ghost node suggestions
- [ ] Offline/sync layer
- [ ] Export API
```
