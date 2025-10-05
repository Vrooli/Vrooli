# Workflow Scheduler Testing Guide

## Test Coverage Summary

This test suite provides comprehensive coverage of the Workflow Scheduler API, following the gold standard patterns from visited-tracker.

### Test Files

1. **test_helpers.go** - Reusable test utilities
   - `setupTestLogger()` - Controlled logging during tests
   - `setupTestDatabase()` - Test database management
   - `setupTestApp()` - Application instance for testing
   - `makeHTTPRequest()` - HTTP request helpers
   - `assertJSONResponse()` - Response validation
   - `createTestSchedule()` - Schedule creation helpers
   - `createTestExecution()` - Execution creation helpers

2. **test_patterns.go** - Systematic error testing patterns
   - `TestScenarioBuilder` - Fluent interface for building test scenarios
   - Error patterns for all endpoints
   - Reusable error scenarios (InvalidUUID, NonExistent, InvalidJSON, etc.)

3. **main_test.go** - Comprehensive HTTP handler tests
   - Health check endpoint
   - Schedule CRUD operations
   - Execution management
   - Analytics and metrics
   - Cron validation and presets
   - Dashboard statistics
   - Edge cases and error paths

4. **scheduler_test.go** - Scheduler logic tests
   - Scheduler initialization
   - Schedule loading and management
   - Job execution tracking
   - Metrics and monitoring
   - Retry logic
   - HTTP target execution

5. **db_init_test.go** - Database initialization tests
   - Schema creation
   - Table structure validation
   - Index verification
   - Foreign key constraints
   - Cascade delete behavior

## Running Tests

### Prerequisites

Tests require a PostgreSQL test database. Set the `TEST_POSTGRES_URL` environment variable:

```bash
export TEST_POSTGRES_URL="postgresql://postgres:postgres@localhost:5432/workflow_scheduler_test"
```

### Run All Tests

```bash
cd api
go test -v ./...
```

### Run with Coverage

```bash
cd api
go test -v -coverprofile=coverage.out -covermode=count ./...
go tool cover -html=coverage.out -o coverage.html
```

### Run Specific Test Suite

```bash
# HTTP handlers only
go test -v -run TestHealthCheck

# Scheduler logic only
go test -v -run TestScheduler

# Database initialization only
go test -v -run TestInitializeDatabase
```

### Run Short Tests (Skip Database)

```bash
go test -v -short ./...
```

## Test Coverage Goals

- **Target**: ≥80% coverage
- **Minimum**: 70% coverage
- **Current**: See coverage report

### Coverage by Component

- **HTTP Handlers**: All endpoints tested with success and error paths
- **Scheduler Logic**: Job management, execution tracking, retry logic
- **Database Operations**: Schema, CRUD, constraints, cascades
- **Helper Functions**: Cron validation, time calculations, response formatting
- **Error Handling**: Systematic error testing for all endpoints

## Test Patterns Used

### 1. Setup-Execute-Assert Pattern

```go
func TestExample(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    app := setupTestApp(t)
    defer cleanupTestData(t, app.DB)

    // Execute
    req := HTTPTestRequest{...}
    rr, _ := makeHTTPRequest(app.Router, req)

    // Assert
    assertJSONResponse(t, rr, http.StatusOK, &response)
}
```

### 2. Table-Driven Tests

```go
t.Run("SpecialExpressions", func(t *testing.T) {
    expressions := []string{"@hourly", "@daily", "@weekly"}

    for _, expr := range expressions {
        // Test each expression
    }
})
```

### 3. Error Test Patterns

```go
t.Run("ErrorPaths", func(t *testing.T) {
    RunErrorScenarios(t, app, GetScheduleErrors())
})
```

### 4. Subtests for Organization

```go
func TestScheduler(t *testing.T) {
    t.Run("Success", func(t *testing.T) { ... })
    t.Run("ErrorCase", func(t *testing.T) { ... })
    t.Run("EdgeCase", func(t *testing.T) { ... })
}
```

## Integration with Centralized Testing

The test suite integrates with Vrooli's centralized testing infrastructure:

- Uses `testing::unit::run_all_tests` from `scripts/scenarios/testing/unit/run-all.sh`
- Integrates with `phase-helpers.sh` for consistent reporting
- Follows coverage thresholds: `--coverage-warn 80 --coverage-error 50`

## Test Quality Checklist

- [x] All HTTP endpoints have success tests
- [x] All HTTP endpoints have error tests
- [x] Edge cases covered (empty data, invalid input, missing fields)
- [x] Database operations tested (CRUD, constraints, cascades)
- [x] Helper functions tested
- [x] Proper cleanup with defer statements
- [x] Isolated test environments
- [x] Response validation (status code AND body)
- [x] Systematic error testing patterns
- [x] Test helpers for code reuse

## Known Limitations

1. **Database Required**: Most tests require a PostgreSQL test database
2. **Network Tests**: HTTP target execution tests may fail without network access
3. **Background Jobs**: Scheduler goroutines not fully tested (requires time-based testing)
4. **Redis**: Redis integration not yet tested

## Future Improvements

1. Add mock HTTP server for testing HTTP target execution
2. Add Redis integration tests
3. Add concurrent execution tests
4. Add performance/load tests
5. Add end-to-end integration tests
