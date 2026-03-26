# Brand Manager — Unit Testing Architecture

## Last Updated
2026-03-26

## Test Organization Status
- [x] Go tests co-located with source files
- [ ] TypeScript tests co-located with source files (UI is template — no tests yet)
- [x] Consistent naming conventions (`*_test.go`)
- [x] Test utilities package exists (`internal/testutil/`)

## Mock Organization Status
- [x] Centralized mock packages (`repository/mocks/`)
- [x] Mock builder pattern (`.Seed()` for pre-loading data)
- [x] Error injection fields (`CreateErr`, `GetErr`, etc.)
- [x] No inline mock definitions in test files

## Testability Status
- [x] Dependency injection used throughout (handlers accept interfaces)
- [x] Interfaces defined for external dependencies (`BrandRepository`, `VersionRepository`, `AssignmentRepository`)
- [x] UUID generation abstracted via `IDFunc` seam
- [ ] Time abstracted — `time.Now()` still called directly in repository layer

## Infrastructure Status
- [ ] Testcontainers configured (not needed — SQLite via temp files is sufficient)
- [x] Test setup centralized (`testutil.SetupTestDB`)
- [ ] CI runs tests successfully (no CI configured yet)

## Test Layers

### Layer 1: Repository tests (integration with real SQLite)
- **Location**: `api/repository/*_test.go`
- **Purpose**: Verify SQL queries, JSON marshalling, and schema correctness
- **Setup**: `testutil.SetupTestDB(t)` creates an isolated temp database per test
- **Coverage**: brands (CRUD + filter + not-found), versions (create, list, get, not-found), assignments (create, get, replace, list, delete, not-found)

### Layer 2: Handler tests — integration (real SQLite)
- **Location**: `api/handlers/brands_test.go`
- **Purpose**: Full HTTP request→handler→repository→database flow
- **Setup**: `setupTestServer(t)` wires real repos on temp DB
- **Coverage**: All CRUD endpoints, validation, version creation, scenario status

### Layer 3: Handler tests — unit (mock repos)
- **Location**: `api/handlers/brands_mock_test.go`
- **Purpose**: Test handler logic in isolation — HTTP parsing, status codes, error handling
- **Setup**: `setupMockServer(t)` wires in-memory mocks with deterministic IDs
- **Coverage**: Create (success + repo error), get (not found), update, delete, assignment (success + brand not found), scenario status

## Mock Implementation Details

### `repository/mocks/` package
Three mock structs implementing the three repository interfaces:

| Mock | Implements | Key Features |
|------|-----------|--------------|
| `BrandRepository` | `repository.BrandRepository` | `.Seed()` builder, error override fields, filter support |
| `VersionRepository` | `repository.VersionRepository` | Append-only storage keyed by brand_id |
| `AssignmentRepository` | `repository.AssignmentRepository` | INSERT OR REPLACE semantics, dual-index (by ID and scenario) |

All mocks are thread-safe (sync.RWMutex) and return defensive copies.

## Seams for Testing

| Seam | Location | Substitution |
|------|----------|-------------|
| Repository interfaces | `repository/interfaces.go` | `mocks/*` in handler unit tests |
| ID generation | `handlers.IDFunc` | `WithIDFunc()` for deterministic IDs |
| Database path | `BM_SQLITE_PATH` env var | `testutil.SetupTestDB` sets temp path |

## Remaining Gaps

1. **Time injection**: Repositories call `time.Now()` directly — prevents testing time-dependent behaviour
2. **UI tests**: Template UI has no test infrastructure yet
3. **CLI tests**: No automated tests for CLI commands (requires running API server)
4. **Error path coverage**: Not all repo error injection paths are tested via handlers

## Priority Improvements

1. [Medium] Abstract time via injectable `TimeFunc` in repositories (mirrors `IDFunc` pattern)
2. [Low] Add Vitest config and component test setup when UI is built out
3. [Low] Add CLI integration test harness using httptest server
