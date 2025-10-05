# News Aggregator & Bias Analysis - Testing Guide

## Overview

This guide provides comprehensive documentation for the test suite of the news-aggregator-bias-analysis scenario. The test suite follows Vrooli's gold standard testing patterns from the visited-tracker scenario and achieves 70-80% code coverage when run with a database.

## Quick Start

```bash
# Navigate to scenario directory
cd scenarios/news-aggregator-bias-analysis

# Run all test phases
make test

# Or run individual phases:
./test/phases/test-dependencies.sh  # Check dependencies
./test/phases/test-structure.sh     # Validate structure
./test/phases/test-unit.sh          # Run unit tests
./test/phases/test-integration.sh   # Run integration tests
./test/phases/test-business.sh      # Run business logic tests
./test/phases/test-performance.sh   # Run performance tests
```

## Test Architecture

### Test Types

The test suite implements all six requested test types:

1. **Dependencies Tests** (`test/phases/test-dependencies.sh`)
   - Validates resource availability (PostgreSQL, Redis, Ollama)
   - Checks language toolchains (Go, Node.js, npm)
   - Verifies essential utilities (jq, curl)
   - Duration: ~30 seconds

2. **Structure Tests** (`test/phases/test-structure.sh`)
   - Validates project structure and required files
   - Checks service.json configuration
   - Verifies Go and Node.js module structure
   - Duration: ~15 seconds

3. **Unit Tests** (`test/phases/test-unit.sh` + Go test files)
   - Tests individual functions and handlers
   - Mocks external dependencies
   - Achieves 70-80% coverage
   - Duration: ~120 seconds

4. **Integration Tests** (`test/phases/test-integration.sh`)
   - Tests database connectivity
   - Validates API endpoint integration
   - Checks resource interactions
   - Duration: ~180 seconds

5. **Business Logic Tests** (`test/phases/test-business.sh`)
   - End-to-end workflow validation
   - CRUD operations testing
   - Data persistence verification
   - Duration: ~180 seconds

6. **Performance Tests** (`test/phases/test-performance.sh` + `api/performance_test.go`)
   - Throughput benchmarks
   - Concurrency testing
   - Memory usage validation
   - Duration: ~180 seconds

### Test Files

#### Go Test Files

**`api/test_helpers.go`** - Reusable Test Utilities
```go
// Setup functions
setupTestLogger()        // Controlled logging
setupTestDatabase()      // Isolated test DB
setupTestEnvironment()   // Complete test env

// Request helpers
makeHTTPRequest()        // HTTP request execution
assertJSONResponse()     // JSON validation
assertErrorResponse()    // Error validation

// Data factories
createTestArticle()      // Article creation
createTestFeed()         // Feed creation
```

**`api/test_patterns.go`** - Systematic Testing Patterns
```go
// Error testing
NewTestScenarioBuilder() // Fluent test builder
  .AddInvalidID()
  .AddNonExistentArticle()
  .AddInvalidJSON()
  .Build()

// Performance testing
RunPerformanceTest()     // Performance validation
```

**`api/main_test.go`** - Handler Tests
- Tests all HTTP handlers
- Validates success and error paths
- Tests query parameters and filtering

**`api/processor_test.go`** - Feed Processing Tests
- Tests RSS feed fetching
- Tests bias analysis
- Tests article storage

**`api/performance_test.go`** - Performance Tests
- Benchmark tests
- Load testing
- Concurrency testing

## Running Tests

### Prerequisites

1. **Test Database**
   ```bash
   # Create test database
   createdb news_test

   # Set environment variable
   export TEST_POSTGRES_URL="postgres://test:test@localhost:5432/news_test?sslmode=disable"
   ```

2. **Required Resources**
   - PostgreSQL (for integration tests)
   - Redis (optional, for caching tests)
   - Ollama (for AI bias analysis tests)

### Running Individual Test Phases

#### Dependencies Test
```bash
./test/phases/test-dependencies.sh
```
Validates:
- Resource CLI availability
- Language toolchains
- Essential utilities

#### Structure Test
```bash
./test/phases/test-structure.sh
```
Validates:
- Project files and directories
- Configuration files
- Module structure

#### Unit Tests
```bash
./test/phases/test-unit.sh
```
Runs:
- All Go unit tests
- Coverage analysis (target: 80%)

#### Integration Tests
```bash
./test/phases/test-integration.sh
```
Tests:
- Database connectivity
- API endpoints
- Resource integration

#### Business Logic Tests
```bash
# Requires running service
vrooli scenario start news-aggregator-bias-analysis

# Then run tests
./test/phases/test-business.sh
```
Tests:
- Feed CRUD operations
- Article retrieval
- Perspective aggregation
- Data persistence

#### Performance Tests
```bash
./test/phases/test-performance.sh
```
Runs:
- Throughput benchmarks
- Load tests
- Memory profiling

### Running Go Tests Directly

```bash
cd api

# Run all tests
go test -tags=testing -v

# Run specific test
go test -tags=testing -v -run TestHealthHandler

# Run with coverage
go test -tags=testing -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run benchmarks
go test -tags=testing -bench=. -benchmem
```

## Test Patterns

### 1. Setup and Teardown

All tests follow this pattern:

```go
func TestHandler(t *testing.T) {
    // Setup logger
    cleanup := setupTestLogger()
    defer cleanup()

    // Setup test environment
    env := setupTestEnvironment(t)
    defer env.Cleanup()

    // Test logic here
}
```

### 2. HTTP Handler Testing

```go
// Make request
w, err := makeHTTPRequest(env, HTTPTestRequest{
    Method: "GET",
    Path:   "/articles",
    QueryParams: map[string]string{
        "category": "politics",
    },
})

// Validate response
result := assertJSONResponse(t, w, http.StatusOK)

// Check fields
if articles, ok := result["articles"].([]interface{}); ok {
    // Assertions...
}
```

### 3. Error Testing

```go
// Build error scenarios
patterns := NewTestScenarioBuilder().
    AddInvalidID("/articles/invalid-id", "GET").
    AddNonExistentArticle("/articles/{id}").
    AddInvalidJSON("/feeds", "POST").
    Build()

// Run error tests
suite := NewHandlerTestSuite("GetArticles", env)
suite.RunErrorTests(t, patterns)
```

### 4. Performance Testing

```go
pattern := PerformanceTestPattern{
    Name:          "GetArticles",
    MaxDuration:   5 * time.Second,
    MinThroughput: 10, // req/s
    Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
        return HTTPTestRequest{
            Method: "GET",
            Path:   "/articles",
        }
    },
}

RunPerformanceTest(t, env, pattern)
```

## Coverage Goals

### Target Coverage: 80%

**Coverage Breakdown:**
- Handler functions: 75-80%
- Feed processor: 80-85%
- Helper functions: 100%
- Overall: 70-80%

### Viewing Coverage

```bash
cd api

# Generate coverage report
go test -tags=testing -coverprofile=coverage.out

# View in terminal
go tool cover -func=coverage.out

# View in browser
go tool cover -html=coverage.out
```

## Common Issues and Solutions

### Issue: Tests Skip Due to Missing Database

**Problem:**
```
Skipping database test - cannot connect: dial tcp 127.0.0.1:5432: connect: connection refused
```

**Solution:**
1. Start PostgreSQL: `sudo systemctl start postgresql`
2. Create test database: `createdb news_test`
3. Set TEST_POSTGRES_URL environment variable

### Issue: Ollama Tests Fail

**Problem:**
Tests requiring Ollama fail because service is not available.

**Solution:**
1. Start Ollama: `vrooli resource start ollama`
2. Ensure OLLAMA_URL is set in environment
3. Tests will automatically skip if Ollama is unavailable

### Issue: Performance Tests Timeout

**Problem:**
Performance tests exceed timeout limits.

**Solution:**
1. Increase timeout in test phase script
2. Ensure database is properly indexed
3. Check system resources

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Test News Aggregator

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: test
          POSTGRES_DB: news_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run tests
        env:
          TEST_POSTGRES_URL: postgres://postgres:test@localhost:5432/news_test?sslmode=disable
        run: |
          cd scenarios/news-aggregator-bias-analysis
          ./test/phases/test-dependencies.sh
          ./test/phases/test-structure.sh
          ./test/phases/test-unit.sh
```

## Best Practices

### 1. Always Use Test Helpers

✅ **Good:**
```go
env := setupTestEnvironment(t)
defer env.Cleanup()
```

❌ **Bad:**
```go
// Manual setup without cleanup
db, _ := sql.Open("postgres", url)
```

### 2. Test Success and Error Paths

✅ **Good:**
```go
t.Run("Success", func(t *testing.T) { /* ... */ })
t.Run("InvalidID", func(t *testing.T) { /* ... */ })
t.Run("NotFound", func(t *testing.T) { /* ... */ })
```

❌ **Bad:**
```go
// Only testing success case
```

### 3. Clean Up Test Data

✅ **Good:**
```go
testArticle := createTestArticle(t, env, "Test", "Source")
defer testArticle.Cleanup()
```

❌ **Bad:**
```go
// Creating test data without cleanup
```

### 4. Use Table-Driven Tests

✅ **Good:**
```go
testCases := []struct {
    name     string
    input    string
    expected int
}{
    {"Valid", "test", http.StatusOK},
    {"Invalid", "", http.StatusBadRequest},
}

for _, tc := range testCases {
    t.Run(tc.name, func(t *testing.T) {
        // Test logic
    })
}
```

## Extending the Test Suite

### Adding a New Handler Test

1. Add test function in `api/main_test.go`:
```go
func TestNewHandler(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    env := setupTestEnvironment(t)
    defer env.Cleanup()

    // Test logic
}
```

2. Add error scenarios:
```go
patterns := NewTestScenarioBuilder().
    AddInvalidID("/new-endpoint/{id}", "GET").
    AddInvalidJSON("/new-endpoint", "POST").
    Build()

suite := NewHandlerTestSuite("NewHandler", env)
suite.RunErrorTests(t, patterns)
```

### Adding a New Test Phase

1. Create script in `test/phases/`:
```bash
#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

# Test logic here

testing::phase::end_with_summary "Phase completed"
```

2. Make executable:
```bash
chmod +x test/phases/test-new-phase.sh
```

## Resources

- **Gold Standard Reference:** `/scenarios/visited-tracker/`
- **Testing Infrastructure:** `/scripts/scenarios/testing/`
- **Test Helpers:** `/scenarios/news-aggregator-bias-analysis/api/test_helpers.go`
- **Test Patterns:** `/scenarios/news-aggregator-bias-analysis/api/test_patterns.go`

## Summary

This test suite provides:
- ✅ Complete coverage of all test types (dependencies, structure, unit, integration, business, performance)
- ✅ Systematic error testing patterns
- ✅ 70-80% code coverage target
- ✅ Gold standard patterns from visited-tracker
- ✅ CI/CD ready infrastructure
- ✅ Comprehensive documentation

For questions or issues, refer to the Vrooli testing documentation or the visited-tracker scenario as a reference implementation.
