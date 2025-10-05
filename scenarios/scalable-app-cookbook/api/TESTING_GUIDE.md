# Testing Guide - Scalable App Cookbook API

## Quick Start

```bash
# Run all tests
cd scenarios/scalable-app-cookbook/api
go test -tags=testing -v

# Run with coverage
go test -tags=testing -coverprofile=coverage.out -covermode=atomic
go tool cover -html=coverage.out

# Run specific test
go test -tags=testing -v -run TestSearchPatternsHandler

# Run all handler tests
go test -tags=testing -v -run "Test.*Handler"
```

## Test Organization

### Test Files

| File | Purpose | Test Count |
|------|---------|------------|
| `test_helpers.go` | Reusable test utilities | N/A |
| `test_patterns.go` | Systematic error patterns | N/A |
| `main_test.go` | HTTP handler tests | 31 |
| `structures_test.go` | Data structure tests | 13 |

### Test Helpers

#### setupTestLogger()
Configures logging for tests with automatic cleanup:
```go
func TestMyHandler(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()
    // Test code...
}
```

#### setupTestDatabase(t)
Creates test database connection with auto-skip if unavailable:
```go
func TestDatabaseOperation(t *testing.T) {
    testDB := setupTestDatabase(t)
    if testDB == nil {
        t.Skip("Database not available")
    }
    defer testDB.Cleanup()
    // Test code...
}
```

#### makeHTTPRequest(handler, request)
Simplified HTTP request testing:
```go
req := HTTPTestRequest{
    Method: "GET",
    Path:   "/api/v1/patterns/search",
    Query:  map[string]string{"limit": "10"},
}
w := makeHTTPRequest(searchPatternsHandler, req)
```

#### assertJSONResponse(t, w, expectedStatus)
Validates JSON response with proper assertions:
```go
response := assertJSONResponse(t, w, http.StatusOK)
if total, ok := response["total"]; !ok {
    t.Error("Expected 'total' field")
}
```

## Writing New Tests

### Basic Handler Test

```go
func TestMyNewHandler(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    testDB := setupTestDatabase(t)
    if testDB == nil {
        t.Skip("Database not available")
    }
    defer testDB.Cleanup()

    t.Run("Success_HappyPath", func(t *testing.T) {
        req := HTTPTestRequest{
            Method: "GET",
            Path:   "/api/v1/my-endpoint",
        }

        w := makeHTTPRequest(myNewHandler, req)
        response := assertJSONResponse(t, w, http.StatusOK)

        // Add specific assertions
        if data, ok := response["data"]; !ok {
            t.Error("Expected 'data' field")
        }
    })

    t.Run("Error_InvalidInput", func(t *testing.T) {
        req := HTTPTestRequest{
            Method: "POST",
            Path:   "/api/v1/my-endpoint",
            Body:   "invalid-json",
        }

        w := makeHTTPRequest(myNewHandler, req)
        assertErrorResponse(t, w, http.StatusBadRequest, "")
    })
}
```

### Using Test Patterns

```go
func TestMyHandlerErrorCases(t *testing.T) {
    scenarios := NewTestScenarioBuilder().
        AddInvalidUUID("/api/v1/patterns/{id}").
        AddNonExistentPattern("/api/v1/patterns/{id}").
        AddInvalidJSON("/api/v1/patterns", "POST").
        Build()

    for _, scenario := range scenarios {
        t.Run(scenario.Name, func(t *testing.T) {
            w := makeHTTPRequest(myHandler, scenario.Request)
            if w.Code != scenario.ExpectedStatus {
                t.Errorf("Expected %d, got %d", scenario.ExpectedStatus, w.Code)
            }
        })
    }
}
```

### Testing with Seeded Data

```go
func TestWithData(t *testing.T) {
    testDB := setupTestDatabase(t)
    if testDB == nil {
        t.Skip("Database not available")
    }
    defer testDB.Cleanup()

    // Seed test data
    seedTestData(t, testDB)
    defer cleanTestData(t, testDB)

    // Your tests here
    req := HTTPTestRequest{
        Method:  "GET",
        Path:    "/api/v1/patterns/test-pattern-1",
        URLVars: map[string]string{"id": "test-pattern-1"},
    }

    w := makeHTTPRequest(getPatternHandler, req)
    response := assertJSONResponse(t, w, http.StatusOK)

    if id := response["id"].(string); id != "test-pattern-1" {
        t.Errorf("Expected id 'test-pattern-1', got %s", id)
    }
}
```

## Test Patterns

### Success Path Testing

Always test the happy path first:

```go
t.Run("Success_BasicOperation", func(t *testing.T) {
    // Setup
    req := HTTPTestRequest{
        Method: "GET",
        Path:   "/api/v1/endpoint",
    }

    // Execute
    w := makeHTTPRequest(handler, req)

    // Validate
    response := assertJSONResponse(t, w, http.StatusOK)

    // Assertions
    if _, ok := response["data"]; !ok {
        t.Error("Expected 'data' field")
    }
})
```

### Error Path Testing

Test all error conditions systematically:

```go
t.Run("Error_MissingParameter", func(t *testing.T) {
    req := HTTPTestRequest{
        Method: "GET",
        Path:   "/api/v1/endpoint",
        // Missing required query parameter
    }

    w := makeHTTPRequest(handler, req)
    assertErrorResponse(t, w, http.StatusBadRequest, "")
})

t.Run("Error_NotFound", func(t *testing.T) {
    req := HTTPTestRequest{
        Method:  "GET",
        Path:    "/api/v1/patterns/non-existent",
        URLVars: map[string]string{"id": "non-existent"},
    }

    w := makeHTTPRequest(handler, req)
    assertErrorResponse(t, w, http.StatusNotFound, "")
})
```

### Edge Case Testing

Test boundary conditions:

```go
t.Run("EdgeCase_NegativeOffset", func(t *testing.T) {
    req := HTTPTestRequest{
        Method: "GET",
        Path:   "/api/v1/patterns/search",
        Query:  map[string]string{"offset": "-5"},
    }

    w := makeHTTPRequest(handler, req)
    response := assertJSONResponse(t, w, http.StatusOK)

    // Should default to 0
    if offset := response["offset"].(float64); offset != 0 {
        t.Errorf("Expected offset 0, got %v", offset)
    }
})
```

## Common Test Scenarios

### Testing Pagination

```go
scenarios := PaginationTestScenarios("/api/v1/patterns/search")

for _, scenario := range scenarios {
    t.Run(scenario.Name, func(t *testing.T) {
        w := makeHTTPRequest(handler, scenario.Request)
        // Validate based on scenario
    })
}
```

### Testing JSON Marshaling

```go
func TestStructureMarshaling(t *testing.T) {
    obj := MyStruct{
        ID:   "test-1",
        Name: "Test Object",
    }

    data, err := json.Marshal(obj)
    if err != nil {
        t.Fatalf("Failed to marshal: %v", err)
    }

    var unmarshaled MyStruct
    if err := json.Unmarshal(data, &unmarshaled); err != nil {
        t.Fatalf("Failed to unmarshal: %v", err)
    }

    if unmarshaled.ID != obj.ID {
        t.Errorf("ID mismatch")
    }
}
```

### Testing with URL Variables (mux)

```go
req := HTTPTestRequest{
    Method:  "GET",
    Path:    "/api/v1/patterns/my-pattern-id",
    URLVars: map[string]string{"id": "my-pattern-id"},
}

w := makeHTTPRequest(getPatternHandler, req)
```

### Testing with Query Parameters

```go
req := HTTPTestRequest{
    Method: "GET",
    Path:   "/api/v1/patterns/search",
    Query: map[string]string{
        "query":  "test",
        "limit":  "10",
        "offset": "0",
    },
}

w := makeHTTPRequest(searchPatternsHandler, req)
```

### Testing POST with JSON Body

```go
requestBody := GenerationRequest{
    RecipeID: "recipe-1",
    Language: "go",
    Parameters: map[string]interface{}{
        "param1": "value1",
    },
}

req := HTTPTestRequest{
    Method: "POST",
    Path:   "/api/v1/recipes/generate",
    Body:   requestBody,
}

w := makeHTTPRequest(generateCodeHandler, req)
```

## Database Testing

### Environment Setup

Required environment variables for database tests:

```bash
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=postgres
export POSTGRES_DB=scalable_app_cookbook_test
```

### Auto-Skip Pattern

Tests automatically skip when database is unavailable:

```go
testDB := setupTestDatabase(t)
if testDB == nil {
    t.Skip("Database not available for testing")
}
defer testDB.Cleanup()
```

This ensures:
- ✅ CI environments without DB don't fail
- ✅ Local development works with or without DB
- ✅ No false positive test failures

### Seeding Test Data

```go
seedTestData(t, testDB)
defer cleanTestData(t, testDB)

// Test data includes:
// - Pattern: test-pattern-1
// - Recipe: test-recipe-1
// - Implementation: test-impl-1
```

## Best Practices

### 1. Always Use Helpers

❌ **Bad**:
```go
req := httptest.NewRequest("GET", "/health", nil)
w := httptest.NewRecorder()
healthHandler(w, req)
```

✅ **Good**:
```go
req := HTTPTestRequest{Method: "GET", Path: "/health"}
w := makeHTTPRequest(healthHandler, req)
```

### 2. Always Clean Up

❌ **Bad**:
```go
func TestSomething(t *testing.T) {
    testDB := setupTestDatabase(t)
    // Forgot to cleanup!
}
```

✅ **Good**:
```go
func TestSomething(t *testing.T) {
    testDB := setupTestDatabase(t)
    if testDB == nil {
        t.Skip("Database not available")
    }
    defer testDB.Cleanup()
}
```

### 3. Use Descriptive Test Names

❌ **Bad**:
```go
t.Run("Test1", func(t *testing.T) {})
```

✅ **Good**:
```go
t.Run("Success_SearchPatternsWithChapterFilter", func(t *testing.T) {})
```

### 4. Use t.Helper() for Helper Functions

```go
func myTestHelper(t *testing.T, value string) {
    t.Helper() // Important!
    if value == "" {
        t.Error("Value should not be empty")
    }
}
```

### 5. Test Both Status Code AND Body

❌ **Bad**:
```go
if w.Code != http.StatusOK {
    t.Error("Wrong status code")
}
// Didn't check response body!
```

✅ **Good**:
```go
response := assertJSONResponse(t, w, http.StatusOK)
if _, ok := response["data"]; !ok {
    t.Error("Expected 'data' field")
}
```

## Coverage Goals

- **Target**: 80% overall coverage
- **Minimum**: 50% (will error below this)
- **Warning**: Below 80%

### Checking Coverage

```bash
# Generate coverage report
go test -tags=testing -coverprofile=coverage.out -covermode=atomic

# View coverage by function
go tool cover -func=coverage.out

# View coverage in browser
go tool cover -html=coverage.out
```

## Troubleshooting

### Tests Skipping Due to Database

**Problem**: All database tests are skipped

**Solution**: Set up PostgreSQL and environment variables:
```bash
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=postgres
export POSTGRES_DB=scalable_app_cookbook_test
```

### Build Tag Required

**Problem**: `go test` doesn't find any tests

**Solution**: Use the testing build tag:
```bash
go test -tags=testing
```

### Nil Pointer Errors

**Problem**: `panic: runtime error: invalid memory address`

**Solution**: Always check for nil before use:
```go
testDB := setupTestDatabase(t)
if testDB == nil {
    t.Skip("Database not available")
}
// Now safe to use testDB
```

## Performance Testing

Run performance tests separately:

```bash
# Requires running scenario
cd scenarios/scalable-app-cookbook
make run

# In another terminal
./test/phases/test-performance.sh
```

Performance thresholds:
- Health check: < 1s
- Pattern search: < 2s
- Statistics: < 3s

## Integration Testing

Run integration tests:

```bash
# Start scenario first
make run

# Run integration tests
./test/phases/test-integration.sh
```

## Next Steps

1. **Run tests with database** to achieve full coverage
2. **Add benchmark tests** for performance analysis
3. **Add fuzz tests** for robustness
4. **Add mock database** for testing DB errors

## Resources

- Gold standard: `/scenarios/visited-tracker/api/`
- Testing guide: `/docs/testing/guides/scenario-unit-testing.md`
- Centralized testing: `/scripts/scenarios/testing/`
