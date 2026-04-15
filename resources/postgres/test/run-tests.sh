#!/usr/bin/env bash
# PostgreSQL Resource Test Runner
# Main test orchestrator following v2.0 contract

set -euo pipefail

# Source test framework
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
source "${REPO_ROOT}/scripts/lib/utils/log.sh"
source "${REPO_ROOT}/scripts/resources/lib/test-framework.sh" 2>/dev/null || true

# Test phase directory
TEST_PHASES_DIR="${RESOURCE_DIR}/test/phases"

# Run specific test phase
run_test_phase() {
    local phase="$1"
    local phase_script="${TEST_PHASES_DIR}/test-${phase}.sh"
    
    if [[ -f "$phase_script" ]]; then
        log::info "Running ${phase} tests..."
        bash "$phase_script"
        return $?
    else
        log::warn "Test phase '${phase}' not implemented"
        return 0
    fi
}

# Main test execution
main() {
    local test_type="${1:-all}"
    local exit_code=0
    
    case "$test_type" in
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
            run_test_phase "smoke" || exit_code=$?
            run_test_phase "unit" || exit_code=$?
            run_test_phase "integration" || exit_code=$?
            ;;
        *)
            log::error "Unknown test type: $test_type"
            log::info "Usage: $0 [smoke|integration|unit|all]"
            exit 1
            ;;
    esac
    
    if [[ $exit_code -eq 0 ]]; then
        log::success "All tests passed!"
    else
        log::error "Some tests failed"
    fi
    
    return $exit_code
}

main "$@"