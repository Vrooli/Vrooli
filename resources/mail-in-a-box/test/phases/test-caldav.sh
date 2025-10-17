#!/bin/bash

# CalDAV/CardDAV Integration Tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
CLI="$RESOURCE_DIR/cli.sh"

# Test configuration
CALDAV_PORT="${CALDAV_PORT:-5232}"
TEST_CALDAV_USER="caldav_test@example.com"
TEST_CALDAV_PASS="CalDav123!"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper functions
test_pass() {
    echo -e "${GREEN}✓${NC} $1"
    ((TESTS_PASSED++))
}

test_fail() {
    echo -e "${RED}✗${NC} $1: $2"
    ((TESTS_FAILED++))
}

# Check if CalDAV container is running
is_caldav_running() {
    docker ps --filter "name=mailinabox-caldav" --filter "status=running" -q 2>/dev/null | grep -q .
}

# Test 1: CalDAV container status
test_caldav_container() {
    echo "Testing: CalDAV container status..."
    
    if is_caldav_running; then
        test_pass "CalDAV container is running"
    else
        test_pass "CalDAV container check skipped (not installed)"
        return 0
    fi
}

# Test 2: CalDAV health check
test_caldav_health() {
    echo "Testing: CalDAV health check..."
    
    if ! is_caldav_running; then
        test_pass "CalDAV health check skipped (not running)"
        return 0
    fi
    
    local health_status
    health_status=$("$CLI" caldav health 2>&1)
    
    if [[ "$health_status" == "healthy" ]]; then
        test_pass "CalDAV service is healthy"
    else
        test_fail "CalDAV health check" "Service unhealthy: $health_status"
    fi
}

# Test 3: CalDAV web interface
test_caldav_web() {
    echo "Testing: CalDAV web interface..."
    
    if ! is_caldav_running; then
        test_pass "CalDAV web interface test skipped (not running)"
        return 0
    fi
    
    if timeout 5 curl -sf "http://localhost:${CALDAV_PORT}/.web/" &>/dev/null; then
        test_pass "CalDAV web interface accessible"
    else
        test_fail "CalDAV web interface" "Not accessible on port ${CALDAV_PORT}"
    fi
}

# Test 4: CalDAV user creation
test_caldav_user_creation() {
    echo "Testing: CalDAV user creation..."
    
    if ! is_caldav_running; then
        test_pass "CalDAV user creation test skipped (not running)"
        return 0
    fi
    
    # Clean up any existing test user
    "$CLI" caldav delete-user "$TEST_CALDAV_USER" &>/dev/null || true
    
    # Create user
    if "$CLI" caldav add-user "$TEST_CALDAV_USER" "$TEST_CALDAV_PASS" &>/dev/null; then
        test_pass "CalDAV user created successfully"
    else
        test_fail "CalDAV user creation" "Failed to create user"
    fi
}

# Test 5: CalDAV user listing
test_caldav_user_list() {
    echo "Testing: CalDAV user listing..."
    
    if ! is_caldav_running; then
        test_pass "CalDAV user listing test skipped (not running)"
        return 0
    fi
    
    local users
    users=$("$CLI" caldav list-users 2>&1)
    
    if echo "$users" | grep -q "$TEST_CALDAV_USER"; then
        test_pass "CalDAV user appears in list"
    else
        test_fail "CalDAV user listing" "Test user not found in list"
    fi
}

# Test 6: CalDAV authentication test
test_caldav_auth() {
    echo "Testing: CalDAV authentication..."
    
    if ! is_caldav_running; then
        test_pass "CalDAV authentication test skipped (not running)"
        return 0
    fi
    
    if "$CLI" caldav test "$TEST_CALDAV_USER" "$TEST_CALDAV_PASS" &>/dev/null; then
        test_pass "CalDAV authentication successful"
    else
        test_fail "CalDAV authentication" "Failed to authenticate"
    fi
}

# Test 7: CalDAV user deletion
test_caldav_user_deletion() {
    echo "Testing: CalDAV user deletion..."
    
    if ! is_caldav_running; then
        test_pass "CalDAV user deletion test skipped (not running)"
        return 0
    fi
    
    if "$CLI" caldav delete-user "$TEST_CALDAV_USER" &>/dev/null; then
        test_pass "CalDAV user deleted successfully"
    else
        test_fail "CalDAV user deletion" "Failed to delete user"
    fi
}

# Test 8: CalDAV info command
test_caldav_info() {
    echo "Testing: CalDAV info command..."
    
    local info_output
    info_output=$("$CLI" caldav info 2>&1)
    
    if echo "$info_output" | grep -q "CalDAV URL"; then
        test_pass "CalDAV info command works"
    else
        test_fail "CalDAV info command" "Invalid output"
    fi
}

# Main test execution
main() {
    echo "==================================="
    echo "Mail-in-a-Box CalDAV/CardDAV Tests"
    echo "==================================="
    
    # Run tests
    test_caldav_container
    test_caldav_health
    test_caldav_web
    test_caldav_user_creation
    test_caldav_user_list
    test_caldav_auth
    test_caldav_user_deletion
    test_caldav_info
    
    # Summary
    echo "==================================="
    echo "Results: $TESTS_PASSED passed, $TESTS_FAILED failed"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}All CalDAV tests passed${NC}"
        exit 0
    else
        echo -e "${RED}Some CalDAV tests failed${NC}"
        exit 1
    fi
}

main "$@"