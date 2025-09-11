#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "${SCRIPT_DIR}")"
PHASES_DIR="${SCRIPT_DIR}/phases"

# Default test type
TEST_TYPE="${1:-all}"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

echo "======================================"
echo "Parlant Resource Test Suite"
echo "======================================"
echo ""

# Run specific test phase
run_test_phase() {
    local phase="$1"
    local script="${PHASES_DIR}/test-${phase}.sh"
    
    if [[ ! -f "$script" ]]; then
        echo -e "${YELLOW}Warning: Test phase '$phase' not found${NC}"
        return 2
    fi
    
    echo -e "${GREEN}Running $phase tests...${NC}"
    
    # Make script executable
    chmod +x "$script"
    
    # Run the test
    if "$script"; then
        echo -e "${GREEN}✓ $phase tests passed${NC}"
        ((PASSED_TESTS++))
        return 0
    else
        echo -e "${RED}✗ $phase tests failed${NC}"
        ((FAILED_TESTS++))
        return 1
    fi
}

# Main test execution
case "$TEST_TYPE" in
    smoke)
        ((TOTAL_TESTS++))
        run_test_phase "smoke" || true
        ;;
    integration)
        ((TOTAL_TESTS++))
        run_test_phase "integration" || true
        ;;
    unit)
        ((TOTAL_TESTS++))
        run_test_phase "unit" || true
        ;;
    all)
        # Run all test phases
        for phase in smoke integration unit; do
            ((TOTAL_TESTS++))
            echo ""
            run_test_phase "$phase" || true
        done
        ;;
    *)
        echo -e "${RED}Error: Unknown test type '$TEST_TYPE'${NC}"
        echo "Usage: $0 [smoke|integration|unit|all]"
        exit 1
        ;;
esac

# Summary
echo ""
echo "======================================"
echo "Test Summary"
echo "======================================"
echo "Total test phases: $TOTAL_TESTS"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"

if [[ $FAILED_TESTS -eq 0 ]]; then
    echo ""
    echo -e "${GREEN}✓ All tests passed!${NC}"
    exit 0
else
    echo ""
    echo -e "${RED}✗ Some tests failed${NC}"
    exit 1
fi