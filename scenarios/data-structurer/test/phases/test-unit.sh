#!/bin/bash
# Data Structurer Unit Tests
# Tests individual API functions and data processing logic

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "=== Data Structurer Unit Tests ==="
echo "Scenario Root: $SCENARIO_ROOT"

# Test 1: Go API Unit Tests
echo ""
echo "Running Go API unit tests..."
if [ -d "$SCENARIO_ROOT/api" ]; then
    cd "$SCENARIO_ROOT/api"

    # Run tests with coverage
    go test -v ./... -short -coverprofile=coverage.out 2>&1 | tee test-output.txt

    # Display coverage summary
    if [ -f coverage.out ]; then
        echo ""
        echo "Coverage Summary:"
        go tool cover -func=coverage.out | tail -1
    fi

    # Check test results
    if grep -q "FAIL" test-output.txt; then
        echo "❌ Go unit tests failed"
        exit 1
    fi

    echo "✅ Go API unit tests passed"
else
    echo "⚠️  API directory not found, skipping Go tests"
fi

# Test 2: CLI BATS Tests
echo ""
echo "Running CLI BATS tests..."
if [ -f "$SCENARIO_ROOT/cli/data-structurer.bats" ]; then
    cd "$SCENARIO_ROOT/cli"

    # Check if bats is available
    if command -v bats &> /dev/null; then
        bats data-structurer.bats
        echo "✅ CLI BATS tests passed"
    else
        echo "⚠️  BATS not installed, skipping CLI tests"
    fi
else
    echo "⚠️  CLI BATS tests not found, skipping"
fi

echo ""
echo "✅ Unit tests completed successfully"
