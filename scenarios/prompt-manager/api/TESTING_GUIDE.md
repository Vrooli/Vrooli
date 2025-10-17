# Testing Guide - Prompt Manager

## Overview

This guide explains the testing infrastructure and patterns used in the prompt-manager scenario.

## Test Architecture

### Test Files

1. **test_helpers.go** - Reusable test utilities
2. **test_patterns.go** - Systematic testing patterns
3. **main_test.go** - Comprehensive unit tests
4. **performance_test.go** - Performance benchmarks

### Test Coverage Areas

- ✅ Health check endpoints
- ✅ Campaign CRUD operations
- ✅ Prompt CRUD operations
- ✅ Search functionality (full-text)
- ✅ Export/Import data portability
- ✅ Helper functions
- ✅ Error conditions
- ✅ Concurrent operations
- ✅ Database connection pooling
- ✅ Performance benchmarks

## Quick Start

### Prerequisites

```bash
# Install PostgreSQL (if not already installed)
# Create test database
createdb prompt_manager_test

# Set test database URL
export TEST_POSTGRES_URL="postgres://user:password@localhost:5432/prompt_manager_test"
```

### Running Tests

```bash
# All tests with coverage
cd scenarios/prompt-manager/api
go test -v -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out

# View coverage summary
go tool cover -func=coverage.out | grep total

# Run specific test
go test -v -run TestCampaignCRUD

# Run all performance tests
go test -v -run TestPerformance

# Run benchmarks
go test -bench=. -benchmem
```

## Test Helpers Reference

### Database Setup

```go
// Setup test database with cleanup
db, cleanup := setupTestDB(t)
defer cleanup()

// Create test schema
setupTestTables(t, db)
```

### Creating Test Fixtures

```go
// Create test campaign
campaign := createTestCampaign(t, db, "Test Campaign")

// Create test prompt
prompt := createTestPrompt(t, db, campaign.ID, "Test Prompt", "Test content")
```

### Making HTTP Requests

```go
// Simple GET request
w := makeHTTPRequest(server, router, HTTPTestRequest{
    Method: "GET",
    Path:   "/api/v1/campaigns",
})

// POST with JSON body
w := makeHTTPRequest(server, router, HTTPTestRequest{
    Method: "POST",
    Path:   "/api/v1/campaigns",
    Body: map[string]interface{}{
        "name": "New Campaign",
        "description": "Description",
    },
})

// With URL variables
w := makeHTTPRequest(server, router, HTTPTestRequest{
    Method:  "GET",
    Path:    "/api/v1/campaigns/" + campaignID,
    URLVars: map[string]string{"id": campaignID},
})
```

### Assertions

```go
// Assert successful JSON response
result := assertJSONResponse(t, w, http.StatusOK)
if result["name"] != "Expected Name" {
    t.Errorf("Unexpected name: %v", result["name"])
}

// Assert error response
assertErrorResponse(t, w, http.StatusNotFound, "Campaign not found")
```

## Test Patterns

### 1. Basic CRUD Test Pattern

```go
func TestEntityCRUD(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    db, dbCleanup := setupTestDB(t)
    defer dbCleanup()

    setupTestTables(t, db)

    server := &APIServer{db: db}
    router := setupRouter(server)

    var createdID string

    t.Run("Create_Success", func(t *testing.T) {
        w := makeHTTPRequest(server, router, HTTPTestRequest{
            Method: "POST",
            Path:   "/api/v1/entity",
            Body:   map[string]interface{}{"name": "Test"},
        })
        result := assertJSONResponse(t, w, http.StatusCreated)
        createdID = result["id"].(string)
    })

    t.Run("Get_Success", func(t *testing.T) {
        w := makeHTTPRequest(server, router, HTTPTestRequest{
            Method:  "GET",
            Path:    "/api/v1/entity/" + createdID,
            URLVars: map[string]string{"id": createdID},
        })
        assertJSONResponse(t, w, http.StatusOK)
    })

    t.Run("Update_Success", func(t *testing.T) {
        w := makeHTTPRequest(server, router, HTTPTestRequest{
            Method:  "PUT",
            Path:    "/api/v1/entity/" + createdID,
            URLVars: map[string]string{"id": createdID},
            Body:    map[string]interface{}{"name": "Updated"},
        })
        assertJSONResponse(t, w, http.StatusOK)
    })

    t.Run("Delete_Success", func(t *testing.T) {
        w := makeHTTPRequest(server, router, HTTPTestRequest{
            Method:  "DELETE",
            Path:    "/api/v1/entity/" + createdID,
            URLVars: map[string]string{"id": createdID},
        })
        if w.Code != http.StatusNoContent {
            t.Errorf("Expected 204, got %d", w.Code)
        }
    })
}
```

### 2. Error Testing Pattern

```go
func TestErrorConditions(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    db, dbCleanup := setupTestDB(t)
    defer dbCleanup()

    setupTestTables(t, db)

    server := &APIServer{db: db}
    router := setupRouter(server)

    t.Run("NotFound", func(t *testing.T) {
        nonExistentID := uuid.New().String()
        w := makeHTTPRequest(server, router, HTTPTestRequest{
            Method:  "GET",
            Path:    "/api/v1/entity/" + nonExistentID,
            URLVars: map[string]string{"id": nonExistentID},
        })
        assertErrorResponse(t, w, http.StatusNotFound, "not found")
    })

    t.Run("InvalidJSON", func(t *testing.T) {
        w := makeHTTPRequest(server, router, HTTPTestRequest{
            Method: "POST",
            Path:   "/api/v1/entity",
            Body:   `{"invalid": "json"`,
        })
        assertErrorResponse(t, w, http.StatusBadRequest, "")
    })
}
```

### 3. Performance Testing Pattern

```go
func TestPerformance_Operation(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    db, dbCleanup := setupTestDB(t)
    defer dbCleanup()

    setupTestTables(t, db)
    server := &APIServer{db: db}
    router := setupRouter(server)

    pattern := PerformanceTestPattern{
        Name:        "OperationName",
        Description: "Test operation performance",
        MaxDuration: 100 * time.Millisecond,
        Iterations:  100,
        Execute: func(t *testing.T, iteration int, setupData interface{}) HTTPTestRequest {
            return HTTPTestRequest{
                Method: "GET",
                Path:   "/api/v1/entity",
            }
        },
        Validate: func(t *testing.T, avgDuration time.Duration, setupData interface{}) {
            if avgDuration > 100*time.Millisecond {
                t.Errorf("Too slow: %v", avgDuration)
            }
        },
    }

    RunPerformanceTest(t, server, router, pattern)
}
```

### 4. Concurrent Testing Pattern

```go
func TestConcurrentOperations(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    db, dbCleanup := setupTestDB(t)
    defer dbCleanup()

    setupTestTables(t, db)
    campaign := createTestCampaign(t, db, "Concurrent Test")

    server := &APIServer{db: db}
    router := setupRouter(server)

    t.Run("ConcurrentWrites", func(t *testing.T) {
        done := make(chan bool, 10)

        for i := 0; i < 10; i++ {
            go func(index int) {
                defer func() { done <- true }()

                w := makeHTTPRequest(server, router, HTTPTestRequest{
                    Method: "POST",
                    Path:   "/api/v1/entity",
                    Body:   map[string]interface{}{"name": "Concurrent"},
                })

                if w.Code != http.StatusCreated {
                    t.Errorf("Expected 201, got %d", w.Code)
                }
            }(i)
        }

        for i := 0; i < 10; i++ {
            <-done
        }
    })
}
```

## Test Organization

### File Structure

```
prompt-manager/
├── api/
│   ├── main.go                    # Production code
│   ├── test_helpers.go            # Test utilities
│   ├── test_patterns.go           # Test patterns
│   ├── main_test.go               # Unit tests
│   └── performance_test.go        # Performance tests
└── test/
    └── phases/
        └── test-unit.sh           # Test phase script
```

### Test Naming

- `Test<Component>` - Basic unit test
- `Test<Component>_<Scenario>` - Specific scenario
- `TestPerformance_<Component>` - Performance test
- `Benchmark<Component>` - Go benchmark

### Subtest Organization

Use `t.Run()` for organized subtests:

```go
func TestComponent(t *testing.T) {
    t.Run("Create_Success", func(t *testing.T) { ... })
    t.Run("Create_InvalidInput", func(t *testing.T) { ... })
    t.Run("Get_Success", func(t *testing.T) { ... })
    t.Run("Get_NotFound", func(t *testing.T) { ... })
}
```

## Coverage Goals

### Targets
- **Overall**: 80%
- **Critical paths**: 90%+
- **Error handling**: 85%+
- **Helper functions**: 100%

### Measuring Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View in browser
go tool cover -html=coverage.out

# View summary
go tool cover -func=coverage.out

# Check specific package
go test -cover ./... | grep prompt-manager-api
```

## Performance Benchmarks

### Running Benchmarks

```bash
# All benchmarks
go test -bench=. -benchmem

# Specific benchmark
go test -bench=BenchmarkHealthCheck -benchmem

# With CPU profiling
go test -bench=. -cpuprofile=cpu.prof

# With memory profiling
go test -bench=. -memprofile=mem.prof
```

### Performance Criteria

| Operation | Target | Maximum |
|-----------|--------|---------|
| Health Check | 10ms | 50ms |
| List Campaigns | 50ms | 100ms |
| Create Campaign | 30ms | 100ms |
| Search Prompts | 100ms | 200ms |
| Export Data | 500ms | 1s |

## Debugging Tests

### Verbose Output

```bash
# Run with verbose output
go test -v

# Show test logs
go test -v -run TestCampaignCRUD 2>&1 | less
```

### Debugging Individual Tests

```bash
# Run single test
go test -run TestCampaignCRUD/Create_Success

# Run with debugger (dlv)
dlv test -- -test.run TestCampaignCRUD
```

### Database Debugging

```bash
# Connect to test database
psql $TEST_POSTGRES_URL

# View test schema
\dt prompt_mgr_test.*

# Check test data
SELECT * FROM prompt_mgr_test.campaigns;
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run tests
        env:
          TEST_POSTGRES_URL: postgres://postgres:test@localhost:5432/postgres
        run: |
          cd scenarios/prompt-manager/api
          go test -v -coverprofile=coverage.out ./...
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./scenarios/prompt-manager/api/coverage.out
```

## Best Practices

### ✅ DO

- Use `defer` for cleanup
- Test both success and error cases
- Use subtests for organization
- Validate status codes AND response bodies
- Create isolated test environments
- Use test fixtures for consistent data
- Write descriptive test names
- Add performance benchmarks for critical paths

### ❌ DON'T

- Share state between tests
- Use hardcoded IDs
- Skip cleanup
- Test implementation details
- Use sleeps for synchronization
- Ignore error cases
- Write brittle tests
- Assume test execution order

## Troubleshooting

### Tests Skipping

**Problem**: Tests skip with "TEST_POSTGRES_URL not set"

**Solution**:
```bash
export TEST_POSTGRES_URL="postgres://user:pass@localhost:5432/test_db"
```

### Database Connection Errors

**Problem**: "connection refused" errors

**Solution**:
1. Verify PostgreSQL is running: `pg_isready`
2. Check connection string format
3. Verify user permissions

### Coverage Not Improving

**Problem**: Coverage stuck at low percentage

**Solution**:
1. Ensure TEST_POSTGRES_URL is set
2. Run with `-count=1` to avoid cache
3. Check which functions are untested:
   ```bash
   go test -coverprofile=coverage.out ./...
   go tool cover -func=coverage.out | grep -v "100.0%"
   ```

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Table Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Testify Library](https://github.com/stretchr/testify) (optional)
- [Vrooli Testing Standards](/docs/testing/guides/scenario-unit-testing.md)
- [Gold Standard: visited-tracker](/scenarios/visited-tracker/api/TESTING_GUIDE.md)

## Contributing

When adding new features:

1. Write tests first (TDD)
2. Aim for 80%+ coverage
3. Include error cases
4. Add performance benchmarks for critical paths
5. Update this guide if introducing new patterns
6. Run full test suite before committing

## Support

For questions or issues:
1. Review this guide
2. Check visited-tracker for examples
3. Consult `/docs/testing/` documentation
4. Review test output with `-v` flag
