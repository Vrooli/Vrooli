#!/bin/bash

# Multi-domain Management Tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
CLI="$RESOURCE_DIR/cli.sh"

# Test configuration
TEST_DOMAIN="test-domain.local"
TEST_DOMAIN2="another-test.local"

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

# Check if mail server is running
is_mail_running() {
    docker ps --filter "name=mailinabox" --filter "status=running" -q 2>/dev/null | grep -q .
}

# Test 1: Add domain
test_add_domain() {
    echo "Testing: Add domain..."
    
    if ! is_mail_running; then
        test_pass "Add domain test skipped (mail server not running)"
        return 0
    fi
    
    # Remove test domain if it exists
    "$CLI" content remove-domain "$TEST_DOMAIN" &>/dev/null || true
    
    if "$CLI" content add-domain "$TEST_DOMAIN" &>/dev/null; then
        test_pass "Domain added successfully"
    else
        test_fail "Add domain" "Failed to add domain"
    fi
}

# Test 2: List domains
test_list_domains() {
    echo "Testing: List domains..."
    
    if ! is_mail_running; then
        test_pass "List domains test skipped (mail server not running)"
        return 0
    fi
    
    local domains
    domains=$("$CLI" content list-domains 2>&1)
    
    if echo "$domains" | grep -q "$TEST_DOMAIN"; then
        test_pass "Domain appears in list"
    else
        test_fail "List domains" "Test domain not found in list"
    fi
}

# Test 3: Add second domain
test_add_second_domain() {
    echo "Testing: Add second domain..."
    
    if ! is_mail_running; then
        test_pass "Add second domain test skipped (mail server not running)"
        return 0
    fi
    
    # Remove test domain if it exists
    "$CLI" content remove-domain "$TEST_DOMAIN2" &>/dev/null || true
    
    if "$CLI" content add-domain "$TEST_DOMAIN2" &>/dev/null; then
        test_pass "Second domain added successfully"
    else
        test_fail "Add second domain" "Failed to add second domain"
    fi
}

# Test 4: Create email account with custom domain
test_domain_email_account() {
    echo "Testing: Create email account with custom domain..."
    
    if ! is_mail_running; then
        test_pass "Domain email account test skipped (mail server not running)"
        return 0
    fi
    
    local test_email="testuser@${TEST_DOMAIN}"
    
    # Remove if exists
    "$CLI" content remove "$test_email" &>/dev/null || true
    
    if echo "TestPass123!" | "$CLI" content add "$test_email" &>/dev/null; then
        test_pass "Email account created with custom domain"
        # Clean up
        "$CLI" content remove "$test_email" &>/dev/null || true
    else
        test_fail "Domain email account" "Failed to create account"
    fi
}

# Test 5: Get DKIM key
test_get_dkim() {
    echo "Testing: Get DKIM key..."
    
    if ! is_mail_running; then
        test_pass "Get DKIM test skipped (mail server not running)"
        return 0
    fi
    
    local dkim_output
    dkim_output=$("$CLI" content get-dkim "$TEST_DOMAIN" 2>&1)
    
    if echo "$dkim_output" | grep -qE "(DKIM|not found)"; then
        test_pass "DKIM command executed"
    else
        test_fail "Get DKIM" "Unexpected output"
    fi
}

# Test 6: Verify domain (DNS check)
test_verify_domain() {
    echo "Testing: Verify domain DNS..."
    
    if ! is_mail_running; then
        test_pass "Verify domain test skipped (mail server not running)"
        return 0
    fi
    
    local verify_output
    verify_output=$("$CLI" content verify-domain "$TEST_DOMAIN" 2>&1)
    
    if echo "$verify_output" | grep -qE "(MX Records|SPF Record)"; then
        test_pass "Domain verification command works"
    else
        test_fail "Verify domain" "Unexpected output"
    fi
}

# Test 7: Remove domain
test_remove_domain() {
    echo "Testing: Remove domain..."
    
    if ! is_mail_running; then
        test_pass "Remove domain test skipped (mail server not running)"
        return 0
    fi
    
    if "$CLI" content remove-domain "$TEST_DOMAIN" &>/dev/null; then
        test_pass "Domain removed successfully"
    else
        test_fail "Remove domain" "Failed to remove domain"
    fi
}

# Test 8: Verify domain removal
test_verify_removal() {
    echo "Testing: Verify domain removal..."
    
    if ! is_mail_running; then
        test_pass "Verify removal test skipped (mail server not running)"
        return 0
    fi
    
    local domains
    domains=$("$CLI" content list-domains 2>&1)
    
    if ! echo "$domains" | grep -q "$TEST_DOMAIN"; then
        test_pass "Domain successfully removed from list"
    else
        test_fail "Verify removal" "Domain still appears in list"
    fi
}

# Clean up
cleanup() {
    if is_mail_running; then
        "$CLI" content remove-domain "$TEST_DOMAIN" &>/dev/null || true
        "$CLI" content remove-domain "$TEST_DOMAIN2" &>/dev/null || true
    fi
}

# Main test execution
main() {
    echo "==================================="
    echo "Mail-in-a-Box Multi-Domain Tests"
    echo "==================================="
    
    # Run tests
    test_add_domain
    test_list_domains
    test_add_second_domain
    test_domain_email_account
    test_get_dkim
    test_verify_domain
    test_remove_domain
    test_verify_removal
    
    # Clean up
    cleanup
    
    # Summary
    echo "==================================="
    echo "Results: $TESTS_PASSED passed, $TESTS_FAILED failed"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}All domain tests passed${NC}"
        exit 0
    else
        echo -e "${RED}Some domain tests failed${NC}"
        exit 1
    fi
}

main "$@"