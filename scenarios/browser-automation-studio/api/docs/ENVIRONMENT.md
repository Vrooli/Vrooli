# Environment Variables

This document describes the environment variables used to configure the browser-automation-studio API.

## Database Configuration

### Backend Selection

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_DB_BACKEND` | Database backend: `postgres` or `sqlite` | `postgres` |

### PostgreSQL

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | Full PostgreSQL connection URL | - |
| `POSTGRES_HOST` | PostgreSQL host | - |
| `POSTGRES_PORT` | PostgreSQL port | - |
| `POSTGRES_USER` | PostgreSQL username | - |
| `POSTGRES_PASSWORD` | PostgreSQL password | - |
| `POSTGRES_DB` | Database name | `browser_automation_studio` |

### SQLite (desktop-friendly)

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_SQLITE_PATH` | Absolute path to SQLite database file | - |
| `DATABASE_URL` | SQLite file URL (e.g., `file:/abs/path/to/bas.db`) | - |
| `SQLITE_DATABASE_PATH` | Override sqlite resource default root | `VROOLI_DATA` or `~/.vrooli/data/sqlite/databases` |

SQLite defaults to `~/.vrooli/data/sqlite/databases/browser-automation-studio.db` with WAL mode and optimized pragmas.

### Connection Pool & Retry

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_DB_MAX_OPEN_CONNS` | Maximum open database connections | `25` |
| `BAS_DB_MAX_IDLE_CONNS` | Maximum idle connections in pool | `5` |
| `BAS_DB_CONN_MAX_LIFETIME_MS` | Connection lifetime in milliseconds | `300000` (5 min) |
| `BAS_DB_MAX_RETRIES` | Maximum connection retry attempts | `10` |
| `BAS_DB_BASE_RETRY_DELAY_MS` | Initial retry delay in milliseconds | `1000` |
| `BAS_DB_MAX_RETRY_DELAY_MS` | Maximum retry delay in milliseconds | `30000` |
| `BAS_DB_RETRY_JITTER_FACTOR` | Random jitter factor for retries (0-1) | `0.25` |

## Timeout Configuration

All timeouts are specified in milliseconds.

### HTTP Request Timeouts

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_TIMEOUT_DEFAULT_REQUEST_MS` | Standard CRUD operations | `5000` |
| `BAS_TIMEOUT_EXTENDED_REQUEST_MS` | Complex queries, bulk operations | `10000` |
| `BAS_TIMEOUT_AI_REQUEST_MS` | AI-powered workflow generation | `45000` |
| `BAS_TIMEOUT_AI_ANALYSIS_MS` | Complex AI analysis (screenshots, DOM) | `60000` |
| `BAS_TIMEOUT_ELEMENT_ANALYSIS_MS` | Element analysis with screenshots | `30000` |
| `BAS_TIMEOUT_EXECUTION_COMPLETION_MS` | Wait for execution completion | `120000` (2 min) |
| `BAS_TIMEOUT_GLOBAL_REQUEST_MS` | Overall HTTP server timeout | `900000` (15 min) |
| `BAS_TIMEOUT_RECORD_MODE_SESSION_MS` | Record mode session operations | `30000` |
| `BAS_TIMEOUT_STARTUP_HEALTH_CHECK_MS` | Startup health checks | `10000` |

### Database Timeouts

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_TIMEOUT_DATABASE_PING_MS` | Health checks and verification | `2000` |
| `BAS_TIMEOUT_DATABASE_QUERY_MS` | Standard database queries | `5000` |
| `BAS_TIMEOUT_DATABASE_MIGRATION_MS` | Schema migrations | `30000` |

### External Service Timeouts

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_TIMEOUT_BROWSER_ENGINE_HEALTH_MS` | Browser engine health checks | `5000` |

## Execution Configuration

Controls workflow execution timeout calculation: `base + (stepCount * perStep)`, clamped to `[min, max]`.

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_EXECUTION_BASE_TIMEOUT_MS` | Minimum overhead for session setup | `30000` |
| `BAS_EXECUTION_PER_STEP_TIMEOUT_MS` | Time allocated per instruction | `10000` |
| `BAS_EXECUTION_PER_STEP_SUBFLOW_TIMEOUT_MS` | Per-step time for nested workflows | `15000` |
| `BAS_EXECUTION_MIN_TIMEOUT_MS` | Minimum execution timeout | `90000` |
| `BAS_EXECUTION_MAX_TIMEOUT_MS` | Maximum execution timeout | `270000` (4.5 min) |
| `BAS_EXECUTION_MAX_SUBFLOW_DEPTH` | Maximum nested subflow recursion | `5` |
| `BAS_EXECUTION_DEFAULT_ENTRY_TIMEOUT_MS` | Default entry selector timeout | `3000` |
| `BAS_EXECUTION_MIN_ENTRY_TIMEOUT_MS` | Minimum entry selector timeout | `250` |
| `BAS_EXECUTION_DEFAULT_RETRY_DELAY_MS` | Initial retry delay | `750` |
| `BAS_EXECUTION_DEFAULT_BACKOFF_FACTOR` | Exponential backoff multiplier | `1.5` |
| `BAS_EXECUTION_COMPLETION_POLL_INTERVAL_MS` | Polling interval for completion | `250` |

## WebSocket Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_WS_CLIENT_SEND_BUFFER_SIZE` | JSON message buffer per client | `256` |
| `BAS_WS_CLIENT_BINARY_BUFFER_SIZE` | Binary frame buffer per client | `120` |
| `BAS_WS_CLIENT_READ_LIMIT` | Maximum client message size | `512` |

## Recording Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_RECORDING_DEFAULT_VIEWPORT_WIDTH` | Default viewport width | `1280` |
| `BAS_RECORDING_DEFAULT_VIEWPORT_HEIGHT` | Default viewport height | `720` |
| `BAS_RECORDING_DEFAULT_STREAM_QUALITY` | JPEG quality (1-100) | `55` |
| `BAS_RECORDING_DEFAULT_STREAM_FPS` | Frames per second (1-60) | `6` |
| `BAS_RECORDING_INPUT_TIMEOUT_MS` | Input event forwarding timeout | `2000` |
| `BAS_RECORDING_CONN_IDLE_TIMEOUT_MS` | Driver connection idle timeout | `90000` |

## AI/DOM Extraction Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_AI_DOM_MAX_DEPTH` | Maximum DOM traversal depth | `6` |
| `BAS_AI_DOM_MAX_CHILDREN_PER_NODE` | Maximum children per node | `12` |
| `BAS_AI_DOM_MAX_TOTAL_NODES` | Maximum total nodes extracted | `800` |
| `BAS_AI_DOM_TEXT_LIMIT` | Text content truncation limit | `120` |
| `BAS_AI_DOM_EXTRACTION_WAIT_MS` | Wait time after navigation | `750` |
| `BAS_AI_PREVIEW_MIN_VIEWPORT` | Minimum viewport dimension | `200` |
| `BAS_AI_PREVIEW_MAX_VIEWPORT` | Maximum viewport dimension | `10000` |
| `BAS_AI_PREVIEW_DEFAULT_WIDTH` | Default preview width | `1920` |
| `BAS_AI_PREVIEW_DEFAULT_HEIGHT` | Default preview height | `1080` |
| `BAS_AI_PREVIEW_WAIT_MS` | Wait time before screenshot | `1200` |
| `BAS_AI_PREVIEW_TIMEOUT_MS` | Preview operation timeout | `20000` |

## Replay/Video Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_REPLAY_CAPTURE_INTERVAL_MS` | Frame capture interval | `40` |
| `BAS_REPLAY_CAPTURE_TAIL_MS` | Additional capture after last action | `320` |
| `BAS_REPLAY_PRESENTATION_WIDTH` | Output video width | `1280` |
| `BAS_REPLAY_PRESENTATION_HEIGHT` | Output video height | `720` |
| `BAS_REPLAY_MAX_CAPTURE_FRAMES` | Maximum frames to capture | `720` |
| `BAS_REPLAY_RENDER_TIMEOUT_MS` | Video rendering timeout | `960000` (16 min) |
| `BAS_EXPORT_FRAME_INTERVAL_MS` | (Legacy) Frame interval override | - |
| `FFMPEG_BIN` | Custom ffmpeg binary path | auto-detect |

## Storage Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_STORAGE_MAX_EMBEDDED_EXTERNAL_BYTES` | Max embedded file size | `5242880` (5 MB) |

## Playwright Driver Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PLAYWRIGHT_DRIVER_URL` | Playwright driver HTTP URL | `http://127.0.0.1:39400` |

## UI/Export Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_EXPORT_PAGE_URL` | Export composer page URL | - |
| `BAS_UI_EXPORT_URL` | UI export URL (fallback) | - |
| `BAS_UI_BASE_URL` | UI base URL | - |
| `UI_SCHEME` | UI protocol scheme | `http` |
| `UI_HOST` | UI host | `127.0.0.1` |
| `UI_PORT` | UI port | - |
| `BAS_EXPORT_PAGE_PATH` | Export page path | `/export/composer.html` |

## API Server Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `API_SCHEME` | API protocol scheme | `http` |
| `API_HOST` | API host | `127.0.0.1` |
| `API_PORT` | API port | `8080` |

## Testing Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `BAS_SKIP_DEMO_SEED` | Skip demo data seeding | `false` |
| `BAS_SKIP_SQLITE_TESTS` | Skip SQLite smoke tests | `false` |
| `BAS_TEST_BACKEND` | Override backend for tests | `BAS_DB_BACKEND` |

## Tuning Guidelines

### Timeout Tuning

**For faster local testing:** Reduce timeouts by 50%
```bash
export BAS_TIMEOUT_DEFAULT_REQUEST_MS=2500
export BAS_TIMEOUT_AI_REQUEST_MS=22500
```

**For flaky/slow networks:** Increase timeouts by 50-100%
```bash
export BAS_TIMEOUT_AI_REQUEST_MS=90000
export BAS_EXECUTION_MAX_TIMEOUT_MS=540000
```

### Database Pool Tuning

**For high-concurrency scenarios:**
```bash
export BAS_DB_MAX_OPEN_CONNS=50
export BAS_DB_MAX_IDLE_CONNS=10
```

**For memory-constrained environments:**
```bash
export BAS_DB_MAX_OPEN_CONNS=10
export BAS_DB_MAX_IDLE_CONNS=2
```

### WebSocket Buffer Tuning

**For high-traffic real-time updates:**
```bash
export BAS_WS_CLIENT_SEND_BUFFER_SIZE=512
export BAS_WS_CLIENT_BINARY_BUFFER_SIZE=240
```

### AI/DOM Extraction Tuning

**For simpler pages (faster extraction):**
```bash
export BAS_AI_DOM_MAX_DEPTH=4
export BAS_AI_DOM_MAX_TOTAL_NODES=400
```

**For complex SPAs (more thorough extraction):**
```bash
export BAS_AI_DOM_MAX_DEPTH=8
export BAS_AI_DOM_MAX_TOTAL_NODES=1500
export BAS_AI_DOM_EXTRACTION_WAIT_MS=1500
```
