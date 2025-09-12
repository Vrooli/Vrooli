#!/usr/bin/env bash
################################################################################
# ERPNext Resource - Main Test Runner
# 
# Orchestrates all test phases for ERPNext resource validation
# Implements v2.0 universal contract test requirements
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
ERPNEXT_TEST_DIR="${APP_ROOT}/resources/erpnext/test"
ERPNEXT_PHASES_DIR="${ERPNEXT_TEST_DIR}/phases"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh" || exit 1
source "${var_LIB_UTILS_DIR}/format.sh" || exit 1
source "${var_LIB_UTILS_DIR}/log.sh" || exit 1

# Test phase to run (default: all)
TEST_PHASE="${1:-all}"

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0
FAILED_PHASES=()

################################################################################
# Helper Functions
################################################################################

run_test_phase() {
    local phase_name="$1"
    local phase_script="${ERPNEXT_PHASES_DIR}/test-${phase_name}.sh"
    
    if [[ ! -f "$phase_script" ]]; then
        log::warn "Test phase '${phase_name}' not found: ${phase_script}"
        return 2  # Not available exit code
    fi
    
    log::info "Running ${phase_name} tests..."
    
    if bash "$phase_script"; then
        log::success "${phase_name} tests passed"
        ((TESTS_PASSED++))
        return 0
    else
        log::error "${phase_name} tests failed"
        ((TESTS_FAILED++))
        FAILED_PHASES+=("$phase_name")
        return 1
    fi
}

################################################################################
# Main Test Execution
################################################################################

main() {
    log::info "=== ERPNext Resource Test Suite ==="
    log::info "Test phase: ${TEST_PHASE}"
    echo
    
    case "$TEST_PHASE" in
        all)
            # Run all test phases in order
            for phase in smoke unit integration; do
                run_test_phase "$phase" || true  # Continue even if a phase fails
            done
            ;;
        smoke|unit|integration)
            # Run specific phase
            run_test_phase "$TEST_PHASE"
            exit $?
            ;;
        *)
            log::error "Unknown test phase: ${TEST_PHASE}"
            log::info "Available phases: all, smoke, unit, integration"
            exit 1
            ;;
    esac
    
    # Report results
    echo
    log::info "=== Test Results ==="
    log::info "Passed: ${TESTS_PASSED}"
    log::info "Failed: ${TESTS_FAILED}"
    
    if [[ ${TESTS_FAILED} -gt 0 ]]; then
        log::error "Failed phases: ${FAILED_PHASES[*]}"
        exit 1
    else
        log::success "All tests passed!"
        exit 0
    fi
}

# Run main function
main "$@"