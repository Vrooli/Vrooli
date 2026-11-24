# Lifestyle Dashboard architecture

The Lifestyle Dashboard is the owner of a shared event store and cross-domain views. Integrating scenarios register as domains and submit events through its HTTP API; the dashboard owns persistence, aggregation, briefs, score configuration, and weekly digests. Product scope and integration intent remain in the [PRD](../../PRD.md).

## Runtime and consumers

- `api/main.go` resolves the scenario's `lifestyle.db` path through `api-core/storage`, builds the SQLite DSN, connects with one open connection and bounded retry, applies the embedded schema with `database.EnsureSchemas`, and starts the API Core server. The filename is a compatibility choice so existing scenario data continues to be found.
- `api/main.go` constructs the SQLite repositories and injects them into `handlers.Handler`; `setupRoutes` owns the HTTP route table and maps `/api/v1` operations to handler methods. `/health` is also exposed for infrastructure checks.
- `api/domain/types.go` owns the cross-domain event envelope and API domain types. Event payloads remain JSON so each source domain can evolve its event-specific fields without changing the shared envelope.
- `api/repository/interfaces.go` defines persistence contracts; `api/repository/sqlite_*.go` implements them. Handlers own HTTP validation and response mapping; repositories own SQL and aggregation. `api/internal/lifestyle/schema.go` embeds `schema.sql` as the startup schema provider.
- `ui/src/App.tsx` owns hash routes. Page components call `ui/src/lib/api.ts`, which resolves the scenario API base with `@vrooli/api-base`; the UI does not connect to SQLite. The same API client serves dashboard, domain, event, brief, score, storage, and digest views.

## Domain and API boundaries

## Key Domain Concepts

### Events

An event keeps its source `domain`, `event_type`, timestamp, JSON payload, intervention flag, and optional hypothesis link. The dashboard stores and queries this shared envelope; event-specific interpretation stays with the producing domain.

### Domains

A registered domain describes a source scenario, its capabilities, and an optional health endpoint. Registration and health state belong to the dashboard; the source scenario owns the meaning and production of its events.

### Handler Layer

The handler layer is the HTTP boundary and owns request validation and response mapping.

### Presentation Layer

Route registration in `api/main.go` dispatches to handlers; handlers depend on repository interfaces, not a database handle.

### Abstraction Layer

The interfaces include separate event, domain, stats, storage, brief, score-configuration, and digest operations. Their current production implementation is SQLite; interface separation supports tests and keeps SQL out of handlers, but does not mean another backend currently exists.

### Aggregates and generated views

Timeline and summary queries aggregate events across domains. Score, briefs, and weekly digests are distinct derived views with their own repositories and API contracts; they are not additional event stores. Their implementation and product requirements are documented in the owning handlers, repositories, and [PRD](../../PRD.md).

## Persistence and portability

SQLite is an embedded, file-backed store. The call names the `lifestyle-dashboard` scenario and retained `lifestyle.db` filename. In a live lifecycle, the assigned `SCENARIO_DATA_DIR` determines the data directory; explicit storage-root overrides can redirect tests and standalone runs. The API Core SQLite storage seam builds the DSN and owns SQLite connection pragmas. The API limits the pool to one connection to respect SQLite's single-writer behavior. Do not infer WAL or other runtime tuning from old comments; inspect the active `api-core/storage` implementation and scenario configuration when changing this contract.

The [storage audit](../internal/STORAGE_AUDIT.md) preserves the historical ADR rationale and audit trail. Its implementation-status tables predate the current schema provider and path resolution, so treat `main.go`, `api-core/storage`, and the embedded schema as current behavior. Keep this architecture page as the ownership map; detailed storage findings and test seams remain in their focused references.

## Change and validation route

Follow a change through the API route, handler, repository interface and SQLite implementation, plus every affected UI or integrating-scenario caller. Preserve the shared event envelope, domain ownership, existing database filename and distinct aggregate-view contracts. Validate API behavior with the API owner tests, UI behavior with the UI suite, and documentation links/contracts with the Lifestyle Dashboard `docs` phase.

See [Quick Start](../QUICKSTART.md), the [storage audit](../internal/STORAGE_AUDIT.md), [test architecture](../internal/UNIT_TEST_ARCHITECTURE.md), [integration seams](../internal/SEAMS.md), and [temporal flows](../internal/TEMPORAL-FLOWS.md) for operational and specialized detail.
