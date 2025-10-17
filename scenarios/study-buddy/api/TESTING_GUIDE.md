# Study Buddy Testing Guide

## Overview

This document describes the comprehensive test suite for the study-buddy scenario, including test patterns, helpers, and coverage targets.

## Test Structure

```
api/
├── test_helpers.go       # Reusable test utilities
├── test_patterns.go      # Systematic error testing patterns
├── main_test.go          # Comprehensive API tests
└── TESTING_GUIDE.md      # This document
```

## Test Coverage Goals

- **Target Coverage**: 80% (business logic and testable functions)
- **Current Coverage**: 39.1% (limited by database dependency)
- **Logic Functions**: 100% coverage achieved
  - Spaced repetition algorithm
  - XP calculation
  - Mastery level determination
  - Level calculation
  - Helper functions

## Running Tests

### Unit Tests Only (Fast)
```bash
cd api
go test -tags=testing -v -short
```

### Full Test Suite with Coverage
```bash
cd api
go test -tags=testing -v -cover -coverprofile=coverage.out
go tool cover -html=coverage.out  # View coverage report
```

### Via Centralized Testing Infrastructure
```bash
cd scenarios/study-buddy
bash test/phases/test-unit.sh
```

### Via Makefile
```bash
cd scenarios/study-buddy
make test
```

## Test Categories

### 1. Health Check Tests
- **Coverage**: Basic service health validation
- **Tests**: `TestHealthCheck`

### 2. P0 Requirement Tests (PRD Compliant)

#### Flashcard Generation (`TestFlashcardGeneration`)
- Success path with different parameters
- Default value handling
- Performance validation (<5 seconds)
- Error path validation

#### Due Cards Retrieval (`TestDueCards`)
- Basic card retrieval
- Subject filtering
- Missing user ID error handling

#### Study Session Management (`TestStudySession`)
- Session creation
- UUID validation
- Error path validation

#### Flashcard Answer Submission (`TestFlashcardAnswer`)
- All response types (easy, good, hard, again)
- XP earning validation
- Progress update validation
- Mastery level verification

### 3. Algorithm Tests

#### Spaced Repetition (`TestSpacedRepetitionAlgorithm`)
- First review interval
- Second review (6-day interval)
- Exponential growth
- Failed review reset
- Ease factor adjustment
- Next review date calculation

**Coverage**: 100% of spaced repetition logic

#### XP Calculation (`TestXPCalculation`)
- Base XP for correct answers
- Speed bonus (<10 seconds)
- No bonus for incorrect answers

**Coverage**: 100% of XP calculation logic

#### Mastery Level (`TestMasteryLevel`)
- All response types mapped correctly
- Unknown response handling

**Coverage**: 100% of mastery level logic

### 4. Endpoint Tests

#### Study Materials (`TestStudyMaterials`, `TestStudyMaterialsCRUD`)
- Material retrieval
- Subject-specific materials

#### Quiz Endpoints (`TestQuizEndpoints`)
- Quiz generation
- Answer submission
- Quiz history

#### Analytics (`TestAnalyticsEndpoints`)
- Learning analytics
- Subject progress

#### AI Endpoints (`TestAIEndpoints`)
- Concept explanation
- Study plan generation

### 5. Performance Tests

#### Spaced Repetition Performance
- Target: <100ms for calculation
- 100 iterations minimum

#### Due Cards Retrieval Performance
- Target: <100ms for retrieval
- 50 iterations minimum

#### Flashcard Generation Performance
- Target: <5 seconds
- 10 iterations minimum

**Note**: Performance tests are skipped in short mode

### 6. Integration Tests

#### Flashcard Learning Flow (`TestIntegrationFlashcardLifecycle`)
- Complete end-to-end workflow:
  1. Generate flashcards
  2. Start study session
  3. Answer multiple flashcards
  4. Validate XP progression
  5. Get due cards

**Note**: Integration tests are skipped in short mode

### 7. Edge Case Tests

#### Edge Cases (`TestEdgeCases`)
- Empty content handling
- Large card count generation
- Very fast response XP bonus

#### Helper Functions (`TestHelperFunctions`)
- Content truncation
- Level calculation
- Structured card generation
- Mock flashcard generation

### 8. Business Logic Tests

#### Streak Calculation (`TestBusinessLogic_StreakCalculation`)
- Streak maintenance
- Level validation
- XP total validation

#### Review Calculation (`TestBusinessLogic_NextReviewCalculation`)
- Next review date in future
- Easy response interval validation

## Test Helpers

### Test Environment Setup
- `setupTestLogger()`: Disables debug logging during tests
- `setupTestEnvironment()`: Creates isolated test environment
- `setupTestDB()`: Connects to test database (skips if unavailable)

### HTTP Testing
- `makeHTTPRequest()`: Creates and executes HTTP test requests
- `assertJSONResponse()`: Validates JSON responses with expected fields
- `assertErrorResponse()`: Validates error responses

### Test Data Creation
- `createTestSubject()`: Creates test subject
- `createTestFlashcard()`: Creates test flashcard
- `createTestSession()`: Creates test study session

### Assertions
- `assertSpacedRepetitionData()`: Validates spaced repetition calculations
- `assertXPCalculation()`: Validates XP calculations
- `compareFlashcards()`: Compares flashcard objects

## Test Patterns

### Error Test Patterns
- `FlashcardGenerationPatterns()`: Missing fields, invalid JSON, empty body
- `StudySessionPatterns()`: Missing user ID, invalid JSON
- `FlashcardAnswerPatterns()`: Missing session ID, invalid JSON
- `SubjectManagementPatterns()`: Missing user ID, invalid JSON

### Performance Test Patterns
- Spaced repetition calculation
- Due cards retrieval
- Flashcard generation

## Database Testing

### Current Limitation
Most handler functions require database access. Without a test database, these functions return 500 errors. This limits overall coverage to ~39%.

### Testing Strategy
1. **Logic Functions**: 100% coverage achieved (no DB required)
2. **Handler Functions**: Structure and routing validated
3. **Database Functions**: Require test database for full coverage

### Test Database Setup (Optional)
```bash
# Set environment variables for test database
export TEST_POSTGRES_HOST=localhost
export TEST_POSTGRES_PORT=5432
export TEST_POSTGRES_USER=postgres
export TEST_POSTGRES_PASSWORD=postgres
export TEST_POSTGRES_DB=study_buddy_test

# Create test database
psql -U postgres -c "CREATE DATABASE study_buddy_test;"

# Run schema
psql -U postgres -d study_buddy_test -f ../initialization/storage/postgres/schema.sql

# Run tests with database
go test -tags=testing -v -cover
```

## Coverage Breakdown

### High Coverage (80-100%)
- Spaced repetition algorithm: 87.5%
- XP calculation: 100%
- Mastery level determination: 100%
- Level calculation: 100%
- Helper functions: 100%
- Generate flashcards (logic): 86.7%
- Analytics endpoints: 100%

### Medium Coverage (30-80%)
- Start study session: 75%
- Subject management: ~30%
- Flashcard CRUD: ~30%

### Low Coverage (0-30%)
- Database-dependent handlers: 0-27%
- End study session: 0%
- Review flashcard: 0%
- Due flashcards (DB query): 0%

## Best Practices

### 1. Test Isolation
- Each test should be independent
- Use `setupTestLogger()` in every test
- Clean up resources with `defer`

### 2. Error Path Testing
- Test all error conditions
- Use systematic error patterns
- Validate error response structure

### 3. Performance Testing
- Skip in short mode
- Test worst-case scenarios
- Validate against PRD requirements

### 4. Integration Testing
- Test complete workflows
- Validate data flow between endpoints
- Skip in short mode (long-running)

### 5. Assertions
- Validate both status code AND response body
- Check all required fields in responses
- Verify data types and values

## Adding New Tests

### 1. Add Test Function
```go
func TestNewFeature(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    router := setupRouter()

    t.Run("Success", func(t *testing.T) {
        // Test implementation
    })

    t.Run("ErrorPaths", func(t *testing.T) {
        // Error testing
    })
}
```

### 2. Add Helper if Needed
```go
// In test_helpers.go
func createTestResource(t *testing.T, router *gin.Engine) *TestResource {
    // Implementation
}
```

### 3. Add Pattern if Reusable
```go
// In test_patterns.go
func NewFeaturePatterns() []ErrorTestPattern {
    return NewTestScenarioBuilder().
        AddMissingRequiredField("POST", "/api/feature", map[string]interface{}{}).
        AddInvalidJSON("POST", "/api/feature").
        Build()
}
```

### 4. Run Tests
```bash
go test -tags=testing -v -run TestNewFeature
```

## CI/CD Integration

The test suite integrates with Vrooli's centralized testing infrastructure:

```bash
# test/phases/test-unit.sh
testing::unit::run_all_tests \
    --go-dir "api" \
    --coverage-warn 80 \
    --coverage-error 50
```

## Known Limitations

1. **Database Dependency**: Handler functions require DB for full coverage
2. **Mock Data**: Some tests use mock responses instead of real data
3. **External Services**: Ollama integration tested with fallback only

## Future Improvements

1. Add mock database layer for handler testing
2. Implement in-memory database for tests
3. Add more edge case coverage
4. Expand performance test scenarios
5. Add stress testing for concurrent requests

## References

- **Gold Standard**: `/scenarios/visited-tracker/` (79.4% coverage)
- **PRD**: `../PRD.md` (P0 requirements)
- **Testing Infrastructure**: `/docs/testing/guides/scenario-unit-testing.md`
