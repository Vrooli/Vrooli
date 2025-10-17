#!/usr/bin/env bash
# Main test runner for NSFW Detector resource

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR

# Test phases directory
readonly PHASES_DIR="${SCRIPT_DIR}/phases"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m' # No Color

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Run a test phase
run_phase() {
    local phase="$1"
    local script="${PHASES_DIR}/test-${phase}.sh"
    
    if [[ ! -f "$script" ]]; then
        echo -e "${YELLOW}Skipping $phase tests (not implemented)${NC}"
        return 2
    fi
    
    echo -e "${YELLOW}Running $phase tests...${NC}"
    
    if bash "$script"; then
        echo -e "${GREEN}✓ $phase tests passed${NC}"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗ $phase tests failed${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Main test execution
main() {
    local test_type="${1:-all}"
    
    echo "=================================="
    echo "NSFW Detector Resource Test Suite"
    echo "=================================="
    echo
    
    case "$test_type" in
        all)
            run_phase "smoke" || true
            run_phase "unit" || true
            run_phase "integration" || true
            ;;
        smoke)
            run_phase "smoke"
            ;;
        unit)
            run_phase "unit"
            ;;
        integration)
            run_phase "integration"
            ;;
        *)
            echo "Error: Unknown test type: $test_type" >&2
            echo "Usage: $0 [all|smoke|unit|integration]" >&2
            exit 1
            ;;
    esac
    
    echo
    echo "=================================="
    echo "Test Results:"
    echo "  Passed: $TESTS_PASSED"
    echo "  Failed: $TESTS_FAILED"
    echo "=================================="
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed${NC}"
        exit 1
    fi
}

# Execute main
main "$@"