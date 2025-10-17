#!/usr/bin/env bash
################################################################################
# VOCR Test Runner - v2.0 Contract Compliant
#
# Main test orchestrator that runs test phases in sequence
################################################################################

set -euo pipefail

# Get script directory
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="${APP_ROOT:-$(cd "${TEST_DIR}/../../.." && pwd)}"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

# Test phase to run (default: all)
PHASE="${1:-all}"

# Track overall results
FAILED_PHASES=0
TOTAL_PHASES=0

################################################################################
# Run a test phase
################################################################################
run_phase() {
    local phase_name="$1"
    local phase_script="${TEST_DIR}/phases/test-${phase_name}.sh"
    
    ((TOTAL_PHASES++))
    
    log::header "Running ${phase_name} tests"
    
    if [[ ! -f "$phase_script" ]]; then
        log::warning "Phase script not found: $phase_script"
        ((FAILED_PHASES++))
        return 1
    fi
    
    if bash "$phase_script"; then
        log::success "✅ ${phase_name} tests passed"
        return 0
    else
        log::error "❌ ${phase_name} tests failed"
        ((FAILED_PHASES++))
        return 1
    fi
}

################################################################################
# Main execution
################################################################################
main() {
    log::header "VOCR Test Suite"
    log::info "Running test phase: ${PHASE}"
    
    case "$PHASE" in
        smoke)
            run_phase "smoke"
            ;;
        integration)
            run_phase "integration"
            ;;
        unit)
            run_phase "unit"
            ;;
        all)
            run_phase "smoke" || true
            run_phase "unit" || true
            run_phase "integration" || true
            ;;
        *)
            log::error "Unknown test phase: $PHASE"
            log::info "Available phases: smoke, integration, unit, all"
            exit 1
            ;;
    esac
    
    # Final summary
    echo ""
    log::header "Test Summary"
    
    if [[ $FAILED_PHASES -eq 0 ]]; then
        log::success "✅ All test phases passed! ($TOTAL_PHASES/$TOTAL_PHASES)"
        exit 0
    else
        log::error "❌ $FAILED_PHASES/$TOTAL_PHASES test phases failed"
        exit 1
    fi
}

# Run main function
main "$@"