# Quick Start: Testing in 5 Minutes

## Prerequisites

```bash
# Ensure you have the Vrooli CLI installed
vrooli help

# Install BATS for CLI testing (if needed)
npm install -g bats
```

## Step 1: Understand Your Testing Context

What are you testing?

- **Scenario**: Complete application with API/UI/CLI → Continue below
- **Resource**: Integration like PostgreSQL, Redis → See [Resource Testing](../architecture/STRATEGY.md)
- **Just Go/Node/Python code**: → See [Scenario Unit Testing](scenario-unit-testing.md)

## Step 2: Run Existing Tests

```bash
# From your scenario directory
cd scenarios/my-scenario/

# Run all tests (recommended)
make test

# Or run specific phase
./test/phases/test-unit.sh
```

## Step 3: Add Your First Test

### For Go Code

Create `api/handler_test.go`:
```go
package main

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
)

type HealthResponse struct {
    Status    string    `json:"status"`
    Service   string    `json:"service"`
    Timestamp time.Time `json:"timestamp"`
    Readiness bool      `json:"readiness"`
}

func TestHealthHandler(t *testing.T) {
    req, _ := http.NewRequest("GET", "/health", nil)
    rr := httptest.NewRecorder()

    handler := http.HandlerFunc(healthHandler)
    handler.ServeHTTP(rr, req)

    if status := rr.Code; status != http.StatusOK {
        t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
    }

    var resp HealthResponse
    if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
        t.Fatalf("failed to parse health response: %v", err)
    }

    if resp.Status != "healthy" {
        t.Errorf("unexpected status: got %q want %q", resp.Status, "healthy")
    }
    if resp.Service == "" {
        t.Error("service should be populated")
    }
    if resp.Timestamp.IsZero() {
        t.Error("timestamp should be set")
    }
    if !resp.Readiness {
        t.Error("readiness should be true when the service is ready")
    }
}
```

Run it:
```bash
cd api/
go test -v
```

### For CLI Commands (BATS)

⚠️ **CRITICAL**: Read [Safety Guidelines](../safety/GUIDELINES.md) first!

Create `test/cli/my-cli.bats`:
```bash
#!/usr/bin/env bats

setup() {
    # Set variables BEFORE skip conditions (safety critical!)
    export TEST_DIR="/tmp/my-cli-test-$$"
    
    # Check prerequisites
    if ! command -v my-cli >/dev/null 2>&1; then
        skip "my-cli not installed"
    fi
    
    mkdir -p "$TEST_DIR"
}

teardown() {
    # Safe cleanup with validation
    if [ -n "${TEST_DIR:-}" ] && [[ "$TEST_DIR" =~ ^/tmp/ ]]; then
        rm -rf "$TEST_DIR" 2>/dev/null || true
    fi
}

@test "CLI shows help" {
    run my-cli --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI processes file" {
    echo "test content" > "$TEST_DIR/input.txt"
    run my-cli process "$TEST_DIR/input.txt"
    [ "$status" -eq 0 ]
    [ -f "$TEST_DIR/output.txt" ]
}
```

Run it:
```bash
bats test/cli/my-cli.bats
```

### For Integration Tests

Create `test/phases/test-integration.sh`:
```bash
#!/bin/bash
set -euo pipefail

# Source testing libraries
APP_ROOT="${APP_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)}"
source "$APP_ROOT/scripts/scenarios/testing/shell/connectivity.sh"

echo "=== Integration Tests ==="

# Test API connectivity
if testing::connectivity::test_api; then
    echo "✅ API is healthy"
else
    echo "❌ API is not responding"
    exit 1
fi

# Test specific endpoint
API_URL=$(testing::connectivity::get_api_url)
response=$(curl -s "${API_URL}/users")
if [[ "$response" =~ "users" ]]; then
    echo "✅ Users endpoint working"
else
    echo "❌ Users endpoint failed"
    exit 1
fi
```

## Step 4: Check Test Safety

**Always run the safety linter before committing:**

```bash
scripts/scenarios/testing/lint-tests.sh test/
```

Fix any issues it finds - they could cause data loss!

## Step 5: Run Full Test Suite

```bash
# From scenario directory
make test

# Or use the comprehensive runner
./test/run-tests.sh
```

## Expected Output

```
=== Running Comprehensive Tests for my-scenario ===

[Phase 1/6] Structure Validation...
✅ All required files present

[Phase 2/6] Dependency Checks...
✅ All dependencies available

[Phase 3/6] Unit Tests...
Running Go tests...
PASS: handler_test.go
Coverage: 75.3%
✅ Unit tests passed

[Phase 4/6] Integration Tests...
✅ API responding on port 8080
✅ All endpoints functional

[Phase 5/6] Business Logic Tests...
✅ Core workflows validated

[Phase 6/6] Performance Tests...
✅ Response times within limits

=== All Tests Passed! ===
Total time: 4m 32s
```

## Common Issues

### "CLI not installed"
```bash
# Install the CLI first
cd cli/
./install.sh
```

### "Port already in use"
```bash
# Stop the running scenario
vrooli scenario stop my-scenario

# Or find and kill the process
lsof -i :8080
kill <PID>
```

### Tests deleting files
**STOP!** You have a dangerous test. Read [Safety Guidelines](../safety/GUIDELINES.md) immediately.

## Next Steps

1. 📈 **Increase Coverage**: Add more test cases
2. 🔄 **Add CI/CD**: Integrate with GitHub Actions
3. 📊 **Track Metrics**: Monitor coverage trends
4. 🏆 **Achieve Gold Standard**: See [Visited Tracker Example](../../../scenarios/visited-tracker/api/TESTING_GUIDE.md)

## Quick Reference

```bash
# Run all tests
make test

# Run specific phase
./test/phases/test-unit.sh

# Check safety
scripts/scenarios/testing/lint-tests.sh

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run BATS tests
bats test/cli/*.bats

# Debug failing test
bash -x ./test/phases/test-integration.sh
```

## Getting Help

- **Safety Issues**: [Safety Guidelines](../safety/GUIDELINES.md)
- **Debugging**: [Troubleshooting Guide](troubleshooting.md)
- **Examples**: [Gold Standard Implementations](../reference/examples.md)
- **Architecture**: [Phased Testing](../architecture/PHASED_TESTING.md)
