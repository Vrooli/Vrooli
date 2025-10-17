#!/usr/bin/env bash
# Nextcloud Test Runner

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHASES_DIR="${SCRIPT_DIR}/phases"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test statistics
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

run_test_phase() {
    local phase="$1"
    local test_file="${PHASES_DIR}/test-${phase}.sh"
    
    if [[ ! -f "$test_file" ]]; then
        echo -e "${YELLOW}⚠ Skipping $phase tests (not found)${NC}"
        return 0
    fi
    
    echo -e "\n${GREEN}▶ Running $phase tests...${NC}"
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if bash "$test_file"; then
        echo -e "${GREEN}✓ $phase tests passed${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}✗ $phase tests failed${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

main() {
    local test_type="${1:-all}"
    local exit_code=0
    
    echo "=============================="
    echo "Nextcloud Resource Test Suite"
    echo "=============================="
    
    case "$test_type" in
        all)
            run_test_phase "smoke" || exit_code=1
            run_test_phase "unit" || exit_code=1
            run_test_phase "integration" || exit_code=1
            ;;
        smoke|unit|integration)
            run_test_phase "$test_type" || exit_code=1
            ;;
        *)
            echo "Error: Unknown test type: $test_type"
            echo "Usage: $0 [all|smoke|unit|integration]"
            exit 1
            ;;
    esac
    
    # Print summary
    echo ""
    echo "=============================="
    echo "Test Summary"
    echo "=============================="
    echo "Tests Run: $TESTS_RUN"
    echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
    if [[ $TESTS_FAILED -gt 0 ]]; then
        echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
    else
        echo -e "Tests Failed: $TESTS_FAILED"
    fi
    echo ""
    
    if [[ $exit_code -eq 0 ]]; then
        echo -e "${GREEN}✓ All tests passed!${NC}"
    else
        echo -e "${RED}✗ Some tests failed${NC}"
    fi
    
    exit $exit_code
}

main "$@"