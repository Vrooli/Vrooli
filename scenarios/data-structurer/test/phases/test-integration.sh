#!/bin/bash
# Data Structurer Integration Tests
# Tests API endpoints, resource integration, and end-to-end workflows

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TESTS_DIR="$SCENARIO_ROOT/tests"

echo "=== Data Structurer Integration Tests ==="
echo "Scenario Root: $SCENARIO_ROOT"

# Check if API is running
API_PORT="${API_PORT:-15770}"
API_BASE_URL="http://localhost:$API_PORT"

echo ""
echo "Checking API availability at $API_BASE_URL..."
if ! curl -sf "$API_BASE_URL/health" > /dev/null 2>&1; then
    echo "❌ API is not running at $API_BASE_URL"
    echo "Please start the API with: cd $SCENARIO_ROOT && make start"
    exit 1
fi
echo "✅ API is running"

# Test 1: Schema API Integration
echo ""
echo "Running Schema API integration tests..."
if [ -x "$TESTS_DIR/test-schema-api.sh" ]; then
    bash "$TESTS_DIR/test-schema-api.sh"
    echo "✅ Schema API tests passed"
else
    echo "❌ Schema API test script not found or not executable"
    exit 1
fi

# Test 2: Processing Pipeline Integration
echo ""
echo "Running Processing Pipeline integration tests..."
if [ -x "$TESTS_DIR/test-processing.sh" ]; then
    bash "$TESTS_DIR/test-processing.sh"
    echo "✅ Processing Pipeline tests passed"
else
    echo "❌ Processing Pipeline test script not found or not executable"
    exit 1
fi

# Test 3: Resource Integration
echo ""
echo "Running Resource integration tests..."
if [ -x "$TESTS_DIR/test-resource-integration.sh" ]; then
    bash "$TESTS_DIR/test-resource-integration.sh"
    echo "✅ Resource integration tests passed"
else
    echo "❌ Resource integration test script not found or not executable"
    exit 1
fi

echo ""
echo "✅ Integration tests completed successfully"
