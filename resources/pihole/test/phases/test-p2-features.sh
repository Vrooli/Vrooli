#!/usr/bin/env bash
# Pi-hole P2 Features Test - Test web interface, groups, and gravity sync
set -euo pipefail

# Get test directory
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${TEST_DIR}/../.." && pwd)"
CLI="${RESOURCE_DIR}/cli.sh"

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Test function
run_test() {
    local test_name="$1"
    local test_cmd="$2"
    
    echo -n "Testing: ${test_name}... "
    
    if eval "$test_cmd" > /dev/null 2>&1; then
        echo "✓"
        ((TESTS_PASSED++))
    else
        echo "✗"
        ((TESTS_FAILED++))
    fi
}

echo "=== Pi-hole P2 Features Tests ==="

# Test Web Interface
echo ""
echo "Web Interface Tests:"
run_test "Check web interface status" "timeout 5 ${CLI} web status"
run_test "Get web interface URL" "${CLI} web url | grep -q 'http://localhost'"
run_test "Get web password" "${CLI} web password"

# Test Group Management
echo ""
echo "Group Management Tests:"
run_test "List groups" "${CLI} groups list"
run_test "Create test group" "${CLI} groups create TestGroup 'Test group for validation'"
run_test "List groups includes TestGroup" "${CLI} groups list | grep -q TestGroup"
run_test "Add client to group" "${CLI} groups add-client 192.168.1.100 TestGroup 'Test client'"
run_test "List clients in group" "${CLI} groups list-clients TestGroup"
run_test "Add domain to group blocklist" "${CLI} groups add-domain TestGroup ads.example.com"
run_test "Disable group" "${CLI} groups disable TestGroup"
run_test "Enable group" "${CLI} groups enable TestGroup"
run_test "Remove client from group" "${CLI} groups remove-client 192.168.1.100"
run_test "Delete test group" "${CLI} groups delete TestGroup"

# Test Gravity Sync
echo ""
echo "Gravity Sync Tests:"
run_test "Initialize sync configuration" "${CLI} gravity-sync init"
run_test "Add remote Pi-hole" "${CLI} gravity-sync add-remote secondary 192.168.1.200 22"
run_test "List remotes" "${CLI} gravity-sync list-remotes | grep -q secondary"
run_test "Export gravity database" "${CLI} gravity-sync export /tmp/test-gravity.db"
run_test "Check exported file exists" "test -f /tmp/test-gravity.db"
run_test "Remove remote" "${CLI} gravity-sync remove-remote secondary"

# Cleanup
rm -f /tmp/test-gravity.db*

# Summary
echo ""
echo "=== Test Summary ==="
echo "Tests Passed: ${TESTS_PASSED}"
echo "Tests Failed: ${TESTS_FAILED}"

if [[ ${TESTS_FAILED} -eq 0 ]]; then
    echo "Status: ALL TESTS PASSED ✓"
    exit 0
else
    echo "Status: SOME TESTS FAILED ✗"
    exit 1
fi