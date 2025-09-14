#!/bin/bash

# Mail-in-a-Box Smoke Tests
# Quick validation that service is running and responsive

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source test helpers
source "$RESOURCE_DIR/lib/core.sh"

# Test configuration
ADMIN_PORT="${MAILINABOX_ADMIN_PORT:-8543}"
SMTP_PORT="${MAILINABOX_SMTP_PORT:-25}"
IMAP_PORT="${MAILINABOX_IMAP_PORT:-143}"
HEALTH_TIMEOUT=5

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
    echo -e "${RED}✗${NC} $1"
    ((TESTS_FAILED++))
}

# Test 1: Container is running
test_container_running() {
    echo "Testing: Container status..."
    
    if docker ps --format "table {{.Names}}" | grep -q "mailinabox"; then
        test_pass "Mail-in-a-Box container is running"
    else
        test_fail "Mail-in-a-Box container is not running"
        return 1
    fi
}

# Test 2: Health check (SMTP based for docker-mailserver)
test_health_endpoint() {
    echo "Testing: Health check..."
    
    # docker-mailserver doesn't have an admin panel, check SMTP instead
    if echo "QUIT" | timeout "$HEALTH_TIMEOUT" nc localhost "$SMTP_PORT" 2>/dev/null | grep -q "220"; then
        test_pass "Mail server is healthy (SMTP responds)"
    else
        test_fail "Mail server health check failed"
        return 1
    fi
}

# Test 3: SMTP port is listening
test_smtp_port() {
    echo "Testing: SMTP service..."
    
    if timeout 2 nc -zv localhost "$SMTP_PORT" &>/dev/null; then
        test_pass "SMTP port $SMTP_PORT is listening"
    else
        test_fail "SMTP port $SMTP_PORT is not accessible"
        return 1
    fi
}

# Test 4: IMAPS port is listening (docker-mailserver uses secure IMAP)
test_imap_port() {
    echo "Testing: IMAP service..."
    
    # docker-mailserver exposes IMAPS (993) not plain IMAP (143)
    local IMAPS_PORT="${MAILINABOX_IMAPS_PORT:-993}"
    if timeout 2 nc -zv localhost "$IMAPS_PORT" &>/dev/null; then
        test_pass "IMAPS port $IMAPS_PORT is listening"
    else
        test_fail "IMAPS port $IMAPS_PORT is not accessible"
        return 1
    fi
}

# Test 5: Email account exists
test_admin_panel() {
    echo "Testing: Email accounts..."
    
    # docker-mailserver doesn't have admin panel, check if accounts exist
    if docker exec mailinabox cat /tmp/docker-mailserver/postfix-accounts.cf 2>/dev/null | grep -q "@"; then
        test_pass "Email accounts configured"
    else
        test_fail "No email accounts found"
        return 1
    fi
}

# Test 6: CLI commands available
test_cli_commands() {
    echo "Testing: CLI commands..."
    
    if "$RESOURCE_DIR/cli.sh" help &>/dev/null; then
        test_pass "CLI help command works"
    else
        test_fail "CLI help command failed"
        return 1
    fi
}

# Test 7: Volume mounts exist
test_volumes() {
    echo "Testing: Docker volumes..."
    
    # Check if container has volume mounts
    if docker inspect mailinabox --format '{{range .Mounts}}{{.Source}}{{"\n"}}{{end}}' 2>/dev/null | grep -q "/"; then
        test_pass "Mail server volumes mounted"
    else
        test_fail "Mail server volumes not found"
        return 1
    fi
}

# Main test execution
main() {
    echo "==================================="
    echo "Mail-in-a-Box Smoke Tests"
    echo "==================================="
    
    # Run all smoke tests
    test_container_running || true
    test_health_endpoint || true
    test_smtp_port || true
    test_imap_port || true
    test_admin_panel || true
    test_cli_commands || true
    test_volumes || true
    
    echo "==================================="
    echo "Results: $TESTS_PASSED passed, $TESTS_FAILED failed"
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        echo -e "${RED}Smoke tests failed${NC}"
        exit 1
    else
        echo -e "${GREEN}All smoke tests passed${NC}"
        exit 0
    fi
}

# Run tests
main