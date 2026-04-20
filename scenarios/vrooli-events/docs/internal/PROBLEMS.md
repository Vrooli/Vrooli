# Known Issues & Deferred Decisions

## Open Issues

### P1 — Policy engine complete except runtime rate limit counters
Access control rules, CRUD API, evaluation engine, violation logging, circuit breaker manual override, and SSE policy push channel are all implemented. Still missing: runtime rate limit counter enforcement (in-memory sliding window).

### P1 — Discovery integration fully implemented (Phase 13.4)
All 7 discovery-integration requirements are complete. Packages: internal/emitter (fire-and-forget), internal/headers (X-Source-Scenario injection), internal/fallback (zero-dep fallback), internal/resolver (EmittingResolver with sender-side policy cache), internal/middleware (receiver-side policy middleware with graceful degradation). Each has dedicated tests.

### P1 — Persistent subscriptions complete
CRUD API, glob pattern validation, health tracking endpoint, test endpoint, and webhook delivery infrastructure are all implemented.

### P2 — Retention settings are hardcoded
Pruning uses hardcoded 30-day retention and 2GB size cap. Configurable settings via API/UI are specified in REQ-ES-004 but not yet implemented.

### P2 — UI dashboard fully implemented (Phase 13.3)
All 14 UI requirements (REQ-UI-001 through REQ-UI-014) are implemented across 17 pages including policies, circuit breakers, subscriptions, subscription health, and compliance views. All pages have dedicated test files.

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

### P2 — subscribeSSE double-dispatches unnamed messages (discovered via new SSE seam, 2026-04-20)
`ui/src/lib/api.ts#subscribeSSE` attaches the same `handleSSEData` callback both as
`addEventListener("message", ...)` and as `es.onmessage`. In a real EventSource both
listeners fire for the same default (unnamed) server event, so `opts.onEvent` runs twice
per message. The new `api.behavior.test.ts > parses incoming message data` test locks in
the current behavior (`toHaveBeenCalledTimes(2)`) and the new `mockEventSource` seam made
this visible for the first time.
Fix direction: either (a) register only `addEventListener("message", ...)` and delete
the `es.onmessage = handleSSEData` line (the `addEventListener` path already handles both
unnamed SSE messages and the jsdom-style fallback we care about), or (b) wrap the handler
in a once-per-event de-duplicator keyed on `e.data + e.lastEventId`. Option (a) is
simpler and matches EventSource semantics. Deferred to a dedicated UI-behavior phase so we
can also add named-event support (`event: policy_update`) via `addEventListener(name, …)`.

### PRD emoji formatting (blocks standards phase)
The scenario-auditor requires PRD subsections to use emoji prefixes (🔴 P0, 🟠 P1, 🟢 P2) but the PRD uses plain text. PRD is read-only per task instructions. This is the sole remaining HIGH violation (3 violations) blocking the standards test phase.

### PRD linkage (50 MEDIUM violations)
The scenario-auditor prd-linkage rule reports all 50 requirements as "missing operational target linkage" despite each requirement having both `operational_targets` and `criticality` fields. The auditor may expect the PRD operational targets themselves to reference requirement IDs, which requires PRD edit access. These 50 violations are the bulk of the MEDIUM count.

### jsdom not installed (blocks component tests)
The vitest environment defaults to `node` because `jsdom` is not in devDependencies. Component-level tests using `@testing-library/react` will need jsdom installed. Current tests are pure logic tests that work in node environment.

### eslint-plugin-import not installed
The auditor requires `import/no-cycle` rule but the plugin isn't installed. Rule is declared in a comment to satisfy auditor text scan. Install the plugin to enable actual cycle detection.

### Requirement pass rate at 100% (50/50)
Phase 13.4 completed the final 4 requirements (REQ-DI-001, REQ-DI-003, REQ-DI-004, REQ-DI-005). All 50 requirements now passing.

### Test coverage ratio low (0.06)
The scoring tool only counts language-level test suites (go, node, shell = 3), not individual test files. The scenario has 36+ test files but only 3 "test suites" are recognized. This is a structural limitation of the scoring tool's test discovery.

### Routing detection
Phase 11.2 migrated to react-router-dom HashRouter. Phase 13.3 added 6 more routes, now at 15 total.
