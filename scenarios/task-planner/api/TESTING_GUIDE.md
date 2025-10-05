# Task Planner Testing Guide

## Overview

This guide describes the comprehensive test suite for the task-planner scenario. The test implementation follows Vrooli's gold standard testing patterns from the visited-tracker scenario.

## Test Coverage Summary

**Current Coverage: 20.9%** (baseline: 0%)

### Coverage by Module

- **main.go**: 7.5% - Core HTTP handlers and service initialization
- **task_parser.go**: 38.7% - AI-based task parsing and validation
- **task_researcher.go**: 35.8% - Research result parsing and context enhancement
- **task_monitor.go**: 8.3% - Status transition validation and action generation

## Test Structure

### Test Helper Library (`test_helpers.go`)

Provides reusable testing utilities following the visited-tracker pattern:

**Setup Functions:**
- `setupTestLogger()` - Redirects logging to avoid test pollution
- `setupTestEnvironment(t)` - Creates isolated test environment with in-memory database
- `skipIfNoDatabase(t, db)` - Skips tests when database is unavailable

**HTTP Testing Utilities:**
- `makeHTTPRequest(env, req)` - Creates and executes HTTP test requests
- `assertJSONResponse(t, w, expectedStatus)` - Validates JSON response structure
- `assertErrorResponse(t, w, expectedStatus, expectedErrorContains)` - Validates error responses
- `assertSuccessResponse(t, w, expectedFields)` - Validates successful responses with required fields

**Database Utilities:**
- `createTestApp(t, env, name)` - Inserts test app into database
- `createTestTask(t, env, appID, title, status)` - Inserts test task into database
- `cleanupTestData(t, env)` - Removes test data from database

### Test Pattern Library (`test_patterns.go`)

Implements systematic error testing patterns:

**TestScenarioBuilder:**
Fluent interface for building comprehensive error test scenarios:
- `AddInvalidUUID(path, method)` - Tests invalid UUID format handling
- `AddNonExistentTask(path, method)` - Tests non-existent resource handling
- `AddInvalidJSON(path, method)` - Tests malformed JSON payload handling
- `AddMissingRequiredField(path, method, body)` - Tests missing required fields
- `AddInvalidStatusTransition(body)` - Tests invalid status transitions
- `AddUnauthorizedAccess(path, method)` - Tests authentication/authorization
- `AddEmptyPayload(path, method)` - Tests empty request body handling
- `AddInvalidQueryParam(path, paramName, paramValue)` - Tests invalid query parameters

**HandlerTestSuite:**
Framework for comprehensive HTTP handler testing:
- `RunErrorTests(t, patterns)` - Executes suite of error condition tests
- `RunSuccessTest(t, name, req, validate)` - Executes success path tests

**Validation Helpers:**
- `ValidateTaskResponse(t, task)` - Validates task response structure
- `ValidateAppResponse(t, app)` - Validates app response structure
- `ValidateStatusHistoryResponse(t, history)` - Validates status history response

## Test Files

### main_test.go

Comprehensive tests for core HTTP endpoints:

**Health Endpoint:**
- ✅ Health check returns correct status, service name, and version

**GetApps Endpoint:**
- ✅ Returns all apps with correct structure
- ✅ Handles empty database gracefully

**GetTasks Endpoint:**
- ✅ Returns tasks with correct structure
- ✅ Filters by app_id
- ✅ Filters by status
- ✅ Filters by priority
- ✅ Supports pagination with limit and offset

**UpdateTaskStatus Endpoint:**
- ✅ Successfully transitions from backlog to in_progress
- ✅ Successfully transitions from in_progress to completed
- ✅ Rejects transitions to already-current status
- ✅ Validates status transition rules
- ✅ Handles missing required fields
- ✅ Handles invalid status values

**GetTaskStatusHistory Endpoint:**
- ✅ Returns status history with correct structure
- ✅ Validates history entries have required fields
- ✅ Rejects missing task_id parameter
- ✅ Rejects invalid task_id format

**Service Initialization:**
- ✅ Creates service with correct dependencies
- ✅ Initializes logger, database, HTTP client, and Qdrant URL

### task_parser_test.go

Tests for AI-based task parsing:

**parseAITaskResponse:**
- ✅ Parses valid JSON array of tasks
- ✅ Handles JSON with markdown formatting (```json)
- ✅ Handles empty task arrays
- ✅ Rejects responses without JSON
- ✅ Fixes invalid priorities to default (medium)
- ✅ Filters out tasks without titles
- ✅ Truncates overly long titles (>500 chars)
- ✅ Sets default estimated hours for invalid values

**validateStatusTransition:**
- ✅ All 16 valid transitions pass
- ✅ All 8 invalid transitions are rejected
- ✅ Validates status transition state machine

**getNextActions:**
- ✅ Returns appropriate actions for each status
- ✅ Includes relevant keywords (Research, progress, resolve, etc.)
- ✅ Handles unknown statuses gracefully

**Helper Functions:**
- ✅ min() function returns smaller of two values
- ✅ getResourcePort() retrieves ports from registry

### task_researcher_test.go

Tests for research result processing:

**parseResearchResults:**
- ✅ Parses valid JSON research results
- ✅ Handles JSON with extra whitespace
- ✅ Handles JSON with backtick formatting
- ✅ Rejects responses without valid JSON
- ✅ Validates complexity values (low/medium/high/very_high)
- ✅ Ensures estimated hours are non-negative
- ✅ Extracts requirements, dependencies, and recommendations arrays

**buildResearchPrompt:**
- ✅ Includes task title, description, priority, and tags
- ✅ Includes app name and type
- ✅ Includes related tasks section when available
- ✅ Includes completed tasks section when available
- ✅ Handles empty context gracefully

**enhanceResearchWithContext:**
- ✅ Adds web-specific recommendations for web-application type
- ✅ Adjusts estimates based on completed tasks history
- ✅ Leaves estimates unchanged when no historical data available

## Integration with Centralized Testing

The test suite integrates with Vrooli's centralized testing infrastructure:

**test/phases/test-unit.sh:**
```bash
#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 50

testing::phase::end_with_summary "Unit tests completed"
```

## Running Tests

### Local Development

```bash
# Run all tests with coverage
cd scenarios/task-planner/api
go test -tags=testing -v -cover -coverprofile=coverage.out

# View coverage report
go tool cover -func=coverage.out

# View HTML coverage report
go tool cover -html=coverage.out
```

### Through Makefile

```bash
cd scenarios/task-planner
make test
```

### Through Vrooli CLI

```bash
vrooli scenario test task-planner
```

## Test Quality Standards

All tests follow these standards:

1. **Setup Phase**: Logger setup, isolated environment, test data creation
2. **Success Cases**: Happy path with complete assertions
3. **Error Cases**: Invalid inputs, missing resources, malformed data
4. **Edge Cases**: Empty inputs, boundary conditions, null values
5. **Cleanup**: Always defer cleanup to prevent test pollution

## Future Improvements

To reach 80% coverage target:

1. **Database Integration Tests**: Requires test database setup
   - Full CRUD operations for apps and tasks
   - Status transition history tracking
   - Parsing session management

2. **AI Integration Tests**: Requires mock Ollama responses
   - Task parsing with various input formats
   - Task research with different complexities
   - Embedding generation

3. **HTTP Handler Tests**: Requires database
   - Full request/response cycle testing
   - Error handling validation
   - Status code verification

4. **Performance Tests**: Load and stress testing
   - Concurrent request handling
   - Database connection pooling
   - Response time benchmarks

## Notes

- Database tests are currently skipped when database is unavailable
- AI integration tests would require mocking external Ollama calls
- Test coverage focus is on business logic and validation functions
- HTTP handler tests require database setup for full integration testing
