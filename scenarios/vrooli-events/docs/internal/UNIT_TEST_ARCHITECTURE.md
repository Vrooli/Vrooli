# vrooli-events Unit Testing Architecture

## Last Updated
2026-04-20

## Test Organization Status
- [x] Go tests co-located with source files (`*_test.go` next to `.go`)
- [x] Consistent naming conventions (`*_test.go` for Go, `*.test.ts` for TS)
- [x] Test utilities package exists (`internal/testutil/`)
- [ ] TypeScript test files exist (infrastructure ready, no tests yet)

## Go Test Layout

```
api/
  handlers.go
  handlers_test.go          # Integration + unit tests (via mockable seams)
internal/
  broker/
    broker.go
    broker_test.go           # Local mockStore (avoids import cycle)
    iface.go                 # EventBroker interface (testability seam)
    matcher.go
    matcher_test.go
  convert/
    convert.go
    convert_test.go
  match/
    match.go
    match_test.go
  store/
    store.go                 # Store interface
    sqlite.go
    sqlite_test.go           # In-memory SQLite via newTestStore()
    pruner.go
    pruner_test.go
  testutil/                  # Shared test helpers (used by api/ tests)
    mock_store.go            # Configurable MockStore with builder pattern
    mock_broker.go           # MockBroker implementing EventBroker
    factories.go             # MakeEvent(), NewTestStore()
```

### Import Cycle Avoidance

Go forbids import cycles even in test files. The `testutil` package imports `store` and `broker`, so:
- `store/` and `broker/` tests use **local** helpers (mockStore, newTestStore, makeEvent)
- `api/` tests import `testutil` freely (no cycle: api→testutil→store/broker, api doesn't export to store/broker)

## TypeScript Test Layout

```
ui/src/
  test-utils/
    setup.ts                 # Vitest setup (cleanup, ResizeObserver/default EventSource stub)
    index.ts                 # Re-exports
    renderWithProviders.tsx   # React render with QueryClientProvider
    factories.ts             # createMockHealthResponse(), createMockEvent()
    mockFetch.ts             # Programmable globalThis.fetch seam (+ .test.ts)
    mockEventSource.ts        # Controllable globalThis.EventSource seam (+ .test.ts)
```

### Vitest Configuration
- Setup file: `src/test-utils/setup.ts` (auto-cleanup, global mocks)
- Environment: jsdom
- Globals: enabled (no explicit imports for describe/it/expect)
- Coverage: v8 provider with json-summary reporter

### Missing: jsdom dependency
The `jsdom` package needs to be installed as a dev dependency (`pnpm add -D jsdom`).

## Mock Organization Status
- [x] Centralized MockStore with builder pattern (`testutil/mock_store.go`)
- [x] Centralized MockBroker for handler tests (`testutil/mock_broker.go`)
- [x] Local mockStore in broker_test.go (minimal, avoids cycle)
- [x] UI factory functions for HealthResponse and EventEnvelope
- [x] UI `mockFetch()` helper (globalThis.fetch seam — Phase 3 iter1, 2026-04-20)
- [x] UI `mockEventSource()` helper (globalThis.EventSource seam — Phase 3 iter1)

## Testability Status
- [x] Store interface used throughout (dependency injection)
- [x] EventBroker interface for handler testability (Phase 3)
- [x] Pruner accepts Store and Logger via config (injectable)
- [x] Time abstracted in SQLiteStore (Phase 20 iter2: `SQLiteConfig.Now` injectable; Prune uses `s.config.Now()`)
- [x] UI HTTP seam tooling (`mockFetch()` swaps `globalThis.fetch` — no production change needed)
- [x] UI SSE seam tooling (`mockEventSource()` swaps `globalThis.EventSource` with controllable instance)

## Infrastructure Status
- [x] In-memory SQLite for store tests (no external deps)
- [x] Vitest configured with jsdom environment
- [ ] jsdom installed as dependency
- [ ] CI test pipeline configured

## Mock Patterns

### Go: Builder Pattern (MockStore)

```go
ms := (&testutil.MockStore{}).
    WithStatsResult(store.Stats{TotalEvents: 42}, nil)
mb := testutil.NewMockBroker().WithSubscriberCount(5)
```

### Go: Local No-Op Mock (broker_test.go)

```go
type mockStore struct{}
func (m *mockStore) Insert(...) (int64, error) { return 0, nil }
// ... minimal implementation for isolation
```

### TypeScript: Factory Functions

```ts
const health = createMockHealthResponse({ subscribers: 3 });
const event = createMockEvent({ eventType: "app.deploy.v1" });
```

### TypeScript: HTTP Seam (`mockFetch`)

```ts
import { mockFetch } from "../test-utils";

const http = mockFetch();
http.respondTo({ urlPattern: "/health" }, { body: { status: "healthy" } });
http.respondTo({ urlPattern: /\/policies\/\d+$/, method: "PUT" }, { status: 204 });
http.rejectWith({ urlPattern: "/events/subscribe" }, new Error("simulated down"));

await fetchHealth();
expect(http.calls[0].url).toContain("/health");

http.restore();
```

Matchers run in insertion order; the first hit wins. Unmatched calls return a 500
with a diagnostic body so tests fail loudly rather than quietly hitting the network.

### TypeScript: SSE Seam (`mockEventSource`)

```ts
import { mockEventSource } from "../test-utils";

const sse = mockEventSource();
subscribeSSE({ onEvent, onError });

sse.instances[0].emitMessage({ eventId: "evt-1", eventType: "x" });
sse.instances[0].emitNamed("policy_update", { version: 7 });
sse.instances[0].emitError();

sse.restore();
```

Each `new EventSource(url)` is recorded; tests drive listeners (`onmessage`,
`addEventListener("message", ...)`, `addEventListener("<name>", ...)`, `onerror`) via the
`.emit*` methods. `close()` is tracked so tests can assert cleanup.

## Priority Improvements

1. ~~**Install jsdom**~~ — Done; 200 UI tests run against jsdom.
2. ~~**Time provider seam**~~ — Done (Phase 20 iter2): `SQLiteConfig.Now func() time.Time`
   defaults to `time.Now` and is used by `pruneByTime()`. Three former `time.Sleep(2*time.Second)`
   calls in `TestPruneByTime`, `TestPruneIdempotent`, and `TestMetaConsistencyThroughPruneCycle`
   were removed; the store suite now runs in ~0.5s (was ~6.5s).
3. ~~**API base URL injection / UI HTTP seam**~~ — Done (Phase 3 iter1, 2026-04-20):
   `mockFetch()` test helper swaps `globalThis.fetch` so behavioral tests can program
   responses, assert on observed URLs, and simulate network failures without changing
   `api.ts`. Backed by `src/test-utils/mockFetch.test.ts` + `src/lib/api.behavior.test.ts`.
4. ~~**UI SSE seam**~~ — Done (Phase 3 iter1): `mockEventSource()` replaces the
   no-op default stub in `setup.ts` with a controllable constructor that exposes
   `.emitMessage` / `.emitNamed` / `.emitError` per instance. Revealed a pre-existing
   double-dispatch bug in `subscribeSSE` (logged in `PROBLEMS.md`).
5. **Fix subscribeSSE double-dispatch** — `onmessage` + `addEventListener("message", ...)`
   both fire for unnamed events, so `onEvent` runs twice per message. Delete the
   `es.onmessage = handleSSEData` assignment once `addEventListener("message", ...)` is
   confirmed sufficient under jsdom + real browsers. Also wire named-event handlers
   (`event: policy_update`) via `addEventListener(name, ...)`.
6. **Add first true component behavior tests** — many existing "component tests" are
   actually module-shape tests in `node` environment. Migrate at least one (e.g.
   `StreamPage.test.ts` or a new `StreamPage.behavior.test.tsx`) to `renderWithProviders`
   + `mockEventSource` to validate render-under-event-stream behavior.

### Time Provider Seam (Phase 20 iter2)

```go
fakeNow := func() time.Time { return time.Now().Add(10 * time.Second) }
s, _ := store.NewSQLiteStore(ctx, store.SQLiteConfig{
    MaxAge: 1 * time.Second,
    Now:    fakeNow, // cutoff = fakeNow - MaxAge, pruning decisions become deterministic
})
```

Only `pruneByTime` currently consults `s.config.Now()`. SQLite's `strftime('%Y-%m-%dT%H:%M:%f','now')`
still governs the `created_at` column, so the seam shifts the *cutoff*, not event timestamps — tests
don't need to freeze row-level time to get deterministic prune behavior.
