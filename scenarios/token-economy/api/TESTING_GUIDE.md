# Token Economy Testing Guide

This document outlines the comprehensive testing strategy for the token-economy scenario.

## Test Structure

### Test Files
- `test_helpers.go` - Reusable test utilities and setup functions
- `test_patterns.go` - Systematic error patterns and test builders
- `main_test.go` - Comprehensive test suite covering all handlers

### Test Coverage Areas

#### 1. Health Checks
- `/health` endpoint validation
- `/api/v1/health` endpoint with service status
- Database connectivity verification
- Redis connectivity verification

#### 2. Token Operations
- **Create Token** (`POST /api/v1/tokens/create`)
  - Success cases with valid data
  - Missing required fields (symbol, name, type)
  - Invalid JSON handling
  - Duplicate symbol prevention

- **Mint Token** (`POST /api/v1/tokens/mint`)
  - Successful minting to wallet
  - Invalid JSON handling
  - Supply tracking validation

- **List Tokens** (`GET /api/v1/tokens`)
  - Household filtering
  - Active token filtering

- **Get Token** (`GET /api/v1/tokens/{id}`)
  - Successful retrieval
  - Non-existent token handling

#### 3. Wallet Operations
- **Create Wallet** (`POST /api/v1/wallets/create`)
  - User wallet creation
  - Scenario wallet creation
  - Invalid JSON handling

- **Get Wallet** (`GET /api/v1/wallets/{wallet_id}`)
  - Successful retrieval
  - Non-existent wallet handling

- **Get Balance** (`GET /api/v1/wallets/{wallet_id}/balance`)
  - Balance retrieval with caching
  - Multiple token balances
  - Non-existent wallet handling

#### 4. Transaction Operations
- **Transfer Token** (`POST /api/v1/tokens/transfer`)
  - Successful transfers
  - Insufficient balance handling
  - Invalid JSON handling
  - Cache invalidation

- **Swap Token** (`POST /api/v1/tokens/swap`)
  - Invalid JSON handling
  - Missing exchange rate handling

- **List Transactions** (`GET /api/v1/transactions`)
  - No filter
  - Wallet filter
  - Token filter
  - Limit filter

- **Get Transaction** (`GET /api/v1/transactions/{hash}`)
  - Successful retrieval
  - Non-existent transaction handling

#### 5. Achievement Operations
- **Get Achievements** (`GET /api/v1/achievements/{user_id}`)
  - Successful retrieval
  - Non-existent user handling

#### 6. Analytics Operations
- **Get Analytics** (`GET /api/v1/admin/analytics`)
  - Total transactions
  - Total volume
  - Active wallets
  - Top earners

#### 7. Utility Functions
- `resolveWalletAddress` - Wallet address resolution
- `sendError` - Error response formatting
- `sendSuccess` - Success response formatting

#### 8. Performance Testing
- Balance retrieval performance (< 100ms)
- Token listing performance (< 100ms)
- Single token retrieval performance (< 50ms)

#### 9. Concurrency Testing
- Concurrent token retrievals (10 concurrent, 50 iterations)

## Test Helpers

### Setup Functions
- `setupTestLogger()` - Configures test logging
- `setupTestDB(t)` - Creates isolated test database
- `cleanupTestData(t, db)` - Removes test data

### Data Creation
- `createTestToken(t, db, householdID)` - Creates test token
- `createTestWallet(t, db, householdID)` - Creates test wallet
- `createTestBalance(t, db, walletID, tokenID, amount)` - Creates test balance
- `createTestTransaction(t, db, from, to, tokenID, amount)` - Creates test transaction

### Request Helpers
- `makeHTTPRequest(req)` - Creates HTTP test request
- `assertJSONResponse(t, w, status, fields)` - Validates JSON response
- `assertJSONArray(t, w, status)` - Validates array response
- `assertErrorResponse(t, w, status, message)` - Validates error response

### Test Data Generators
- `TestData.TokenRequest(symbol, name, type)` - Token creation request
- `TestData.WalletRequest(type)` - Wallet creation request
- `TestData.MintRequest(tokenID, toWallet, amount)` - Mint request
- `TestData.TransferRequest(from, to, tokenID, amount)` - Transfer request

## Test Patterns

### Error Testing
```go
patterns := NewTestScenarioBuilder().
    AddInvalidJSON("POST", "/api/v1/tokens/create").
    AddMissingRequiredField("POST", "/api/v1/tokens/create", "symbol").
    AddNonExistentResource("GET", "/api/v1/tokens", "token").
    Build()
```

### Performance Testing
```go
RunPerformanceTest(t, PerformanceTestPattern{
    Name:        "Operation_Performance",
    MaxDuration: 100 * time.Millisecond,
    Execute:     func(t *testing.T, data interface{}) time.Duration { ... },
})
```

### Concurrency Testing
```go
RunConcurrencyTest(t, ConcurrencyTestPattern{
    Name:        "Concurrent_Operations",
    Concurrency: 10,
    Iterations:  50,
    Execute:     func(t *testing.T, data interface{}, i int) error { ... },
})
```

## Running Tests

### Run All Tests
```bash
cd /home/matthalloran8/Vrooli/scenarios/token-economy
make test
```

### Run Unit Tests Only
```bash
cd api
go test -v -cover
```

### Run Specific Test
```bash
cd api
go test -v -run TestHealthHandler
```

### Generate Coverage Report
```bash
cd api
go test -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Run Performance Tests
```bash
cd api
go test -v -run TestPerformance
```

### Run Concurrency Tests
```bash
cd api
go test -v -run TestConcurrency
```

## Coverage Goals

- **Target**: 80% code coverage
- **Minimum**: 50% code coverage (error threshold)
- **Warning**: Below 80% coverage

## Test Database

Tests use a dedicated test database to avoid polluting production data:
- Database: `token_economy` (or `$POSTGRES_DB`)
- Household ID for tests: `00000000-0000-0000-0000-000000000099`
- Automatic cleanup before and after tests

## Best Practices

1. **Isolation**: Each test should be independent
2. **Cleanup**: Always use `defer` for cleanup
3. **Assertions**: Test both status code AND response body
4. **Edge Cases**: Test invalid inputs, missing data, non-existent resources
5. **Performance**: Validate response times for critical endpoints
6. **Concurrency**: Test thread safety of shared resources

## Integration with Vrooli Testing Infrastructure

The test suite integrates with Vrooli's centralized testing infrastructure:
- Uses `scripts/scenarios/testing/shell/phase-helpers.sh`
- Sources `scripts/scenarios/testing/unit/run-all.sh`
- Follows phase-based test organization
- Reports coverage with standardized thresholds

## Next Steps for Test Enhancement

1. Add more edge case testing for complex business logic
2. Implement integration tests with actual database migrations
3. Add load testing for high-traffic scenarios
4. Implement fuzz testing for input validation
5. Add mutation testing to verify test effectiveness
