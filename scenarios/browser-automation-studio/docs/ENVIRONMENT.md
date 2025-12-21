# Environment Variables (Single Source)

One canonical reference for configuring the browser-automation-studio scenario (API + UI). Required values are fail-fast; optional values have defaults but can be overridden for deployments. For a curated control-surface view with recommended profiles and tradeoffs, see `docs/CONTROL_SURFACE.md`.

## Required Variables

- `API_PORT` – API server port (20000-24999)
- `UI_PORT` – UI server port (40000-44999)
- `WS_PORT` – WebSocket port (25000-29999)
- `DATABASE_URL` – PostgreSQL connection string (or SQLite file URL)
- `BAS_DB_BACKEND` – `postgres` (default) or `sqlite`
- `MINIO_PORT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET_NAME`
- `BROWSERLESS_PORT`
- `VROOLI_LIFECYCLE_MANAGED` – must be `true`

## Optional Overrides

- Hosts: `API_HOST`, `UI_HOST`, `WS_HOST`, `MINIO_HOST`, `BROWSERLESS_HOST` (default: `localhost`)
- CORS: `CORS_ALLOWED_ORIGINS` (or legacy `ALLOWED_ORIGINS`, `CORS_ALLOWED_ORIGIN`)
- Screenshot defaults: `SCREENSHOT_DEFAULT_WIDTH`, `SCREENSHOT_DEFAULT_HEIGHT`
- Full URLs: `BROWSERLESS_URL`, `MINIO_ENDPOINT`, `BROWSER_AUTOMATION_API_URL`, `BAS_EXPORT_PAGE_URL`, `BAS_UI_BASE_URL`, `UI_SCHEME`, `UI_HOST`, `UI_PORT`, `BAS_EXPORT_PAGE_PATH` (default `/export/composer.html`)

## Database Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_DB_BACKEND` | Backend selector (`postgres` or `sqlite`) | `postgres` |
| `DATABASE_URL` | Full connection URL (Postgres or SQLite) | — |
| `POSTGRES_HOST` / `POSTGRES_PORT` / `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | Postgres parts | `browser_automation_studio` (DB name) |
| `BAS_SQLITE_PATH` | Absolute SQLite path (desktop/electron) | `~/.vrooli/data/browser-automation-studio/bas.sqlite` |
| `SQLITE_DATABASE_PATH` | Override sqlite resource default root | `~/.vrooli/data/sqlite/databases` |
| `BAS_DB_MAX_OPEN_CONNS` | Max open connections | `25` |
| `BAS_DB_MAX_IDLE_CONNS` | Max idle connections | `5` |
| `BAS_DB_CONN_MAX_LIFETIME_MS` | Conn lifetime | `300000` |
| `BAS_DB_MAX_RETRIES` / `BAS_DB_BASE_RETRY_DELAY_MS` / `BAS_DB_MAX_RETRY_DELAY_MS` / `BAS_DB_RETRY_JITTER_FACTOR` | Retry tuning | `10` / `1000` / `30000` / `0.25` |

SQLite defaults to `~/.vrooli/data/sqlite/databases/browser-automation-studio.db` with WAL mode and tuned pragmas.

## Timeout Hierarchy

Keep inner timeouts smaller than outers to get actionable errors.

- Go API HTTP client: 5 min (`playwright_engine.go:48`)
- Playwright driver request: 5 min (`playwright-driver/src/config.ts`)
- Workflow execution: 90–120s default (`simple_executor.go:918-935`), configurable via `executionTimeoutMs`
- Step timeout: execution timeout + 2s buffer (`simple_executor.go:783-784`)
- Startup health check: 5s (`main.go:performStartupHealthCheck`)

### API Timeout Variables (ms)

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_TIMEOUT_DEFAULT_REQUEST_MS` | Standard CRUD | `5000` |
| `BAS_TIMEOUT_EXTENDED_REQUEST_MS` | Complex/bulk | `10000` |
| `BAS_TIMEOUT_AI_REQUEST_MS` | AI workflow gen | `45000` |
| `BAS_TIMEOUT_AI_ANALYSIS_MS` | AI analysis (DOM, screenshots) | `60000` |
| `BAS_TIMEOUT_ELEMENT_ANALYSIS_MS` | Element analysis | `30000` |
| `BAS_TIMEOUT_EXECUTION_COMPLETION_MS` | Wait for execution done | `120000` |
| `BAS_TIMEOUT_GLOBAL_REQUEST_MS` | Overall HTTP server timeout | `900000` |
| `BAS_TIMEOUT_RECORD_MODE_SESSION_MS` | Record mode ops | `30000` |
| `BAS_TIMEOUT_STARTUP_HEALTH_CHECK_MS` | Startup health checks | `10000` |
| `BAS_TIMEOUT_DATABASE_PING_MS` | DB health checks | `2000` |
| `BAS_TIMEOUT_DATABASE_QUERY_MS` | DB queries | `5000` |
| `BAS_TIMEOUT_DATABASE_MIGRATION_MS` | DB migrations | `30000` |
| `BAS_TIMEOUT_BROWSER_ENGINE_HEALTH_MS` | Browser engine health | `5000` |

### Execution Tuning

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_EXECUTION_BASE_TIMEOUT_MS` | Base overhead | `30000` |
| `BAS_EXECUTION_PER_STEP_TIMEOUT_MS` | Per instruction | `10000` |
| `BAS_EXECUTION_PER_STEP_SUBFLOW_TIMEOUT_MS` | Per subflow step | `15000` |
| `BAS_EXECUTION_MIN_TIMEOUT_MS` / `BAS_EXECUTION_MAX_TIMEOUT_MS` | Clamp window | `90000` / `270000` |
| `BAS_EXECUTION_MAX_SUBFLOW_DEPTH` | Max subflow recursion | `5` |
| `BAS_EXECUTION_DEFAULT_ENTRY_TIMEOUT_MS` / `BAS_EXECUTION_MIN_ENTRY_TIMEOUT_MS` | Entry selector timeouts | `3000` / `250` |
| `BAS_EXECUTION_DEFAULT_RETRY_DELAY_MS` / `BAS_EXECUTION_DEFAULT_BACKOFF_FACTOR` | Retry/backoff | `750` / `1.5` |
| `BAS_EXECUTION_COMPLETION_POLL_INTERVAL_MS` | Completion poll | `250` |
| `BAS_EXECUTION_HEARTBEAT_INTERVAL_MS` | Mid-step heartbeat cadence (ms, `0` disables) | `2000` |

`BROWSERLESS_HEARTBEAT_INTERVAL` (Go duration string, e.g. `2s`) is also accepted for backward compatibility.

## WebSocket & Recording

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_WS_CLIENT_SEND_BUFFER_SIZE` | Per-client JSON buffer | `256` |
| `BAS_WS_CLIENT_BINARY_BUFFER_SIZE` | Per-client binary buffer | `120` |
| `BAS_WS_CLIENT_READ_LIMIT` | Max client message size | `512` |
| `BAS_RECORDING_DEFAULT_VIEWPORT_WIDTH` / `BAS_RECORDING_DEFAULT_VIEWPORT_HEIGHT` | Default viewport | `1280` / `720` |
| `BAS_RECORDING_DEFAULT_STREAM_QUALITY` | JPEG quality 1-100 | `55` |
| `BAS_RECORDING_DEFAULT_STREAM_FPS` | FPS 1-60 | `6` |
| `BAS_RECORDING_INPUT_TIMEOUT_MS` | Input forwarding timeout | `2000` |
| `BAS_RECORDING_CONN_IDLE_TIMEOUT_MS` | Driver idle timeout | `90000` |

## Events / Backpressure

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_EVENTS_PER_EXECUTION_BUFFER` | Buffered automation events per execution before telemetry drops | `200` |
| `BAS_EVENTS_PER_ATTEMPT_BUFFER` | Buffered telemetry events per step attempt | `50` |

## AI / DOM Extraction

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_AI_DOM_MAX_DEPTH` | DOM traversal depth | `6` |
| `BAS_AI_DOM_MAX_CHILDREN_PER_NODE` | Max children per node | `12` |
| `BAS_AI_DOM_MAX_TOTAL_NODES` | Max nodes | `800` |
| `BAS_AI_DOM_TEXT_LIMIT` | Text truncation limit | `120` |
| `BAS_AI_DOM_EXTRACTION_WAIT_MS` | Wait after navigation | `750` |
| `BAS_AI_PREVIEW_MIN_VIEWPORT` / `BAS_AI_PREVIEW_MAX_VIEWPORT` | Min/max viewport | `200` / `10000` |
| `BAS_AI_PREVIEW_DEFAULT_WIDTH` / `BAS_AI_PREVIEW_DEFAULT_HEIGHT` | Preview size | `1920` / `1080` |
| `BAS_AI_PREVIEW_WAIT_MS` | Wait before screenshot | `1200` |
| `BAS_AI_PREVIEW_TIMEOUT_MS` | Preview timeout | `20000` |

## Replay / Export

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_REPLAY_CAPTURE_INTERVAL_MS` | Frame capture interval | `40` |
| `BAS_REPLAY_CAPTURE_TAIL_MS` | Extra capture after last action | `320` |
| `BAS_REPLAY_PRESENTATION_WIDTH` / `BAS_REPLAY_PRESENTATION_HEIGHT` | Output size | `1280` / `720` |
| `BAS_REPLAY_MAX_CAPTURE_FRAMES` | Max frames | `720` |
| `BAS_REPLAY_RENDER_TIMEOUT_MS` | Video render timeout | `960000` |
| `BAS_EXPORT_FRAME_INTERVAL_MS` | Legacy frame override | — |
| `FFMPEG_BIN` | Custom ffmpeg path | auto-detect |

## Storage

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_STORAGE_MAX_EMBEDDED_EXTERNAL_BYTES` | Max embedded file size | `5242880` (5 MB) |

## Playwright Driver

| Variable | Description | Default |
|----------|-------------|---------|
| `PLAYWRIGHT_DRIVER_URL` | Driver HTTP URL | `http://127.0.0.1:39400` |

## Entitlement Configuration (optional)

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_ENTITLEMENT_ENABLED` | Enable entitlement checks | `false` |
| `BAS_ENTITLEMENT_SERVICE_URL` | Entitlement service URL | — |
| `BAS_ENTITLEMENT_DEFAULT_TIER` | Fallback tier | `free` |
| `BAS_ENTITLEMENT_CACHE_TTL_MS` | Cache TTL | `300000` |
| `BAS_ENTITLEMENT_REQUEST_TIMEOUT_MS` | Request timeout | `5000` |
| `BAS_ENTITLEMENT_OFFLINE_GRACE_PERIOD_MS` | Offline grace period | `86400000` |
| `BAS_ENTITLEMENT_TIER_LIMITS_JSON` | Monthly execution limits JSON | `{"free":50,"solo":200,"pro":-1,"studio":-1,"business":-1}` |
| `BAS_ENTITLEMENT_WATERMARK_TIERS` | Watermarked tiers | `free,solo` |
| `BAS_ENTITLEMENT_AI_TIERS` | AI access tiers | `pro,studio,business` |
| `BAS_ENTITLEMENT_RECORDING_TIERS` | Recording access tiers | `solo,pro,studio,business` |

## Lifecycle Variables

Set automatically by Vrooli lifecycle:

- `VROOLI_LIFECYCLE_MANAGED`
- `LOG_LEVEL`

## Examples

```bash
# Core (set by lifecycle)
export API_PORT=20100
export UI_PORT=40100
export WS_PORT=25100
export DATABASE_URL="postgresql://user:pass@localhost:5432/browser_automation"
export MINIO_PORT=9000
export MINIO_ACCESS_KEY="minioadmin"
export MINIO_SECRET_KEY="minioadmin"
export MINIO_BUCKET_NAME="browser-automation-screenshots"
export BROWSERLESS_PORT=3000
export VROOLI_LIFECYCLE_MANAGED=true

# Optional tuning
export BAS_TIMEOUT_AI_REQUEST_MS=90000
export BAS_DB_MAX_OPEN_CONNS=50
export CORS_ALLOWED_ORIGINS="https://app.example.com,https://dashboard.example.com"
```

### Tuning Cheatsheet

- Faster local tests: `BAS_TIMEOUT_DEFAULT_REQUEST_MS=2500`, `BAS_TIMEOUT_AI_REQUEST_MS=22500`
- High concurrency: `BAS_DB_MAX_OPEN_CONNS=50`, `BAS_DB_MAX_IDLE_CONNS=10`
- Memory constrained: `BAS_DB_MAX_OPEN_CONNS=10`, `BAS_DB_MAX_IDLE_CONNS=2`
- Heavier DOMs: increase `BAS_AI_DOM_MAX_DEPTH`, `BAS_AI_DOM_MAX_TOTAL_NODES`, `BAS_AI_DOM_EXTRACTION_WAIT_MS`
