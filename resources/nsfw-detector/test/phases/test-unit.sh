#!/usr/bin/env bash
# Unit tests for NSFW Detector resource
# Library function validation - must complete in <60s

set -euo pipefail

# Script directory
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
readonly LIB_DIR="${SCRIPT_DIR}/lib"

# Source the library
source "${LIB_DIR}/core.sh"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly NC='\033[0m'

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper function
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -n "Testing $test_name... "
    
    if eval "$test_command" > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
    fi
}

# Main unit tests
main() {
    echo "Running NSFW Detector unit tests..."
    echo "===================================="
    
    # Test 1: Port allocation function
    run_test "port allocation" \
        "[[ $(get_port) =~ ^[0-9]+$ ]]"
    
    # Test 2: Resource name is set
    run_test "resource name" \
        "[[ '$RESOURCE_NAME' == 'nsfw-detector' ]]"
    
    # Test 3: Config directory exists
    run_test "config directory" \
        "[[ -d '$CONFIG_DIR' ]]"
    
    # Test 4: Test directory exists
    run_test "test directory" \
        "[[ -d '$TEST_DIR' ]]"
    
    # Test 5: Default configuration loaded
    run_test "defaults loaded" \
        "[[ -n '${NSFW_DETECTOR_DEFAULT_MODEL:-}' ]]"
    
    # Test 6: Runtime config exists
    run_test "runtime.json exists" \
        "[[ -f '${CONFIG_DIR}/runtime.json' ]]"
    
    # Test 7: Runtime config valid JSON
    run_test "runtime.json valid" \
        "jq -e . '${CONFIG_DIR}/runtime.json'"
    
    # Test 8: CLI script exists and executable
    run_test "CLI executable" \
        "[[ -x '${SCRIPT_DIR}/cli.sh' ]]"
    
    # Test 9: Test runner executable
    run_test "test runner executable" \
        "[[ -x '${TEST_DIR}/run-tests.sh' ]]"
    
    # Test 10: Package.json exists
    run_test "package.json exists" \
        "[[ -f '${SCRIPT_DIR}/package.json' ]]"
    
    # Summary
    echo "===================================="
    echo "Unit Test Results:"
    echo "  Passed: $TESTS_PASSED"
    echo "  Failed: $TESTS_FAILED"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}All unit tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some unit tests failed${NC}"
        exit 1
    fi
}

# Run tests
main "$@"