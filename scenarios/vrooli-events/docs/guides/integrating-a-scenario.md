# Integrating a Scenario

How to connect your scenario to vrooli-events for automatic event emission and policy enforcement.

Implementation (scenario-side SDK — currently `scenarios/vrooli-events/internal/*`; see [DOC: internal/SEAMS.md#architecture-alignment] for the planned promotion to `packages/api-core/eventbus-sdk/`):
EmittingResolver: [CODE: internal/resolver/resolver.go#EmittingResolver] | Event emitter: [CODE: internal/emitter/emitter.go] | Receiver middleware: [CODE: internal/middleware/policy.go] | Graceful fallback: [CODE: internal/fallback/fallback.go] | Source header: [CODE: internal/headers/headers.go]

## Automatic Integration (via Discovery Package)

Once the discovery package is updated with `EmittingResolver`, **all existing scenarios get event emission for free**. No code changes needed — every call through `discovery.ResolveScenarioURL()` automatically emits an event.

### What Changes

Before (current):
```go
import "github.com/vrooli/api-core/discovery"

url, err := discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
// Makes HTTP call to agent-manager — nobody knows about it
```

After (with EmittingResolver):
```go
import "github.com/vrooli/api-core/discovery"

// At scenario startup, wrap the resolver
resolver := discovery.NewEmittingResolver(
    discovery.DefaultResolver(),
    "http://localhost:15000",  // vrooli-events URL (auto-discovered)
    "my-scenario",            // This scenario's name
)

url, err := resolver.ResolveScenarioURLDefault(ctx, "agent-manager")
// 1. Checks local policy cache — denies fast if forbidden
// 2. Resolves port (same as before)
// 3. Emits discovery event in background (zero latency added)
// 4. Returns URL to caller
```

## Adding Receiver-Side Policy Enforcement

Optional but recommended. Protects your scenario from unauthorized callers.

```go
import "github.com/vrooli/api-core/discovery"

// In your router setup
router := http.NewServeMux()
handler := discovery.PolicyMiddleware("http://localhost:15000", "my-scenario")(router)
// Incoming requests now checked against receiver-side policy cache
// Requests without X-Source-Scenario header treated as "external" policy class
```

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

- **EmittingResolver**: Falls back to default behavior — resolves ports normally, events are silently dropped
- **PolicyMiddleware**: Uses last-known policy or configured default (allow/deny)
- **Standard Resolver**: Works unchanged — no vrooli-events dependency

This makes adoption incremental and non-breaking.
