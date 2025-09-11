#!/usr/bin/env bash
################################################################################
# SimPy Test Runner - v2.0 Contract Compliant
# 
# Main test orchestrator that delegates to phase-specific test scripts
################################################################################

set -euo pipefail

# Get script directory
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="$(cd "$RESOURCE_DIR/../.." && pwd)"

# Source common utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Test phase to run
PHASE="${1:-all}"

# Run specific test phase
run_test_phase() {
    local phase="$1"
    local script="$TEST_DIR/phases/test-${phase}.sh"
    
    if [[ ! -f "$script" ]]; then
        log::error "Test phase script not found: $script"
        return 1
    fi
    
    log::header "Running SimPy ${phase} tests"
    
    if bash "$script"; then
        log::success "SimPy ${phase} tests passed"
        return 0
    else
        log::error "SimPy ${phase} tests failed"
        return 1
    fi
}

# Main test execution
main() {
    case "$PHASE" in
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
            
            # Run all test phases in order
            for phase in smoke integration unit; do
                if ! run_test_phase "$phase"; then
                    ((failed++))
                fi
            done
            
            if [[ $failed -eq 0 ]]; then
                log::success "All SimPy tests passed"
                exit 0
            else
                log::error "$failed test phase(s) failed"
                exit 1
            fi
            ;;
        *)
            log::error "Unknown test phase: $PHASE"
            log::info "Available phases: smoke, integration, unit, all"
            exit 1
            ;;
    esac
}

main "$@"