# Testing Guide for Vrooli Assistant

## Overview

This guide describes the testing infrastructure for the Vrooli Assistant scenario. The test suite follows Vrooli's gold standard testing patterns as demonstrated in the visited-tracker scenario.

## Test Structure

```
vrooli-assistant/
├── api/
│   ├── main.go              # Main application code
│   ├── test_helpers.go      # Reusable test utilities
│   ├── test_patterns.go     # Systematic error patterns
│   └── main_test.go         # Comprehensive test suite
├── test/
│   └── phases/
│       └── test-unit.sh     # Integrated with centralized testing
```

## Test Coverage

**Target Coverage**: 80% (minimum 50%)

### Current Coverage Areas

1. **Health Endpoint** (100%)
   - Health check returns correct status
   - JSON response format validation

2. **Model Structs** (100%)
   - Issue model creation and validation
   - AgentSession model creation and validation

3. **Database Tests** (Conditional)
   - Connection handling
   - Table creation
   - CRUD operations for issues
   - CRUD operations for agent sessions
   - Concurrent request handling

4. **HTTP Handlers** (Database-dependent)
   - Status handler with metrics
   - Capture handler with validation
   - Spawn agent handler
   - History handler with pagination
   - Individual issue retrieval
   - Status update operations

## Running Tests

### With Database (Full Coverage)

```bash
# Ensure PostgreSQL is running and accessible
export TEST_POSTGRES_URL="postgres://user:pass@localhost:5432/vrooli_assistant_test"

# Or use individual components
export TEST_POSTGRES_HOST="localhost"
export TEST_POSTGRES_PORT="5432"
export TEST_POSTGRES_USER="postgres"
export TEST_POSTGRES_PASSWORD="postgres"
export TEST_POSTGRES_DB="vrooli_assistant_test"

# Run tests
cd api
go test -v -tags=testing -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Without Database (Partial Coverage)

```bash
# Tests will gracefully skip database-dependent tests
cd api
go test -v -tags=testing
```

### Using Centralized Test Runner

```bash
# From scenario root
cd test/phases
./test-unit.sh

# Or use Make
cd ../..
make test
```

## Test Helpers

### setupTestLogger()
Silences logs during tests unless `TEST_VERBOSE=1` is set.

```go
cleanup := setupTestLogger()
defer cleanup()
```

### setupTestDB(t)
Creates isolated test database with automatic cleanup.

```go
testDB := setupTestDB(t)
if testDB == nil {
    t.Skip("Database not available")
    return
}
defer testDB.Cleanup()
```

### setupTestRouter()
Creates a router with all handlers registered.

```go
router := setupTestRouter()
```

### makeHTTPRequest(req, handler)
Executes HTTP test requests with automatic JSON marshaling.

```go
w, err := makeHTTPRequest(HTTPTestRequest{
    Method: "POST",
    Path:   "/api/v1/assistant/capture",
    Body:   issueData,
    URLVars: map[string]string{"id": "123"},
}, router)
```

### Assertion Helpers

- `assertJSONResponse(t, w, expectedStatus)` - Validates JSON response
- `assertErrorResponse(t, w, expectedStatus, expectedMessage)` - Validates error format
- `assertFieldExists(t, response, field)` - Checks field presence
- `assertFieldEquals(t, response, field, expected)` - Validates field value

### Test Data Creation

- `createTestIssue(t, db)` - Creates a test issue, returns ID
- `createTestAgentSession(t, db, issueID)` - Creates a test session, returns ID

## Test Patterns

### Error Testing with TestScenarioBuilder

```go
patterns := NewTestScenarioBuilder().
    AddInvalidUUID("/api/v1/issues/{id}", "GET").
    AddNonExistentIssue("/api/v1/issues/{id}", "GET").
    AddInvalidJSON("/api/v1/capture", "POST").
    Build()

suite := NewHandlerTestSuite("IssueHandler", router, "/api/v1/issues/{id}")
suite.RunErrorTests(t, patterns)
```

### Success Testing

```go
suite.RunSuccessTest(t, "Success", HTTPTestRequest{
    Method: "GET",
    Path:   "/health",
}, func(t *testing.T, w *httptest.ResponseRecorder) {
    response := assertJSONResponse(t, w, http.StatusOK)
    assertFieldEquals(t, response, "status", "healthy")
})
```

## Best Practices

### 1. Test Isolation
Every test should be independent and not rely on other tests' state.

```go
t.Run("TestCase", func(t *testing.T) {
    // Setup
    testDB := setupTestDB(t)
    defer testDB.Cleanup()

    // Test logic
    // ...
})
```

### 2. Cleanup with Defer
Always use defer for cleanup to ensure resources are released even if tests fail.

```go
cleanup := setupTestLogger()
defer cleanup()
```

### 3. Skip Gracefully
Tests should skip gracefully when dependencies are unavailable.

```go
testDB := setupTestDB(t)
if testDB == nil {
    t.Skip("Database not available for testing")
    return
}
defer testDB.Cleanup()
```

### 4. Verbose Error Messages
Include context in error messages to make debugging easier.

```go
if w.Code != http.StatusOK {
    t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
}
```

### 5. Test Both Happy and Error Paths
Every handler should have tests for:
- Success cases
- Invalid input
- Missing required fields
- Non-existent resources
- Edge cases (empty values, etc.)

## Integration with Centralized Testing

The `test/phases/test-unit.sh` script integrates with Vrooli's centralized testing infrastructure:

```bash
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 50 \
    --verbose
```

## Extending Tests

### Adding New Handler Tests

1. Create test function: `TestNewHandler(t *testing.T)`
2. Set up logger, database (if needed), and router
3. Add success test cases
4. Add error test cases using TestScenarioBuilder
5. Verify database state changes (if applicable)

### Adding New Test Helpers

Add helpers to `test_helpers.go` following these patterns:
- Return cleanup functions from setup helpers
- Use descriptive names
- Include error handling
- Document expected behavior

### Adding New Error Patterns

Add patterns to `test_patterns.go`:
- Implement `ErrorTestPattern` interface
- Provide clear name and description
- Set expected HTTP status code
- Include validation logic

## Continuous Improvement

This test suite should be continuously improved as the application evolves:
- Add tests for new features
- Increase coverage for edge cases
- Refactor helpers for better reusability
- Update patterns based on common error scenarios

## References

- Gold Standard: `/scenarios/visited-tracker/` (79.4% Go coverage)
- Testing Guide: `/docs/testing/guides/scenario-unit-testing.md`
- Phase Helpers: `/scripts/scenarios/testing/shell/phase-helpers.sh`
