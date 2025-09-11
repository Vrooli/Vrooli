#!/bin/bash
set -euo pipefail

# FFmpeg Test Runner
# Orchestrates all test phases for the FFmpeg resource

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"
PHASES_DIR="$SCRIPT_DIR/phases"

# Test configuration
export TEST_TIMEOUT="${TEST_TIMEOUT:-120}"
export TEST_PHASE="${1:-all}"

# Load test utilities
source "$RESOURCE_DIR/lib/core.sh"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test results tracking
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

log_test() {
    local level="$1"
    shift
    case "$level" in
        "SUCCESS") echo -e "${GREEN}✅ $*${NC}" ;;
        "FAIL") echo -e "${RED}❌ $*${NC}" ;;
        "INFO") echo -e "${YELLOW}ℹ️  $*${NC}" ;;
        *) echo "$*" ;;
    esac
}

run_test_phase() {
    local phase="$1"
    local phase_script="$PHASES_DIR/test-${phase}.sh"
    
    if [[ ! -f "$phase_script" ]]; then
        log_test "FAIL" "Test phase '$phase' not found at $phase_script"
        return 1
    fi
    
    log_test "INFO" "Running $phase tests..."
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if timeout "$TEST_TIMEOUT" bash "$phase_script"; then
        log_test "SUCCESS" "$phase tests passed"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        log_test "FAIL" "$phase tests failed"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Main test execution
main() {
    log_test "INFO" "FFmpeg Resource Test Suite"
    log_test "INFO" "Test phase: $TEST_PHASE"
    
    case "$TEST_PHASE" in
        smoke)
            run_test_phase "smoke"
            ;;
        integration)
            run_test_phase "integration"
            ;;
        unit)
            run_test_phase "unit"
            ;;
        all)
            # Run all phases in order
            local failed=0
            for phase in smoke unit integration; do
                if ! run_test_phase "$phase"; then
                    failed=1
                fi
            done
            if [[ $failed -eq 1 ]]; then
                log_test "FAIL" "Some test phases failed"
            fi
            ;;
        *)
            log_test "FAIL" "Unknown test phase: $TEST_PHASE"
            echo "Usage: $0 [smoke|integration|unit|all]"
            exit 1
            ;;
    esac
    
    # Test summary
    echo ""
    log_test "INFO" "Test Summary:"
    log_test "INFO" "  Tests run: $TESTS_RUN"
    log_test "SUCCESS" "  Passed: $TESTS_PASSED"
    if [[ $TESTS_FAILED -gt 0 ]]; then
        log_test "FAIL" "  Failed: $TESTS_FAILED"
        exit 1
    fi
    
    log_test "SUCCESS" "All tests passed!"
    exit 0
}

main "$@"