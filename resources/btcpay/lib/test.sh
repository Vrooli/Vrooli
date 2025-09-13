#!/usr/bin/env bash
# BTCPay Test Operations - v2.0 Contract Compliant
# Delegates to test phase scripts per universal contract

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

btcpay::test::smoke() {
    # Delegate to test phase script per v2.0 contract
    if [[ -f "${RESOURCE_DIR}/test/phases/test-smoke.sh" ]]; then
        bash "${RESOURCE_DIR}/test/phases/test-smoke.sh"
        return $?
    else
        log::error "Smoke test script not found: ${RESOURCE_DIR}/test/phases/test-smoke.sh"
        return 1
    fi
}

btcpay::test::integration() {
    # Delegate to test phase script per v2.0 contract
    if [[ -f "${RESOURCE_DIR}/test/phases/test-integration.sh" ]]; then
        bash "${RESOURCE_DIR}/test/phases/test-integration.sh"
        return $?
    else
        log::error "Integration test script not found: ${RESOURCE_DIR}/test/phases/test-integration.sh"
        return 1
    fi
}

btcpay::test::unit() {
    # Delegate to test phase script per v2.0 contract
    if [[ -f "${RESOURCE_DIR}/test/phases/test-unit.sh" ]]; then
        bash "${RESOURCE_DIR}/test/phases/test-unit.sh"
        return $?
    else
        log::error "Unit test script not found: ${RESOURCE_DIR}/test/phases/test-unit.sh"
        return 2  # Return 2 for "not available" per contract
    fi
}

btcpay::test::all() {
    # Delegate to main test runner per v2.0 contract
    if [[ -f "${RESOURCE_DIR}/test/run-tests.sh" ]]; then
        bash "${RESOURCE_DIR}/test/run-tests.sh" all
        return $?
    else
        log::error "Test runner not found: ${RESOURCE_DIR}/test/run-tests.sh"
        return 1
    fi
}

btcpay::test::performance() {
    log::info "Running BTCPay performance tests..."
    
    # Test API response time
    log::info "Testing API response time..."
    local start_time=$(date +%s%N)
    curl -sf "${BTCPAY_BASE_URL}/api/v1/health" &>/dev/null
    local end_time=$(date +%s%N)
    local response_time=$(( (end_time - start_time) / 1000000 ))
    
    if [[ $response_time -lt 1000 ]]; then
        log::success "API response time: ${response_time}ms (good)"
    elif [[ $response_time -lt 3000 ]]; then
        log::warning "API response time: ${response_time}ms (acceptable)"
    else
        log::error "API response time: ${response_time}ms (poor)"
        return 1
    fi
    
    log::success "BTCPay performance tests completed"
    return 0
}

export -f btcpay::test::smoke
export -f btcpay::test::integration
export -f btcpay::test::unit
export -f btcpay::test::all
export -f btcpay::test::performance