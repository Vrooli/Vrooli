#!/usr/bin/env bash
# Splink Smoke Tests - Quick health validation (<30s)

set -euo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(dirname "$SCRIPT_DIR")"
RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Test results
PASSED=0
FAILED=0

# Helper function for test assertions
assert() {
    local test_name="$1"
    local condition="$2"
    
    echo -n "  $test_name... "
    if eval "$condition"; then
        echo "✓"
        ((PASSED++))
    else
        echo "✗"
        ((FAILED++))
    fi
}

echo "Running Splink Smoke Tests"
echo "=========================="

# Test 1: Service health check
echo "1. Health Check Tests"
assert "Service responds to health check" \
    "timeout 5 curl -sf http://localhost:${SPLINK_PORT}/health &>/dev/null"

assert "Health check returns quickly (<1s)" \
    "timeout 1 curl -sf http://localhost:${SPLINK_PORT}/health &>/dev/null"

# Test 2: API availability
echo ""
echo "2. API Availability Tests"
assert "API root endpoint accessible" \
    "curl -sf http://localhost:${SPLINK_PORT}/ &>/dev/null || curl -sf http://localhost:${SPLINK_PORT}/docs &>/dev/null"

# Test 3: Container/Process status (if using Docker)
echo ""
echo "3. Process Status Tests"
if command -v docker &>/dev/null; then
    assert "Docker container running" \
        "docker ps --format '{{.Names}}' | grep -q 'vrooli-splink'"
else
    assert "Splink process running" \
        "pgrep -f 'splink|uvicorn.*8096' &>/dev/null"
fi

# Test 4: Port binding
echo ""
echo "4. Network Tests"
assert "Port ${SPLINK_PORT} is listening" \
    "netstat -tln 2>/dev/null | grep -q ':${SPLINK_PORT}' || ss -tln | grep -q ':${SPLINK_PORT}'"

# Test 5: Basic configuration
echo ""
echo "5. Configuration Tests"
assert "Configuration files exist" \
    "[[ -f ${RESOURCE_DIR}/config/defaults.sh && -f ${RESOURCE_DIR}/config/runtime.json ]]"

assert "CLI script is executable" \
    "[[ -x ${RESOURCE_DIR}/cli.sh ]]"

# Summary
echo ""
echo "=========================="
echo "Smoke Test Summary:"
echo "  Passed: $PASSED"
echo "  Failed: $FAILED"
echo "=========================="

if [[ $FAILED -gt 0 ]]; then
    exit 1
fi

exit 0