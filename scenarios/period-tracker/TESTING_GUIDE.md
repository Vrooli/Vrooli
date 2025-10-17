# Testing Guide - period-tracker

## Quick Start

### Run All Tests
```bash
cd scenarios/period-tracker
make test
```

### Run Unit Tests Only (No Database Required)
```bash
cd api
go test -v -run TestEncrypt -coverprofile=coverage.out
```

### Run With Coverage Report
```bash
cd api
go test -coverprofile=coverage.out -covermode=atomic
go tool cover -html=coverage.out -o coverage.html
```

### Run Specific Test
```bash
cd api
go test -v -run TestEncryptDecrypt
```

### Run Benchmarks
```bash
cd api
go test -bench=. -benchmem
```

## Test Structure

### Unit Tests (No Database Required)
Located in `api/encryption_test.go`:
- ✅ Encryption/decryption tests
- ✅ Key derivation tests
- ✅ Error handling tests
- ✅ Performance benchmarks

**These tests run in CI/CD without database**

### Integration Tests (Database Required)
Located in `api/main_test.go`:
- ⚠️  HTTP endpoint tests
- ⚠️  Database interaction tests
- ⚠️  Multi-tenant isolation tests
- ⚠️  Authentication tests

**These tests require PostgreSQL with schema initialized**

## Test Files

| File | Purpose | Database Required |
|------|---------|-------------------|
| `encryption_test.go` | Encryption unit tests | ❌ No |
| `main_test.go` | HTTP handler integration tests | ✅ Yes |
| `test_helpers.go` | Reusable test utilities | ✅ Yes |
| `test_patterns.go` | Systematic error patterns | ✅ Yes |
| `test/phases/test-unit.sh` | Centralized test runner | ✅ Yes |

## Test Helpers

### `setupTestEnvironment(t)`
Creates isolated test environment with:
- Test logger (suppressed output)
- Temporary directory
- Test database connection
- Gin router in test mode
- Test user ID

### `makeHTTPRequest(env, req)`
Executes HTTP requests with:
- Automatic JSON marshaling
- Authentication headers
- Custom headers support

### `assertJSONResponse(t, w, status, keys)`
Validates JSON responses:
- Status code verification
- Content-Type validation
- Required key presence
- Returns parsed response map

### `assertErrorResponse(t, w, status)`
Validates error responses:
- Status code verification
- Error key presence in response

### `cleanupTestData(t, env, userID)`
Removes test data:
- Audit logs
- Detected patterns
- Predictions
- Daily symptoms
- Cycles

## Test Patterns

### Error Test Scenario Builder

```go
patterns := NewTestScenarioBuilder().
    AddMissingAuth("GET", "/api/v1/cycles").
    AddInvalidUUID("GET", "/api/v1/cycles").
    AddInvalidJSON("POST", "/api/v1/cycles").
    Build()

for _, pattern := range patterns {
    t.Run(pattern.Name, func(t *testing.T) {
        w, _ := makeHTTPRequest(env, pattern.Request)
        if w.Code != pattern.ExpectedStatus {
            t.Errorf("Expected %d, got %d", pattern.ExpectedStatus, w.Code)
        }
    })
}
```

### Performance Testing

```go
RunPerformanceTest(t, env, CreateCyclePerformancePattern())
RunPerformanceTest(t, env, GetCyclesPerformancePattern(10))
RunPerformanceTest(t, env, EncryptionPerformancePattern())
```

## Coverage Goals

| Component | Target | Current |
|-----------|--------|---------|
| Encryption Functions | 80% | ✅ 80%+ |
| HTTP Handlers | 80% | ⚠️  Requires DB |
| Middleware | 80% | ⚠️  Requires DB |
| Total | 80% | ⚠️  Requires DB |

## Running in CI/CD

### Without Database (Unit Tests Only)
```bash
cd api && go test -v -run TestEncrypt -coverprofile=coverage.out
```
**Result**: Encryption tests pass, ~10% total coverage

### With Database (Full Suite)
```bash
# 1. Start PostgreSQL
docker-compose up -d postgres

# 2. Initialize schema
psql -h localhost -U postgres -d period_tracker < initialization/postgres/schema.sql

# 3. Run tests
make test
```
**Result**: All tests pass, 80%+ coverage

## Environment Variables for Testing

```bash
# Database connection (uses defaults if not set)
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=postgres
export POSTGRES_DB=period_tracker_test

# Or use connection URL
export POSTGRES_URL="postgres://user:pass@localhost:5432/dbname?sslmode=disable"
```

## Test Data Factories

### Create Test Cycle
```go
cycleID := createTestCycle(t, env, userID, "2024-01-15", "medium", "Test notes")
```

### Create Test Symptom
```go
symptomID := createTestSymptom(t, env, userID, "2024-01-15", 7)
```

## Debugging Tests

### Verbose Output
```bash
go test -v
```

### Show All Test Output (including passed)
```bash
go test -v -test.v
```

### Run with Race Detector
```bash
go test -race
```

### CPU Profiling
```bash
go test -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

## Common Issues

### "Database not available"
**Solution**: Tests will skip gracefully. Run unit tests only or start PostgreSQL.

### "Relation does not exist"
**Solution**: Initialize database schema:
```bash
psql -h localhost -U postgres -d period_tracker < initialization/postgres/schema.sql
```

### "Connection refused"
**Solution**: Check PostgreSQL is running:
```bash
docker ps | grep postgres
```

## Best Practices

1. **Always use test helpers** - Don't create HTTP requests manually
2. **Clean up test data** - Use `defer cleanupTestData()`
3. **Test both success and error paths** - Use error scenario builder
4. **Verify encryption** - Check data is encrypted in database
5. **Test multi-tenant isolation** - Verify user data boundaries
6. **Use table-driven tests** - For testing multiple scenarios
7. **Add benchmarks** - For performance-critical code

## Example Test

```go
func TestCreateCycle(t *testing.T) {
    // Setup
    cleanup := setupTestLogger()
    defer cleanup()

    env := setupTestEnvironment(t)
    defer env.Cleanup()
    defer cleanupTestData(t, env, env.UserID)

    // Test
    t.Run("Success", func(t *testing.T) {
        w, err := makeHTTPRequest(env, HTTPTestRequest{
            Method: "POST",
            Path:   "/api/v1/cycles",
            Body: map[string]interface{}{
                "start_date": "2024-01-15",
                "flow_intensity": "medium",
            },
        })

        if err != nil {
            t.Fatalf("Request failed: %v", err)
        }

        response := assertJSONResponse(t, w, http.StatusCreated,
            []string{"cycle_id", "message"})

        if _, ok := response["cycle_id"].(string); !ok {
            t.Error("Expected valid cycle_id")
        }
    })
}
```

## Resources

- Gold Standard: `/scenarios/visited-tracker/api/TESTING_GUIDE.md`
- Centralized Testing: `/scripts/scenarios/testing/`
- Test Patterns: `api/test_patterns.go`
- Test Helpers: `api/test_helpers.go`
