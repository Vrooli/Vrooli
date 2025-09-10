#!/usr/bin/env bash
set -euo pipefail

# Main test runner for Pandas-AI

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
TEST_DIR="${APP_ROOT}/resources/pandas-ai/test"
PHASES_DIR="${TEST_DIR}/phases"

source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Run specific test phase
run_test_phase() {
    local phase="${1:-}"
    local phase_script="${PHASES_DIR}/test-${phase}.sh"
    
    if [[ ! -f "${phase_script}" ]]; then
        log::warn "Test phase '${phase}' not found at ${phase_script}"
        return 2
    fi
    
    log::info "Running ${phase} tests..."
    bash "${phase_script}"
}

# Main test execution
main() {
    local test_type="${1:-all}"
    
    case "${test_type}" in
        all)
            log::header "Running all Pandas-AI tests"
            local failed=0
            
            # Run each phase
            for phase in smoke integration unit; do
                if run_test_phase "${phase}"; then
                    log::success "✓ ${phase} tests passed"
                else
                    log::error "✗ ${phase} tests failed"
                    failed=1
                fi
            done
            
            if [[ ${failed} -eq 0 ]]; then
                log::success "All tests passed!"
                exit 0
            else
                log::error "Some tests failed"
                exit 1
            fi
            ;;
        smoke|integration|unit)
            run_test_phase "${test_type}"
            ;;
        *)
            log::error "Unknown test type: ${test_type}"
            echo "Usage: $0 [all|smoke|integration|unit]"
            exit 1
            ;;
    esac
}

main "$@"