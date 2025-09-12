#!/usr/bin/env bash
################################################################################
# Keycloak Test Runner - v2.0 Contract Compliant
# Main test orchestrator for all test phases
################################################################################

set -euo pipefail

# Determine script location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source common utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"

# Test phase mapping
declare -A TEST_PHASES=(
    ["smoke"]="test-smoke.sh"
    ["integration"]="test-integration.sh"  
    ["unit"]="test-unit.sh"
)

# Default to all tests
TEST_TYPE="${1:-all}"

# Helper functions
run_test_phase() {
    local phase="$1"
    local script="${TEST_PHASES[$phase]}"
    local test_file="${SCRIPT_DIR}/phases/${script}"
    
    if [[ ! -f "$test_file" ]]; then
        log::warning "Test phase '${phase}' not found at ${test_file}"
        return 2
    fi
    
    log::info "Running ${phase} tests..."
    if bash "$test_file"; then
        log::success "${phase} tests passed"
        return 0
    else
        log::error "${phase} tests failed"
        return 1
    fi
}

# Main test execution
main() {
    local exit_code=0
    
    case "$TEST_TYPE" in
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
            log::info "Running all test phases..."
            
            # Run tests in order: smoke -> unit -> integration
            for phase in smoke unit integration; do
                if ! run_test_phase "$phase"; then
                    exit_code=1
                    # Continue running other tests even if one fails
                fi
            done
            
            if [[ $exit_code -eq 0 ]]; then
                log::success "All tests passed"
            else
                log::error "Some tests failed"
            fi
            ;;
        *)
            log::error "Unknown test type: $TEST_TYPE"
            echo "Usage: $0 [smoke|integration|unit|all]"
            exit 1
            ;;
    esac
    
    return $exit_code
}

# Execute main function
main "$@"