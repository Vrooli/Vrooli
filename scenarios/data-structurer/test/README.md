# Data Structurer Test Suite

Comprehensive automated testing for the data-structurer scenario, covering unit tests, integration tests, and end-to-end workflows.

## Test Structure

```
test/
├── phases/
│   ├── test-unit.sh           # Unit tests (Go API + CLI BATS)
│   └── test-integration.sh    # Integration tests (API + Resources)
├── cli/                        # CLI-specific test utilities
├── utils/                      # Shared test utilities
└── README.md                   # This file

api/
└── main_test.go               # Go unit tests

cli/
└── data-structurer.bats       # CLI BATS tests

tests/
├── test-schema-api.sh         # Schema API integration tests
├── test-processing.sh         # Processing pipeline tests
└── test-resource-integration.sh  # Resource dependency tests
```

## Running Tests

### Quick Start

```bash
# Run all tests
cd /path/to/data-structurer
make test

# Run specific test phases
make test-unit              # Unit tests only
make test-integration       # Integration tests only
```

### Manual Execution

#### Unit Tests

```bash
# Go API unit tests
cd api
go test -v ./... -short

# Go tests with coverage
go test -v ./... -short -coverprofile=coverage.out
go tool cover -html=coverage.out

# CLI BATS tests
cd cli
bats data-structurer.bats
```

#### Integration Tests

```bash
# Ensure API is running first
make start

# Run integration tests
bash test/phases/test-integration.sh

# Or run individual test scripts
bash tests/test-schema-api.sh
bash tests/test-processing.sh
bash tests/test-resource-integration.sh
```

## Test Coverage

### Unit Tests (api/main_test.go)

**Data Structure Tests:**
- ✅ Schema struct validation
- ✅ ProcessedData struct validation
- ✅ ProcessingRequest validation with multiple input types
- ✅ SchemaTemplate structure validation
- ✅ HealthResponse structure validation

**Business Logic Tests:**
- ✅ Healthy dependencies counter
- ✅ Confidence score boundary validation (0.0 - 1.0)
- ✅ UUID generation and uniqueness
- ✅ Batch mode flag handling
- ✅ Empty schema definition handling

**Serialization Tests:**
- ✅ JSON marshaling/unmarshaling for all data structures
- ✅ Timestamp format preservation
- ✅ ProcessingRequest deserialization
- ✅ Error response structure

**Validation Tests:**
- ✅ Input type validation (text, file, url)
- ✅ Invalid input type rejection
- ✅ ProcessingResponse error handling

**Performance Tests:**
- ✅ Schema JSON marshaling benchmark
- ✅ ProcessingRequest creation benchmark

### CLI Tests (cli/data-structurer.bats)

**Basic Commands (10 tests):**
- ✅ Help display with no arguments
- ✅ Help command
- ✅ Version information
- ✅ Version JSON format
- ✅ Status information
- ✅ Status JSON format

**Schema Management (11 tests):**
- ✅ List schemas
- ✅ List schemas in JSON
- ✅ Create schema with valid file
- ✅ Schema name requirement validation
- ✅ Schema file requirement validation
- ✅ Invalid schema file rejection
- ✅ Get schema by ID
- ✅ Handle non-existent schema
- ✅ Delete schema
- ✅ Update schema (if implemented)

**Schema Templates (4 tests):**
- ✅ List templates
- ✅ List templates in JSON
- ✅ Get template by ID
- ✅ Create schema from template

**Data Processing (5 tests):**
- ✅ Process text data
- ✅ Schema ID requirement
- ✅ Input data requirement
- ✅ Get processed data
- ✅ Handle processing errors

**Processing Jobs (3 tests):**
- ✅ List processing jobs
- ✅ List jobs in JSON
- ✅ Get job by ID

**Error Handling (4 tests):**
- ✅ Unknown command handling
- ✅ Missing required flags
- ✅ UUID format validation
- ✅ API connection failure

**Output Formats (3 tests):**
- ✅ JSON flag support for all commands
- ✅ Human-readable default output
- ✅ Verbose mode support

### Integration Tests

**Schema API Tests (tests/test-schema-api.sh):**
- ✅ Health check endpoint
- ✅ List schemas (empty and populated)
- ✅ Create schema with valid definition
- ✅ Get created schema by ID
- ✅ Delete schema
- ✅ List schema templates
- ✅ Invalid schema creation (400 error)
- ✅ Non-existent schema retrieval (404 error)

**Processing Pipeline Tests (tests/test-processing.sh):**
- ✅ API health verification
- ✅ Create test schema for processing
- ✅ Process text data
- ✅ Retrieve processed data
- ✅ Batch processing support check
- ✅ Processing status verification

**Resource Integration Tests (tests/test-resource-integration.sh):**
- ✅ PostgreSQL connectivity
- ✅ Ollama availability
- ✅ Unstructured-io availability
- ✅ N8N availability
- ✅ Qdrant availability (optional)
- ✅ API resource dependency status
- ✅ Database schema validation

## Test Requirements

### Prerequisites

**Required Tools:**
- Go 1.21+ (for API unit tests)
- BATS (Bash Automated Testing System)
- jq (JSON processor)
- curl
- PostgreSQL client tools

**Required Resources:**
- PostgreSQL (running)
- Ollama (running)
- N8N (running)
- Unstructured-io (running)

**Installation:**

```bash
# Install BATS
git clone https://github.com/bats-core/bats-core.git
cd bats-core
sudo ./install.sh /usr/local

# Install BATS helpers (for CLI tests)
cd cli
git clone https://github.com/bats-core/bats-support.git test_helper/bats-support
git clone https://github.com/bats-core/bats-assert.git test_helper/bats-assert

# Install Go testing dependencies
cd api
go get github.com/stretchr/testify/assert
go mod tidy
```

### Environment Variables

```bash
# API configuration
export API_PORT=15770
export DATA_STRUCTURER_API_URL="http://localhost:15770"

# Database configuration
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_DB=vrooli
export POSTGRES_USER=vrooli
export POSTGRES_PASSWORD=your_password

# Resource ports (auto-configured by Vrooli)
export RESOURCE_PORTS[ollama]=11434
export RESOURCE_PORTS[n8n]=5678
export RESOURCE_PORTS[unstructured-io]=11450
export RESOURCE_PORTS[qdrant]=6333
```

## Test Scenarios

### Scenario 1: Fresh Installation Validation

Tests that run after initial setup to verify the system is properly configured.

```bash
# 1. Verify structure
ls -la .vrooli/service.json api/main.go cli/data-structurer

# 2. Verify resources
vrooli resource status postgres
vrooli resource status ollama

# 3. Run unit tests (no API required)
cd api && go test -v ./... -short

# 4. Start API and run integration tests
make start
make test-integration
```

### Scenario 2: Continuous Integration

Tests suitable for CI/CD pipelines.

```bash
# CI-friendly test execution
set -e

# Unit tests (fast, no dependencies)
cd api && go test -v ./... -short -coverprofile=coverage.out

# Generate coverage report
go tool cover -func=coverage.out

# Fail if coverage is below threshold (e.g., 60%)
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
if (( $(echo "$COVERAGE < 60" | bc -l) )); then
    echo "Coverage $COVERAGE% is below 60% threshold"
    exit 1
fi
```

### Scenario 3: Pre-Deployment Validation

Comprehensive tests before deploying to production.

```bash
# Full test suite
make test

# Performance tests
cd api
go test -bench=. -benchmem

# Load testing (if available)
bash tests/load-test.sh

# Security scanning
gosec ./...
```

### Scenario 4: Regression Testing

Tests to run after code changes to ensure no existing functionality broke.

```bash
# Run all integration tests
bash test/phases/test-integration.sh

# Verify specific workflows
bash tests/test-schema-api.sh
bash tests/test-processing.sh

# CLI compatibility tests
cd cli && bats data-structurer.bats
```

## Writing New Tests

### Adding Go Unit Tests

```go
// api/my_feature_test.go
package main

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestMyNewFeature(t *testing.T) {
    // Arrange
    input := "test data"

    // Act
    result := myNewFunction(input)

    // Assert
    assert.NotNil(t, result)
    assert.Equal(t, "expected", result)
}
```

### Adding CLI BATS Tests

```bash
# cli/data-structurer.bats
@test "My new CLI feature works" {
    run bash "$CLI_SCRIPT" new-command --flag value
    [ "$status" -eq 0 ]
    [[ "$output" =~ "expected output" ]]
}
```

### Adding Integration Tests

```bash
# tests/test-my-feature.sh
#!/bin/bash
set -euo pipefail

API_BASE_URL="${API_BASE_URL:-http://localhost:15770}"

# Test new endpoint
response=$(curl -sf "$API_BASE_URL/api/v1/new-endpoint")
echo "$response" | jq -e '.status == "success"'

echo "✅ New feature integration test passed"
```

## Troubleshooting

### Common Issues

**1. BATS tests skip with "API server not running"**

```bash
# Start the API first
make start

# Verify API is running
curl http://localhost:15770/health

# Run tests again
cd cli && bats data-structurer.bats
```

**2. Go tests fail with "cannot connect to database"**

```bash
# Verify PostgreSQL is running
vrooli resource status postgres

# Check database connection
resource-postgres exec -c "SELECT 1;"

# Verify environment variables
echo $POSTGRES_HOST $POSTGRES_PORT
```

**3. Integration tests timeout**

```bash
# Increase health check timeout
export HEALTH_CHECK_TIMEOUT=30

# Or check resource status
vrooli resource status ollama
vrooli resource status unstructured-io
```

**4. Coverage report not generated**

```bash
# Ensure tests run with coverage flag
cd api
go test ./... -coverprofile=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html
```

## Performance Metrics

### Expected Test Execution Times

| Test Suite | Duration | Notes |
|------------|----------|-------|
| Go Unit Tests | < 5s | Fast, no external dependencies |
| CLI BATS Tests | 30-60s | Requires running API |
| Schema API Integration | 10-15s | Requires API + PostgreSQL |
| Processing Pipeline | 15-30s | Requires all resources |
| Resource Integration | 10-20s | Checks all resource health |
| **Full Test Suite** | **2-3 minutes** | All tests combined |

### Coverage Targets

- **Go API Unit Tests:** > 60% code coverage
- **CLI Commands:** > 80% command coverage
- **Integration Tests:** 100% critical path coverage
- **Error Handling:** 100% error path coverage

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Data Structurer Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_DB: vrooli
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
        ports:
          - 5432:5432

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install dependencies
        run: |
          sudo apt-get update
          sudo apt-get install -y jq curl
          npm install -g bats

      - name: Run unit tests
        run: |
          cd api
          go test -v ./... -short -coverprofile=coverage.out
          go tool cover -func=coverage.out

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./api/coverage.out
```

## Contributing

When adding new features to data-structurer:

1. **Write unit tests first** (TDD approach)
2. **Add integration tests** for new endpoints
3. **Update CLI BATS tests** if CLI changes
4. **Document test scenarios** in this README
5. **Verify all tests pass** before committing

## References

- [BATS Documentation](https://bats-core.readthedocs.io/)
- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Assert](https://pkg.go.dev/github.com/stretchr/testify/assert)
- [Vrooli Testing Standards](../../docs/testing/README.md)
