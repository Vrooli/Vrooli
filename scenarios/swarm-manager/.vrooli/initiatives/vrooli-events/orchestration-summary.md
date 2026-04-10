# Meta-Orchestrator Summary: vrooli-events

## Source
Planning session covering the design of a central event bus and policy engine for all inter-scenario communication in Vrooli.

## Decisions Made
- All inter-scenario calls through the discovery package automatically emit events (no opt-in per scenario)
- Event emission is fire-and-forget (background goroutine, zero latency impact on caller)
- Policy enforcement happens on BOTH ends: sender-side (in discovery package, fail-fast) and receiver-side (middleware, defense-in-depth)
- Policy cache loaded at scenario startup, updated in real-time via SSE from vrooli-events
- If vrooli-events is unreachable, last-known policy holds (configurable fail-open or fail-closed per rule)
- Structured event IDs: {scenario}.{domain}.{action}.{version} (e.g. swarm-manager.backlog.item-completed.v1)
- Subscribers use glob patterns for matching (e.g. swarm-manager.backlog.*)
- SSE chosen over WebSocket for pub/sub (simpler, HTTP-native, auto-reconnects)
- Storage: SQLite + file-based (no Postgres, no Redis) for portability
- Proto contract follows standard approach in packages/proto/schemas/vrooli-events/v1/
- Policy scopes: access control, rate limiting, circuit breaking
- Durable event storage with configurable retention window and size-based pruning

## Architecture

```
discovery package (sender) --fire-and-forget--> vrooli-events (event store + policy engine)
                                                      |
                                              SSE push (policy updates + event subscriptions)
                                                      |
                                              notification-hub (subscribes to events, delivers to devices)
                                                      |
                                              ntfy resource --> APNs --> iPhone
```

## Dependency Notes
- vrooli-events-core-runtime must exist before discovery integration or notification-hub event subscriptions
- notification-hub-signal-extraction and ntfy-resource have no dependency on vrooli-events and can start in parallel
- archive-legacy-notification-hub only depends on signal extraction completing

## Unresolved Questions Deferred To Workshop
- Exact SSE protocol details (heartbeat interval, reconnection strategy, backpressure)
- Event envelope proto schema specifics (what metadata fields beyond scenario/domain/action/version)
- Policy rule schema details (how rules are expressed, stored, and evaluated)
- Circuit breaker thresholds and recovery strategy
- Event retention defaults (days? size cap?)
- Whether event emission should buffer locally during vrooli-events outages or drop
