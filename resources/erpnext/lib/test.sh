#!/usr/bin/env bash
################################################################################
# ERPNext Resource - Test Library
# 
# Test functions for ERPNext resource validation
# Delegates to test/phases/* scripts per v2.0 contract
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
ERPNEXT_RESOURCE_DIR="${APP_ROOT}/resources/erpnext"
ERPNEXT_TEST_DIR="${ERPNEXT_RESOURCE_DIR}/test"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh" || return 1
# shellcheck disable=SC2154  # var_LIB_UTILS_DIR is set by var.sh
source "${var_LIB_UTILS_DIR}/format.sh" || return 1
source "${var_LIB_UTILS_DIR}/log.sh" || return 1

################################################################################
# Test Delegation Functions (v2.0 Contract)
################################################################################

# Smoke test - delegates to test/phases/test-smoke.sh
erpnext::test::smoke() {
    log::info "Running ERPNext smoke tests..."
    
    if [[ ! -f "${ERPNEXT_TEST_DIR}/phases/test-smoke.sh" ]]; then
        log::error "Smoke test script not found"
        return 2  # Not available
    fi
    
    bash "${ERPNEXT_TEST_DIR}/phases/test-smoke.sh"
    return $?
}

# Integration test - delegates to test/phases/test-integration.sh
erpnext::test::integration() {
    log::info "Running ERPNext integration tests..."
    
    if [[ ! -f "${ERPNEXT_TEST_DIR}/phases/test-integration.sh" ]]; then
        log::error "Integration test script not found"
        return 2  # Not available
    fi
    
    bash "${ERPNEXT_TEST_DIR}/phases/test-integration.sh"
    return $?
}

# Unit test - delegates to test/phases/test-unit.sh
erpnext::test::unit() {
    log::info "Running ERPNext unit tests..."
    
    if [[ ! -f "${ERPNEXT_TEST_DIR}/phases/test-unit.sh" ]]; then
        log::error "Unit test script not found"
        return 2  # Not available
    fi
    
    bash "${ERPNEXT_TEST_DIR}/phases/test-unit.sh"
    return $?
}

# All tests - delegates to test/run-tests.sh
erpnext::test::all() {
    log::info "Running all ERPNext tests..."
    
    if [[ ! -f "${ERPNEXT_TEST_DIR}/run-tests.sh" ]]; then
        log::error "Main test runner not found"
        return 2  # Not available
    fi
    
    bash "${ERPNEXT_TEST_DIR}/run-tests.sh" all
    return $?
}

# Performance test - custom test phase (beyond v2.0 requirements)
erpnext::test::performance() {
    log::info "Running ERPNext performance tests..."
    
    # Create performance test script if needed
    local perf_test="${ERPNEXT_TEST_DIR}/phases/test-performance.sh"
    if [[ ! -f "$perf_test" ]]; then
        log::warn "Performance test not yet implemented"
        return 2  # Not available
    fi
    
    bash "$perf_test"
    return $?
}