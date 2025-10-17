#!/usr/bin/env bash
################################################################################
# Cline Test Runner - v2.0 Universal Contract Compliant
# 
# Orchestrates all test phases for the Cline resource
#
# Usage:
#   ./test/run-tests.sh [phase]
#   
# Phases:
#   smoke       - Quick health check (<30s)
#   integration - Full functionality test (<120s)
#   unit        - Library function tests (<60s)
#   all         - Run all test phases
#
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"
TEST_PHASES_DIR="${SCRIPT_DIR}/phases"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

# Test configuration
export TEST_TIMEOUT_SMOKE=30
export TEST_TIMEOUT_INTEGRATION=120
export TEST_TIMEOUT_UNIT=60
export TEST_TIMEOUT_ALL=180

# Test results tracking
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

#######################################
# Run a test phase with timeout
# Arguments:
#   phase: Test phase to run
#   timeout: Timeout in seconds
# Returns:
#   0 if test passes, 1 otherwise
#######################################
run_test_phase() {
    local phase="$1"
    local timeout_val="$2"
    local test_script="${TEST_PHASES_DIR}/test-${phase}.sh"
    
    if [[ ! -f "$test_script" ]]; then
        log::warn "Test phase '${phase}' not found at ${test_script}"
        ((TESTS_SKIPPED++))
        return 0
    fi
    
    log::header "Running ${phase} tests (timeout: ${timeout_val}s)"
    
    if timeout "${timeout_val}" bash "${test_script}"; then
        log::success "✅ ${phase} tests passed"
        ((TESTS_PASSED++))
        return 0
    else
        log::error "❌ ${phase} tests failed"
        ((TESTS_FAILED++))
        return 1
    fi
}

#######################################
# Main test execution
#######################################
main() {
    local phase="${1:-all}"
    local exit_code=0
    
    log::header "🧪 Cline Resource Test Suite"
    log::info "Test phase: ${phase}"
    echo ""
    
    case "$phase" in
        smoke)
            run_test_phase "smoke" "$TEST_TIMEOUT_SMOKE" || exit_code=1
            ;;
            
        integration)
            run_test_phase "integration" "$TEST_TIMEOUT_INTEGRATION" || exit_code=1
            ;;
            
        unit)
            run_test_phase "unit" "$TEST_TIMEOUT_UNIT" || exit_code=1
            ;;
            
        all)
            # Run all phases in sequence
            run_test_phase "smoke" "$TEST_TIMEOUT_SMOKE" || exit_code=1
            echo ""
            run_test_phase "unit" "$TEST_TIMEOUT_UNIT" || exit_code=1
            echo ""
            run_test_phase "integration" "$TEST_TIMEOUT_INTEGRATION" || exit_code=1
            ;;
            
        *)
            log::error "Unknown test phase: ${phase}"
            echo "Usage: $0 [smoke|integration|unit|all]"
            exit 1
            ;;
    esac
    
    # Test summary
    echo ""
    log::header "📊 Test Summary"
    log::info "Tests Passed: ${TESTS_PASSED}"
    [[ $TESTS_FAILED -gt 0 ]] && log::error "Tests Failed: ${TESTS_FAILED}"
    [[ $TESTS_SKIPPED -gt 0 ]] && log::warn "Tests Skipped: ${TESTS_SKIPPED}"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        log::success "✅ All tests passed!"
    else
        log::error "❌ Some tests failed"
    fi
    
    exit $exit_code
}

# Execute main function
main "$@"