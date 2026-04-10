# Unit Testing Architecture

This document describes the unit testing infrastructure for the development-toolchain-validator scenario.

## Last Updated
2026-03-11

## Test Organization Status
- [x] Go tests co-located with source files
- [x] Consistent naming conventions (`*_test.go`)
- [x] Test utilities package exists (`api/internal/testutil/`)
- [x] Mock package exists (`api/internal/mocks/`)

## Directory Structure

```
api/
├── domain/
│   └── reference/
│       ├── model.go
│       ├── repository.go
│       ├── service.go
│       └── service_test.go      # ← Co-located service tests
├── handlers/
│   ├── reference.go
│   └── reference_test.go        # ← Co-located handler tests
├── infrastructure/
│   └── postgres/
│       └── reference_repo.go    # ← Integration tests (future)
└── internal/
    ├── mocks/
    │   └── repository.go        # ← Centralized mock implementations
    └── testutil/
        ├── helpers.go           # ← Test assertions and utilities
        └── fixtures.go          # ← Test data factories
```

## Mock Organization

### MockRepository (`internal/mocks/repository.go`)

A builder-pattern mock for `reference.Repository`:

```go
// Create with builder pattern
repo := mocks.NewMockRepository().
    WithReference(testutil.NewReferenceFactory().WithID("123").Build()).
    WithCreateError(errors.New("database error"))

// Use in tests
service := reference.NewService(repo)
```

Features:
- Builder methods for state setup (`WithReference`, `WithCreateError`, etc.)
- Call tracking for verification (`CreateCallCount`, `DeleteCallCount`)
- Thread-safe for parallel tests

## Test Fixtures (`internal/testutil/fixtures.go`)

Factory pattern for creating test data:

```go
// Create with defaults
ref := testutil.NewReferenceFactory().Build()

// Customize as needed
ref := testutil.NewReferenceFactory().
    WithID("custom-id").
    WithSlug("custom-slug").
    WithTemplate("go-api").
    Build()
```

Available factories:
- `ReferenceFactory` - Creates `*reference.Reference`
- `CreateInputFactory` - Creates `reference.CreateInput`

## Test Helpers (`internal/testutil/helpers.go`)

Common assertions and utilities:

```go
// Status assertions
testutil.AssertStatus(t, rec, http.StatusOK)

// Content type assertions
testutil.AssertContentType(t, rec, "application/json")

// JSON parsing
testutil.AssertJSON(t, rec, &response)

// Request creation
req := testutil.MakeJSONRequest(t, http.MethodPost, "/api/v1/references", input)

// Pointer helpers for UpdateInput
testutil.StringPtr("value")
```

## Test Patterns

### Table-Driven Tests

All tests use table-driven format with category markers:

```go
tests := []struct {
    name      string
    input     reference.CreateInput
    setupMock func(*mocks.MockRepository)
    wantErr   error
    category  string // happy_path, boundary, error, edge_case
}{
    {
        name:     "valid_input_creates_reference",
        input:    reference.CreateInput{...},
        wantErr:  nil,
        category: "happy_path",
    },
    // ...
}

for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) {
        // ARRANGE
        repo := mocks.NewMockRepository()
        tc.setupMock(repo)
        service := reference.NewService(repo)

        // ACT
        result, err := service.Create(ctx, tc.input)

        // ASSERT
        if tc.wantErr != nil {
            if !errors.Is(err, tc.wantErr) {
                t.Fatalf("expected error %v, got %v", tc.wantErr, err)
            }
            return
        }
        // ...
    })
}
```

### External Test Package Pattern

Service tests use `reference_test` package to avoid import cycles:

```go
package reference_test  // External test package

import (
    "development-toolchain-validator/domain/reference"
    "development-toolchain-validator/internal/mocks"
)
```

### Handler Tests

Handler tests use a test router setup:

```go
func setupTestRouter(repo *mocks.MockRepository) *mux.Router {
    service := reference.NewService(repo)
    handler := NewReferenceHandler(service)
    router := mux.NewRouter()
    handler.RegisterRoutes(router)
    return router
}
```

## Test Coverage by Layer

| Layer | File | Tests | Status |
|-------|------|-------|--------|
| Service | `domain/reference/service.go` | `service_test.go` | ✅ 50+ test cases |
| Handler | `handlers/reference.go` | `reference_test.go` | ✅ 25+ test cases |
| Repository | `infrastructure/postgres/reference_repo.go` | - | ⏳ Needs testcontainers |

## Running Tests

```bash
# Run all tests
cd scenarios/development-toolchain-validator/api
go test ./... -v

# Run with coverage
go test ./... -cover

# Run specific package
go test ./domain/reference -v
go test ./handlers -v
```

## Future Improvements

1. **Integration Tests**: Add testcontainers for PostgreSQL repository tests
2. **Coverage Metrics**: Target 80%+ line coverage
3. **Benchmark Tests**: Add performance benchmarks for critical paths
4. **Fuzz Tests**: Add fuzz testing for input validation
