# Vrooli Events

Central event bus for inter-scenario communication in the Vrooli ecosystem. Provides durable event storage, real-time SSE pub/sub, and a CLI for event management.

## Architecture

```
Producers (any scenario)
    |
    v
POST /api/v1/events  -->  SQLite Store (WAL mode)  -->  Pruner (6h cycle)
    |                           |
    v                           v
SSE Broker  <--  notify  <--  Insert
    |
    v
GET /api/v1/events/subscribe  -->  Consumers (SSE clients)
```

- **Event Store**: SQLite with WAL mode, structured event IDs, metadata, proto-serialized payloads
- **SSE Broker**: Real-time pub/sub with 30s heartbeat, glob-pattern filtering, 64-capacity channels with backpressure
- **Pruner**: Dual-trigger background goroutine (30-day retention + 2GB size cap)

## Quick Start

```bash
cd scenarios/vrooli-events
make start    # Start via Vrooli lifecycle
make test     # Run tests
make logs     # View logs
make stop     # Stop
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/events | Ingest event (202 Accepted) |
| GET | /api/v1/events | Query events (filters: type, source, correlation_id, since, limit) |
| GET | /api/v1/events/subscribe | SSE stream (filters: type, source, target) |
| GET | /health | Health check with store stats |

## CLI

```bash
vrooli-events query --type "swarm-manager.**" --limit 10
vrooli-events subscribe --type "*.created.*"
vrooli-events stats
vrooli-events status
vrooli-events configure api_base http://localhost:15000/api/v1
```

## Event ID Format

Events use structured IDs: `{scenario}.{domain}.{action}.{version}`

Pattern matching:
- `*` matches exactly one segment
- `**` matches one or more segments
- Empty pattern matches everything

## Integration

Other scenarios publish events via HTTP POST:

```bash
curl -X POST http://localhost:${API_PORT}/api/v1/events \
  -H "Content-Type: application/json" \
  -d '{"eventId":"swarm-manager.backlog.created.v1","eventType":"backlog.created.v1","sourceScenario":"swarm-manager","payload":{}}'
```

Subscribe to events via SSE:

```bash
curl -N "http://localhost:${API_PORT}/api/v1/events/subscribe?type=swarm-manager.**"
```

## Storage

Uses embedded SQLite (no external database required). Data stored at runtime in the scenario directory. WAL mode enables concurrent readers with a single writer.

## Related

- Proto schemas: `packages/proto/schemas/vrooli-events/v1/`
- Policy API (planned): `scenarios/swarm-manager/execute/vrooli-events-policy-api-and-middleware/`
- Discovery integration (planned): `scenarios/swarm-manager/execute/discovery-event-emission-and-policy-cache/`
