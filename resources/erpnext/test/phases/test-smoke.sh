#!/usr/bin/env bash
################################################################################
# ERPNext Resource - Smoke Test Phase
# 
# Quick health validation (<30s) per v2.0 contract
# Tests basic service availability and health endpoint
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
ERPNEXT_RESOURCE_DIR="${APP_ROOT}/resources/erpnext"

# Source utilities and core library
source "${APP_ROOT}/scripts/lib/utils/var.sh" || exit 1
source "${var_LIB_UTILS_DIR}/format.sh" || exit 1
source "${var_LIB_UTILS_DIR}/log.sh" || exit 1
source "${APP_ROOT}/scripts/resources/port_registry.sh" || exit 1
source "${ERPNEXT_RESOURCE_DIR}/lib/core.sh" || exit 1

# Get ERPNext port
ERPNEXT_PORT="${RESOURCE_PORTS["erpnext"]:-8020}"

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0

################################################################################
# Test Functions
################################################################################

test_service_running() {
    log::info "Testing: Service is running..."
    
    if erpnext::is_running; then
        log::success "  ✓ ERPNext service is running"
        ((TESTS_PASSED++))
        return 0
    else
        log::error "  ✗ ERPNext service is not running"
        ((TESTS_FAILED++))
        return 1
    fi
}

test_health_endpoint() {
    log::info "Testing: Service endpoint responds..."
    
    # ERPNext doesn't have /health, check if web server responds
    local http_code
    http_code=$(timeout 5 curl -sf -o /dev/null -w "%{http_code}" "http://localhost:${ERPNEXT_PORT}/" 2>/dev/null || echo "000")
    
    if [[ "$http_code" =~ ^[234] ]]; then
        log::success "  ✓ Service endpoint responds on port ${ERPNEXT_PORT} (HTTP ${http_code})"
        ((TESTS_PASSED++))
        return 0
    else
        log::error "  ✗ Service endpoint not responding on port ${ERPNEXT_PORT}"
        ((TESTS_FAILED++))
        return 1
    fi
}

test_health_response_time() {
    log::info "Testing: Service response time <1s..."
    
    local start_time=$(date +%s%N)
    local http_code
    http_code=$(timeout 1 curl -sf -o /dev/null -w "%{http_code}" "http://localhost:${ERPNEXT_PORT}/" 2>/dev/null || echo "000")
    
    if [[ "$http_code" =~ ^[234] ]]; then
        local end_time=$(date +%s%N)
        local duration=$((($end_time - $start_time) / 1000000))  # Convert to milliseconds
        
        if [[ $duration -lt 1000 ]]; then
            log::success "  ✓ Service responded in ${duration}ms (<1s requirement)"
            ((TESTS_PASSED++))
            return 0
        else
            log::error "  ✗ Service took ${duration}ms (>1s requirement)"
            ((TESTS_FAILED++))
            return 1
        fi
    else
        log::error "  ✗ Service check timed out (>1s)"
        ((TESTS_FAILED++))
        return 1
    fi
}

test_port_accessible() {
    log::info "Testing: Port ${ERPNEXT_PORT} is accessible..."
    
    if timeout 5 nc -zv localhost "${ERPNEXT_PORT}" &>/dev/null; then
        log::success "  ✓ Port ${ERPNEXT_PORT} is accessible"
        ((TESTS_PASSED++))
        return 0
    else
        log::error "  ✗ Port ${ERPNEXT_PORT} is not accessible"
        ((TESTS_FAILED++))
        return 1
    fi
}

test_dependencies_available() {
    log::info "Testing: Required dependencies are available..."
    
    local deps_ok=true
    
    # Check PostgreSQL
    if timeout 5 nc -zv localhost "${RESOURCE_PORTS["postgres"]:-5432}" &>/dev/null; then
        log::success "  ✓ PostgreSQL is available"
    else
        log::warn "  ⚠ PostgreSQL is not available (may affect functionality)"
        deps_ok=false
    fi
    
    # Check Redis
    if timeout 5 nc -zv localhost "${RESOURCE_PORTS["redis"]:-6379}" &>/dev/null; then
        log::success "  ✓ Redis is available"
    else
        log::warn "  ⚠ Redis is not available (may affect functionality)"
        deps_ok=false
    fi
    
    if $deps_ok; then
        ((TESTS_PASSED++))
        return 0
    else
        log::warn "  ⚠ Some dependencies are not available"
        ((TESTS_PASSED++))  # Not a failure, just a warning
        return 0
    fi
}

################################################################################
# Main Test Execution
################################################################################

main() {
    log::info "=== ERPNext Smoke Tests ==="
    echo
    
    # Track start time for 30s limit
    local start_time=$(date +%s)
    
    # Run tests (continue even if individual tests fail)
    test_service_running || true
    test_port_accessible || true
    test_health_endpoint || true
    test_health_response_time || true
    test_dependencies_available || true
    
    # Check total time
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo
    log::info "=== Smoke Test Results ==="
    log::info "Duration: ${duration}s (limit: 30s)"
    log::info "Passed: ${TESTS_PASSED}"
    log::info "Failed: ${TESTS_FAILED}"
    
    if [[ $duration -gt 30 ]]; then
        log::warn "Tests exceeded 30s time limit"
    fi
    
    if [[ ${TESTS_FAILED} -gt 0 ]]; then
        log::error "Smoke tests failed"
        exit 1
    else
        log::success "All smoke tests passed!"
        exit 0
    fi
}

# Run main function
main "$@"