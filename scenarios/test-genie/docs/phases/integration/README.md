# Integration Phase

**ID**: `integration`
**Timeout**: 120 seconds
**Optional**: Yes (when runtime not available)
**Requires Runtime**: Yes

The integration phase tests runtime liveness of a running scenario. It validates CLI functionality and WebSocket connections.

## What Gets Tested

```mermaid
graph TB
    subgraph "Integration Checks"
        CLI[CLI Validation<br/>Commands work correctly]
        WS[WebSocket<br/>Real-time connections]
    end

    START[Start] --> RUNTIME{Scenario<br/>Running?}
    RUNTIME -->|Yes| CLI
    RUNTIME -->|No| SKIP[Skip Phase]

    CLI --> WS
    WS --> DONE[Complete]

    CLI -.->|broken| FAIL[Fail]
    WS -.->|no connection| FAIL

    style CLI fill:#fff3e0
    style WS fill:#f3e5f5
```

## Runtime Requirement

The integration phase requires the scenario to be running:

```bash
# Start scenario first
vrooli scenario start my-scenario

# Then run integration tests
test-genie execute my-scenario --phases integration
```

If the scenario isn't running, the phase is skipped (exit code 2).

The running API URL is automatically detected from:
1. Lifecycle metadata (`~/.vrooli/processes/scenarios/<name>/start-api.json`)
2. Environment variables configured in `service.json`

It is used to derive the WebSocket URL (see below); the integration phase no
longer performs an HTTP health check of its own — scenario health is already
guaranteed by the lifecycle health gate before this phase runs.

## CLI Validation

Checks CLI functionality:

| Check | Command | Expected |
|-------|---------|----------|
| Binary exists | `which <cli>` | Found in PATH |
| Help works | `<cli> help` | Shows usage |
| Version works | `<cli> version` | Shows version with "version" in output |

## WebSocket Validation

Validates WebSocket connectivity for scenarios with real-time features.

### How WebSocket URL is Derived

WebSocket URLs are **derived from the API URL** following the pattern established by `@vrooli/api-base`. This is because scenarios typically proxy WebSocket connections through the same server as HTTP requests:

```
API URL:       http://localhost:8080
WebSocket URL: ws://localhost:8080/api/v1/ws  (derived)
```

The derivation process:
1. **Protocol conversion**: `http://` → `ws://`, `https://` → `wss://`
2. **Path appended**: API URL + configured WebSocket path

This follows the `@vrooli/api-base` architecture where the UI server's `proxyWebSocketUpgrade()` function proxies WebSocket connections to the API server.

**Reference**: See `packages/api-base/docs/concepts/websocket-support.md` for the full WebSocket architecture.

### Configuration

Configure WebSocket validation in `.vrooli/testing.json`:

```json
{
  "integration": {
    "websocket": {
      "enabled": true,
      "path": "/api/v1/ws",
      "max_connection_ms": 2000
    }
  }
}
```

| Field | Description | Default |
|-------|-------------|---------|
| `enabled` | Whether to run WebSocket validation | `true` (if path is set) |
| `path` | WebSocket endpoint path (appended to API URL) | (none - skips if empty) |
| `max_connection_ms` | Maximum connection time in milliseconds | 2000 |

### When WebSocket Validation Runs

WebSocket validation only runs when:
1. The scenario is running (API URL is detected)
2. A `websocket.path` is configured in `testing.json`
3. `websocket.enabled` is not explicitly set to `false`

If any of these conditions are not met, WebSocket validation is gracefully skipped.

### Example: Scenario with WebSocket Support

For a chat scenario using `@vrooli/api-base`:

**`.vrooli/testing.json`**:
```json
{
  "integration": {
    "websocket": {
      "enabled": true,
      "path": "/api/v1/ws",
      "max_connection_ms": 3000
    }
  }
}
```

**What happens**:
1. WebSocket validation: `ws://localhost:8080/api/v1/ws` (derived from the running API URL)

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All integration tests pass |
| 1 | Integration failures |
| 2 | Skipped (runtime not available) |

## Full Configuration Reference

```json
{
  "integration": {
    "websocket": {
      "enabled": true,
      "path": "/api/v1/ws",
      "max_connection_ms": 2000
    }
  }
}
```

| Section | Field | Description | Default |
|---------|-------|-------------|---------|
| `websocket` | `enabled` | Enable WebSocket checks | `true` if path set |
| `websocket` | `path` | WebSocket endpoint path | (none) |
| `websocket` | `max_connection_ms` | Max connection time (ms) | 2000 |

## Related Documentation

- [@vrooli/api-base WebSocket Support](/packages/api-base/docs/concepts/websocket-support.md) - WebSocket architecture

## See Also

- [Phases Overview](../README.md) - All phases
- [Unit Phase](../unit/README.md) - Previous phase
- [Playbooks Phase](../playbooks/README.md) - Next phase
