# Architecture

## Overview

The Stream of Consciousness Analyzer is a three-tier application: a Go REST API backed by PostgreSQL, a React + Vite UI, and a Go CLI for automation.

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   React UI   │────▶│   Go API     │────▶│  PostgreSQL   │
│  (Vite PWA)  │     │  (gorilla)   │     │  (schemes,    │
│              │     │              │     │   thoughts)   │
└──────────────┘     └──────────────┘     └──────────────┘
                            │
                     ┌──────┴──────┐
                     │   Ollama    │
                     │ (suggestions│
                     │  + fallback)│
                     └─────────────┘
```

## Data Model

Four PostgreSQL tables form the core:

- **schemes** — Capture workspaces. Each scheme is an independent thought-capture session.
- **information** — Canvas items (text, voice, URLs, etc.) positioned spatially within a scheme.
- **thoughts** — Higher-order graph nodes that represent structured ideas. Can span schemes via nullable `scheme_id`.
- **thought_edges** — Directional connections between thoughts with labels (cause/effect, decision/outcome).

See [CODE: api/schema.go] for the DDL and [CODE: api/models.go] for Go struct definitions.

## API Layer

The API is a flat Go package using gorilla/mux with handler factory functions. Each domain has its own service file:

| Service | File | Responsibility |
|---------|------|----------------|
| SchemeService | [CODE: api/scheme_service.go] | Scheme CRUD |
| InformationService | [CODE: api/information_service.go] | Canvas item CRUD |
| ThoughtService | [CODE: api/thought_service.go] | Thought + edge CRUD |
| ExportService | [CODE: api/export_service.go] | Graph export for cross-scenario use |
| SuggestionService | [CODE: api/suggestion_service.go] | LLM-powered ghost node suggestions |

Routes are registered in [CODE: api/main.go#setupRoutes]. Handlers are in [CODE: api/handlers.go].

## UI Layer

React + TypeScript + Vite, mobile-first. Key components:

| Component | File | Purpose |
|-----------|------|---------|
| SchemeList | [CODE: ui/src/components/SchemeList.tsx] | Sidebar for scheme selection/creation |
| TextCapture | [CODE: ui/src/components/TextCapture.tsx] | Zero-friction text input |
| CanvasView | [CODE: ui/src/components/CanvasView.tsx] | Spatial canvas with drag/pan/zoom |
| GraphView | [CODE: ui/src/components/GraphView.tsx] | Thought relationship visualization |

## LLM Integration

The SuggestionService uses Ollama as the primary LLM provider with OpenRouter as a fallback. Suggestions are generated as "ghost nodes" — dismissible thought proposals placed in the graph view. Generation is throttled (configurable interval, max 1 pending request).

## Health Checks

- `/health` — Infrastructure health (database connectivity, critical)
- `/api/v1/health` — Client-facing health endpoint

Both use the `api-core/health` package with database ping as a critical check.
