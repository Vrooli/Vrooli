# Swarm Manager Testing Guide

## Overview

The swarm-manager test suite provides comprehensive coverage of the application's functionality, focusing on file-based task management, utility functions, and API endpoints.

## Test Structure

### Test Files

- `test_helpers.go` - Reusable test utilities and setup functions
- `test_patterns.go` - Systematic error testing patterns
- `main_test.go` - Comprehensive test cases

### Test Organization

```
api/
├── test_helpers.go         # Reusable test utilities
├── test_patterns.go        # Systematic error patterns
├── main_test.go            # Comprehensive tests
└── TESTING_GUIDE.md        # This file
```

## Running Tests

### Basic Test Execution

```bash
# Run all tests
cd api && go test -tags=testing -v

# Run with coverage
cd api && go test -tags=testing -coverprofile=coverage.out -covermode=atomic

# View coverage report
go tool cover -html=coverage.out

# Run short tests (skip performance/integration tests)
go test -tags=testing -short
```

### Using Test Phases

```bash
# Run unit tests through centralized testing infrastructure
cd .. && test/phases/test-unit.sh
```

## Test Patterns

### 1. Helper Functions

The test suite provides several helper functions to simplify test writing:

#### Setup Functions

- `setupTestLogger()` - Configures logging for tests
- `setupTestDirectory(t)` - Creates isolated test environment
- `setupTestDB(t)` - Attempts database connection (skips if unavailable)
- `setupTestTask(t, env, taskType, status)` - Creates test task
- `setupTestAgent(t, name)` - Creates test agent
- `setupTestProblem(t, severity, status)` - Creates test problem

#### Request Helpers

- `makeFiberRequest(app, req)` - Executes Fiber HTTP request
- `assertJSONResponse(t, w, status, fields)` - Validates JSON response
- `assertErrorResponse(t, w, status, message)` - Validates error response

### 2. Test Scenarios

#### Success Cases
Test happy paths with valid inputs and expected outputs.

```go
t.Run("Success", func(t *testing.T) {
    // Setup
    env := setupTestDirectory(t)
    defer env.Cleanup()

    // Execute
    w, err := makeFiberRequest(app, FiberTestRequest{
        Method: "GET",
        Path:   "/health",
    })

    // Assert
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
})
```

#### Error Cases
Test error handling with invalid inputs, missing resources, etc.

```go
t.Run("Error_InvalidJSON", func(t *testing.T) {
    w, err := makeFiberRequest(app, FiberTestRequest{
        Method: "POST",
        Path:   "/api/tasks",
        Body:   `{"invalid": "json"`,
    })

    if w.Code != http.StatusBadRequest {
        t.Errorf("Expected status 400, got %d", w.Code)
    }
})
```

#### Edge Cases
Test boundary conditions, empty inputs, null values, etc.

```go
t.Run("Edge_EmptyBody", func(t *testing.T) {
    w, err := makeFiberRequest(app, FiberTestRequest{
        Method: "POST",
        Path:   "/api/tasks",
        Body:   nil,
    })

    if w.Code != http.StatusBadRequest {
        t.Errorf("Expected status 400, got %d", w.Code)
    }
})
```

### 3. Systematic Error Patterns

Use `TestScenarioBuilder` for systematic error testing:

```go
patterns := NewTestScenarioBuilder().
    AddNonExistentTask("GET", "/api/tasks/%s").
    AddInvalidJSON("POST", "/api/tasks").
    AddEmptyBody("POST", "/api/tasks").
    Build()

suite := &HandlerTestSuite{
    HandlerName: "createTask",
    BaseURL:     "/api/tasks",
}

suite.RunErrorTests(t, app, patterns)
```

## Database-Dependent Tests

Some tests require a PostgreSQL database. These tests will be skipped if the database is not available.

### Running with Database

Set environment variables:

```bash
export TEST_POSTGRES_HOST=localhost
export TEST_POSTGRES_PORT=5432
export TEST_POSTGRES_USER=postgres
export TEST_POSTGRES_PASSWORD=postgres
export TEST_POSTGRES_DB=swarm_manager_test
```

Then run tests:

```bash
go test -tags=testing -v
```

### Tests that require database:

- `TestCreateTask/Success` - Creates task and logs to database
- Other database-dependent endpoints

## Coverage Goals

- **Target**: ≥80% coverage
- **Minimum**: 70% coverage
- **Current**: Run `go test -cover` to see current coverage

### Coverage by Component

- Utility functions: 100%
- File operations: 95%
- HTTP handlers: 80%
- Database operations: 70% (optional, depends on test DB)

## Performance Testing

Performance tests are skipped in short mode:

```bash
# Run including performance tests
go test -tags=testing -v

# Skip performance tests
go test -tags=testing -v -short
```

Performance benchmarks:

- Task creation: < 100ms per task
- Task retrieval: < 50ms per request

## Best Practices

### 1. Always Clean Up

```go
env := setupTestDirectory(t)
defer env.Cleanup()
```

### 2. Use Table-Driven Tests

```go
tests := []struct {
    name        string
    input       interface{}
    expectedVal float64
}{
    {"ValidFloat64", float64(5.5), 5.5},
    {"ValidInt", 10, 10.0},
    {"NilValue", nil, 7.5},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        result := getFloat(tt.input, tt.defaultVal)
        if result != tt.expectedVal {
            t.Errorf("Expected %f, got %f", tt.expectedVal, result)
        }
    })
}
```

### 3. Test Both Success and Failure

Every endpoint should have tests for:
- Success with valid input
- Error with invalid input
- Error with missing input
- Error with malformed input

### 4. Validate Response Structure

```go
var response map[string]interface{}
if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
    t.Fatalf("Failed to parse response: %v", err)
}

if _, ok := response["expected_field"]; !ok {
    t.Error("Expected 'expected_field' in response")
}
```

## Common Issues

### Database Connection Errors

If you see database connection errors and don't need database testing:

```bash
# Run without database-dependent tests
go test -tags=testing -short -v
```

### Build Tag Issues

Always use `-tags=testing` flag when running tests:

```bash
# Correct
go test -tags=testing -v

# Wrong (will fail to compile)
go test -v
```

### Test Isolation

Each test should be independent. Use `setupTestDirectory(t)` to create isolated environments.

## Contributing

When adding new tests:

1. Follow the existing patterns
2. Add appropriate cleanup with `defer`
3. Test success, error, and edge cases
4. Update this guide if introducing new patterns
5. Ensure coverage remains above 80%
