#!/bin/bash

# GeoNode Test Runner

set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${TEST_DIR}/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Run test phase
run_test_phase() {
    local phase="$1"
    local script="${TEST_DIR}/phases/test-${phase}.sh"
    
    if [[ ! -f "$script" ]]; then
        echo -e "${YELLOW}Skipping ${phase} tests (script not found)${NC}"
        return 0
    fi
    
    echo -e "${GREEN}Running ${phase} tests...${NC}"
    
    if bash "$script"; then
        echo -e "${GREEN}✓ ${phase} tests passed${NC}"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗ ${phase} tests failed${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Main test execution
main() {
    echo "========================================="
    echo "GeoNode Resource Test Suite"
    echo "========================================="
    echo ""
    
    local start_time=$(date +%s)
    
    # Run test phases in order
    run_test_phase "smoke" || true
    run_test_phase "unit" || true
    run_test_phase "integration" || true
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "========================================="
    echo "Test Results"
    echo "========================================="
    echo -e "Passed: ${GREEN}${TESTS_PASSED}${NC}"
    echo -e "Failed: ${RED}${TESTS_FAILED}${NC}"
    echo "Duration: ${duration} seconds"
    echo ""
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        echo -e "${RED}TEST SUITE FAILED${NC}"
        exit 1
    else
        echo -e "${GREEN}ALL TESTS PASSED${NC}"
        exit 0
    fi
}

main "$@"