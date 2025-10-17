#!/usr/bin/env bash
################################################################################
# BTCPay Test Runner - v2.0 Universal Contract Compliant
# Orchestrates all test phases for BTCPay Server resource
################################################################################

set -euo pipefail

# Script directory and resource root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Test phases configuration
declare -A TEST_PHASES=(
    ["smoke"]="test-smoke.sh"
    ["unit"]="test-unit.sh"
    ["integration"]="test-integration.sh"
)

# Run specific test phase
run_test_phase() {
    local phase="$1"
    local script="${TEST_PHASES[$phase]}"
    local script_path="${SCRIPT_DIR}/phases/${script}"
    
    if [[ ! -f "$script_path" ]]; then
        log::warning "Test phase '${phase}' not found at: ${script_path}"
        return 2
    fi
    
    log::info "Running ${phase} tests..."
    if bash "$script_path"; then
        log::success "${phase} tests passed"
        return 0
    else
        log::error "${phase} tests failed"
        return 1
    fi
}

# Run all test phases
run_all_tests() {
    local failed=0
    local total=0
    
    log::info "Running all BTCPay test phases..."
    
    for phase in smoke unit integration; do
        ((total++))
        if run_test_phase "$phase"; then
            log::success "✓ ${phase} tests passed"
        else
            ((failed++))
            log::error "✗ ${phase} tests failed"
        fi
    done
    
    log::info "Test Summary: $((total - failed))/${total} phases passed"
    
    if [[ $failed -eq 0 ]]; then
        log::success "All tests passed!"
        return 0
    else
        log::error "${failed} test phase(s) failed"
        return 1
    fi
}

# Main execution
main() {
    local phase="${1:-all}"
    
    case "$phase" in
        all)
            run_all_tests
            ;;
        smoke|unit|integration)
            run_test_phase "$phase"
            ;;
        *)
            log::error "Unknown test phase: ${phase}"
            echo "Usage: $0 [all|smoke|unit|integration]"
            exit 1
            ;;
    esac
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi