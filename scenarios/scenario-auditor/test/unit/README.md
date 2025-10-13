# Unit Tests

## Overview
Unit tests for scenario-auditor are co-located with the source code following Go conventions:
- `api/*_test.go` - API handler and business logic tests
- `api/rules/*_test.go` - Individual rule validation tests (run with `-tags=ruletests`)
- `api/internal/*_test.go` - Internal package tests

## Running Tests

### All Unit Tests
```bash
cd test && ./phases/test-unit.sh
```

### Specific Package Tests
```bash
cd api && go test ./... -v
cd api && go test -tags=ruletests ./rules/... -v  # Rule tests
```

## Test Coverage

Current coverage (as of 2025-10-05):
- **Rules package**: 75.2% coverage
- **Overall**: 34.4% coverage
- **Target**: >50% coverage ✅

See `test/phases/test-unit.sh` for detailed coverage reporting.

## Test Organization

- **Unit tests**: Co-located `*_test.go` files test individual functions and packages
- **Integration tests**: `test/phases/test-integration.sh` tests API endpoints and CLI
- **Business tests**: `test/phases/test-business.sh` validates P0 requirements
- **Performance tests**: `test/phases/test-performance.sh` checks response times and throughput

## Adding New Tests

1. Create `*_test.go` file next to the code being tested
2. Use standard Go testing patterns
3. For rule tests, add `//go:build ruletests` build tag
4. Run `make test` to verify all test suites pass
