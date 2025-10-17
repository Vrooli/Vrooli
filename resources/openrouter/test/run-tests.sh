#!/usr/bin/env bash
################################################################################
# OpenRouter Test Runner - v2.0 Universal Contract Compliant
# 
# Orchestrates test execution for the OpenRouter resource
#
# Usage:
#   ./test/run-tests.sh [smoke|integration|unit|all]
################################################################################

set -euo pipefail

# Determine script location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"
APP_ROOT="$(cd "$RESOURCE_DIR/../.." && pwd)"

# Source common utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/config/defaults.sh"

# Test phase to run
TEST_PHASE="${1:-all}"

# Run requested test phase
run_test_phase() {
    local phase="$1"
    local test_script="${SCRIPT_DIR}/phases/test-${phase}.sh"
    
    if [[ ! -f "$test_script" ]]; then
        log::warn "Test phase '$phase' not found at $test_script"
        return 2
    fi
    
    log::info "Running $phase tests..."
    
    # Execute test phase
    if bash "$test_script"; then
        log::success "$phase tests passed"
        return 0
    else
        log::error "$phase tests failed"
        return 1
    fi
}

# Main test execution
main() {
    local exit_code=0
    
    case "$TEST_PHASE" in
        smoke)
            run_test_phase "smoke" || exit_code=$?
            ;;
        integration)
            run_test_phase "integration" || exit_code=$?
            ;;
        unit)
            run_test_phase "unit" || exit_code=$?
            ;;
        all)
            # Run all test phases in order
            for phase in smoke integration unit; do
                if ! run_test_phase "$phase"; then
                    exit_code=1
                    # Continue running other phases even if one fails
                fi
            done
            ;;
        *)
            log::error "Unknown test phase: $TEST_PHASE"
            echo "Usage: $0 [smoke|integration|unit|all]"
            exit 1
            ;;
    esac
    
    # Report overall status
    if [[ $exit_code -eq 0 ]]; then
        log::success "All requested tests passed"
    else
        log::error "Some tests failed"
    fi
    
    exit $exit_code
}

main "$@"