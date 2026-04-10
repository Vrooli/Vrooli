# Assumptions

## Last Updated
2026-04-05

## Validated Assumptions
- **Single-node deployment**: vrooli-events runs on one server alongside all scenarios. SQLite is sufficient. *(Validated: PRD non-goals explicitly state no clustering.)*
- **Trust model**: Scenarios trust each other within the Vrooli ecosystem. Policy enforcement is governance, not adversarial security. *(Validated: PRD non-goals.)*
- **Proto payload format**: Events use Protocol Buffer JSON encoding via `protojson`. *(Validated: working end-to-end in tests.)*

## Unvalidated Assumptions
- **Subscriber count stays small**: Backpressure handling (64-buffer, drop+notify) assumes tens of subscribers, not thousands. No load testing has been done.
- **Pruning completes within one interval**: The hourly pruner assumes each prune operation finishes in seconds. With millions of events, this may not hold.
- **EventSource browser API is sufficient**: The UI uses native `EventSource` which lacks custom headers (no auth). If auth is added to SSE endpoints, a polyfill or fetch-based SSE client will be needed.
- **Hash routing is sufficient**: The UI uses hash-based routing. If SEO or SSR becomes relevant, react-router with history API would be needed.
- **SQLite concurrent read performance**: WAL mode allows concurrent reads, but many simultaneous query+subscribe handlers could still contend on the Go-level mutex in `SQLiteStore`.

## Dependency Assumptions
- **`@vrooli/api-base`**: Provides `resolveApiBase` and `buildApiUrl`. Assumed to be stable and available in all scenario UIs.
- **`@vrooli/iframe-bridge`**: Provides storage shim for iframe-embedded UI. Assumed to handle `localStorage`/`sessionStorage` proxying.
- **`packages/proto`**: Generated Go types for `EventEnvelope` are assumed to be kept in sync with proto schema changes.
