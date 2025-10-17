# Visitor Intelligence Testing Guide

This guide explains the comprehensive test suite for the visitor-intelligence scenario.

## Test Structure

```
api/
├── main.go                  # Main application code
├── main_test.go             # Comprehensive test suite
├── test_helpers.go          # Reusable test utilities
├── test_patterns.go         # Systematic error testing patterns
└── coverage.out             # Generated coverage report
```

## Test Categories

### 1. **Configuration Tests** (`TestInitConfig`)
- Tests environment variable parsing
- Tests component-based configuration
- Validates required field enforcement
- **Coverage**: Config initialization logic

### 2. **Utility Function Tests**
- `TestGetClientIP`: IP address extraction from headers
- Tests X-Forwarded-For, X-Real-IP, and RemoteAddr fallback
- **Coverage**: Client IP detection logic

### 3. **HTTP Handler Tests**

#### Health Check (`TestHealthHandler`)
- Tests healthy database and Redis connections
- Tests degraded state when services are down
- Validates JSON response structure
- **Coverage**: Health monitoring endpoint

#### Event Tracking (`TestTrackHandler`)
- Tests new visitor creation
- Tests existing visitor tracking
- Tests session management
- Error cases:
  - Missing fingerprint
  - Missing event_type
  - Missing scenario
  - Invalid JSON
  - Invalid HTTP method
- **Coverage**: Core tracking functionality

#### Visitor Retrieval (`TestGetVisitorHandler`)
- Tests visitor profile retrieval
- Tests non-existent visitor handling
- Uses systematic error patterns (invalid UUID, not found)
- **Coverage**: Visitor data retrieval

#### Analytics (`TestGetAnalyticsHandler`)
- Tests analytics with various timeframes (1d, 7d, 30d, 90d)
- Tests default timeframe
- Tests scenarios with no data
- Validates bounce rate calculation
- **Coverage**: Analytics aggregation

#### Tracker Script (`TestTrackerScriptHandler`)
- Tests JavaScript file serving
- Validates content-type and caching headers
- Gracefully skips if file doesn't exist
- **Coverage**: Static file serving

### 4. **Business Logic Tests**

#### Visitor Management (`TestGetOrCreateVisitor`)
- Tests new visitor creation
- Tests existing visitor retrieval
- Tests Redis caching behavior
- **Coverage**: Visitor management logic

#### Session Management (`TestGetOrCreateSession`)
- Tests new session creation
- Tests active session retrieval
- Tests UTM parameter extraction
- **Coverage**: Session tracking logic

### 5. **Performance Tests**

#### `TestPerformance_TrackingEndpoint`
- Max duration: 500ms
- Tests single event tracking performance
- **Coverage**: Performance validation

#### `TestPerformance_BulkTracking`
- Max duration: 5 seconds
- Tests 100 events (10 unique visitors)
- **Coverage**: Bulk operation performance

### 6. **Edge Case Tests**

#### `TestEdgeCase_EmptyProperties`
- Tests tracking with empty property maps
- Validates JSON marshaling

#### `TestEdgeCase_NullOptionalFields`
- Tests null pointer handling
- Validates JSON response for optional fields

## Test Patterns and Helpers

### Test Helpers (`test_helpers.go`)

#### `setupTestLogger()`
Configures controlled logging for tests with cleanup.

```go
cleanup := setupTestLogger()
defer cleanup()
```

#### `setupTestDatabase(t *testing.T)`
Creates isolated test database connections with automatic cleanup.

```go
env := setupTestDatabase(t)
if env == nil {
    t.Skip("Test database not available")
    return
}
defer env.Cleanup()
```

#### `setupTestVisitor(t, fingerprint)`
Creates a pre-configured visitor with sample data.

```go
visitor := setupTestVisitor(t, "test-fingerprint")
defer visitor.Cleanup()
```

#### `makeHTTPRequest(req HTTPTestRequest)`
Creates and configures HTTP test requests.

```go
w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
    Method: "POST",
    Path: "/api/v1/visitor/track",
    Body: event,
})
```

#### `assertJSONResponse(t, w, status, expectedFields)`
Validates JSON responses with expected status and fields.

```go
response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
    "id": visitor.ID,
    "fingerprint": visitor.Fingerprint,
})
```

### Test Patterns (`test_patterns.go`)

#### `TestScenarioBuilder`
Fluent interface for building systematic error tests.

```go
patterns := NewTestScenarioBuilder().
    AddInvalidUUID("GET", "/api/v1/visitor/invalid-uuid").
    AddNonExistentVisitor("GET", "/api/v1/visitor/{id}").
    AddInvalidJSON("POST", "/api/v1/visitor/track").
    Build()

suite := &HandlerTestSuite{
    HandlerName: "getVisitorHandler",
    Handler: getVisitorHandler,
    BaseURL: "/api/v1/visitor/{id}",
}
suite.RunErrorTests(t, patterns)
```

#### Error Patterns
- `invalidUUIDPattern`: Tests with malformed UUIDs
- `nonExistentVisitorPattern`: Tests with non-existent IDs
- `invalidJSONPattern`: Tests with malformed JSON
- `missingRequiredFieldsPattern`: Tests with incomplete data
- `invalidMethodPattern`: Tests with wrong HTTP methods

#### Performance Patterns
```go
pattern := PerformanceTestPattern{
    Name: "TrackingEndpoint",
    MaxDuration: 500 * time.Millisecond,
    Execute: func(t *testing.T, setupData interface{}) time.Duration {
        start := time.Now()
        // ... test code ...
        return time.Since(start)
    },
}
RunPerformanceTest(t, pattern)
```

## Running Tests

### Local Development

```bash
# Run all tests
cd api
go test -v

# Run with coverage
go test -v -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out

# Run specific test
go test -v -run TestTrackHandler

# Run performance tests only
go test -v -run Performance
```

### Through Makefile

```bash
# Run all tests (recommended)
cd scenarios/visitor-intelligence
make test

# Run unit tests
test/phases/test-unit.sh

# Run integration tests
test/phases/test-integration.sh

# Run performance tests
test/phases/test-performance.sh
```

## Test Requirements

### Environment Variables

Tests gracefully skip when database is not available:
- `POSTGRES_URL` or individual components (HOST, PORT, USER, PASSWORD, DB)
- `REDIS_ADDR` (defaults to localhost:6379)
- `REDIS_DB` (defaults to 1 for tests)

### Dependencies

- PostgreSQL database with schema initialized
- Redis server running
- Go 1.16+ with required modules

## Coverage Goals

| Category | Target Coverage |
|----------|----------------|
| Overall | ≥80% |
| Config | 100% |
| Handlers | ≥80% |
| Business Logic | ≥80% |
| Utilities | 100% |

## Best Practices

1. **Always use cleanup functions** with defer to prevent test pollution
2. **Isolate tests** using setupTestDatabase for database-dependent tests
3. **Use table-driven tests** for multiple similar scenarios
4. **Test both success and error paths** comprehensively
5. **Validate both status codes and response bodies**
6. **Use descriptive test names** following the pattern `Test<Function>_<Scenario>`
7. **Skip tests gracefully** when dependencies are unavailable

## Adding New Tests

When adding tests for new handlers:

1. **Add to main_test.go**:
```go
func TestNewHandler(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    env := setupTestDatabase(t)
    if env == nil {
        t.Skip("Test database not available")
        return
    }
    defer env.Cleanup()

    t.Run("Success", func(t *testing.T) { /* ... */ })
    t.Run("Error_Cases", func(t *testing.T) { /* ... */ })
}
```

2. **Use error patterns**:
```go
suite := &HandlerTestSuite{
    HandlerName: "newHandler",
    Handler: newHandler,
}
patterns := NewTestScenarioBuilder().
    AddInvalidUUID("GET", "/path/{id}").
    Build()
suite.RunErrorTests(t, patterns)
```

3. **Add performance tests if needed**:
```go
func TestPerformance_NewEndpoint(t *testing.T) {
    pattern := PerformanceTestPattern{
        Name: "NewEndpoint",
        MaxDuration: 100 * time.Millisecond,
        Execute: func(t *testing.T, data interface{}) time.Duration {
            start := time.Now()
            // test code
            return time.Since(start)
        },
    }
    RunPerformanceTest(t, pattern)
}
```

## Troubleshooting

### Tests Skip Due to Missing Database
**Solution**: Ensure PostgreSQL and Redis are running with correct env vars

### Coverage Lower Than Expected
**Solution**: Check which functions aren't covered with `go tool cover -func=coverage.out`

### Tests Timeout
**Solution**: Increase timeout with `-timeout` flag: `go test -timeout=5m`

### Database Pollution Between Tests
**Solution**: Ensure all tests use `defer cleanup()` and proper test isolation
