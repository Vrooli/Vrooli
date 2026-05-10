# Configuration Reference

The Stream of Consciousness Analyzer exposes a small, intentional set of tunable levers. These are the controls an operator or developer can adjust without changing business logic.

## Environment Variables (Runtime)

These are set on the host or in the scenario's environment before starting.

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_URL` | `http://localhost:11434` | Ollama LLM endpoint. Override to point at a remote Ollama instance. |
| `OPENROUTER_API_KEY` | *(unset — provider disabled)* | Set to any non-empty value to activate OpenRouter as a fallback LLM provider. |
| `API_PORT` | Allocated by Vrooli | Port the Go API listens on. Managed by the lifecycle system. |
| `UI_PORT` | Allocated by Vrooli | Port the Vite UI dev server or production server listens on. |
| `POSTGRES_*` | Via `api-core` | Standard PostgreSQL connection variables (host, port, user, password, dbname). |

## API Constants (`api/config.go`)

These are compile-time constants. Changing them requires rebuilding the API binary.

| Constant | Value | Purpose |
|----------|-------|---------|
| `AppVersion` | `1.0.0` | Semantic version reported by the `/health` endpoint. Bump on releases. |
| `ExportFormatVersion` | `vrooli-graph-v1` | Schema version tag in export payloads. Consumers use this to select the right parser. Change only when the export schema changes. |
| `DefaultOllamaURL` | `http://localhost:11434` | Fallback when `OLLAMA_URL` env var is unset. |
| `OpenRouterURL` | `https://openrouter.ai/api/v1` | Fixed endpoint for the OpenRouter fallback provider. |

## UI Constants (`ui/src/lib/config.ts`)

These are importable TypeScript constants. Changing them requires rebuilding the UI bundle.

### Canvas (spatial view)

| Constant | Default | Impact |
|----------|---------|--------|
| `CANVAS_ZOOM_MIN` | `0.25` | Minimum zoom (25%). Lower = see more items at once. |
| `CANVAS_ZOOM_MAX` | `4` | Maximum zoom (400%). Higher = closer inspection. |
| `CANVAS_ZOOM_IN_FACTOR` | `1.1` | Multiplier per scroll-up tick. Closer to 1 = smoother. |
| `CANVAS_ZOOM_OUT_FACTOR` | `0.9` | Multiplier per scroll-down tick. Closer to 1 = smoother. |

### Initial Placement

Controls where newly created items appear on the canvas. Items are placed randomly within a rectangle of the given size, anchored at the canvas origin.

| Constant | Default | Scope |
|----------|---------|-------|
| `INFO_PLACEMENT_WIDTH` | `600` | Width (px) of placement area for information items. |
| `INFO_PLACEMENT_HEIGHT` | `400` | Height (px) of placement area for information items. |
| `THOUGHT_PLACEMENT_WIDTH` | `500` | Width (px) of placement area for thought nodes. |
| `THOUGHT_PLACEMENT_HEIGHT` | `300` | Height (px) of placement area for thought nodes. |

### Graph View

| Constant | Default | Impact |
|----------|---------|--------|
| `EDGE_STROKE_COLOR` | `rgba(148,163,184,0.3)` | SVG line color for thought connections. |
| `EDGE_STROKE_WIDTH` | `2` | SVG line width (px) for thought connections. |
| `GRAPH_MIN_HEIGHT` | `400` | Minimum height (px) of the thought graph container. |
| `LINK_MODE_WAITING` | `__waiting__` | Internal sentinel for link-mode state. Not user-facing. |

### Text Capture

| Constant | Default | Impact |
|----------|---------|--------|
| `TEXT_CAPTURE_ROWS` | `2` | Visible rows in the quick-capture textarea. |

## Shared Utilities (`ui/src/lib/utils.ts`)

| Function | Signature | Purpose |
|----------|-----------|---------|
| `randomCanvasPosition` | `(width, height) → {x, y}` | Generates a random position within the given bounds. Used by TextCapture and GraphView to place new items. |
| `cn` | `(...inputs) → string` | Tailwind class merge utility (clsx + twMerge). |

## What Is NOT Exposed (and Why)

| Decision | Rationale |
|----------|-----------|
| Database schema names | Managed by the lifecycle system; changing them would break initialization. |
| API route paths | Changing routes would break UI and cross-scenario consumers. Versioned via `/api/v1/` prefix. |
| Health check intervals/timeouts | Defined in `.vrooli/service.json` and managed by the lifecycle system, not the app. |
| Textarea auto-expand | Currently fixed rows; auto-expand adds complexity without clear demand. Revisit if users report friction. |
| Edge rendering offsets (70px, 20px) | Tightly coupled to node dimensions in GraphView. Parameterizing without a layout engine would create fragile config. |
