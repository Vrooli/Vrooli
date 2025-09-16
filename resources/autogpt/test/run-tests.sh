#!/usr/bin/env bash
################################################################################
# AutoGPT Test Runner - v2.0 Contract Compliant
# Main test orchestration script
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"
PHASES_DIR="${SCRIPT_DIR}/phases"

# Source common utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"

# Test phase definitions
declare -A TEST_PHASES=(
    ["smoke"]="Quick health validation"
    ["integration"]="End-to-end functionality"
    ["unit"]="Library function tests"
)

# Execute test phase
run_test_phase() {
    local phase="${1}"
    local script="${PHASES_DIR}/test-${phase}.sh"
    
    if [[ ! -f "${script}" ]]; then
        log::error "Test phase '${phase}' not found at: ${script}"
        return 1
    fi
    
    log::info "Running ${phase} tests..."
    if bash "${script}"; then
        log::success "${phase} tests passed"
        return 0
    else
        log::error "${phase} tests failed"
        return 1
    fi
}

# Main test execution
main() {
    local phase="${1:-all}"
    local exit_code=0
    
    log::header "AutoGPT Test Suite"
    
    if [[ "${phase}" == "all" ]]; then
        # Run all phases in order
        for test_phase in smoke integration unit; do
            if ! run_test_phase "${test_phase}"; then
                exit_code=1
                # Continue running other tests even if one fails
            fi
        done
    elif [[ -n "${TEST_PHASES[${phase}]:-}" ]]; then
        # Run specific phase
        if ! run_test_phase "${phase}"; then
            exit_code=1
        fi
    else
        log::error "Unknown test phase: ${phase}"
        log::info "Available phases: ${!TEST_PHASES[*]} all"
        exit 1
    fi
    
    # Summary
    if [[ ${exit_code} -eq 0 ]]; then
        log::success "All tests passed"
    else
        log::error "Some tests failed"
    fi
    
    return ${exit_code}
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi