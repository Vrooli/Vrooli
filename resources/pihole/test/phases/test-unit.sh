#!/usr/bin/env bash
# Pi-hole Unit Tests - Library function validation
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

echo "Pi-hole Unit Tests"
echo "=================="

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper function
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -n "Testing: $test_name... "
    if eval "$test_command"; then
        echo "PASS"
        ((TESTS_PASSED++))
    else
        echo "FAIL"
        ((TESTS_FAILED++))
    fi
}

# Test 1: Configuration files exist
run_test "defaults.sh exists" "test -f '${RESOURCE_DIR}/config/defaults.sh'"
run_test "runtime.json exists" "test -f '${RESOURCE_DIR}/config/runtime.json'"
run_test "schema.json exists" "test -f '${RESOURCE_DIR}/config/schema.json'"

# Test 2: Core library exists
run_test "core.sh exists" "test -f '${RESOURCE_DIR}/lib/core.sh'"
run_test "test.sh exists" "test -f '${RESOURCE_DIR}/lib/test.sh'"

# Test 3: CLI script exists and is executable
run_test "cli.sh exists" "test -f '${RESOURCE_DIR}/cli.sh'"
run_test "cli.sh is executable" "test -x '${RESOURCE_DIR}/cli.sh'"

# Test 4: Test scripts exist
run_test "run-tests.sh exists" "test -f '${RESOURCE_DIR}/test/run-tests.sh'"
run_test "test-smoke.sh exists" "test -f '${RESOURCE_DIR}/test/phases/test-smoke.sh'"
run_test "test-integration.sh exists" "test -f '${RESOURCE_DIR}/test/phases/test-integration.sh'"

# Test 5: Port configuration is valid
run_test "DNS port is 53" "[[ '${PIHOLE_DNS_PORT}' == '53' ]]"
run_test "API port is 8087" "[[ '${PIHOLE_API_PORT}' == '8087' ]]"
run_test "DHCP port is 67" "[[ '${PIHOLE_DHCP_PORT}' == '67' ]]"

# Test 6: Container name is set
run_test "Container name defined" "[[ -n '${PIHOLE_CONTAINER_NAME}' ]]"
run_test "Container name correct" "[[ '${PIHOLE_CONTAINER_NAME}' == 'vrooli-pihole' ]]"

# Test 7: Data directory configuration
run_test "Data directory defined" "[[ -n '${PIHOLE_DATA_DIR}' ]]"

# Test 8: Upstream DNS configuration
run_test "Primary DNS set" "[[ '${PIHOLE_UPSTREAM_DNS_1}' == '1.1.1.1' ]]"
run_test "Secondary DNS set" "[[ '${PIHOLE_UPSTREAM_DNS_2}' == '1.0.0.1' ]]"

# Test 9: Feature flags
run_test "Query logging enabled" "[[ '${PIHOLE_QUERY_LOGGING}' == 'true' ]]"
run_test "IPv6 enabled" "[[ '${PIHOLE_ENABLE_IPV6}' == 'true' ]]"
run_test "DHCP disabled by default" "[[ '${PIHOLE_ENABLE_DHCP}' == 'false' ]]"

# Test 10: Runtime.json is valid JSON
run_test "runtime.json valid JSON" "jq -e . '${RESOURCE_DIR}/config/runtime.json' >/dev/null"

# Test 11: Schema.json is valid JSON
run_test "schema.json valid JSON" "jq -e . '${RESOURCE_DIR}/config/schema.json' >/dev/null"

# Test 12: Documentation exists
run_test "README.md exists" "test -f '${RESOURCE_DIR}/README.md'"
run_test "PRD.md exists" "test -f '${RESOURCE_DIR}/PRD.md'"

# Summary
echo ""
echo "Results:"
echo "  Passed: $TESTS_PASSED"
echo "  Failed: $TESTS_FAILED"

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo ""
    echo "✓ All unit tests passed!"
    exit 0
else
    echo ""
    echo "✗ Some unit tests failed"
    exit 1
fi