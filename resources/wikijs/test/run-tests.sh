#!/usr/bin/env bash
################################################################################
# Wiki.js Test Runner - v2.0 Universal Contract Compliant
# 
# Orchestrates all test phases for Wiki.js resource validation
#
# Usage:
#   ./test/run-tests.sh [phase]
#
# Phases:
#   smoke       - Quick health validation (<30s)
#   integration - End-to-end functionality (<120s)
#   unit        - Library function tests (<60s)
#   all         - Run all test phases (default)
#
################################################################################

set -euo pipefail

# Resolve paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source common test utilities
source "${APP_ROOT}/scripts/lib/utils/testing.sh" 2>/dev/null || true

# Test phase to run
PHASE="${1:-all}"

# Track test results
PASSED=0
FAILED=0
SKIPPED=0

# ANSI color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print test header
print_header() {
    echo "=================================="
    echo "Wiki.js Resource Test Suite"
    echo "Phase: ${PHASE}"
    echo "=================================="
    echo ""
}

# Run test phase
run_phase() {
    local phase="$1"
    local phase_script="${SCRIPT_DIR}/phases/test-${phase}.sh"
    
    if [[ ! -f "$phase_script" ]]; then
        echo -e "${YELLOW}[SKIP]${NC} ${phase} tests not implemented"
        ((SKIPPED++))
        return 2
    fi
    
    echo -e "${GREEN}[RUN]${NC} Starting ${phase} tests..."
    
    if bash "$phase_script"; then
        echo -e "${GREEN}[PASS]${NC} ${phase} tests completed successfully"
        ((PASSED++))
        return 0
    else
        echo -e "${RED}[FAIL]${NC} ${phase} tests failed"
        ((FAILED++))
        return 1
    fi
}

# Print test summary
print_summary() {
    echo ""
    echo "=================================="
    echo "Test Summary"
    echo "=================================="
    echo -e "Passed:  ${GREEN}${PASSED}${NC}"
    echo -e "Failed:  ${RED}${FAILED}${NC}"
    echo -e "Skipped: ${YELLOW}${SKIPPED}${NC}"
    echo ""
    
    if [[ $FAILED -gt 0 ]]; then
        echo -e "${RED}TESTS FAILED${NC}"
        return 1
    elif [[ $SKIPPED -gt 0 && $PASSED -eq 0 ]]; then
        echo -e "${YELLOW}NO TESTS RUN${NC}"
        return 2
    else
        echo -e "${GREEN}ALL TESTS PASSED${NC}"
        return 0
    fi
}

# Main test execution
main() {
    print_header
    
    case "$PHASE" in
        smoke)
            run_phase "smoke" || true
            ;;
        integration)
            run_phase "integration" || true
            ;;
        unit)
            run_phase "unit" || true
            ;;
        all)
            run_phase "smoke" || true
            run_phase "integration" || true
            run_phase "unit" || true
            ;;
        *)
            echo "Unknown test phase: $PHASE"
            echo "Valid phases: smoke, integration, unit, all"
            exit 1
            ;;
    esac
    
    print_summary
}

# Execute main function
main "$@"