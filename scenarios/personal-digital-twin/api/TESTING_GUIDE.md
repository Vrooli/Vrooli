# Testing Guide for Personal Digital Twin

## Overview

This testing suite provides comprehensive coverage for the Personal Digital Twin scenario, including unit tests, integration tests, and performance tests.

## Test Structure

### Test Files

- **`test_helpers.go`** - Reusable test utilities and helpers
  - `setupTestLogger()` - Controlled logging during tests
  - `setupTestDB()` - Test database connection management
  - `setupTestPersona()` - Creates test personas with cleanup
  - `assertJSONResponse()` - Validates JSON responses
  - `assertErrorResponse()` - Validates error responses
  - `TestDataGenerator` - Utilities for generating test data

- **`test_patterns.go`** - Systematic error testing patterns
  - `TestScenarioBuilder` - Fluent interface for building test scenarios
  - `ErrorTestPattern` - Systematic error condition testing
  - `PerformanceTestPattern` - Performance testing scenarios
  - `runConcurrentRequests()` - Concurrency testing utilities

- **`main_test.go`** - Comprehensive unit tests
  - Tests for all API handlers
  - Success cases, error cases, and edge cases
  - Concurrent request testing
  - Configuration testing
  - Performance benchmarks

## Running Tests

### Prerequisites

1. **PostgreSQL Database**: Tests require a PostgreSQL database
   ```bash
   export POSTGRES_URL="postgres://user:pass@localhost:5432/testdb"
   ```

2. **Start Required Resources**:
   ```bash
   # Start PostgreSQL if not running
   vrooli resource start postgres
   ```

### Running Unit Tests

```bash
# Run all unit tests
cd api && go test -v

# Run with coverage
go test -v -coverprofile=coverage.out -covermode=atomic

# View coverage report
go tool cover -html=coverage.out

# View coverage summary
go tool cover -func=coverage.out
```

### Running via Test Phases

```bash
# Unit tests
./test/phases/test-unit.sh

# Integration tests (requires running scenario)
./test/phases/test-integration.sh

# Performance tests (requires running scenario)
./test/phases/test-performance.sh
```

### Running via Makefile

```bash
# Run all tests
make test

# Or use shorthand
make t
```

## Test Coverage

### Current Coverage: 31.7% (without database)

**Note**: Coverage is limited without database connection. With proper database setup, coverage should reach **80%+**.

### Coverage by Handler

- `createPersona`: Validates request parsing and error handling
- `getPersona`: Tests retrieval, not found, and invalid UUID scenarios
- `listPersonas`: Tests listing and empty results
- `connectDataSource`: Tests data source connection and validation
- `getDataSources`: Tests retrieval of data sources
- `searchDocuments`: Tests search functionality with various parameters
- `startTraining`: Tests training job creation
- `getTrainingJobs`: Tests training job retrieval
- `createAPIToken`: Tests token creation with default permissions
- `getAPITokens`: Tests token listing
- `handleChat`: Tests chat functionality and session management
- `getChatHistory`: Tests conversation history retrieval
- Health endpoint: Fully tested

## Test Patterns

### 1. Success Cases

Each handler test includes success scenarios:

```go
t.Run("Success", func(t *testing.T) {
    testPersona := setupTestPersona(t, "Test Persona")
    defer testPersona.Cleanup()

    // Execute request
    // Validate response
})
```

### 2. Error Cases

Systematic error testing:

- Missing required fields
- Invalid JSON
- Empty request body
- Non-existent resources
- Invalid UUIDs

### 3. Edge Cases

- Default values (e.g., default permissions, default limit)
- New session generation
- Empty lists

### 4. Concurrency Tests

Tests concurrent request handling:

```go
t.Run("ConcurrentPersonaCreation", func(t *testing.T) {
    errors := runConcurrentRequests(t, 10, func(i int) error {
        // Make request
        // Validate response
    })
    validateNoConcurrencyErrors(t, errors)
})
```

### 5. Performance Tests

Benchmarks for response times:

- Health endpoint: < 0.1s
- Persona retrieval: < 0.5s
- List personas: < 1.0s
- Chat response: < 2.0s
- Concurrent requests (10): < 5.0s

## Integration Tests

### Test Flow

1. **Setup**: Start scenario, wait for API readiness
2. **Test Sequence**:
   - Health check
   - Create persona
   - Retrieve persona
   - List personas
   - Connect data source
   - Start training job
   - Create API token
   - Search documents
   - Chat interaction
   - Chat history retrieval
3. **Cleanup**: Automatic via test lifecycle

### Running Integration Tests

```bash
# Ensure scenario is running
make run

# Run integration tests
./test/phases/test-integration.sh
```

## Performance Tests

### Metrics

- **Response Time**: Maximum acceptable latency for each endpoint
- **Throughput**: Concurrent request handling capacity
- **Consistency**: Performance under load

### Running Performance Tests

```bash
# Run performance tests
./test/phases/test-performance.sh
```

## Test Data Management

### Test Personas

All test personas are prefixed with `test-` for easy identification and cleanup:

```go
personaID := "test-" + uuid.New().String()
```

### Cleanup

Tests use `defer` statements to ensure cleanup:

```go
testPersona := setupTestPersona(t, "Test Persona")
defer testPersona.Cleanup()  // Guaranteed cleanup
```

### Database Isolation

Tests clean up data in the cleanup phase:

```go
testDB.Exec("DELETE FROM conversations WHERE persona_id LIKE 'test-%'")
testDB.Exec("DELETE FROM api_tokens WHERE persona_id LIKE 'test-%'")
testDB.Exec("DELETE FROM training_jobs WHERE persona_id LIKE 'test-%'")
testDB.Exec("DELETE FROM data_sources WHERE persona_id LIKE 'test-%'")
testDB.Exec("DELETE FROM personas WHERE id LIKE 'test-%'")
```

## Troubleshooting

### Database Connection Issues

If tests fail with "relation does not exist":

1. Ensure PostgreSQL is running
2. Set `POSTGRES_URL` environment variable
3. Run database migrations:
   ```bash
   psql $POSTGRES_URL -f initialization/postgres/schema.sql
   ```

### Coverage Too Low

To increase coverage:

1. Ensure database is connected
2. Run with `-covermode=atomic` for accurate coverage
3. Check which functions are uncovered:
   ```bash
   go tool cover -func=coverage.out | grep -E '^[^[:space:]]' | grep -v '100.0%'
   ```

### Integration Tests Timeout

If integration tests timeout:

1. Check if scenario is running: `make status`
2. Check logs: `make logs`
3. Increase timeout in test phase script

## Best Practices

1. **Always use test helpers**: Don't duplicate setup code
2. **Use defer for cleanup**: Ensures cleanup even on test failure
3. **Test both success and failure**: Cover happy path and error cases
4. **Use descriptive test names**: Makes failures easy to diagnose
5. **Keep tests isolated**: Each test should be independent
6. **Mock external services**: Don't rely on external APIs in unit tests

## Extending Tests

### Adding New Handler Tests

```go
func TestNewHandler(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    env := setupTestDB(t)
    defer env.Cleanup()

    t.Run("Success", func(t *testing.T) {
        // Test success case
    })

    t.Run("ErrorCases", func(t *testing.T) {
        // Test error cases
    })
}
```

### Adding New Test Patterns

```go
func customTestPattern() ErrorTestPattern {
    return ErrorTestPattern{
        Name:           "CustomPattern",
        Description:    "Description of the test",
        ExpectedStatus: http.StatusBadRequest,
        Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
            return &HTTPTestRequest{
                Method: "POST",
                Path:   "/api/custom",
                Body:   "test data",
            }
        },
    }
}
```

## Continuous Integration

The test suite integrates with Vrooli's centralized testing infrastructure:

- **Phase-based execution**: Tests run in phases (unit → integration → performance)
- **Coverage thresholds**: Warn at 80%, error at 50%
- **Automated cleanup**: Test environments are cleaned automatically
- **Parallel execution**: Independent tests can run concurrently

## Coverage Goals

- **Target**: 80% statement coverage
- **Minimum**: 50% statement coverage
- **Current**: 31.7% (without database), expected 80%+ with database

## Additional Resources

- [Vrooli Testing Guide](/docs/testing/guides/scenario-unit-testing.md)
- [Gold Standard: visited-tracker](/scenarios/visited-tracker/api/TESTING_GUIDE.md)
- [Centralized Testing Infrastructure](/scripts/scenarios/testing/)
