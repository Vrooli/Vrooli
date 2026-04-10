# Stream of Consciousness Analyzer - Unit Testing Architecture

## Last Updated
2026-03-20

## Test Organization

### API (Go)

Tests are co-located with source files in `api/` and split by domain concern:

| File | Purpose | Test Count |
|------|---------|------------|
| `handlers_test.go` | Scheme + Information handler tests, route registration | ~20 |
| `thought_handlers_test.go` | Thought + Edge handler tests | ~17 |
| `export_handlers_test.go` | Export handler tests | 3 |
| `suggestion_handlers_test.go` | Suggestion/Provider handler tests | 4 |
| `suggestion_service_test.go` | SuggestionService unit tests (env seam, fallback) | 10 |
| `model_test.go` | Model serialization roundtrip tests | ~19 |
| `mocks_test.go` | Centralized mock implementations + interface checks | 0 (infra) |
| `testhelpers_test.go` | Shared test utilities (fakeUUID, assertStatus, decodeJSON) | 0 (infra) |

### UI (TypeScript)

| File | Purpose |
|------|---------|
| `ui/src/App.test.tsx` | App component rendering with scheme list |

### CLI (Go)

| File | Purpose |
|------|---------|
| `cli/app_test.go` | CLI command registration |

## Mock Organization

All mocks are centralized in `api/mocks_test.go`:

| Mock | Interface | Builder Methods |
|------|-----------|-----------------|
| `mockSchemes` | `SchemeStore` | `seed()`, `WithListError()` |
| `mockInfo` | `InformationStore` | `seed()`, `WithListError()` |
| `mockThoughts` | `ThoughtStore` | `seedThought()`, `seedEdge()`, `WithListError()` |
| `mockExport` | `ExportStore` | `seed()` |
| `mockSuggestions` | `SuggestionProvider` | `WithGenerateError()`, `WithSuggestions()` |

**Pattern**: Builder methods return `*self` for chaining. `seed()` methods populate internal maps and return the created entity for use in assertions.

## Testability Seams

- **Service interfaces** (`interfaces.go`): Handlers accept interfaces, enabling mock injection
- **EnvReader** (`suggestion_service.go`): `NewSuggestionServiceWithEnv(db, env)` injects environment access
- **Compile-time checks** (`mocks_test.go`): `var _ Interface = (*Implementation)(nil)` assertions

## Test Helpers

Centralized in `api/testhelpers_test.go`:

- `fakeUUID(n)` — Deterministic UUID generation for test reproducibility
- `decodeJSON` — Generic JSON response decoder with `t.Helper()`
- `assertStatus(t, rec, expected)` — HTTP status assertion with body in error message

## Status Checklist

- [x] Go tests co-located with source files
- [x] TypeScript tests co-located with source files
- [x] Consistent naming conventions (`*_test.go`, `*.test.tsx`)
- [x] Test utilities centralized (`testhelpers_test.go`)
- [x] Centralized mock package (`mocks_test.go`)
- [x] Mock builder patterns (WithXxx chainable methods)
- [x] No inline mock definitions in test files
- [x] Dependency injection via interfaces
- [x] EnvReader abstraction for environment-dependent code
- [ ] Testcontainers for database integration tests
- [ ] MSW or fetch mock setup for UI tests
- [ ] Test configuration files (vitest.config.ts with coverage)

## Priority Improvements

1. **Testcontainers setup** — Add `setupTestDB(t)` for service-level integration tests
2. **UI test infrastructure** — Add MSW handlers and `renderWithProviders()` helper
3. **Coverage reporting** — Configure `go test -coverprofile` and vitest coverage
