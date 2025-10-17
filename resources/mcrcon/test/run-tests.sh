#!/usr/bin/env bash
# mcrcon test runner - executes all test phases

set -euo pipefail

# Get test directory
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly TEST_DIR

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m' # No Color

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Run test phase
run_phase() {
    local phase_name="$1"
    local phase_script="${TEST_DIR}/phases/test-${phase_name}.sh"
    
    echo -e "${YELLOW}Running ${phase_name} tests...${NC}"
    
    if [[ ! -f "$phase_script" ]]; then
        echo -e "${RED}Test phase script not found: $phase_script${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
    
    if bash "$phase_script"; then
        echo -e "${GREEN}✓ ${phase_name} tests passed${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}✗ ${phase_name} tests failed${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Main test execution
main() {
    echo "================================"
    echo "mcrcon Resource Test Suite"
    echo "================================"
    echo ""
    
    local exit_code=0
    
    # Run test phases in order
    run_phase "smoke" || exit_code=1
    run_phase "unit" || exit_code=1
    run_phase "integration" || exit_code=1
    
    # Summary
    echo ""
    echo "================================"
    echo "Test Results Summary"
    echo "================================"
    echo -e "Passed: ${GREEN}${TESTS_PASSED}${NC}"
    echo -e "Failed: ${RED}${TESTS_FAILED}${NC}"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}All tests passed!${NC}"
    else
        echo -e "${RED}Some tests failed${NC}"
    fi
    
    exit $exit_code
}

main "$@"