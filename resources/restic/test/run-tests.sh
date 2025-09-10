#!/usr/bin/env bash
# Restic Resource - Test Runner

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Test type
TEST_TYPE="${1:-all}"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly NC='\033[0m' # No Color

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

#######################################
# Run a test phase
#######################################
run_test_phase() {
    local phase_name="$1"
    local phase_script="$2"
    
    echo ""
    echo "Running $phase_name tests..."
    
    if [[ -f "$phase_script" ]]; then
        if bash "$phase_script"; then
            echo -e "${GREEN}✓${NC} $phase_name tests passed"
            ((TESTS_PASSED++))
        else
            echo -e "${RED}✗${NC} $phase_name tests failed"
            ((TESTS_FAILED++))
        fi
    else
        echo "  $phase_name test script not found, skipping"
    fi
}

# Main test execution
case "$TEST_TYPE" in
    smoke)
        run_test_phase "smoke" "${SCRIPT_DIR}/phases/test-smoke.sh"
        ;;
    integration)
        run_test_phase "integration" "${SCRIPT_DIR}/phases/test-integration.sh"
        ;;
    unit)
        run_test_phase "unit" "${SCRIPT_DIR}/phases/test-unit.sh"
        ;;
    all)
        run_test_phase "smoke" "${SCRIPT_DIR}/phases/test-smoke.sh"
        run_test_phase "integration" "${SCRIPT_DIR}/phases/test-integration.sh"
        run_test_phase "unit" "${SCRIPT_DIR}/phases/test-unit.sh"
        ;;
    *)
        echo "Error: Unknown test type: $TEST_TYPE"
        echo "Valid types: smoke, integration, unit, all"
        exit 1
        ;;
esac

# Summary
echo ""
echo "Test Summary:"
echo -e "  Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "  Failed: ${RED}$TESTS_FAILED${NC}"

if [[ $TESTS_FAILED -gt 0 ]]; then
    exit 1
fi

exit 0