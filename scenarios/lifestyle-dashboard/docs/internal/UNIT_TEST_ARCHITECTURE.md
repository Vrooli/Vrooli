# Lifestyle Dashboard - Unit Testing Architecture

This document describes the unit testing infrastructure for the Lifestyle Dashboard API. It serves as a guide for maintaining consistent testing practices.

## Last Updated
2026-03-10

## Test Organization Status

### File Organization
- [x] Go tests co-located with source files (`*_test.go` pattern)
- [x] Tests in same package for white-box testing access
- [x] Consistent naming: `TestFunctionName_Scenario`
- [x] `[REQ:ID]` annotations for requirement traceability

### Mock Organization
- [x] Centralized mock packages (`api/internal/testutil/`)
- [x] Mock builder patterns (chainable configuration)
- [x] Interface-based mocks for all repositories

### Testability
- [x] Dependency injection via constructors
- [x] Repository interfaces defined for all storage
- [x] Handler accepts interfaces, not concrete types
- [ ] Time provider abstraction (weak seam, documented in SEAMS.md)
- [ ] UUID generator abstraction (weak seam, documented in SEAMS.md)

### Infrastructure
- [x] Test database helpers (`SetupTestDB`, `SetupInMemoryDB`)
- [x] Standard test patterns documented
- [x] All tests pass in CI

## Directory Structure

```
api/
├── main.go
├── main_test.go                 # Integration tests (Server level)
├── internal/
│   └── testutil/                # Centralized test utilities
│       ├── db.go                # Database setup helpers
│       ├── mocks.go             # Mock repository implementations
│       └── mocks_test.go        # Tests for mocks
├── domain/
│   ├── types.go
│   ├── types_test.go            # Type serialization tests
│   └── schema.go
├── handlers/
│   ├── handlers.go
│   ├── handlers_test.go         # Handler unit tests
│   ├── events.go
│   ├── domains.go
│   └── stats.go
└── repository/
    ├── interfaces.go
    ├── repository_test.go       # Repository tests (with real SQLite)
    ├── sqlite_events.go
    ├── sqlite_domains.go
    └── sqlite_stats.go
```

## Test Patterns

### 1. Table-Driven Tests

Use table-driven tests for systematic coverage:

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name     string
        input    InputType
        wantErr  bool
        errMsg   string
        category string  // "happy_path", "boundary", "error"
    }{
        {"valid_input", validInput, false, "", "happy_path"},
        {"empty_required", emptyInput, true, "required", "error"},
        // ...
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            err := Validate(tc.input)
            if tc.wantErr {
                if err == nil {
                    t.Error("Expected error")
                }
                return
            }
            if err != nil {
                t.Errorf("Unexpected error: %v", err)
            }
        })
    }
}
```

### 2. Arrange-Act-Assert Pattern

Structure ALL tests with clear sections:

```go
func TestCreateEvent_Success(t *testing.T) {
    // ARRANGE: Set up dependencies and test data
    repo := testutil.NewMockEventRepository()
    handler := handlers.New(repo, mockDomains, mockStats)
    body := `{"domain": "test", "event_type": "test.event"}`

    // ACT: Execute the code under test
    req := httptest.NewRequest("POST", "/api/v1/events", strings.NewReader(body))
    rr := httptest.NewRecorder()
    handler.CreateEvent(rr, req)

    // ASSERT: Verify the results
    if rr.Code != http.StatusCreated {
        t.Errorf("Expected 201, got %d", rr.Code)
    }
}
```

### 3. Test Helpers

Use `t.Helper()` in all test helper functions:

```go
func setupTestHandler(t *testing.T) (*handlers.Handler, func()) {
    t.Helper()  // Mark as helper for better error reporting

    db, cleanup := testutil.SetupTestDB(t)
    eventRepo := repository.NewSQLiteEventRepository(db)
    domainRepo := repository.NewSQLiteDomainRepository(db)
    statsRepo := repository.NewSQLiteStatsRepository(db)
    h := handlers.New(eventRepo, domainRepo, statsRepo)

    return h, cleanup
}
```

### 4. Requirement Annotations

Link tests to requirements for traceability:

```go
// TestEventStorage_SQLitePersistence validates events are stored in SQLite
// [REQ:LD-EVENT-STORAGE] Events stored in lifestyle.db SQLite file
func TestEventStorage_SQLitePersistence(t *testing.T) {
    // ...
}
```

## Test Categories

### Unit Tests (Package-Level)

| Package | Test File | Focus |
|---------|-----------|-------|
| domain | `types_test.go` | JSON serialization |
| handlers | `handlers_test.go` | HTTP handler logic |
| repository | `repository_test.go` | SQLite operations |
| testutil | `mocks_test.go` | Mock correctness |

### Integration Tests (Server-Level)

| File | Focus |
|------|-------|
| `main_test.go` | Full HTTP request/response flow |

## Mock Usage Guidelines

### When to Use Mocks

✅ Use mocks for:
- Testing handler logic in isolation
- Testing error handling paths
- Speeding up test execution
- Avoiding database state pollution

❌ Don't use mocks for:
- Testing SQL query correctness
- Testing database constraints
- E2E/integration tests

### Mock Builder Pattern

Mocks support chainable configuration:

```go
// Pre-populate with data
repo := testutil.NewMockEventRepository().
    WithEvent(&domain.Event{ID: "e1", Domain: "test"}).
    WithEvent(&domain.Event{ID: "e2", Domain: "test"})

// Configure error injection
repo := testutil.NewMockEventRepository().
    WithCreateError(errors.New("database full"))
```

### Error Injection

Test error paths by configuring mock errors:

```go
func TestCreateEvent_RepositoryError(t *testing.T) {
    repo := testutil.NewMockEventRepository().
        WithCreateError(errors.New("connection lost"))

    handler := handlers.New(repo, mockDomains, mockStats)

    req := httptest.NewRequest("POST", "/api/v1/events", validBody)
    rr := httptest.NewRecorder()
    handler.CreateEvent(rr, req)

    if rr.Code != http.StatusInternalServerError {
        t.Errorf("Expected 500 on repo error, got %d", rr.Code)
    }
}
```

## Test Database Options

### In-Memory SQLite (Fast)

For quick unit tests:

```go
db := testutil.SetupInMemoryDB(t)
// Automatically cleaned up via t.Cleanup()
```

### File-Based SQLite (WAL Mode)

For tests requiring WAL mode or file persistence:

```go
db, cleanup := testutil.SetupTestDB(t)
defer cleanup()
```

## Running Tests

```bash
# All tests
cd api && go test ./... -v

# Specific package
go test ./handlers/... -v

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific test
go test ./handlers/... -run TestCreateEvent -v
```

## Test Counts by Package

| Package | Test Count | Requirement Coverage |
|---------|------------|---------------------|
| main (integration) | 22 | P0/P1 operational targets |
| handlers | 10 | Handler-level behavior |
| repository | 15 | Storage operations |
| domain | 4 | Type serialization |
| testutil | 13 | Mock correctness |
| **Total** | **64** | |

## Issues and Improvements

### Current Issues
- None blocking

### Completed Improvements
- [x] Centralized test utilities in `internal/testutil/`
- [x] Mock repository implementations
- [x] Builder pattern for mock configuration
- [x] Documented seams in SEAMS.md

### Future Improvements
- [ ] Add `TimeProvider` interface if time-dependent testing needed
- [ ] Add handler tests using mock repositories (currently use real SQLite)
- [ ] Add benchmark tests for hot paths

## Related Documentation

- [SEAMS.md](SEAMS.md) - Integration boundaries and testability
- [ARCHITECTURE.md](../concepts/ARCHITECTURE.md) - Overall system design
- [STORAGE_AUDIT.md](STORAGE_AUDIT.md) - Database architecture decisions
