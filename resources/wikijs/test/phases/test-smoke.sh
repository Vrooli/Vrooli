#!/usr/bin/env bash
################################################################################
# Wiki.js Smoke Test - v2.0 Universal Contract Compliant
# 
# Quick health validation that must complete in <30 seconds
#
################################################################################

set -euo pipefail

# Resolve paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
RESOURCE_DIR="$(cd "${TEST_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source common functions
source "${RESOURCE_DIR}/lib/common.sh"

# Test timeout
TIMEOUT=30

# ANSI color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test result tracking
TESTS_PASSED=0
TESTS_FAILED=0

# Run a test with timeout
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -n "  Testing ${test_name}... "
    
    if timeout "$TIMEOUT" bash -c "$test_command" &>/dev/null; then
        echo -e "${GREEN}✓${NC}"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Main smoke test
main() {
    echo "Running Wiki.js smoke tests..."
    echo ""
    
    # Test 1: Check if Wiki.js is installed
    run_test "installation" "is_installed" || true
    
    # Test 2: Check if Wiki.js is running
    run_test "container running" "is_running" || true
    
    # Test 3: Check health endpoint with proper timeout
    if is_running; then
        local port=$(get_wikijs_port)
        run_test "health endpoint" "timeout 5 curl -sf http://localhost:${port}/ >/dev/null" || true
    else
        echo -e "  Testing health endpoint... ${YELLOW}SKIP${NC} (service not running)"
    fi
    
    # Test 4: Check GraphQL endpoint availability
    if is_running; then
        local port=$(get_wikijs_port)
        run_test "GraphQL endpoint" "timeout 5 curl -sf http://localhost:${port}/graphql -H 'Content-Type: application/json' -d '{\"query\":\"{info{version}}\"}'" || true
    else
        echo -e "  Testing GraphQL endpoint... ${YELLOW}SKIP${NC} (service not running)"
    fi
    
    # Test 5: Check port allocation
    run_test "port registry" "get_wikijs_port | grep -E '^[0-9]+$'" || true
    
    # Test 6: Check data directory
    run_test "data directory" "[[ -d '$WIKIJS_DATA_DIR' ]]" || true
    
    echo ""
    echo "Smoke test summary: ${TESTS_PASSED} passed, ${TESTS_FAILED} failed"
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        return 1
    fi
    
    return 0
}

# Execute smoke tests
main "$@"