#!/usr/bin/env bash
################################################################################
# Unstructured.io Test Runner - v2.0 Contract Compliant
# 
# Main test orchestrator for unstructured-io resource validation
# Delegates to phase-specific test scripts as per universal.yaml
#
# Usage:
#   ./test/run-tests.sh [phase]
#   
# Phases:
#   smoke       - Quick health validation (<30s)
#   integration - End-to-end functionality (<120s)
#   unit        - Library function validation (<60s)
#   all         - Run all test phases (<600s)
#
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"

# Test phase directory
PHASES_DIR="${SCRIPT_DIR}/phases"

# Test execution function
run_test_phase() {
    local phase="$1"
    local script="${PHASES_DIR}/test-${phase}.sh"
    
    if [[ ! -f "$script" ]]; then
        log::error "Test script not found: $script"
        return 2
    fi
    
    if [[ ! -x "$script" ]]; then
        chmod +x "$script"
    fi
    
    log::info "Running ${phase} tests..."
    
    if bash "$script"; then
        log::success "${phase} tests passed"
        return 0
    else
        log::error "${phase} tests failed"
        return 1
    fi
}

# Main test runner
main() {
    local phase="${1:-all}"
    
    case "$phase" in
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
            local failed=0
            
            # Run all phases in order
            for test_phase in smoke unit integration; do
                if ! run_test_phase "$test_phase"; then
                    failed=1
                fi
            done
            
            if [[ $failed -eq 0 ]]; then
                log::success "All tests passed"
                return 0
            else
                log::error "Some tests failed"
                return 1
            fi
            ;;
        *)
            log::error "Unknown test phase: $phase"
            echo "Usage: $0 [smoke|integration|unit|all]"
            return 1
            ;;
    esac
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi