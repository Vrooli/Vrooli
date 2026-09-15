# Integrating a Scenario

How to connect your scenario to vrooli-events for automatic event emission and policy enforcement.

The platform integration lives in `packages/api-core/eventbus`, not in a scenario-local SDK. It provides a local policy snapshot cache, a background refresher, HTTP middleware, and non-blocking receipt publication. The detailed platform rules are in `path:../../docs/concepts/VROOLI_EVENTS_PLATFORM_CONTRACT.md`.

## Standard API Integration

The standard API server path can use `eventbus.Middleware`. It evaluates only the latest locally cached policy snapshot; it does not make a policy network call while serving the request. A fresh cached denial blocks the request. Missing, stale, or unreachable Vrooli Events state fails open for ordinary traffic.

### What Changes

Before (current):
```go
import "github.com/vrooli/api-core/discovery"

url, err := discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
// Makes HTTP call to agent-manager — nobody knows about it
```

With a scenario-specific safe projection:
```go
import "github.com/vrooli/api-core/eventbus"

cache := eventbus.NewCache()
client := eventbus.Client{BaseURL: eventsURL}
eventbus.StartRefresher(ctx, client, cache, eventbus.RefreshConfig{})

handler := eventbus.Middleware(eventbus.MiddlewareConfig{
    Cache: cache,
    TargetScenario: "my-scenario",
    Receipts: client,
    Projection: func(r *http.Request, status int) (eventbus.Projection, bool) {
        // Return only bounded, explicitly approved fields. Never scrape bodies.
        return eventbus.Projection{"resource": r.PathValue("id")}, true
    },
})(router)
```

Receipt emission is post-response and best effort. A receipt that cannot be observed is **not** evidence that the underlying operation failed.

## Agent-attributed receipts

`eventbus` derives a run correlation only from verified provenance placed in request context by `api-core/provenance`. It never trusts a caller-provided run header. Vrooli Events accepts `vrooli.events.receipt.v1` only with a verified Agent Manager identity, a run correlation, operation/outcome metadata, and a bounded safe projection.

## Publishing Custom Events

Your scenario can publish domain-specific events beyond automatic discovery events:

```go
import eventspb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"

// Use the events client (available from EmittingResolver or standalone)
client := discovery.NewEventsClient("http://localhost:15000", "my-scenario")

client.Emit(ctx, &eventspb.EventEnvelope{
    EventId:        "my-scenario.orders.completed.v1",
    EventType:      "orders.completed.v1",
    SourceScenario: "my-scenario",
    CorrelationId:  traceID,
    Payload:        orderPayload,
})
```

## What If vrooli-events Is Not Running?

Everything still works:

- **Cached policy**: a fresh cached denial is still enforced; otherwise ordinary traffic proceeds.
- **Receipt publication**: it is skipped or dropped after the response; it never changes the request result.
- **Refresh**: retries with bounded exponential backoff and jitter.

This makes adoption incremental and non-breaking.
