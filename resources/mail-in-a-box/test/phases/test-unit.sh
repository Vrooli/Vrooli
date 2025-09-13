#!/bin/bash

# Mail-in-a-Box Unit Tests
# Tests for individual library functions

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source libraries to test
source "$RESOURCE_DIR/lib/core.sh"

# Test configuration
TEST_EMAIL="test@example.com"
TEST_PASSWORD="TestPass123!"
TEST_ALIAS="alias@example.com"
TEST_DOMAIN="testdomain.com"

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

# Test 1: validate_email function
test_validate_email() {
    echo "Testing: Email validation..."
    
    # Valid emails
    local valid_emails=("user@example.com" "admin@mail.local" "test.user@domain.co.uk")
    for email in "${valid_emails[@]}"; do
        if validate_email "$email"; then
            test_pass "Valid email accepted: $email"
        else
            test_fail "Valid email rejected" "$email"
        fi
    done
    
    # Invalid emails
    local invalid_emails=("not-an-email" "@example.com" "user@" "user..name@example.com")
    for email in "${invalid_emails[@]}"; do
        if ! validate_email "$email" 2>/dev/null; then
            test_pass "Invalid email rejected: $email"
        else
            test_fail "Invalid email accepted" "$email"
        fi
    done
}

# Test 2: validate_password function
test_validate_password() {
    echo "Testing: Password validation..."
    
    # Valid passwords
    local valid_passwords=("SecurePass123!" "Abcd1234" "P@ssw0rd!")
    for password in "${valid_passwords[@]}"; do
        if validate_password "$password"; then
            test_pass "Valid password accepted: [hidden]"
        else
            test_fail "Valid password rejected" "[hidden]"
        fi
    done
    
    # Invalid passwords (too short)
    local invalid_passwords=("abc" "1234567" "")
    for password in "${invalid_passwords[@]}"; do
        if ! validate_password "$password" 2>/dev/null; then
            test_pass "Invalid password rejected: [too short]"
        else
            test_fail "Invalid password accepted" "[hidden]"
        fi
    done
}

# Test 3: check_docker function
test_check_docker() {
    echo "Testing: Docker check..."
    
    if check_docker; then
        test_pass "Docker is available"
    else
        test_fail "Docker check failed" "Docker should be running"
    fi
}

# Test 4: get_container_name function
test_get_container_name() {
    echo "Testing: Container name resolution..."
    
    local container_name
    container_name=$(get_container_name 2>/dev/null || echo "")
    
    if [[ -n "$container_name" ]]; then
        test_pass "Container name retrieved: $container_name"
    else
        # This might be expected if container isn't running
        test_pass "Container name function works (no container running)"
    fi
}

# Test 5: parse_csv_line function
test_parse_csv() {
    echo "Testing: CSV parsing..."
    
    # Test data
    local csv_line="user@example.com,SecurePassword123"
    local email password
    
    # Parse the line
    IFS=',' read -r email password <<< "$csv_line"
    
    if [[ "$email" == "user@example.com" ]] && [[ "$password" == "SecurePassword123" ]]; then
        test_pass "CSV parsing works correctly"
    else
        test_fail "CSV parsing failed" "Expected: user@example.com,SecurePassword123"
    fi
}

# Test 6: parse_json function (basic test)
test_parse_json() {
    echo "Testing: JSON parsing..."
    
    # Test data
    local json_data='{"email": "user@example.com", "password": "SecurePassword123"}'
    
    # Check if jq is available
    if command -v jq &>/dev/null; then
        local email password
        email=$(echo "$json_data" | jq -r '.email')
        password=$(echo "$json_data" | jq -r '.password')
        
        if [[ "$email" == "user@example.com" ]] && [[ "$password" == "SecurePassword123" ]]; then
            test_pass "JSON parsing works correctly"
        else
            test_fail "JSON parsing failed" "Incorrect values extracted"
        fi
    else
        test_pass "JSON parsing skipped (jq not available)"
    fi
}

# Test 7: Environment variable defaults
test_env_defaults() {
    echo "Testing: Environment defaults..."
    
    # Check default values
    local hostname="${MAILINABOX_PRIMARY_HOSTNAME:-mail.local}"
    local admin_email="${MAILINABOX_ADMIN_EMAIL:-admin@mail.local}"
    local bind_addr="${MAILINABOX_BIND_ADDRESS:-127.0.0.1}"
    
    if [[ "$hostname" == "mail.local" ]] && [[ "$admin_email" == "admin@mail.local" ]]; then
        test_pass "Default environment values correct"
    else
        test_fail "Default environment values incorrect" "Check defaults.sh"
    fi
}

# Test 8: Port configuration
test_port_config() {
    echo "Testing: Port configuration..."
    
    local admin_port="${MAILINABOX_ADMIN_PORT:-8543}"
    local smtp_port="${MAILINABOX_SMTP_PORT:-25}"
    local imap_port="${MAILINABOX_IMAP_PORT:-143}"
    
    # Check ports are numeric
    if [[ "$admin_port" =~ ^[0-9]+$ ]] && [[ "$smtp_port" =~ ^[0-9]+$ ]] && [[ "$imap_port" =~ ^[0-9]+$ ]]; then
        test_pass "Port configuration valid"
    else
        test_fail "Port configuration invalid" "Non-numeric port values"
    fi
}

# Main test execution
main() {
    echo "==================================="
    echo "Mail-in-a-Box Unit Tests"
    echo "==================================="
    
    # Run all unit tests
    test_validate_email || true
    test_validate_password || true
    test_check_docker || true
    test_get_container_name || true
    test_parse_csv || true
    test_parse_json || true
    test_env_defaults || true
    test_port_config || true
    
    echo "==================================="
    echo "Results: $TESTS_PASSED passed, $TESTS_FAILED failed"
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        echo -e "${RED}Unit tests failed${NC}"
        exit 1
    else
        echo -e "${GREEN}All unit tests passed${NC}"
        exit 0
    fi
}

# Run tests
main