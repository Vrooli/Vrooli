# Testing Guide for Vrooli Orchestrator

## Overview

This test suite provides comprehensive coverage for the vrooli-orchestrator service, following the gold standard patterns from visited-tracker (79.4% coverage).

## Test Infrastructure

### Test Files

- **test_helpers.go** - Reusable test utilities for database setup, HTTP requests, and assertions
- **test_patterns.go** - Systematic error testing patterns using TestScenarioBuilder
- **main_test.go** - Comprehensive HTTP handler tests (all API endpoints)
- **profiles_test.go** - Profile manager database operation tests
- **orchestrator_test.go** - Orchestration logic and activation/deactivation tests
- **unit_test.go** - Unit tests that don't require database (structs, constants, helpers)

### Test Structure

Tests are organized following the visited-tracker gold standard:

1. **Setup Phase**: Logger initialization, isolated test database, test data
2. **Success Cases**: Happy path with complete assertions
3. **Error Cases**: Invalid inputs, missing resources, malformed data
4. **Edge Cases**: Empty inputs, boundary conditions, null values
5. **Cleanup**: Defer cleanup to prevent test pollution

## Running Tests

### Quick Run (No Database Required)

```bash
# Run unit tests only (tests that don't require database)
cd api
go test -v -run "^Test(Profile|ActivationResult|DeactivationResult|Constants|ContainsIgnoreCase|TestScenarioBuilder|ProfileCRUDErrorPatterns|ProfileActivationErrorPatterns|HTTPTestRequest|HTTPError|NewLogger|LoggerMethods|GetResourcePort)$"
```

### Full Test Suite (Requires PostgreSQL)

```bash
# Ensure postgres resource is running
cd ../../
vrooli resource start postgres

# Set environment variables
export POSTGRES_PASSWORD=postgres  # Or your actual password
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5433
export POSTGRES_USER=postgres

# Run all tests
cd scenarios/vrooli-orchestrator/api
go test -v -coverprofile=coverage.out -covermode=atomic ./...

# View coverage report
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Using Centralized Test Infrastructure

```bash
# Run through scenario test system
cd scenarios/vrooli-orchestrator
make test

# Or via vrooli CLI
vrooli scenario test vrooli-orchestrator
```

## Test Coverage Targets

- **Target Coverage**: 80% (specified in issue)
- **Minimum Coverage**: 50% (error threshold)
- **Warning Coverage**: 80%

## Test Categories

### 1. HTTP Handler Tests (main_test.go)

Tests all API endpoints with comprehensive scenarios:

- `TestHealth` - Health endpoint validation
- `TestListProfiles` - Profile listing (empty and populated)
- `TestGetProfile` - Profile retrieval (success, not found, empty name)
- `TestCreateProfile` - Profile creation (success, validation, duplicates)
- `TestUpdateProfile` - Profile updates (all field types, partial updates)
- `TestDeleteProfile` - Profile deletion (inactive, active, not found)
- `TestGetStatus` - Status endpoint (with/without active profile)
- `TestActivateProfile` - Profile activation endpoint
- `TestDeactivateProfile` - Profile deactivation endpoint

### 2. Profile Manager Tests (profiles_test.go)

Tests all database operations:

- `TestProfileManagerListProfiles` - List with ordering
- `TestProfileManagerGetProfile` - Single profile retrieval
- `TestProfileManagerCreateProfile` - Creation with all field types
- `TestProfileManagerUpdateProfile` - Update operations
- `TestProfileManagerDeleteProfile` - Deletion with constraints
- `TestProfileManagerGetActiveProfile` - Active profile tracking
- `TestProfileManagerSetActiveProfile` - Activation
- `TestProfileManagerClearActiveProfile` - Deactivation
- `TestProfileJSONParsing` - Complex JSON field handling

### 3. Orchestrator Tests (orchestrator_test.go)

Tests orchestration logic:

- `TestActivateProfile` - Full activation flow
- `TestDeactivateCurrentProfile` - Deactivation flow
- `TestStartResource/StopResource` - Resource management
- `TestStartScenario/StopScenario` - Scenario management
- `TestValidateProfile` - Profile validation
- `TestProfileActivationDeactivationCycle` - Complete lifecycle

### 4. Unit Tests (unit_test.go)

Tests without database dependency:

- `TestProfile` - Profile struct validation
- `TestActivationResultStructure` - Result structures
- `TestConstants` - Constant values
- `TestContainsIgnoreCase` - Helper functions
- `TestTestScenarioBuilder` - Error pattern builder
- `TestHTTPTestRequest` - Request structures

## Test Helper Functions

### setupTestLogger()
Initializes controlled logging during tests to reduce noise.

### setupTestDB(t)
Creates isolated test database with:
- Unique database name per test run
- Complete schema initialization
- Automatic cleanup on test completion
- Router setup with all endpoints

### makeHTTPRequest(router, req)
Simplified HTTP request creation:
- Automatic JSON marshaling
- Header management
- Response recording

### assertJSONResponse(t, rr, expectedStatus)
Validates JSON responses:
- Status code verification
- Content-Type validation
- JSON parsing
- Returns parsed response map

### assertErrorResponse(t, rr, expectedStatus, errorSubstring)
Validates error responses:
- All assertJSONResponse validations
- Error message presence
- Error message content verification

### createTestProfile(t, env, name, status)
Creates test profiles with predefined data:
- Standard resource/scenario configuration
- Customizable name and status
- Returns created profile

### cleanupProfiles(env)
Removes all test profiles from database.

## Error Test Patterns

### TestScenarioBuilder

Fluent interface for building systematic error tests:

```go
patterns := NewTestScenarioBuilder().
    AddInvalidJSON("POST", "/api/v1/profiles").
    AddNonExistentProfile("GET", "/api/v1/profiles/test").
    AddMissingRequiredField("POST", "/api/v1/profiles", "name").
    Build()
```

### Common Patterns

- `AddInvalidJSON` - Malformed JSON requests
- `AddMissingRequiredField` - Required field validation
- `AddNonExistentProfile` - 404 errors
- `AddDuplicateProfile` - Conflict errors
- `AddDeleteActiveProfile` - Constraint violations

## Performance Considerations

- Tests complete in < 60 seconds (target time)
- Database tests use isolated databases (no interference)
- Parallel test execution supported
- Cleanup uses defer for reliability

## Integration with CI/CD

The test suite integrates with:

- **Centralized Testing Library**: `scripts/scenarios/testing/`
- **Phase-based Runner**: `test/phases/test-unit.sh`
- **Coverage Thresholds**: `--coverage-warn 80 --coverage-error 50`

## Known Limitations

1. **Database Dependency**: Most comprehensive tests require PostgreSQL
2. **External Commands**: Tests for resource/scenario management depend on `vrooli` CLI
3. **Environment Setup**: Requires proper environment variables for database access

## Future Improvements

1. Mock database for faster unit tests
2. Integration tests for actual resource/scenario management
3. Performance benchmarks
4. Load testing for concurrent operations

## Coverage Report Generation

```bash
# Generate coverage report
go test -coverprofile=coverage.out -covermode=atomic ./...

# View in terminal
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# Open in browser
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

## Best Practices

1. **Always defer cleanup** to prevent test pollution
2. **Test both status AND body** for HTTP handlers
3. **Use table-driven tests** for multiple scenarios
4. **Verify all error cases** not just success paths
5. **Keep tests independent** - no shared state between tests
6. **Use meaningful test names** describing what is tested

## Example Test Pattern

```go
func TestMyHandler(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()

    env := setupTestDB(t)
    defer env.Cleanup()

    t.Run("Success", func(t *testing.T) {
        // Happy path
    })

    t.Run("ErrorCase", func(t *testing.T) {
        // Error handling
    })

    t.Run("EdgeCase", func(t *testing.T) {
        // Boundary conditions
    })
}
```
