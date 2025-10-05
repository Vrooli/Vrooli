# Testing Guide for Tech Tree Designer

## Test Structure

The test suite for tech-tree-designer follows Vrooli's centralized testing infrastructure and gold standard patterns from visited-tracker.

### Test Files

1. **test_helpers.go** - Reusable test utilities
   - `setupTestLogger()` - Controlled logging during tests
   - `setupTestDB()` - Database connection with automatic cleanup
   - `setupTestRouter()` - Gin router configuration for testing
   - `makeHTTPRequest()` - Simplified HTTP request creation
   - `assertJSONResponse()` - Validate JSON responses
   - `assertErrorResponse()` - Validate error responses
   - Database test data creators: `createTestTechTree`, `createTestSector`, `createTestStage`, `createTestScenarioMapping`

2. **test_patterns.go** - Systematic error testing patterns
   - `TestScenarioBuilder` - Fluent interface for building test scenarios
   - `ErrorTestPattern` - Systematic error condition testing
   - `HandlerTestSuite` - Comprehensive HTTP handler testing

3. **main_test.go** - HTTP handler tests
   - Health endpoint testing
   - Tech tree retrieval (GET /api/v1/tech-tree)
   - Sectors management (GET /api/v1/tech-tree/sectors, GET /api/v1/tech-tree/sectors/:id)
   - Stages management (GET /api/v1/tech-tree/stages/:id)
   - Scenario mappings (GET/POST /api/v1/progress/scenarios, PUT /api/v1/progress/scenarios/:scenario)
   - Strategic analysis (POST /api/v1/tech-tree/analyze)
   - Recommendations (GET /api/v1/recommendations)
   - Dependencies and connections
   - Complete workflow integration tests

4. **helpers_test.go** - Business logic unit tests
   - `TestGenerateStrategicRecommendations` - Recommendation generation logic
   - `TestCalculateProjectedTimeline` - Timeline calculation logic
   - `TestIdentifyBottlenecks` - Bottleneck identification logic
   - `TestAnalyzeCrossSectorImpacts` - Cross-sector impact analysis
   - `TestAnalysisRequestValidation` - Input validation
   - `TestDataStructures` - Data model validation
   - `TestRecommendationQuality` - Recommendation prioritization
   - `TestTimelineProjections` - Timeline chronology and confidence

5. **business_logic_test.go** - Domain validation tests
   - Sector category validation
   - Stage type progression validation
   - Completion status progression rules
   - Progress percentage validation (0-100%)
   - Contribution weight validation (0-1)
   - Impact score validation (0-1)
   - JSON marshalling/unmarshalling
   - Cross-sector impact logic
   - Recommendation consistency
   - Bottleneck actionability
   - Timeline calculations
   - Dependency strength validation
   - Milestone type validation

## Running Tests

### Unit Tests
```bash
cd scenarios/tech-tree-designer
make test
```

Or directly:
```bash
cd scenarios/tech-tree-designer/api
go test -v -tags=testing ./...
```

### With Coverage
```bash
cd scenarios/tech-tree-designer/api
go test -tags=testing -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Using Centralized Testing Infrastructure
```bash
cd scenarios/tech-tree-designer
./test/phases/test-unit.sh
```

## Test Organization

### Success Cases
- Happy path testing with valid inputs
- Complete response validation (status code + body)
- Field presence and type checking

### Error Cases
- Invalid UUIDs
- Non-existent resources (404 errors)
- Malformed JSON requests
- Missing required fields
- Invalid HTTP methods

### Edge Cases
- Empty arrays and nil values
- Boundary conditions (0%, 100% progress)
- Negative values and out-of-range inputs
- Zero resources, negative time horizons
- Long time horizons and high resource counts

## Coverage Target

- **Target**: 80% coverage
- **Minimum**: 50% coverage
- **Current**: 19.2% (limited by database dependency in handlers)

### Coverage Breakdown

**Well-covered components:**
- Strategic recommendation logic (100%)
- Timeline calculation (100%)
- Bottleneck identification (100%)
- Cross-sector impact analysis (100%)
- Business logic validation (100%)
- Data structure validation (100%)

**Requires database for coverage:**
- Database-backed HTTP handlers (getTechTree, getSectors, etc.)
- Helper functions that query database
- Integration workflows

## Database Testing

Many tests are designed to skip gracefully when the database is unavailable:

```go
testDB, dbCleanup := setupTestDB(t)
if testDB == nil {
    return // Skip if database not available
}
defer dbCleanup()
```

To run with database tests:
1. Ensure PostgreSQL is running
2. Create test database: `createdb vrooli_test`
3. Run database initialization scripts
4. Execute tests

## Test Quality Standards

### Each Test Should Include:

1. **Setup Phase**
   - Logger initialization
   - Test environment/database setup
   - Test data creation

2. **Success Cases**
   - Happy path with complete assertions
   - Validate all expected fields
   - Check data types and values

3. **Error Cases**
   - Invalid inputs
   - Missing resources
   - Malformed data
   - HTTP method validation

4. **Edge Cases**
   - Empty inputs
   - Boundary conditions
   - Null/nil values
   - Extreme values

5. **Cleanup**
   - Always defer cleanup functions
   - Prevent test pollution
   - Restore original state

### HTTP Handler Testing Pattern

```go
t.Run("Success case", func(t *testing.T) {
    // Setup test data
    treeID := createTestTechTree(t, testDB)

    // Make request
    w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree", nil)

    // Validate response
    response := assertJSONResponse(t, w, http.StatusOK)

    // Check expected fields
    if id, ok := response["id"].(string); !ok || id != treeID {
        t.Errorf("Expected tree ID '%s', got '%v'", treeID, response["id"])
    }
})

t.Run("Error cases", func(t *testing.T) {
    // Test invalid methods, missing resources, etc.
    errorPattern := NewErrorTestPattern(t, router)
    errorPattern.TestInvalidMethods("/api/v1/tech-tree", "GET")
})
```

## Integration with Centralized Testing

The test suite integrates with Vrooli's centralized testing infrastructure:

- **Phase-based execution**: `test/phases/test-unit.sh`
- **Centralized runners**: Sources from `scripts/scenarios/testing/unit/run-all.sh`
- **Phase helpers**: Uses `scripts/scenarios/testing/shell/phase-helpers.sh`
- **Coverage thresholds**: `--coverage-warn 80 --coverage-error 50`

## Future Improvements

1. **Mock Database Layer**: Create database mocks to test handlers without PostgreSQL
2. **Integration Tests**: Add full end-to-end workflow tests with real database
3. **Performance Tests**: Add benchmarking for strategic analysis and recommendations
4. **Stress Tests**: Test with large tech trees (1000+ nodes)
5. **CLI Tests**: Add BATS tests for CLI commands
6. **UI Tests**: Add React component tests when UI is developed

## Test Maintenance

- Tests use build tag `// +build testing` to separate from production code
- Test helpers are reusable across all test files
- Systematic error patterns reduce code duplication
- Database cleanup is automatic with defer statements
- Tests skip gracefully when dependencies unavailable
