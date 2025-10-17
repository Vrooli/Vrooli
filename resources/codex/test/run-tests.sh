#!/usr/bin/env bash
################################################################################
# Codex Resource Test Runner
# Main test orchestration script for all test phases
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CODEX_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${CODEX_DIR}/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Test configuration
TEST_TIMEOUT="${TEST_TIMEOUT:-600}"
PHASE="${1:-all}"

# Test phase functions
run_smoke_tests() {
    log::info "Running smoke tests..."
    timeout 30 bash "${SCRIPT_DIR}/phases/test-smoke.sh"
}

run_integration_tests() {
    log::info "Running integration tests..."
    timeout 120 bash "${SCRIPT_DIR}/phases/test-integration.sh"
}

run_unit_tests() {
    log::info "Running unit tests..."
    timeout 60 bash "${SCRIPT_DIR}/phases/test-unit.sh"
}

run_all_tests() {
    local failed=0
    
    run_smoke_tests || ((failed++))
    run_unit_tests || ((failed++))
    run_integration_tests || ((failed++))
    
    if [ $failed -gt 0 ]; then
        log::error "Tests failed: $failed phase(s) had errors"
        return 1
    fi
    
    log::success "All tests passed!"
    return 0
}

# Main execution
main() {
    log::info "Starting Codex test suite (phase: $PHASE)"
    
    case "$PHASE" in
        smoke)
            run_smoke_tests
            ;;
        integration)
            run_integration_tests
            ;;
        unit)
            run_unit_tests
            ;;
        all)
            run_all_tests
            ;;
        *)
            log::error "Unknown test phase: $PHASE"
            echo "Usage: $0 [smoke|integration|unit|all]"
            exit 1
            ;;
    esac
    
    exit $?
}

main "$@"