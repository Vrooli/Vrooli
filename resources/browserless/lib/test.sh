#!/usr/bin/env bash
################################################################################
# Browserless Test Implementation - v2.0 Contract Compliant
# 
# Implements test functions that are called by the CLI framework.
# This file does NOT run tests directly - it delegates to test phases.
#
# Required by v2.0 universal contract at:
# /scripts/resources/contracts/v2.0/universal.yaml
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCRIPT_DIR}/../../.." && builtin pwd)}"
TEST_DIR="${SCRIPT_DIR}/../test"
TEST_PHASES_DIR="${TEST_DIR}/phases"

# Source logging utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh" || { echo "FATAL: Failed to load variable definitions" >&2; exit 1; }
# shellcheck disable=SC1091
source "${var_LOG_FILE}" || { echo "FATAL: Failed to load logging library" >&2; exit 1; }

################################################################################
# Test Phase: Smoke (Quick health validation)
# Duration: <30s per v2.0 contract
################################################################################
browserless::test::smoke() {
    log::info "Running Browserless smoke test..."
    
    if [[ ! -f "${TEST_PHASES_DIR}/test-smoke.sh" ]]; then
        log::error "Smoke test script not found at ${TEST_PHASES_DIR}/test-smoke.sh"
        return 1
    fi
    
    # Delegate to smoke test phase
    bash "${TEST_PHASES_DIR}/test-smoke.sh"
    return $?
}

################################################################################
# Test Phase: Integration (End-to-end functionality)
# Duration: <120s per v2.0 contract
################################################################################
browserless::test::integration() {
    log::info "Running Browserless integration test..."
    
    if [[ ! -f "${TEST_PHASES_DIR}/test-integration.sh" ]]; then
        log::error "Integration test script not found at ${TEST_PHASES_DIR}/test-integration.sh"
        return 1
    fi
    
    # Delegate to integration test phase
    bash "${TEST_PHASES_DIR}/test-integration.sh"
    return $?
}

################################################################################
# Test Phase: Unit (Library function validation)
# Duration: <60s per v2.0 contract
################################################################################
browserless::test::unit() {
    log::info "Running Browserless unit tests..."
    
    if [[ ! -f "${TEST_PHASES_DIR}/test-unit.sh" ]]; then
        log::error "Unit test script not found at ${TEST_PHASES_DIR}/test-unit.sh"
        return 1
    fi
    
    # Delegate to unit test phase
    bash "${TEST_PHASES_DIR}/test-unit.sh"
    return $?
}

################################################################################
# Test Phase: All (Run all test phases)
# Duration: <600s per v2.0 contract
################################################################################
browserless::test::all() {
    log::info "Running all Browserless tests..."
    
    local overall_status=0
    local test_results=""
    
    # Run smoke tests
    log::info "Phase 1/3: Smoke tests..."
    if browserless::test::smoke; then
        test_results="${test_results}✓ Smoke tests passed\n"
    else
        test_results="${test_results}✗ Smoke tests failed\n"
        overall_status=1
    fi
    
    # Run integration tests
    log::info "Phase 2/3: Integration tests..."
    if browserless::test::integration; then
        test_results="${test_results}✓ Integration tests passed\n"
    else
        test_results="${test_results}✗ Integration tests failed\n"
        overall_status=1
    fi
    
    # Run unit tests
    log::info "Phase 3/3: Unit tests..."
    if browserless::test::unit; then
        test_results="${test_results}✓ Unit tests passed\n"
    else
        test_results="${test_results}✗ Unit tests failed\n"
        overall_status=1
    fi
    
    # Display summary
    echo ""
    echo "Test Results Summary:"
    echo "────────────────────"
    echo -e "$test_results"
    
    if [[ $overall_status -eq 0 ]]; then
        log::success "🎉 All Browserless tests PASSED"
    else
        log::error "💥 Some Browserless tests FAILED"
    fi
    
    return $overall_status
}

# Export functions for CLI framework
export -f browserless::test::smoke
export -f browserless::test::integration
export -f browserless::test::unit
export -f browserless::test::all