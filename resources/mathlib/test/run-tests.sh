#!/usr/bin/env bash
# Mathlib Resource - Main Test Runner

set -euo pipefail

# Test directory
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHASES_DIR="${TEST_DIR}/phases"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Run a test phase
run_phase() {
    local phase_name="$1"
    local phase_script="${PHASES_DIR}/test-${phase_name}.sh"
    
    echo -e "\n${YELLOW}Running ${phase_name} tests...${NC}"
    
    if [[ ! -f "${phase_script}" ]]; then
        echo -e "${RED}Error: Test script ${phase_script} not found${NC}"
        return 1
    fi
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if bash "${phase_script}"; then
        echo -e "${GREEN}✓ ${phase_name} tests passed${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗ ${phase_name} tests failed${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

# Main test execution
main() {
    echo "========================================="
    echo "Mathlib Resource Test Suite"
    echo "========================================="
    
    # Run test phases in order
    run_phase "smoke"
    run_phase "unit"
    run_phase "integration"
    
    # Summary
    echo ""
    echo "========================================="
    echo "Test Results Summary"
    echo "========================================="
    echo "Total:  ${TOTAL_TESTS}"
    echo -e "${GREEN}Passed: ${PASSED_TESTS}${NC}"
    if [[ ${FAILED_TESTS} -gt 0 ]]; then
        echo -e "${RED}Failed: ${FAILED_TESTS}${NC}"
    else
        echo -e "Failed: 0"
    fi
    echo "========================================="
    
    # Exit code based on failures
    if [[ ${FAILED_TESTS} -gt 0 ]]; then
        exit 1
    fi
    
    exit 0
}

# Execute main
main "$@"