#!/usr/bin/env bash
################################################################################
# BTCPay Smoke Tests - Quick health validation
# Must complete in <30 seconds per v2.0 contract
################################################################################

set -euo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities and BTCPay libraries
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/common.sh"

# Test timeout (30 seconds max for smoke tests)
SMOKE_TEST_TIMEOUT=30

# Test: BTCPay is installed
test_btcpay_installed() {
    log::info "Testing: BTCPay is installed..."
    
    if btcpay::is_installed; then
        log::success "BTCPay Docker image found"
        return 0
    else
        log::error "BTCPay is not installed"
        return 1
    fi
}

# Test: BTCPay is running
test_btcpay_running() {
    log::info "Testing: BTCPay is running..."
    
    if btcpay::is_running; then
        log::success "BTCPay container is running"
        return 0
    else
        log::error "BTCPay container is not running"
        return 1
    fi
}

# Test: Health check responds
test_health_check() {
    log::info "Testing: Health endpoint responds..."
    
    if timeout 5 curl -sf "${BTCPAY_BASE_URL}/api/v1/health" &>/dev/null; then
        log::success "Health endpoint responding"
        return 0
    else
        log::error "Health endpoint not responding"
        return 1
    fi
}

# Test: PostgreSQL container running
test_postgres_running() {
    log::info "Testing: PostgreSQL container running..."
    
    if docker ps --filter "name=^${BTCPAY_POSTGRES_CONTAINER}$" --format "{{.Names}}" | grep -q "^${BTCPAY_POSTGRES_CONTAINER}$"; then
        log::success "PostgreSQL container is running"
        return 0
    else
        log::error "PostgreSQL container is not running"
        return 1
    fi
}

# Test: Port is accessible
test_port_accessible() {
    log::info "Testing: Port ${BTCPAY_PORT} is accessible..."
    
    if timeout 5 nc -zv localhost "${BTCPAY_PORT}" &>/dev/null; then
        log::success "Port ${BTCPAY_PORT} is accessible"
        return 0
    else
        log::error "Port ${BTCPAY_PORT} is not accessible"
        return 1
    fi
}

# Main test execution
main() {
    local start_time=$(date +%s)
    local failed=0
    local total=5
    
    log::info "========================================="
    log::info "BTCPay Smoke Tests"
    log::info "========================================="
    
    # Run each test
    test_btcpay_installed || ((failed++))
    test_btcpay_running || ((failed++))
    test_health_check || ((failed++))
    test_postgres_running || ((failed++))
    test_port_accessible || ((failed++))
    
    # Calculate elapsed time
    local end_time=$(date +%s)
    local elapsed=$((end_time - start_time))
    
    # Check timeout
    if [[ $elapsed -gt $SMOKE_TEST_TIMEOUT ]]; then
        log::warning "Smoke tests took ${elapsed}s (exceeded ${SMOKE_TEST_TIMEOUT}s limit)"
    else
        log::info "Smoke tests completed in ${elapsed}s"
    fi
    
    # Report results
    log::info "========================================="
    if [[ $failed -eq 0 ]]; then
        log::success "PASSED: All ${total} smoke tests passed"
        exit 0
    else
        log::error "FAILED: ${failed}/${total} tests failed"
        exit 1
    fi
}

# Execute tests
main "$@"