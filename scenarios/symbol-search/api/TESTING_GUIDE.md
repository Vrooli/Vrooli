# Symbol Search API Testing Guide

## Overview

This guide documents the comprehensive testing approach for the symbol-search scenario, following Vrooli's gold standard testing patterns from visited-tracker.

## Test Coverage Goals

- **Target**: 80% code coverage
- **Minimum**: 50% code coverage (error threshold)
- **Current**: Tests cover all major API endpoints and utility functions

## Test Architecture

### Test Files

1. **test_helpers.go** - Reusable test utilities
   - Database setup and teardown
   - HTTP request helpers
   - Response assertion helpers
   - Test data seeding functions

2. **test_patterns.go** - Systematic error testing patterns
   - TestScenarioBuilder - Fluent interface for error scenarios
   - ErrorTestScenario - Structured error test cases
   - HandlerTestSuite - Comprehensive handler testing
   - CommonEdgeCases - Reusable edge case patterns

3. **main_test.go** - Comprehensive unit tests
   - Health check tests
   - Search endpoint tests (with filters, pagination, edge cases)
   - Character detail tests (Unicode/decimal formats)
   - Categories and blocks listing tests
   - Bulk range operation tests
   - Helper function tests
   - Performance tests

## Test Helpers

### Database Setup

```go
db, cleanup := setupTestDB(t)
defer cleanup()
```

Creates a test database connection using `POSTGRES_TEST_URL` or `POSTGRES_URL` environment variable.

### API Setup

```go
router, api, cleanup := setupTestRouter(t)
defer cleanup()
```

Creates a fully configured test router with all API endpoints.

### Making Requests

```go
req := HTTPTestRequest{
    Method: "GET",
    Path:   "/api/search",
    QueryParams: map[string]string{
        "q": "LATIN",
        "limit": "10",
    },
}
w := makeHTTPRequest(router, req)
```

### Response Assertions

```go
// JSON response with field validation
response := assertJSONResponse(t, w, http.StatusOK, "status", "database")

// Search response validation
searchResp := assertSearchResponse(t, w, http.StatusOK)

// Character detail validation
charResp := assertCharacterResponse(t, w, http.StatusOK)

// Error response validation
assertErrorResponse(t, w, http.StatusBadRequest)
```

## Test Patterns

### Error Testing with Builder Pattern

```go
scenarios := NewTestScenarioBuilder().
    AddInvalidCodepoint("/api/character/invalid").
    AddNonExistentCharacter("/api/character/U+FFFFFF").
    AddInvalidRange("/api/bulk/range").
    AddTooManyRanges("/api/bulk/range").
    Build()

suite := &HandlerTestSuite{
    Name:      "GetCharacterDetail",
    Router:    router,
    API:       api,
    Scenarios: scenarios,
}

suite.RunErrorTests(t)
```

### Edge Case Testing

```go
edgeCases := CommonEdgeCases()
for _, edgeCase := range edgeCases {
    t.Run("EdgeCase_"+edgeCase.Name, func(t *testing.T) {
        edgeCase.Test(t, router, api)
    })
}
```

## Test Coverage by Endpoint

### 1. Health Check (`/health`)

**Success Cases:**
- Basic health check returns healthy status
- Database connection verification
- Characters loaded verification

**Coverage:** 100%

### 2. Search Characters (`/api/search`)

**Success Cases:**
- Basic search with query parameter
- Search with category filter
- Search with block filter
- Search with Unicode version filter
- Search with pagination (limit/offset)
- Empty results for non-existent queries
- Default pagination values

**Edge Cases:**
- Empty query string
- Negative limit (defaults to 100)
- Excessive limit (caps at 1000)
- Negative offset (defaults to 0)
- Special characters in query
- Unicode characters in query

**Coverage:** ~95%

### 3. Character Detail (`/api/character/:codepoint`)

**Success Cases:**
- Unicode format (U+1F600)
- Decimal format (128512)
- Related characters (same block, max 5)
- Usage examples generation

**Error Cases:**
- Invalid codepoint format
- Non-existent character (404)

**Coverage:** ~90%

### 4. Categories List (`/api/categories`)

**Success Cases:**
- List all categories with counts
- Verify required fields
- Verify character counts

**Coverage:** 100%

### 5. Blocks List (`/api/blocks`)

**Success Cases:**
- List all blocks with counts
- Verify required fields
- Verify ordering by start_codepoint

**Coverage:** 100%

### 6. Bulk Range (`/api/bulk/range`)

**Success Cases:**
- Single range query
- Multiple ranges query
- Decimal format ranges
- Verify characters within range

**Error Cases:**
- Empty ranges array
- Too many ranges (>10)
- Invalid range (start > end)
- Malformed JSON

**Coverage:** ~85%

## Helper Function Tests

### parseCodepoint()

- Unicode format (U+1F600)
- Decimal format (128512)
- Lowercase unicode (u+41)
- Invalid format error

### parseCodepointRange()

- Valid Unicode range
- Valid decimal range
- Error: start > end
- Error: invalid start
- Error: invalid end

### generateUsageExamples()

- Basic character (HTML + Unicode)
- Character with optional fields (HTML Entity, CSS)

## Performance Tests

Performance tests run when not in short mode (`go test -short` skips them).

**Targets:**
- Search response time: < 50ms average
- Character detail: < 25ms average
- Bulk operations: < 200ms average

**Methodology:**
- 5 warmup iterations
- 100 measurement iterations
- Average response time calculation

## Running Tests

### Unit Tests Only (No Database Required)

```bash
cd api
go test -tags=testing -v
```

Tests helper functions without database:
- parseCodepoint
- parseCodepointRange
- generateUsageExamples

**Coverage:** ~9-10% (helper functions only)

### Full Unit Tests (With Database)

```bash
cd api
export POSTGRES_URL="postgres://user:pass@localhost:5432/symbol_search_test"
go test -tags=testing -v -coverprofile=coverage.out
```

**Expected Coverage:** 70-85%

### With Coverage Report

```bash
cd api
go test -tags=testing -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Performance Tests

```bash
cd api
export POSTGRES_URL="..."
go test -tags=testing -v -run TestPerformance
```

### Via Test Phase Integration

```bash
cd /path/to/symbol-search
./test/phases/test-unit.sh        # Unit tests with coverage
./test/phases/test-integration.sh  # Integration tests (requires running API)
./test/phases/test-performance.sh  # Performance benchmarks
```

## Integration with Vrooli Testing Infrastructure

The test suite integrates with Vrooli's centralized testing system:

- **Unit Runner**: `scripts/scenarios/testing/unit/run-all.sh`
- **Phase Helpers**: `scripts/scenarios/testing/shell/phase-helpers.sh`
- **Coverage Thresholds**: 80% warning, 50% error

## Best Practices

1. **Always use defer for cleanup**
   ```go
   cleanup := setupTestLogger()
   defer cleanup()
   ```

2. **Test both success and error paths**
   - Use TestScenarioBuilder for systematic error testing
   - Include edge cases from CommonEdgeCases()

3. **Verify response structure AND content**
   ```go
   response := assertSearchResponse(t, w, http.StatusOK)
   if response.Total < 0 {
       t.Errorf("Invalid total: %d", response.Total)
   }
   ```

4. **Use table-driven tests for multiple scenarios**
   ```go
   scenarios := NewTestScenarioBuilder().
       AddInvalidCodepoint(...).
       AddNonExistentCharacter(...).
       Build()
   ```

5. **Clean up test data**
   ```go
   seedTestCharacter(t, db, "U+TEST", 999999, "TEST CHAR")
   defer cleanupTestCharacter(t, db, "U+TEST")
   ```

## Known Limitations

1. **Database Dependency**: Most tests require a PostgreSQL database with seeded Unicode data
2. **Performance Variability**: Performance tests may vary based on database size and system load
3. **Test Data Requirements**: Integration tests assume standard Unicode database population

## Future Improvements

1. **Mock Database**: Add in-memory database option for faster unit tests
2. **Parameterized Tests**: Expand table-driven tests for more scenarios
3. **Concurrent Testing**: Add tests for concurrent API access
4. **Load Testing**: Add sustained load tests beyond current performance tests
5. **CLI Testing**: Add BATS tests for CLI interface

## References

- Gold Standard: `/scenarios/visited-tracker/api/` (79.4% coverage)
- Testing Guide: `/docs/testing/guides/scenario-unit-testing.md`
- Centralized Testing: `/scripts/scenarios/testing/`
