#!/usr/bin/env bash
################################################################################
# MinIO Test Runner - v2.0 Contract Compliant
#
# Main test orchestrator that delegates to test phases
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
MINIO_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
PHASES_DIR="${SCRIPT_DIR}/phases"

# Source logging
APP_ROOT="${APP_ROOT:-$(builtin cd "${MINIO_DIR}/../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Test phase to run
PHASE="${1:-all}"

################################################################################
# Test Execution Functions
################################################################################

run_smoke_tests() {
    log::info "Running MinIO smoke tests..."
    
    if [[ -f "${PHASES_DIR}/test-smoke.sh" ]]; then
        bash "${PHASES_DIR}/test-smoke.sh"
    else
        log::error "Smoke test file not found"
        return 1
    fi
}

run_integration_tests() {
    log::info "Running MinIO integration tests..."
    
    if [[ -f "${PHASES_DIR}/test-integration.sh" ]]; then
        bash "${PHASES_DIR}/test-integration.sh"
    else
        log::error "Integration test file not found"
        return 1
    fi
}

run_unit_tests() {
    log::info "Running MinIO unit tests..."
    
    if [[ -f "${PHASES_DIR}/test-unit.sh" ]]; then
        bash "${PHASES_DIR}/test-unit.sh"
    else
        log::warning "Unit test file not found (optional)"
        return 2
    fi
}

run_all_tests() {
    log::info "Running all MinIO test phases..."
    
    local failed=0
    
    # Run smoke tests
    if ! run_smoke_tests; then
        log::error "Smoke tests failed"
        ((failed++))
    fi
    
    # Run integration tests
    if ! run_integration_tests; then
        log::error "Integration tests failed"
        ((failed++))
    fi
    
    # Run unit tests (optional)
    run_unit_tests || true
    
    if [[ $failed -gt 0 ]]; then
        log::error "Test execution failed: $failed phases failed"
        return 1
    else
        log::success "All test phases completed successfully"
        return 0
    fi
}

################################################################################
# Main Execution
################################################################################

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
        echo "Usage: $0 {smoke|integration|unit|all}"
        exit 1
        ;;
esac