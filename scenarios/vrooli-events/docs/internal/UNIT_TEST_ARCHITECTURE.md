# vrooli-events Unit Testing Architecture

## Last Updated
2026-04-05

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
    setup.ts                 # Vitest setup (cleanup, ResizeObserver/EventSource mocks)
    index.ts                 # Re-exports
    renderWithProviders.tsx   # React render with QueryClientProvider
    factories.ts             # createMockHealthResponse(), createMockEvent()
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
- [ ] UI component mocks (not yet needed — no component tests)

## Testability Status
- [x] Store interface used throughout (dependency injection)
- [x] EventBroker interface for handler testability (Phase 3)
- [x] Pruner accepts Store and Logger via config (injectable)
- [ ] Time abstracted (Prune uses time.Now() directly)
- [ ] API base URL injectable in UI (module-level singleton)

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

## Priority Improvements

1. **Install jsdom** — unlocks UI unit tests with existing infrastructure
2. **Time provider seam** — inject time source into SQLiteStore.Prune() for deterministic tests (currently uses time.Sleep in TestPruneByTime)
3. **API base URL injection** — make fetchHealth/fetchEvents accept optional base URL for test overrides
4. **Add first UI component tests** — use renderWithProviders + factories to test StatCard, EventTable
