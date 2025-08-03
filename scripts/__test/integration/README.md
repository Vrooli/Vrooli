# Integration Test Framework

## Overview

This directory contains the consolidated integration testing framework for Vrooli services and resources. Unlike unit tests (which use mocks), these tests interact with real services to validate actual functionality.

## Directory Structure

```
integration/
├── run.sh              # Main test runner
├── health-check.sh     # Service health validation
├── data/              # Test data and fixtures
├── services/          # Individual service tests
│   ├── ollama.sh
│   ├── whisper.sh
│   └── ...
└── scenarios/         # Multi-service integration tests
    ├── multi-ai-pipeline.sh
    └── ...
```

## Quick Start

### Run All Integration Tests
```bash
./run.sh
```

### Run Health Checks Only
```bash
./run.sh --health
# or directly:
./health-check.sh
```

### Test Specific Service
```bash
./run.sh --service ollama
```

### Run Specific Scenario
```bash
./run.sh --scenario multi-ai-pipeline
```

### Advanced Options
```bash
# Run with verbose output
./run.sh --verbose

# Run tests in parallel
./run.sh --parallel

# Set custom timeout (in seconds)
./run.sh --timeout 600

# Combine options
./run.sh --service ollama --verbose --timeout 120
```

## Writing New Tests

### Service Tests

Service tests validate individual service functionality. Create a new file in `services/`:

```bash
#!/usr/bin/env bash
# services/myservice.sh

set -euo pipefail

# Configuration
readonly SERVICE_NAME="myservice"
readonly BASE_URL="${MYSERVICE_URL:-http://localhost:8080}"

# Test tracking
TESTS_PASSED=0
TESTS_FAILED=0

# Test functions
test_service_health() {
    echo -n "Testing service health... "
    if curl -sf "$BASE_URL/health" > /dev/null; then
        echo "PASS"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo "FAIL"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Main
main() {
    echo "Testing $SERVICE_NAME"
    test_service_health
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        exit 0
    else
        exit 1
    fi
}

main
```

### Scenario Tests

Scenarios test integration between multiple services. Create in `scenarios/`:

```bash
#!/usr/bin/env bash
# scenarios/myscenario.sh

set -euo pipefail

# Test multi-service workflow
step_1_service_a() {
    # Test service A
}

step_2_service_b() {
    # Test service B with data from A
}

# Main workflow
main() {
    step_1_service_a
    step_2_service_b
    # Validate results
}

main
```

## Health Checks

The health check system validates service availability before running tests:

### Supported Check Types
- **HTTP**: Check HTTP/HTTPS endpoints
- **TCP**: Check port availability
- **Docker**: Check container status
- **Process**: Check if process is running

### Adding New Health Checks

Edit `health-check.sh` and add to the `SERVICE_CHECKS` array:

```bash
["myservice"]="http|http://localhost:8080/health"
["mydb"]="tcp|localhost|5432"
["mycontainer"]="docker|mycontainer_name"
```

### Required vs Optional Services

Required services (Postgres, Redis) must be running for tests to proceed. Optional services generate warnings but don't block test execution.

## Test Data

Place test fixtures and data files in the `data/` directory:

```
data/
├── audio/          # Test audio files
├── images/         # Test images
├── json/           # JSON fixtures
└── sql/            # SQL test data
```

## Best Practices

1. **Real Services Only**: Integration tests should use real services, not mocks
2. **Idempotent Tests**: Tests should clean up after themselves
3. **Timeout Handling**: Always set appropriate timeouts for long operations
4. **Error Reporting**: Provide clear error messages and debugging info
5. **Service Dependencies**: Check service health before testing functionality

## Environment Variables

Tests respect these environment variables:

```bash
# Service URLs
OLLAMA_URL=http://localhost:11434
WHISPER_URL=http://localhost:9000
MINIO_URL=http://localhost:9001

# Test configuration
SCENARIO_TIMEOUT=60
RETRY_COUNT=3
RETRY_DELAY=2
```

## Troubleshooting

### Service Not Available
```bash
# Check if service is running
./health-check.sh --verbose

# Check specific service
docker ps | grep service_name
```

### Test Timeouts
```bash
# Increase timeout for slow operations
./run.sh --timeout 600
```

### Debugging Failed Tests
```bash
# Run with verbose output
./run.sh --service ollama --verbose

# Check service logs
docker logs ollama_container
```

## Migration Notes

This framework consolidates tests previously scattered across:
- `resources/single/` → `services/`
- `resources/fixtures/workflows/` → `scenarios/`
- Various health checks → `health-check.sh`

The new structure provides:
- Centralized test runner
- Standardized health checks
- Clear separation between service and scenario tests
- Better error reporting and debugging