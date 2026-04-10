# Stream of Consciousness Analyzer

A frictionless thought-capture app for deep thinkers whose cognition is naturally graph-structured (decision trees, cause/effect chains). Eliminates the tradeoff between thinking deeply and capturing accurately.

**Core insight:** Combine a zero-friction spatial canvas for raw information capture with an explicit thought graph for structuring relationships. LLM-powered refinement and passive suggestions reduce the cognitive load of organizing without interrupting flow.

## Quick Start

```bash
cd scenarios/stream-of-consciousness-analyzer
make start    # Start all services
# Open browser to the UI URL shown in output
# Record your first voice note with the microphone button
```

## Features

- **Canvas view** — Freeform spatial layout with virtualized rendering and spatial indexing (quadtree). Drag items to position, filter by type or thought association.
- **Thought graph view** — Explicit relationship visualization with directional edges (cause/effect, decision/outcome). Cross-scheme thought linking.
- **Voice capture** — Tap to record, real-time Whisper transcription (WebSocket streaming + HTTP batch fallback), LLM refinement using nearby canvas context.
- **Agent chat** — Agent-manager-powered embedded chat with quick-action buttons (ask/suggest/enhance) and free-text conversation. Agent has read/write access to scheme data.
- **Export** — One-tap flow: detect connected scenarios, generate context prompt with CLI read command, open target app with pre-loaded context.
- **Ghost node suggestions** — Periodic LLM-generated thought suggestions as dismissible ghost nodes in graph view, throttled with batching.

## Architecture

- **API**: Go REST server with repository-pattern storage abstraction
- **UI**: React + TypeScript + Vite, mobile-first PWA
- **CLI**: Go CLI with full API parity (schemes, thoughts, edges, info, export, providers, suggestions)
- **Storage**: PostgreSQL (server) via repository interfaces; SQLite-ready for future mobile
- **Sync**: IndexedDB local queue -> service worker -> API server (last-write-wins)

### Data Model

- **Scheme** — Capture workspace (auto-saved, renamable)
- **Information** — Canvas items: voice recordings, text, photos/videos, URLs, files, tables, todos
- **Thought** — Higher-order nodes in the relationship graph (can span schemes)
- **ThoughtEdge** — Directional (or undirected) connections between thoughts

## Configuration

| Variable | Purpose |
|----------|---------|
| `POSTGRES_HOST/PORT/USER/PASSWORD/DB` | PostgreSQL connection |
| `REDIS_URL` | Caching and sync state |
| `OLLAMA_URL` | Local LLM for refinement and ghost nodes |
| `WHISPER_STT_URL` | Voice transcription (default: port 8090) |
| `OPENROUTER_API_KEY` | Fallback LLM provider |
| `GHOST_NODE_INTERVAL` | Ghost node generation interval (default: 30s) |

## API Reference

- `GET/POST/PUT/DELETE /api/v1/schemes` — Scheme CRUD
- `GET/POST/PUT/DELETE /api/v1/schemes/{id}/information` — Information items
- `GET/POST/PUT/DELETE /api/v1/thoughts` — Thought management
- `POST /api/v1/thoughts/{id}/edges` — Edge management
- `GET /api/v1/schemes/{id}/export` — Export scheme graph
- `WS /api/v1/voice/stream` — WebSocket voice streaming

## CLI Commands

```bash
# Health
stream-of-consciousness-analyzer status

# Schemes
stream-of-consciousness-analyzer scheme list [--json]
stream-of-consciousness-analyzer scheme get <id> [--json]
stream-of-consciousness-analyzer scheme create --name NAME [--json]
stream-of-consciousness-analyzer scheme update <id> --name NAME [--json]
stream-of-consciousness-analyzer scheme delete <id>
stream-of-consciousness-analyzer scheme export <id> [--json]

# Thoughts
stream-of-consciousness-analyzer thought list [--scheme ID] [--json]
stream-of-consciousness-analyzer thought get <id> [--json]
stream-of-consciousness-analyzer thought create --title TITLE [--body BODY] [--scheme ID] [--json]
stream-of-consciousness-analyzer thought update <id> [--title TITLE] [--body BODY] [--json]
stream-of-consciousness-analyzer thought delete <id>

# Edges
stream-of-consciousness-analyzer edge list <thought-id> [--json]
stream-of-consciousness-analyzer edge create <source-id> --target TARGET_ID [--label LABEL] [--json]
stream-of-consciousness-analyzer edge delete <edge-id> --thought THOUGHT_ID

# Information
stream-of-consciousness-analyzer info list <scheme-id> [--json]
stream-of-consciousness-analyzer info create --scheme ID --content TEXT [--type TYPE] [--json]
stream-of-consciousness-analyzer info update <info-id> --scheme ID [--content TEXT] [--type TYPE] [--json]
stream-of-consciousness-analyzer info delete <info-id> --scheme ID

# Suggestions
stream-of-consciousness-analyzer provider list [--json]
stream-of-consciousness-analyzer suggestion generate <scheme-id> [--json]
```

## Development

```bash
make start    # Start services
make test     # Run test suite
make logs     # View logs
make stop     # Stop services
```

## Documentation

- [Quick Start](docs/QUICKSTART.md) — get running in 5 minutes
- [Architecture](docs/concepts/ARCHITECTURE.md) — system design and data flow
- [API Reference](docs/reference/api-endpoints.md) — full REST API documentation
- [PRD](PRD.md) — operational targets and product vision
- [Requirements](requirements/README.md) — detailed requirements per target
