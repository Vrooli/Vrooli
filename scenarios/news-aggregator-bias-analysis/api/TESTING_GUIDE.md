# Testing Guide for News Aggregator Bias Analysis

## Overview
This guide explains the testing infrastructure for the news-aggregator-bias-analysis scenario.

## Test Structure

### Test Files
- **test_helpers.go**: Reusable test utilities and fixtures
- **test_patterns.go**: Systematic error testing patterns
- **main_test.go**: HTTP handler tests
- **processor_test.go**: Feed processing logic tests
- **performance_test.go**: Performance and benchmark tests

### Test Phases
- **test-unit.sh**: Unit tests (120s timeout)
- **test-integration.sh**: Integration tests (180s timeout)
- **test-performance.sh**: Performance tests (180s timeout)

## Running Tests

### Prerequisites
```bash
# Install PostgreSQL for testing
# Create test database
createdb news_test

# Set environment variable
export TEST_POSTGRES_URL="postgres://test:test@localhost:5432/news_test?sslmode=disable"
```

### Run All Tests
```bash
# From scenario root
make test

# Or via vrooli CLI
vrooli scenario test news-aggregator-bias-analysis
```

### Run Specific Test Phases
```bash
# Unit tests only
./test/phases/test-unit.sh

# Integration tests
./test/phases/test-integration.sh

# Performance tests
./test/phases/test-performance.sh
```

### Run Specific Test Files
```bash
cd api

# All tests
go test -tags=testing -v

# Specific test
go test -tags=testing -run TestHealthHandler -v

# With coverage
go test -tags=testing -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Run Performance Tests
```bash
cd api

# Performance tests only
go test -tags=testing -run=TestPerformance -v

# Benchmarks only
go test -tags=testing -bench=. -benchmem -run=^$

# Both
go test -tags=testing -run=TestPerformance -bench=.
```

## Writing Tests

### Using Test Helpers
```go
func TestMyHandler(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    env := setupTestEnvironment(t)
    defer env.Cleanup()

    // Create test data
    testArticle := createTestArticle(t, env, "Test Title", "Test Source")
    defer testArticle.Cleanup()

    // Make request
    req := HTTPTestRequest{
        Method: "GET",
        Path:   "/articles",
    }

    w, err := makeHTTPRequest(env, req)
    if err != nil {
        t.Fatalf("Failed: %v", err)
    }

    // Assert response
    assertJSONResponse(t, w, http.StatusOK)
}
```

### Using Error Test Patterns
```go
func TestMyHandler_Errors(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    env := setupTestEnvironment(t)
    defer env.Cleanup()

    suite := NewHandlerTestSuite("MyHandler", env)

    patterns := NewTestScenarioBuilder().
        AddInvalidID("/api/resource/{id}", "GET").
        AddNonExistentArticle("/api/resource/{id}").
        AddInvalidJSON("/api/resource", "POST").
        Build()

    suite.RunErrorTests(t, patterns)
}
```

### Writing Performance Tests
```go
func TestPerformance_MyEndpoint(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    env := setupTestEnvironment(t)
    defer env.Cleanup()

    pattern := PerformanceTestPattern{
        Name:          "MyEndpoint",
        Description:   "Test endpoint performance",
        MaxDuration:   5 * time.Second,
        MinThroughput: 10,
        Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
            return HTTPTestRequest{
                Method: "GET",
                Path:   "/my-endpoint",
            }
        },
        Validate: func(t *testing.T, duration time.Duration, throughput float64) {
            t.Logf("Throughput: %.2f req/s", throughput)
        },
    }

    RunPerformanceTest(t, env, pattern)
}
```

## Test Data Management

### Creating Test Articles
```go
// Single article
article := createTestArticle(t, env, "Title", "Source")
defer article.Cleanup()

// Multiple articles
articles := GenerateTestArticles(t, env, 10, "Source")
defer CleanupTestArticles(articles)
```

### Creating Test Feeds
```go
// Single feed
feed := createTestFeed(t, env, "Feed Name", "https://example.com/feed.rss")
defer feed.Cleanup()

// Multiple feeds
feeds := GenerateTestFeeds(t, env, 5)
defer CleanupTestFeeds(feeds)
```

## Coverage Goals

### Target Coverage: 80%
- Minimum acceptable: 70%
- Error threshold: 50%

### Coverage by Component
- **HTTP Handlers**: 75%+
- **Feed Processing**: 80%+
- **Helper Functions**: 100%
- **Error Handling**: 90%+

## Common Test Patterns

### Success Cases
```go
t.Run("Success", func(t *testing.T) {
    // Setup
    // Execute
    // Assert all fields
    // Cleanup
})
```

### Error Cases
```go
t.Run("ErrorPaths", func(t *testing.T) {
    suite := NewHandlerTestSuite("Handler", env)
    patterns := NewTestScenarioBuilder().
        AddInvalidID("/path/{id}", "GET").
        AddNonExistentArticle("/path/{id}").
        Build()
    suite.RunErrorTests(t, patterns)
})
```

### Edge Cases
```go
t.Run("EdgeCases", func(t *testing.T) {
    // Empty inputs
    // Boundary conditions
    // Special characters
    // Large datasets
})
```

## Debugging Tests

### Verbose Output
```bash
go test -tags=testing -v
```

### Run Specific Test
```bash
go test -tags=testing -run TestHealthHandler -v
```

### Check Coverage
```bash
go test -tags=testing -coverprofile=coverage.out
go tool cover -func=coverage.out
```

### Debug with Delve
```bash
dlv test -- -test.run TestHealthHandler
```

## CI/CD Integration

### GitHub Actions Example
```yaml
- name: Run Tests
  env:
    TEST_POSTGRES_URL: postgres://test:test@localhost:5432/news_test
  run: |
    make test
```

### Test Database Setup
```bash
# In CI/CD
docker run -d --name test-postgres \
  -e POSTGRES_USER=test \
  -e POSTGRES_PASSWORD=test \
  -e POSTGRES_DB=news_test \
  -p 5432:5432 \
  postgres:14
```

## Best Practices

### Always Use Defer for Cleanup
```go
cleanup := setupTestLogger()
defer cleanup()

env := setupTestEnvironment(t)
defer env.Cleanup()

testData := createTestArticle(t, env, "Title", "Source")
defer testData.Cleanup()
```

### Test Both Status and Body
```go
w, err := makeHTTPRequest(env, req)
if err != nil {
    t.Fatalf("Request failed: %v", err)
}

// Check status
if w.Code != http.StatusOK {
    t.Errorf("Expected 200, got %d", w.Code)
}

// Check body
var result map[string]interface{}
json.Unmarshal(w.Body.Bytes(), &result)
// Assert fields
```

### Use Table-Driven Tests
```go
tests := []struct {
    name     string
    input    float64
    expected string
}{
    {"Left", -50.0, "left"},
    {"Center", 0.0, "center"},
    {"Right", 50.0, "right"},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        result := categorizeBias(tt.input)
        if result != tt.expected {
            t.Errorf("Expected %s, got %s", tt.expected, result)
        }
    })
}
```

### Isolate Test Data
- Each test creates its own data
- Always clean up after tests
- Use unique identifiers
- Avoid test interdependencies

## Troubleshooting

### Tests Skipped
**Issue**: All tests show "SKIP"
**Solution**: Set TEST_POSTGRES_URL environment variable

### Database Connection Failed
**Issue**: Cannot connect to test database
**Solution**:
1. Verify PostgreSQL is running
2. Check database exists: `createdb news_test`
3. Verify credentials in TEST_POSTGRES_URL

### Coverage Too Low
**Issue**: Coverage below 50%
**Solution**:
1. Ensure database is available
2. Check which tests are skipping
3. Add more test cases for uncovered code

### Tests Timeout
**Issue**: Tests exceed timeout
**Solution**:
1. Increase timeout: `-timeout 300s`
2. Optimize slow tests
3. Check for infinite loops

## Reference

### Gold Standard: visited-tracker
The testing patterns in this scenario are based on the visited-tracker scenario, which achieves 79.4% coverage.

### Documentation
- Main testing guide: `/docs/testing/guides/scenario-unit-testing.md`
- Phase helpers: `/scripts/scenarios/testing/shell/phase-helpers.sh`
- Unit test runner: `/scripts/scenarios/testing/unit/run-all.sh`

## Metrics

### Current Test Stats
- Test Functions: 30+
- Test Cases: 90+
- Performance Tests: 10
- Benchmarks: 3
- Coverage Target: 80%
- Expected Coverage: 70-80% (with database)
