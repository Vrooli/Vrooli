#!/bin/bash
set -euo pipefail

echo "=== Integration Tests Phase ==="
echo "Starting integration tests..."

# Test counters
error_count=0
test_count=0

echo "Testing API integration..."
if curl -sf --max-time 5 "http://localhost:17695/health" >/dev/null 2>&1; then
    echo "API integration tests passed"
    test_count=$((test_count + 1))
else
    echo "API integration tests failed"
    error_count=$((error_count + 1))
fi

echo "Testing CLI integration..."
# Get scenario directory
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CLI_BINARY="$SCENARIO_DIR/cli/visited-tracker"

if [ -f "$CLI_BINARY" ] && [ -x "$CLI_BINARY" ]; then
    if "$CLI_BINARY" version >/dev/null 2>&1; then
        echo "CLI integration tests passed"
        test_count=$((test_count + 1))
    else
        echo "CLI integration tests failed"
        error_count=$((error_count + 1))
    fi
else
    echo "CLI not available at $CLI_BINARY - skipping"
fi

echo "Testing database integration..."
if command -v resource-postgres >/dev/null 2>&1; then
    if resource-postgres test smoke >/dev/null 2>&1; then
        echo "Database integration tests passed"
        test_count=$((test_count + 1))
    else
        echo "Database integration tests failed"
        error_count=$((error_count + 1))
    fi
else
    echo "Database resource not available - skipping"
fi

echo "Summary: $test_count passed, $error_count failed"

if [ $error_count -eq 0 ]; then
    echo "SUCCESS: All integration tests passed"
    exit 0
else
    echo "ERROR: Some integration tests failed"
    exit 1
fi