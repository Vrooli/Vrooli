#!/usr/bin/env bash
################################################################################
# Wiki.js Unit Test - v2.0 Universal Contract Compliant
# 
# Library function validation that must complete in <60 seconds
#
################################################################################

set -euo pipefail

# Resolve paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
RESOURCE_DIR="$(cd "${TEST_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source library functions to test
source "${RESOURCE_DIR}/lib/common.sh"

# Test timeout
TIMEOUT=60

# ANSI color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test result tracking
TESTS_PASSED=0
TESTS_FAILED=0

# Run a test
run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected="${3:-}"
    
    echo -n "  Testing ${test_name}... "
    
    local result
    if result=$(timeout "$TIMEOUT" bash -c "$test_command" 2>/dev/null); then
        if [[ -z "$expected" ]] || [[ "$result" == "$expected" ]]; then
            echo -e "${GREEN}✓${NC}"
            ((TESTS_PASSED++))
            return 0
        else
            echo -e "${RED}✗${NC} (expected: $expected, got: $result)"
            ((TESTS_FAILED++))
            return 1
        fi
    else
        echo -e "${RED}✗${NC} (command failed)"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Test port functions
test_port_functions() {
    echo "  Port functions:"
    
    # Test get_wikijs_port returns valid port
    local port=$(get_wikijs_port)
    if [[ "$port" =~ ^[0-9]+$ ]] && [[ "$port" -gt 1024 ]] && [[ "$port" -lt 65535 ]]; then
        echo -e "    get_wikijs_port... ${GREEN}✓${NC} (port: $port)"
        ((TESTS_PASSED++))
    else
        echo -e "    get_wikijs_port... ${RED}✗${NC} (invalid port: $port)"
        ((TESTS_FAILED++))
    fi
}

# Test database config functions
test_db_functions() {
    echo "  Database functions:"
    
    # Test get_db_config returns valid config
    local db_config=$(get_db_config)
    if [[ "$db_config" =~ host=.+\ port=[0-9]+\ name=.+\ user=.+\ pass=.+ ]]; then
        echo -e "    get_db_config... ${GREEN}✓${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "    get_db_config... ${RED}✗${NC} (invalid format)"
        ((TESTS_FAILED++))
    fi
}

# Test container status functions
test_container_functions() {
    echo "  Container functions:"
    
    # Test get_container_status returns valid status
    local status=$(get_container_status)
    if [[ "$status" == "running" ]] || [[ "$status" == "stopped" ]] || [[ "$status" == "not_found" ]]; then
        echo -e "    get_container_status... ${GREEN}✓${NC} (status: $status)"
        ((TESTS_PASSED++))
    else
        echo -e "    get_container_status... ${RED}✗${NC} (invalid status: $status)"
        ((TESTS_FAILED++))
    fi
}

# Test configuration constants
test_constants() {
    echo "  Configuration constants:"
    
    # Test WIKIJS_CONTAINER is set
    if [[ -n "$WIKIJS_CONTAINER" ]]; then
        echo -e "    WIKIJS_CONTAINER... ${GREEN}✓${NC} ($WIKIJS_CONTAINER)"
        ((TESTS_PASSED++))
    else
        echo -e "    WIKIJS_CONTAINER... ${RED}✗${NC} (not set)"
        ((TESTS_FAILED++))
    fi
    
    # Test WIKIJS_IMAGE is set
    if [[ -n "$WIKIJS_IMAGE" ]]; then
        echo -e "    WIKIJS_IMAGE... ${GREEN}✓${NC} ($WIKIJS_IMAGE)"
        ((TESTS_PASSED++))
    else
        echo -e "    WIKIJS_IMAGE... ${RED}✗${NC} (not set)"
        ((TESTS_FAILED++))
    fi
    
    # Test WIKIJS_DATA_DIR is set
    if [[ -n "$WIKIJS_DATA_DIR" ]]; then
        echo -e "    WIKIJS_DATA_DIR... ${GREEN}✓${NC} ($WIKIJS_DATA_DIR)"
        ((TESTS_PASSED++))
    else
        echo -e "    WIKIJS_DATA_DIR... ${RED}✗${NC} (not set)"
        ((TESTS_FAILED++))
    fi
}

# Main unit test
main() {
    echo "Running Wiki.js unit tests..."
    echo ""
    
    test_port_functions
    test_db_functions
    test_container_functions
    test_constants
    
    echo ""
    echo "Unit test summary: ${TESTS_PASSED} passed, ${TESTS_FAILED} failed"
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        return 1
    fi
    
    return 0
}

# Execute unit tests
main "$@"