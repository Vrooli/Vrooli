# Notification Hub - Testing Guide

## Quick Start

### Run All Tests
```bash
cd scenarios/notification-hub
make test
```

### Run Specific Test Phases
```bash
# Unit tests only
bash test/phases/test-unit.sh

# Integration tests
bash test/phases/test-integration.sh

# Performance tests
bash test/phases/test-performance.sh
```

## Test Structure

### Test Files

```
notification-hub/
├── api/
│   ├── test_helpers.go       # Reusable test utilities
│   ├── test_patterns.go      # Systematic error patterns
│   ├── main_test.go          # API endpoint tests
│   ├── processor_test.go     # Notification processing tests
│   └── performance_test.go   # Performance benchmarks
├── test/
│   ├── phases/
│   │   ├── test-unit.sh      # Unit test runner
│   │   ├── test-integration.sh
│   │   └── test-performance.sh
│   └── run-tests.sh          # Main test runner
└── TEST_IMPLEMENTATION_SUMMARY.md
```

## Test Environment Setup

### Prerequisites

1. **PostgreSQL Database**
   ```bash
   # Create test database
   createdb notification_hub_test

   # Run migrations
   psql notification_hub_test < initialization/postgres/schema.sql
   ```

2. **Redis Server**
   ```bash
   # Ensure Redis is running
   redis-cli ping
   # Should return: PONG
   ```

3. **Environment Variables**
   ```bash
   # Database connection
   export TEST_POSTGRES_URL="postgres://user:pass@localhost:5432/notification_hub_test?sslmode=disable"

   # Or use individual components
   export TEST_POSTGRES_HOST="localhost"
   export TEST_POSTGRES_PORT="5432"
   export TEST_POSTGRES_USER="postgres"
   export TEST_POSTGRES_PASSWORD="postgres"
   export TEST_POSTGRES_DB="notification_hub_test"

   # Redis connection (optional, has defaults)
   export TEST_REDIS_URL="redis://localhost:6379/15"
   ```

## Running Tests

### Basic Test Execution

```bash
# Run all tests with verbose output
cd api
go test -tags=testing -v ./...

# Run with coverage
go test -tags=testing -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run specific test
go test -tags=testing -v -run TestHealthCheck

# Run specific test file
go test -tags=testing -v ./... -run TestNotificationSending
```

### Performance Testing

```bash
# Run all benchmarks
go test -tags=performance -bench=. -benchmem ./...

# Run specific benchmark
go test -tags=performance -bench=BenchmarkHealthCheck

# Run with custom benchmark time
go test -tags=performance -bench=. -benchtime=10s
```

### Integration Testing

```bash
# Run integration tests (requires database/Redis)
bash test/phases/test-integration.sh

# Or with go test
go test -tags=integration -v ./...
```

## Test Patterns

### Using Test Helpers

```go
func TestMyEndpoint(t *testing.T) {
    // Setup logger
    cleanup := setupTestLogger()
    defer cleanup()

    // Setup environment (database, Redis, test profile)
    env := setupTestEnvironment(t)
    defer env.Cleanup()

    // Make HTTP request
    w, err := makeHTTPRequest(env, HTTPTestRequest{
        Method:  "GET",
        Path:    "/api/v1/profiles/" + env.TestProfile.ID.String() + "/notifications",
        Headers: map[string]string{"X-API-Key": env.TestAPIKey},
    })

    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }

    // Assert response
    response := assertJSONResponse(t, w, http.StatusOK, []string{"notifications"})

    // Validate response data
    notifications, ok := response["notifications"].([]interface{})
    if !ok {
        t.Fatal("Expected notifications to be an array")
    }
}
```

### Using Error Patterns

```go
func TestErrorHandling(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    env := setupTestEnvironment(t)
    defer env.Cleanup()

    // Build systematic error test suite
    patterns := NewTestScenarioBuilder().
        AddMissingAPIKey("/api/v1/profiles/%s/notifications", "GET").
        AddInvalidAPIKey("/api/v1/profiles/%s/notifications", "GET").
        AddInvalidJSON("/api/v1/profiles/%s/notifications/send", "POST").
        AddNonExistentContact("/api/v1/profiles/%s/contacts/%s", "GET").
        Build()

    // Run all error patterns
    RunErrorTests(t, env, patterns)
}
```

### Using Performance Patterns

```go
func TestPerformance(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    env := setupTestEnvironment(t)
    defer env.Cleanup()

    // Build performance test suite
    tests := NewPerformanceTestBuilder().
        AddEndpointLatencyTest("health_check", "/health", 200*time.Millisecond).
        AddEndpointLatencyTest("profile_list", "/api/v1/admin/profiles", 500*time.Millisecond).
        Build()

    RunPerformanceTests(t, env, tests)
}
```

## Writing New Tests

### Test File Template

```go
// +build testing

package main

import (
    "testing"
)

func TestNewFeature(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    env := setupTestEnvironment(t)
    defer env.Cleanup()

    t.Run("SuccessCase", func(t *testing.T) {
        // Test happy path
    })

    t.Run("ErrorCases", func(t *testing.T) {
        // Test error conditions
    })

    t.Run("EdgeCases", func(t *testing.T) {
        // Test boundary conditions
    })
}
```

### Best Practices

1. **Always Use Helpers**
   - Use `setupTestLogger()` for controlled logging
   - Use `setupTestEnvironment()` for complete test setup
   - Use `makeHTTPRequest()` for HTTP calls
   - Use assertion helpers for validation

2. **Defer Cleanup**
   ```go
   env := setupTestEnvironment(t)
   defer env.Cleanup()  // Always defer cleanup
   ```

3. **Test Structure**
   - Setup phase
   - Success cases
   - Error cases
   - Edge cases
   - Cleanup (deferred)

4. **Use Sub-tests**
   ```go
   t.Run("SubTestName", func(t *testing.T) {
       // Sub-test logic
   })
   ```

5. **Table-Driven Tests**
   ```go
   testCases := []struct{
       name     string
       input    interface{}
       expected interface{}
   }{
       {"case1", input1, expected1},
       {"case2", input2, expected2},
   }

   for _, tc := range testCases {
       t.Run(tc.name, func(t *testing.T) {
           // Test logic
       })
   }
   ```

## Troubleshooting

### Tests Skip with "Test database not available"

**Problem**: Database connection failed

**Solutions**:
1. Ensure PostgreSQL is running
2. Check `TEST_POSTGRES_URL` is correct
3. Verify test database exists
4. Run migrations on test database

### Tests Skip with "Test Redis not available"

**Problem**: Redis connection failed

**Solutions**:
1. Ensure Redis is running (`redis-cli ping`)
2. Check `TEST_REDIS_URL` is correct
3. Verify Redis is accessible

### Tests Fail with "table does not exist"

**Problem**: Database schema not initialized

**Solution**:
```bash
psql $TEST_POSTGRES_URL < initialization/postgres/schema.sql
```

### Tests Hang or Timeout

**Problem**: Database/Redis operations blocking

**Solutions**:
1. Increase test timeout: `go test -timeout 5m`
2. Check database connections: `SELECT * FROM pg_stat_activity`
3. Verify no deadlocks or long-running queries

### Coverage Report Shows 0%

**Problem**: Tests not running with correct build tags

**Solution**:
```bash
go test -tags=testing -cover ./...
```

## Coverage Goals

- **Target**: 80% coverage
- **Minimum**: 50% coverage
- **Current**: See `make test` output

### View Coverage Report

```bash
cd api
go test -tags=testing -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Coverage by Package

```bash
go test -tags=testing -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## CI/CD Integration

### GitHub Actions Example

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
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: notification_hub_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      redis:
        image: redis:7
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run migrations
        run: |
          psql -h localhost -U postgres notification_hub_test < initialization/postgres/schema.sql
        env:
          PGPASSWORD: postgres
      - name: Run tests
        run: |
          cd scenarios/notification-hub
          make test
        env:
          TEST_POSTGRES_URL: postgres://postgres:postgres@localhost:5432/notification_hub_test?sslmode=disable
          TEST_REDIS_URL: redis://localhost:6379/15
```

## Additional Resources

- [TEST_IMPLEMENTATION_SUMMARY.md](./TEST_IMPLEMENTATION_SUMMARY.md) - Detailed implementation summary
- [Vrooli Testing Guide](/docs/testing/guides/scenario-unit-testing.md) - General testing guidelines
- [Visited Tracker Tests](/scenarios/visited-tracker/) - Gold standard reference

---

**Last Updated**: 2025-10-04
**Test Framework**: Go testing + Vrooli centralized infrastructure
