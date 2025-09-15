#!/usr/bin/env bash
# Pi-hole Integration Tests - End-to-end functionality validation
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Source core library
source "${RESOURCE_DIR}/lib/core.sh"

# Get actual DNS port from configuration
DNS_PORT="${PIHOLE_DNS_PORT:-53}"
if [[ -f "${PIHOLE_DATA_DIR}/.port_config" ]]; then
    source "${PIHOLE_DATA_DIR}/.port_config"
    DNS_PORT="${DNS_PORT:-${PIHOLE_DNS_PORT}}"
fi
echo "Using DNS port: ${DNS_PORT}"

echo "Pi-hole Integration Tests"
echo "========================="

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

# Ensure Pi-hole is running
if ! check_health; then
    echo "Error: Pi-hole is not running. Start it first with:"
    echo "  vrooli resource pihole manage start --wait"
    exit 1
fi

# Test 1: DNS resolution for legitimate domain
run_test "DNS resolves google.com" "timeout 5 dig @localhost -p ${DNS_PORT} google.com +short | grep -E '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$'"

# Test 2: DNS resolution for localhost
run_test "DNS resolves localhost" "timeout 5 dig @localhost -p ${DNS_PORT} localhost +short | grep -q '127.0.0.1'"

# Test 3: API summary endpoint
run_test "API summary data" "docker exec '${CONTAINER_NAME}' pihole api stats/summary | jq -e '.gravity.domains_being_blocked'"

# Test 4: API status shows enabled
run_test "API status enabled" "docker exec '${CONTAINER_NAME}' pihole status | grep -q 'enabled'"

# Test 5: Blacklist management
TEST_DOMAIN="test-block-$(date +%s).com"
run_test "Add to blacklist" "docker exec '${CONTAINER_NAME}' pihole -b '$TEST_DOMAIN'"
run_test "Remove from blacklist" "docker exec '${CONTAINER_NAME}' pihole -b -d '$TEST_DOMAIN'"

# Test 6: Whitelist management
TEST_DOMAIN="test-allow-$(date +%s).com"
run_test "Add to whitelist" "docker exec '${CONTAINER_NAME}' pihole -w '$TEST_DOMAIN'"
run_test "Remove from whitelist" "docker exec '${CONTAINER_NAME}' pihole -w -d '$TEST_DOMAIN'"

# Test 7: Query log exists
run_test "Query log file exists" "docker exec '${CONTAINER_NAME}' test -f /var/log/pihole/pihole.log"

# Test 8: Gravity database exists
run_test "Gravity database" "docker exec '${CONTAINER_NAME}' test -f /etc/pihole/gravity.db"

# Test 9: FTL (Faster Than Light) daemon running
run_test "FTL daemon running" "docker exec '${CONTAINER_NAME}' pgrep pihole-FTL"

# Test 10: DNS cache working
run_test "DNS cache functional" "docker exec '${CONTAINER_NAME}' pihole-FTL dns-cache show | head -1"

# Test 11: Statistics are being collected
run_test "Statistics collection" "docker exec '${CONTAINER_NAME}' pihole api stats/summary | jq -e '.queries.total >= 0'"

# Test 12: Password file exists
run_test "Password file exists" "test -f '${PIHOLE_DATA_DIR}/.webpassword'"

# Summary
echo ""
echo "Results:"
echo "  Passed: $TESTS_PASSED"
echo "  Failed: $TESTS_FAILED"

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo ""
    echo "✓ All integration tests passed!"
    exit 0
else
    echo ""
    echo "✗ Some integration tests failed"
    exit 1
fi