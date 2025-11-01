# Campaign Content Studio - Test Suite

## Overview

Comprehensive test suite for the Campaign Content Studio scenario, following Vrooli's gold standard testing patterns (based on visited-tracker).

## Test Coverage

- **Without Database**: ~10% (all non-database dependent code)
- **With Database**: ~80%+ (full handler coverage)

## Quick Start

### Running Tests Without Database

```bash
cd api
go test -tags=testing -v
```

This runs all tests that don't require database integration:
- Structure validation
- Constants validation
- Logger functionality
- HTTP error handling
- Service initialization
- Test data generators

### Running Tests With Database

1. Start PostgreSQL (if not already running):
   ```bash
   docker run -d \
     -e POSTGRES_USER=testuser \
     -e POSTGRES_PASSWORD=testpass \
     -e POSTGRES_DB=campaign_test \
     -p 5432:5432 \
     postgres:15-alpine
   ```

2. Set environment variable:
   ```bash
   export TEST_POSTGRES_URL="postgres://testuser:testpass@localhost:5432/campaign_test?sslmode=disable"
   ```

3. Run tests:
   ```bash
   cd api
   go test -tags=testing -v
   ```

### Running with Coverage

```bash
cd api
go test -tags=testing -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Via Testing Infrastructure

```bash
# From the scenario root
test/run-tests.sh                # Run all phases
test/run-tests.sh quick          # Structure + unit smoke test
test/phases/test-unit.sh         # Invoke a single phase
```

The shared runner (see `test/run-tests.sh`) wires in all standard phases:

- `structure` – validates required files and directories
- `dependencies` – checks automation assets and manifest health
- `unit` – delegates to the centralized Go test runner with coverage gates
- `integration` – confirms API/CLI wiring without starting the runtime
- `business` – verifies seed data and automation workflows
- `performance` – placeholder until benchmarks land

## Test Structure

### Test Files

- **basic_test.go** - Tests without database dependencies
  - Data structures
  - Constants
  - Logger
  - HTTP error handling
  - Service initialization

- **main_test.go** - Comprehensive handler tests (requires database)
  - Health endpoint
  - List campaigns
  - Create campaign
  - List documents
  - Search documents
  - Generate content

- **performance_test.go** - Performance and load tests
  - Response time validation
  - Concurrency testing
  - Bulk operations
  - Database connection pooling

- **test_helpers.go** - Reusable test utilities
  - Test environment setup
  - Database helpers
  - HTTP request helpers
  - Response validation helpers
  - Test data generators

- **test_patterns.go** - Systematic error testing patterns
  - Error test patterns
  - Handler test suites
  - Test scenario builder
  - Performance patterns
  - Concurrency patterns

## Test Scenarios

### Success Cases
- ✅ Health endpoint returns healthy status
- ✅ List campaigns returns empty array when no campaigns
- ✅ List campaigns returns all campaigns
- ✅ Create campaign with valid data
- ✅ List documents for campaign
- ✅ All data structures properly validated

### Error Cases
- ✅ Invalid JSON returns 400 error
- ✅ Missing required fields returns 400 error
- ✅ Invalid UUID format returns 400 error
- ✅ Non-existent campaign returns 404 error
- ✅ Empty request body returns 400 error

### Performance Cases
- ✅ Health endpoint responds in < 50ms
- ✅ List campaigns (empty) in < 100ms
- ✅ List campaigns (10 items) in < 200ms
- ✅ Create campaign in < 200ms
- ✅ Concurrent requests (100 requests, 10 concurrent)
- ✅ Bulk operations (50 sequential, 50 concurrent)

## Running Specific Tests

### By Test Name
```bash
# Run health tests only
go test -tags=testing -v -run TestHealth

# Run all basic tests
go test -tags=testing -v -run TestBasic

# Run performance tests
go test -tags=testing -v -run TestPerformance
```

### Skip Slow Tests
```bash
go test -tags=testing -v -short
```

### Run Benchmarks
```bash
go test -tags=testing -bench=. -benchmem
```

## Test Configuration

### Environment Variables

- `TEST_POSTGRES_URL` - PostgreSQL connection string for tests
  - Format: `postgres://user:pass@host:port/dbname?sslmode=disable`
  - If not set, database-dependent tests are skipped

### Coverage Thresholds

Configured in `test/phases/test-unit.sh`:
- Warning threshold: 80%
- Error threshold: 50%

## Continuous Integration

The test suite integrates with Vrooli's centralized testing infrastructure:

```bash
# Runs all test phases
APP_ROOT=/path/to/Vrooli bash test/phases/test-unit.sh
```

This automatically:
- Sets up test environment
- Runs all tests with coverage
- Reports results with thresholds
- Integrates with CI/CD pipeline

## Test Data

### Test Campaigns
```go
campaign := setupTestCampaign(t, env, "test-campaign")
defer campaign.Cleanup()
```

### Test Documents
```go
doc := setupTestDocument(t, env, campaignID, "test.pdf")
defer doc.Cleanup()
```

### Test Requests
```go
// Campaign request
campaignReq := TestData.CampaignRequest("Name", "Description")

// Content generation request
contentReq := TestData.GenerateContentRequest(campaignID, "blog_post", "prompt")

// Search request
searchReq := TestData.SearchRequest("query", 10)
```

## Debugging Tests

### Verbose Output
```bash
go test -tags=testing -v
```

### Specific Test with Output
```bash
go test -tags=testing -v -run TestCreateCampaign
```

### With Race Detection
```bash
go test -tags=testing -race -v
```

### With Memory Profiling
```bash
go test -tags=testing -memprofile=mem.prof
go tool pprof mem.prof
```

## Common Issues

### Database Connection Failed
**Problem**: `TEST_POSTGRES_URL not set, skipping database tests`

**Solution**: Set the environment variable:
```bash
export TEST_POSTGRES_URL="postgres://testuser:testpass@localhost:5432/campaign_test?sslmode=disable"
```

### Tests Hanging
**Problem**: Tests timeout or hang

**Solution**: Increase timeout or check for deadlocks:
```bash
go test -tags=testing -v -timeout 5m
```

### Coverage Too Low
**Problem**: Coverage below threshold

**Solution**:
1. Ensure TEST_POSTGRES_URL is set
2. Run with coverage report to identify gaps:
```bash
go tool cover -html=coverage.out
```

## Best Practices

1. **Always clean up**: Use `defer` for cleanup functions
2. **Isolate tests**: Each test should be independent
3. **Use helpers**: Leverage test_helpers.go utilities
4. **Test errors**: Include error cases, not just success
5. **Performance matters**: Set reasonable time limits
6. **Document**: Add comments for complex test scenarios

## Contributing

When adding new tests:

1. Follow the gold standard pattern (visited-tracker)
2. Add to appropriate test file (basic, main, or performance)
3. Include success, error, and edge cases
4. Use existing helpers or create new reusable ones
5. Update this README if adding new test categories
6. Ensure tests pass both with and without database

## References

- Gold Standard: `/scenarios/visited-tracker/api/`
- Testing Guide: `/docs/testing/guides/scenario-unit-testing.md`
- Centralized Testing: `/scripts/scenarios/testing/`
