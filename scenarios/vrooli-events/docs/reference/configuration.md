# Configuration

## Environment Variables

These configure vrooli-events at startup. They are typically set via the Vrooli lifecycle system (`.vrooli/service.json`).

| Variable | Default | Description |
|----------|---------|-------------|
| `API_PORT` | `15000` | HTTP API listen port |
| `DB_PATH` | `./data/events.db` | SQLite database file path |
| `LOG_LEVEL` | `info` | Log verbosity: debug, info, warn, error |

## Runtime Settings

These are stored in the SQLite `settings` table and changeable via API or CLI without restart.

### Retention

| Setting | Default | API Field | Description |
|---------|---------|-----------|-------------|
| Retention window | 30 days | `retention_days` | Maximum age of stored events |
| Size cap | 2 GB | `max_size_bytes` | Maximum total payload size |
| Prune interval | 6 hours | `prune_interval_hours` | How often the background pruner runs |

```bash
# View current settings
vrooli-events retention

# Update
vrooli-events retention --retention-days 60 --max-size-gb 4

# Or via API
curl -X PUT http://localhost:${API_PORT}/api/v1/settings/retention \
  -H "Content-Type: application/json" \
  -d '{"retention_days": 60, "max_size_bytes": 4294967296}'
```

### Policy Defaults

| Setting | Default | Description |
|---------|---------|-------------|
| Default policy | `allow` | Behavior when no access control rules match: `allow` or `deny` |
| Policy cache TTL | — | No TTL — caches are updated in real-time via SSE |
| Fail mode | `open` | Behavior when vrooli-events is unreachable: `open` (allow) or `closed` (deny). Configurable per-rule. |

### SSE Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| Heartbeat interval | 30s | SSE heartbeat frequency |
| Retry interval | 5000ms | Recommended client retry (sent in SSE `retry:` field) |
| Subscriber channel capacity | 64 | Buffer size before backpressure drops events |

### EmittingResolver Configuration

These settings apply in the discovery package (`packages/api-core/discovery/`) when using EmittingResolver:

| Setting | Default | Description |
|---------|---------|-------------|
| Event buffer capacity | 256 | Local channel size for fire-and-forget emission |
| Batch size | 10 | Max events per HTTP POST to vrooli-events |
| Batch window | 100ms | Max wait before flushing a partial batch |
| Retry attempts | 3 | Retries on failed POST to vrooli-events |
| Retry backoff | 1s | Base backoff between retries |
| SSE reconnect backoff | 1s → 30s | Exponential backoff for policy SSE reconnection |
| Policy stale threshold | 60s | SSE disconnect duration before reporting cache as stale |
