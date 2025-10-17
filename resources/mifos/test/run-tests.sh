#!/usr/bin/env bash
################################################################################
# Mifos Test Runner
# 
# Main test orchestration script for Mifos resource
################################################################################

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIFOS_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${MIFOS_DIR}/../.." && pwd)"

# Source test framework
source "${APP_ROOT}/scripts/lib/utils/test-framework.sh"

# Run specific test phase
run_test_phase() {
    local phase="${1}"
    local test_script="${SCRIPT_DIR}/phases/test-${phase}.sh"
    
    if [[ -f "${test_script}" ]]; then
        log::header "Running ${phase} tests"
        bash "${test_script}"
        return $?
    else
        log::error "Test phase '${phase}' not found"
        return 1
    fi
}

# Main test runner
main() {
    local phase="${1:-all}"
    
    case "${phase}" in
        smoke|integration|unit)
            run_test_phase "${phase}"
            ;;
        all)
            local total_errors=0
            
            for test_phase in smoke integration unit; do
                if ! run_test_phase "${test_phase}"; then
                    ((total_errors++))
                fi
                echo ""
            done
            
            if [[ ${total_errors} -eq 0 ]]; then
                log::success "All test phases passed!"
                exit 0
            else
                log::error "${total_errors} test phase(s) failed"
                exit 1
            fi
            ;;
        *)
            log::error "Unknown test phase: ${phase}"
            echo "Usage: $0 [smoke|integration|unit|all]"
            exit 1
            ;;
    esac
}

main "$@"