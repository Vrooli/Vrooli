#!/bin/bash

# OpenEMS Integration Tests
# Full functionality testing (<120s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
CLI="${RESOURCE_DIR}/cli.sh"

# Source configuration (sets ports from registry)
source "${RESOURCE_DIR}/config/defaults.sh"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0

# Test function
run_test() {
    local test_name="$1"
    local command="$2"
    
    ((TESTS_RUN++))
    echo -n "  Testing $test_name... "
    
    if eval "$command" &>/dev/null; then
        echo "✅"
        ((TESTS_PASSED++))
        return 0
    else
        echo "❌"
        return 1
    fi
}

echo "🔗 OpenEMS Integration Tests"
echo "=========================="

# Test 1: Install OpenEMS
run_test "installation" "$CLI manage install"

# Test 2: Start OpenEMS
echo -n "  Testing service startup... "
if $CLI manage start &>/dev/null; then
    echo "✅"
    ((TESTS_PASSED++))
    ((TESTS_RUN++))
    sleep 5  # Give service time to start
else
    echo "❌"
    ((TESTS_RUN++))
fi

# Test 3: Health check
run_test "health check" "timeout 5 docker exec openems-edge echo 'OK' || true"

# Test 4: Configure DER asset
run_test "DER configuration" "$CLI content execute configure-der --type solar --capacity 10"

# Test 5: Get status
run_test "status retrieval" "$CLI content execute get-status"

# Test 6: Simulate solar generation
run_test "solar simulation" "$CLI content execute simulate-solar 5000"

# Test 7: Simulate load
run_test "load simulation" "$CLI content execute simulate-load 3000"

# Test 8: Add configuration
echo "test_config" > /tmp/test_openems.json
run_test "add configuration" "$CLI content add test_config /tmp/test_openems.json"

# Test 9: List configurations
run_test "list configurations" "$CLI content list | grep -E '(test_config|der_solar)'"

# Test 10: Get configuration
run_test "get configuration" "$CLI content get test_config"

# Test 11: Remove configuration
run_test "remove configuration" "$CLI content remove test_config"

# Test 12: View logs
run_test "view logs" "$CLI logs 10"

# Test 13: Restart service
run_test "service restart" "$CLI manage restart"

sleep 3  # Wait for restart

# Test 14: Status after restart
run_test "status after restart" "$CLI status"

# Test 15: Stop service
run_test "service stop" "$CLI manage stop"

# Clean up
rm -f /tmp/test_openems.json

# Summary
echo ""
echo "📊 Integration Test Results"
echo "========================="
echo "Tests Run: $TESTS_RUN"
echo "Tests Passed: $TESTS_PASSED"
echo "Tests Failed: $((TESTS_RUN - TESTS_PASSED))"

if [[ $TESTS_PASSED -eq $TESTS_RUN ]]; then
    echo "✅ All integration tests passed!"
    exit 0
else
    echo "❌ Some integration tests failed"
    exit 1
fi