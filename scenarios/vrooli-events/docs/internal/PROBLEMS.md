# Known Issues & Deferred Decisions

## Open Issues

### P1 — Policy engine not yet implemented
The core event bus (store, SSE, API, CLI) is complete, but the policy engine (access control, rate limiting, circuit breaking) is not yet built. This is tracked by backlog item `execute/vrooli-events-core-runtime` P1 operational targets and `execute/discovery-event-emission-and-policy-cache`.

### P1 — Discovery integration not yet implemented
The EmittingResolver and PolicyMiddleware in `packages/api-core/discovery/` do not exist yet. Currently, scenarios must manually POST events to vrooli-events. Automatic event emission requires the discovery package update tracked by `execute/discovery-event-emission-and-policy-cache`.

### P1 — Persistent subscriptions not yet implemented
Only ephemeral SSE subscriptions exist. Persistent named subscriptions with webhook delivery, health tracking, and auto-disable are planned but not built.

### P2 — Retention settings are hardcoded
Pruning uses hardcoded 30-day retention and 2GB size cap. Configurable settings via API/UI are specified in REQ-ES-004 but not yet implemented.

### P2 — UI dashboard not started
The React UI exists as a scaffold only. All 14 UI requirements (REQ-UI-001 through REQ-UI-014) are planned but not implemented.

## Deferred Decisions

### Event replay mechanism
OT-P2-008 specifies event replay (re-emit historical events to a subscription). Design questions deferred:
- Should replay be rate-limited to avoid overwhelming subscribers?
- Should replayed events be marked differently (e.g., `X-VrooliEvents-Replayed: true`)?
- Should replay support time-range filtering or pattern filtering?

### Metrics export format
OT-P2-009 specifies Prometheus-compatible /metrics endpoint. Deferred until there's a concrete Grafana/monitoring integration need.

### SQLite single-writer scaling
At very high throughput, SQLite's single-writer model could become a bottleneck. Current mitigations (WAL mode, async ingestion) should handle the current scale. If this becomes a real problem, options include:
- Write-ahead buffer with batch commits
- Sharding events across multiple SQLite files by time window
- Switching to a different embedded store (e.g., BadgerDB)

This is explicitly deferred as a non-goal for the current phase.

## Tech Debt

### None currently
The codebase is freshly initialized. No accumulated tech debt yet.
