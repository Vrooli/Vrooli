# Configuration

Bootstrap: [CODE: api/main.go#main] | Config: [CODE: internal/config/config.go#Load] | Pruning: [CODE: internal/store/pruner.go#StartPruner]

All tunable levers are centralized in the `internal/config` package. Every parameter has a sane default and can be overridden via environment variable. The UI has corresponding client-side constants in `ui/src/lib/constants.ts`.

## Environment Variables

These configure vrooli-events at startup. They are typically set via the Vrooli lifecycle system (`.vrooli/service.json`).

### Core

| Variable | Default | Description |
|----------|---------|-------------|
| `API_PORT` | (lifecycle) | HTTP API listen port (managed by Vrooli) |
| `DB_PATH` | `~/.vrooli/vrooli-events/events.db` | SQLite database file path |

### Retention & Pruning

Controls how long events are kept and how aggressively the store is cleaned up. Higher retention = more history but more disk usage.

| Variable | Default | Description | Impact |
|----------|---------|-------------|--------|
| `RETENTION_MAX_AGE` | `720h` (30 days) | Maximum age of stored events (Go duration) | Higher = more history, more disk |
| `RETENTION_MAX_SIZE_BYTES` | `2147483648` (2 GB) | Maximum total payload size | Higher = more storage before pruning |
| `PRUNE_INTERVAL` | `6h` | How often the background pruner runs (Go duration) | Lower = more frequent cleanup, slightly more CPU |

### SSE / Broker

Controls the Server-Sent Events (SSE) pub-sub system behavior. These affect real-time streaming performance and client reconnection.

| Variable | Default | Description | Impact |
|----------|---------|-------------|--------|
| `SSE_SUBSCRIBER_BUF_SIZE` | `64` | Per-subscriber channel buffer size | Higher = more burst tolerance, more memory per connection |
| `SSE_HEARTBEAT_INTERVAL` | `30s` | SSE heartbeat frequency (Go duration) | Lower = faster disconnect detection, more traffic |
| `SSE_RETRY_MS` | `5000` | Client retry directive in SSE stream (ms) | Lower = faster reconnection, more server load |
| `SSE_REPLAY_LIMIT` | `1000` | Max events replayed on reconnect via Last-Event-ID | Higher = better catch-up after disconnect, slower reconnect |

### API Limits

Controls request handling and query behavior.

| Variable | Default | Description | Impact |
|----------|---------|-------------|--------|
| `API_MAX_BODY_BYTES` | `1048576` (1 MB) | Max request body size for event ingestion | Higher = larger payloads accepted |
| `API_QUERY_LIMIT_DEFAULT` | `100` | Default query limit when not specified by client | Higher = more results by default |
| `API_QUERY_LIMIT_MAX` | `1000` | Maximum allowed query limit (caps client requests) | Higher = allows larger result sets |

## UI Client Constants

Client-side tunable values are in `ui/src/lib/constants.ts`. These do not require a server restart.

| Constant | Default | Description |
|----------|---------|-------------|
| `HEALTH_POLL_INTERVAL_MS` | `10000` | How often pages poll `/health` (ms) |
| `METRICS_POLL_INTERVAL_MS` | `5000` | Analytics dashboard poll interval (ms) |
| `STREAM_MAX_EVENTS` | `200` | Max events retained in the live stream buffer |
| `QUERY_LIMIT_OPTIONS` | `[25, 50, 100, 500]` | Available limit choices in the event log |
| `INPUT_CLASS` | (Tailwind classes) | Shared CSS class for form inputs |
| `STATUS_COLORS` | healthy/degraded/unhealthy/unknown | Health status indicator color map |

## What Is NOT Configurable (and Why)

| Aspect | Reason |
|--------|--------|
| SQLite pragmas (WAL, busy_timeout, synchronous) | Safety-critical settings that ensure data integrity; changing them risks corruption |
| `MaxOpenConns(1)` | SQLite's single-writer constraint; increasing it causes locking errors |
| Glob fetch multiplier (3x) | Internal optimization detail; the 3x over-fetch ensures accurate glob results after SQL LIKE approximation |
| Write timeout = 0 | Required for SSE; any timeout would kill long-lived streams |
| Navigation routes | Application structure; adding/removing pages requires code changes |

## Configuration Examples

```bash
# High-throughput deployment (more buffer, faster heartbeat)
SSE_SUBSCRIBER_BUF_SIZE=256 SSE_HEARTBEAT_INTERVAL=10s API_MAX_BODY_BYTES=5242880 vrooli scenario start vrooli-events

# Long retention, large store
RETENTION_MAX_AGE=2160h RETENTION_MAX_SIZE_BYTES=10737418240 vrooli scenario start vrooli-events

# Minimal footprint (short retention, small buffer)
RETENTION_MAX_AGE=72h RETENTION_MAX_SIZE_BYTES=536870912 SSE_SUBSCRIBER_BUF_SIZE=16 vrooli scenario start vrooli-events
```

## EmittingResolver Configuration

These settings apply in the discovery package (`packages/api-core/discovery/`) when using EmittingResolver:

| Setting | Default | Description |
|---------|---------|-------------|
| Event buffer capacity | 256 | Local channel size for fire-and-forget emission |
| Batch size | 10 | Max events per HTTP POST to vrooli-events |
| Batch window | 100ms | Max wait before flushing a partial batch |
| Retry attempts | 3 | Retries on failed POST to vrooli-events |
| Retry backoff | 1s | Base backoff between retries |
| SSE reconnect backoff | 1s - 30s | Exponential backoff for policy SSE reconnection |
| Policy stale threshold | 60s | SSE disconnect duration before reporting cache as stale |
