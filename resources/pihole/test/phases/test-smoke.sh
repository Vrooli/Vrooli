#!/usr/bin/env bash
# Pi-hole Smoke Tests - Quick health validation
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Source core library
source "${RESOURCE_DIR}/lib/core.sh"

echo "Pi-hole Smoke Tests"
echo "==================="

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper function
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -n "Testing: $test_name... "
    if eval "$test_command" >/dev/null 2>&1; then
        echo "PASS"
        ((TESTS_PASSED++))
    else
        echo "FAIL"
        ((TESTS_FAILED++))
    fi
}

# Test 1: Container exists
run_test "Container exists" "docker ps -a --format '{{.Names}}' | grep -q '^${CONTAINER_NAME}$'"

# Test 2: Container is running
run_test "Container running" "docker ps --format '{{.Names}}' | grep -q '^${CONTAINER_NAME}$'"

# Test 3: DNS port is listening
run_test "DNS port (53) listening" "timeout 5 nc -z localhost 53"

# Test 4: API port is listening
run_test "API port (8087) listening" "timeout 5 nc -z localhost 8087"

# Test 5: Health check passes
run_test "Health check" "check_health"

# Test 6: API status endpoint
run_test "API status endpoint" "timeout 5 curl -sf 'http://localhost:8087/admin/api.php?status'"

# Test 7: Container logs accessible
run_test "Container logs" "docker logs --tail 1 '${CONTAINER_NAME}'"

# Test 8: Data directory exists
run_test "Data directory exists" "test -d '${PIHOLE_DATA_DIR}'"

# Summary
echo ""
echo "Results:"
echo "  Passed: $TESTS_PASSED"
echo "  Failed: $TESTS_FAILED"

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo ""
    echo "✓ All smoke tests passed!"
    exit 0
else
    echo ""
    echo "✗ Some smoke tests failed"
    exit 1
fi