# Testing Guide - typing-test API

## Overview

This guide explains the testing infrastructure for the typing-test scenario, following Vrooli's gold standard testing patterns.

## Test Files Structure

```
api/
├── test_helpers.go         # Reusable test utilities
├── test_patterns.go        # Systematic error patterns
├── main_test.go            # HTTP handler tests
├── typing_processor_test.go # Business logic tests
└── TESTING_GUIDE.md        # This file
```

## Quick Start

### Running Tests

```bash
# Run all tests
go test -v -tags=testing ./...

# Run with coverage
go test -v -tags=testing -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out

# Run specific test
go test -v -tags=testing -run TestGetPracticeText

# Run benchmarks
go test -v -tags=testing -bench=. -benchmem
```

### With Database

```bash
# Set test database URL
export TEST_POSTGRES_URL="postgres://user:pass@localhost:5432/test_db?sslmode=disable"

# Run full test suite
go test -v -tags=testing -coverprofile=coverage.out ./...
```

## Test Helper Library

### setupTestLogger()

Suppress logs during testing for cleaner output:

```go
func TestMyHandler(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    // Your test code
}
```

### setupTestDatabase(t)

Create isolated test database connection:

```go
func TestWithDatabase(t *testing.T) {
    testDB := setupTestDatabase(t)
    if testDB == nil {
        t.Skip("Database not available")
    }
    defer testDB.Cleanup()

    // Use testDB.DB for database operations
}
```

**Features**:
- Automatic skip if `TEST_POSTGRES_URL` not set
- Connection pooling configured for tests
- Automatic schema setup
- Cleanup with truncate on defer

### makeHTTPRequest(t, router, req)

Simplified HTTP request testing:

```go
w := makeHTTPRequest(t, router, HTTPTestRequest{
    Method: "GET",
    Path:   "/api/endpoint",
    Body:   requestData,
})

var response ResponseType
assertJSONResponse(t, w, http.StatusOK, &response)
```

**Supports**:
- All HTTP methods
- JSON body marshaling
- Custom headers
- URL variables (for mux)

### Assertion Helpers

#### assertJSONResponse(t, w, status, target)

Validate JSON responses:

```go
var response MyResponse
assertJSONResponse(t, w, http.StatusOK, &response)

// Validates:
// - Status code matches
// - Content-Type is application/json
// - Response can be decoded to target
```

#### assertErrorResponse(t, w, status)

Validate error responses:

```go
assertErrorResponse(t, w, http.StatusBadRequest)
```

#### assertValidCoachingResponse(t, response)

Validate coaching response structure:

```go
response := processor.ProvideCoaching(ctx, stats, "medium")
assertValidCoachingResponse(t, response)

// Validates:
// - Feedback is not empty
// - Tips array has content
// - EncouragingNote present
// - NextChallenge present
```

#### assertValidProcessedStats(t, stats)

Validate processed stats structure:

```go
result, err := processor.ProcessStats(ctx, stats)
assertValidProcessedStats(t, result)

// Validates:
// - SessionID present
// - Timestamp present
// - Metrics within valid ranges
// - Analysis has performance level
```

#### assertValidAdaptiveResponse(t, response)

Validate adaptive text response:

```go
response := processor.GenerateAdaptiveText(ctx, request)
assertValidAdaptiveResponse(t, response)

// Validates:
// - Text is not empty
// - WordCount is positive
// - Difficulty is set
// - IsAdaptive flag is true
// - Timestamp is present
```

### Test Data Generators

#### createTestScore(name, wpm, accuracy)

Generate test score:

```go
score := createTestScore("TestPlayer", 60, 90)
// Returns Score with calculated score field
```

#### createTestStats(sessionID, wpm, accuracy)

Generate session stats:

```go
stats := createTestStats(uuid.New().String(), 50, 88.0)
// Returns SessionStats with all fields populated
```

#### createTestAdaptiveRequest(userID, difficulty)

Generate adaptive text request:

```go
request := createTestAdaptiveRequest(uuid.New().String(), "medium")
// Returns AdaptiveTextRequest with defaults
```

## Test Pattern Library

### TestScenarioBuilder

Fluent interface for building error test scenarios:

```go
scenarios := NewTestScenarioBuilder().
    AddInvalidJSON("POST", "/api/endpoint").
    AddEmptyBody("POST", "/api/endpoint").
    Build()

for _, scenario := range scenarios {
    t.Run(scenario.Name, func(t *testing.T) {
        w := makeHTTPRequest(t, router, scenario.Request)
        assertErrorResponse(t, w, scenario.ExpectedStatus)
    })
}
```

**Available Builders**:
- `AddInvalidJSON(method, path)` - Malformed JSON
- `AddEmptyBody(method, path)` - Empty request body
- `AddMissingRequiredField(method, path, field, body)` - Missing field
- `AddInvalidQueryParam(path, param, value)` - Invalid parameter

### Edge Case Generators

#### createEdgeCaseScore(caseType)

Generate edge case scores:

```go
cases := []string{"minimal", "maximum", "empty_name", "long_name"}
for _, caseType := range cases {
    score := createEdgeCaseScore(caseType)
    // Test with edge case score
}
```

#### createEdgeCaseAdaptiveRequest(caseType)

Generate edge case adaptive requests:

```go
cases := []string{"no_target_words", "many_target_words", "many_mistakes", "empty_strings"}
for _, caseType := range cases {
    request := createEdgeCaseAdaptiveRequest(caseType)
    // Test with edge case request
}
```

### Bulk Data Generators

#### generateTestScores(count)

Generate multiple test scores:

```go
scores := generateTestScores(10)
for _, score := range scores {
    insertTestScore(t, testDB.DB, score)
}
```

#### generateTestSessions(count)

Generate multiple test sessions:

```go
sessions := generateTestSessions(5)
// Each session has varied WPM, accuracy, difficulty
```

### Boundary Value Testing

#### generateWPMBoundaryTests()

WPM boundary test cases:

```go
cases := generateWPMBoundaryTests()
// Returns: ZeroWPM, NegativeWPM, MaxWPM, ExtremeWPM
```

#### generateAccuracyBoundaryTests()

Accuracy boundary test cases:

```go
cases := generateAccuracyBoundaryTests()
// Returns: ZeroAccuracy, NegativeAccuracy, PerfectAccuracy, OverAccuracy
```

## Writing Tests

### Basic Handler Test

```go
func TestMyHandler(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    router := setupTestRouter()

    t.Run("SuccessCase", func(t *testing.T) {
        w := makeHTTPRequest(t, router, HTTPTestRequest{
            Method: "GET",
            Path:   "/api/my-endpoint",
        })

        var response MyResponse
        assertJSONResponse(t, w, http.StatusOK, &response)

        // Additional assertions
        if response.Field != "expected" {
            t.Errorf("Expected 'expected', got '%s'", response.Field)
        }
    })

    t.Run("ErrorPaths", func(t *testing.T) {
        scenarios := NewTestScenarioBuilder().
            AddInvalidJSON("POST", "/api/my-endpoint").
            Build()

        for _, scenario := range scenarios {
            t.Run(scenario.Name, func(t *testing.T) {
                w := makeHTTPRequest(t, router, scenario.Request)
                assertErrorResponse(t, w, scenario.ExpectedStatus)
            })
        }
    })
}
```

### Database Handler Test

```go
func TestDatabaseHandler(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    testDB := setupTestDatabase(t)
    if testDB == nil {
        t.Skip("Database not available")
    }
    defer testDB.Cleanup()

    // Set global db for handlers
    db = testDB.DB
    typingProcessor = NewTypingProcessor(testDB.DB)

    router := setupTestRouter()

    t.Run("WithData", func(t *testing.T) {
        // Insert test data
        scores := generateTestScores(5)
        for _, score := range scores {
            insertTestScore(t, testDB.DB, score)
        }

        // Test handler
        w := makeHTTPRequest(t, router, HTTPTestRequest{
            Method: "GET",
            Path:   "/api/leaderboard",
        })

        var response LeaderboardResponse
        assertJSONResponse(t, w, http.StatusOK, &response)

        if len(response.Scores) == 0 {
            t.Error("Expected scores in leaderboard")
        }
    })
}
```

### Business Logic Test

```go
func TestBusinessLogic(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    processor := &TypingProcessor{}
    ctx := context.Background()

    t.Run("DifferentInputs", func(t *testing.T) {
        cases := []struct {
            name     string
            input    InputType
            expected ExpectedType
        }{
            {"Case1", input1, expected1},
            {"Case2", input2, expected2},
        }

        for _, tc := range cases {
            t.Run(tc.name, func(t *testing.T) {
                result := processor.Method(ctx, tc.input)

                if result != tc.expected {
                    t.Errorf("Expected %v, got %v", tc.expected, result)
                }
            })
        }
    })
}
```

### Integration Test

```go
func TestFullUserFlow(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    testDB := setupTestDatabase(t)
    if testDB == nil {
        t.Skip("Database not available for integration test")
    }
    defer testDB.Cleanup()

    db = testDB.DB
    typingProcessor = NewTypingProcessor(testDB.DB)
    router := setupTestRouter()

    sessionID := uuid.New().String()

    // Step 1: Get practice text
    t.Run("1_GetPracticeText", func(t *testing.T) {
        w := makeHTTPRequest(t, router, HTTPTestRequest{
            Method: "GET",
            Path:   "/api/practice-text?difficulty=easy",
        })

        var response PracticeTextResponse
        assertJSONResponse(t, w, http.StatusOK, &response)
    })

    // Step 2: Submit stats
    t.Run("2_SubmitStats", func(t *testing.T) {
        stats := StatsRequest{
            SessionID: sessionID,
            WPM:       45,
            Accuracy:  88,
            Text:      "Practice text",
        }

        w := makeHTTPRequest(t, router, HTTPTestRequest{
            Method: "POST",
            Path:   "/api/stats",
            Body:   stats,
        })

        var response ProcessedStats
        assertJSONResponse(t, w, http.StatusOK, &response)
    })

    // Continue with other steps...
}
```

## Best Practices

### 1. Test Isolation

Each test should be independent:

```go
✅ Good:
func TestHandler(t *testing.T) {
    testDB := setupTestDatabase(t)
    defer testDB.Cleanup() // Cleanup after test

    // Test code
}

❌ Bad:
func TestHandler(t *testing.T) {
    // Relies on data from previous test
}
```

### 2. Proper Cleanup

Always use defer for cleanup:

```go
✅ Good:
cleanup := setupTestLogger()
defer cleanup()

testDB := setupTestDatabase(t)
defer testDB.Cleanup()

❌ Bad:
cleanup := setupTestLogger()
// Forgot defer - cleanup may not run
```

### 3. Clear Test Names

Use descriptive test names:

```go
✅ Good:
t.Run("ValidScore", func(t *testing.T) { ... })
t.Run("InvalidJSON", func(t *testing.T) { ... })
t.Run("EdgeCase_EmptyName", func(t *testing.T) { ... })

❌ Bad:
t.Run("Test1", func(t *testing.T) { ... })
t.Run("Test2", func(t *testing.T) { ... })
```

### 4. Helpful Error Messages

Provide context in assertions:

```go
✅ Good:
if result.WPM != expected {
    t.Errorf("WPM mismatch: expected %d, got %d", expected, result.WPM)
}

❌ Bad:
if result.WPM != expected {
    t.Error("wrong WPM")
}
```

### 5. Test Coverage

Aim for comprehensive coverage:

- ✅ Success paths
- ✅ Error paths
- ✅ Edge cases
- ✅ Boundary values
- ✅ Integration flows

## Performance Testing

### Benchmark Tests

```go
func BenchmarkMyHandler(b *testing.B) {
    router := setupTestRouter()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        w := httptest.NewRecorder()
        req := httptest.NewRequest("GET", "/api/endpoint", nil)
        router.ServeHTTP(w, req)
    }
}
```

### Running Benchmarks

```bash
# Run all benchmarks
go test -v -tags=testing -bench=. -benchmem

# Run specific benchmark
go test -v -tags=testing -bench=BenchmarkHealthHandler -benchmem

# Compare benchmarks
go test -bench=. -benchmem > old.txt
# Make changes
go test -bench=. -benchmem > new.txt
benchcmp old.txt new.txt
```

## Coverage Goals

- **Overall**: 70-85% (with database)
- **Business Logic**: 100%
- **HTTP Handlers**: 80%+
- **Helper Functions**: 60%+

## Troubleshooting

### Tests Skip Due to Database

**Problem**: `TEST_POSTGRES_URL not set, skipping database tests`

**Solution**:
```bash
export TEST_POSTGRES_URL="postgres://user:pass@localhost:5432/test_db?sslmode=disable"
```

### Build Tag Issues

**Problem**: `no buildable Go source files`

**Solution**: Always use `-tags=testing` flag:
```bash
go test -v -tags=testing ./...
```

### Coverage Too Low

**Problem**: Coverage at 33.5%

**Cause**: Database tests are skipped

**Solution**: Set `TEST_POSTGRES_URL` or implement database mocking

## CI/CD Integration

### GitHub Actions Example

```yaml
- name: Run Tests
  env:
    TEST_POSTGRES_URL: postgres://postgres:postgres@localhost:5432/test?sslmode=disable
  run: |
    cd scenarios/typing-test/api
    go test -v -tags=testing -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out
```

## Additional Resources

- **Gold Standard**: `/scenarios/visited-tracker/` - Reference implementation
- **Test Patterns**: `/docs/testing/guides/scenario-unit-testing.md`
- **Centralized Testing**: `/scripts/scenarios/testing/`

## Summary

This testing infrastructure provides:

1. ✅ **Comprehensive Coverage** - All critical paths tested
2. ✅ **Reusable Components** - Helper library and patterns
3. ✅ **Systematic Approach** - Builder patterns for test scenarios
4. ✅ **Database Ready** - Full database test support
5. ✅ **Performance Monitoring** - Benchmark tests included
6. ✅ **CI/CD Integration** - Ready for automated testing

Follow these patterns to maintain high-quality, maintainable tests.
