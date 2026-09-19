# Offer Desk seams

| Seam | Interface | Owner | Failure contract |
|---|---|---|---|
| Routed database | `*database.RoutedDB` | API composition root | isolated test pools; database failure is explicit |
| Evaluation clock | `schedule.Clock` | API composition root | scheduler uses the injected clock and records evaluation time |
| Money Ledger read client | generated `JournalServiceClient` and `PositionServiceClient` | board handler | short deadline; source becomes visible unavailable, never zero |
| Catalog store | `catalog.Store` | catalog domain | typed edge matrix, lifecycle adjacency, and trigger preconditions are enforced server-side |
| Scheduler | `Service.startScheduler` | gates composition root | failed scheduled evaluations are logged and remain retryable |
| Connect transport | generated `offers_v1connect` handlers | handlers | typed proto payloads and Connect status |
| CLI manifest bindings | `cli/manifest.json` | CLI composition root | every RPC has a declared command |
| Projection bus | `space --projection offers --json` | CLI/fleet adapter | schema-valid denominator document with source attribution |
