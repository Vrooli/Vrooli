#!/usr/bin/env bash
################################################################################
# Haystack Test Runner - v2.0 Universal Contract Compliant
# 
# Main test orchestrator for Haystack resource
# Delegates to phase-specific test scripts
################################################################################

set -euo pipefail

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
HAYSTACK_TEST_DIR="${APP_ROOT}/resources/haystack/test"
HAYSTACK_CLI_DIR="${APP_ROOT}/resources/haystack"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Test phase to run (default: all)
TEST_PHASE="${1:-all}"

# Test result tracking
TESTS_PASSED=0
TESTS_FAILED=0
FAILED_TESTS=()

# Run a test phase
run_test_phase() {
    local phase="$1"
    local script="${HAYSTACK_TEST_DIR}/phases/test-${phase}.sh"
    
    if [[ ! -f "${script}" ]]; then
        log::error "Test phase '${phase}' not found at ${script}"
        return 1
    fi
    
    log::info "Running ${phase} tests..."
    
    if bash "${script}"; then
        log::success "${phase} tests passed"
        ((TESTS_PASSED++))
        return 0
    else
        log::error "${phase} tests failed"
        ((TESTS_FAILED++))
        FAILED_TESTS+=("${phase}")
        return 1
    fi
}

# Main test execution
main() {
    log::header "Haystack Test Suite"
    
    case "${TEST_PHASE}" in
        smoke)
            run_test_phase "smoke" || true
            ;;
        integration)
            run_test_phase "integration" || true
            ;;
        unit)
            run_test_phase "unit" || true
            ;;
        all)
            # Run all test phases in order
            for phase in smoke integration unit; do
                run_test_phase "${phase}" || true
            done
            ;;
        *)
            log::error "Unknown test phase: ${TEST_PHASE}"
            echo "Usage: $0 [smoke|integration|unit|all]"
            exit 1
            ;;
    esac
    
    # Report results
    log::header "Test Results"
    log::info "Tests passed: ${TESTS_PASSED}"
    
    if [[ ${TESTS_FAILED} -gt 0 ]]; then
        log::error "Tests failed: ${TESTS_FAILED}"
        log::error "Failed phases: ${FAILED_TESTS[*]}"
        exit 1
    else
        log::success "All tests passed!"
        exit 0
    fi
}

# Run main
main "$@"