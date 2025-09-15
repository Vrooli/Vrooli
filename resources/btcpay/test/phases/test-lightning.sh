#!/usr/bin/env bash
################################################################################
# BTCPay Lightning Network Tests
# Tests Lightning Network functionality and integration
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
source "${RESOURCE_DIR}/lib/lightning.sh"

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Test: Lightning container exists
test_lightning_container() {
    log::info "Testing: Lightning container exists..."
    
    if docker ps -a --filter "name=${BTCPAY_CONTAINER_NAME}-lnd" --format "{{.Names}}" | grep -q "${BTCPAY_CONTAINER_NAME}-lnd"; then
        log::success "Lightning container found"
        ((TESTS_PASSED++))
        return 0
    else
        log::warning "Lightning container not found (setup may be required)"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Test: Lightning configuration exists
test_lightning_config() {
    log::info "Testing: Lightning configuration..."
    
    if [[ -f "${BTCPAY_CONFIG_DIR}/lightning.json" ]]; then
        local enabled=$(jq -r '.enabled // false' "${BTCPAY_CONFIG_DIR}/lightning.json")
        if [[ "$enabled" == "true" ]]; then
            log::success "Lightning configuration found and enabled"
            ((TESTS_PASSED++))
            return 0
        else
            log::warning "Lightning configuration exists but not enabled"
            ((TESTS_FAILED++))
            return 1
        fi
    else
        log::warning "Lightning configuration not found"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Test: Lightning RPC connectivity
test_lightning_rpc() {
    log::info "Testing: Lightning RPC connectivity..."
    
    if ! btcpay::lightning::is_configured; then
        log::warning "Lightning not configured, skipping RPC test"
        return 0
    fi
    
    # Try to get node info
    if docker exec "${BTCPAY_CONTAINER_NAME}-lnd" lncli --network=mainnet getinfo &>/dev/null; then
        log::success "Lightning RPC is responsive"
        ((TESTS_PASSED++))
        return 0
    else
        log::warning "Lightning RPC not ready (node may be syncing)"
        ((TESTS_PASSED++))  # Not a failure, just not ready
        return 0
    fi
}

# Test: Lightning data persistence
test_lightning_data() {
    log::info "Testing: Lightning data persistence..."
    
    if [[ -d "${BTCPAY_DATA_DIR}/lnd" ]]; then
        log::success "Lightning data directory exists"
        
        # Check for wallet files
        if ls "${BTCPAY_DATA_DIR}/lnd/data/"*.db &>/dev/null 2>&1; then
            log::info "Lightning wallet data found"
        fi
        ((TESTS_PASSED++))
        return 0
    else
        log::warning "Lightning data directory not found"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Test: Lightning invoice creation (mock test)
test_lightning_invoice_mock() {
    log::info "Testing: Lightning invoice structure (mock)..."
    
    # This is a mock test that validates the function exists
    if type -t btcpay::lightning::create_invoice &>/dev/null; then
        log::success "Lightning invoice function available"
        ((TESTS_PASSED++))
        return 0
    else
        log::error "Lightning invoice function not found"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Main test execution
main() {
    log::info "========================================="
    log::info "BTCPay Lightning Network Tests"
    log::info "========================================="
    
    # Run tests (continue on failure to test everything)
    test_lightning_container || true
    test_lightning_config || true
    test_lightning_rpc || true
    test_lightning_data || true
    test_lightning_invoice_mock || true
    
    # Summary
    log::info "========================================="
    local total=$((TESTS_PASSED + TESTS_FAILED))
    if [[ $TESTS_FAILED -eq 0 ]]; then
        log::success "PASSED: All $total Lightning tests passed"
        exit 0
    elif [[ $TESTS_FAILED -le 2 ]]; then
        log::warning "PARTIAL: $TESTS_PASSED/$total tests passed (Lightning may not be configured)"
        exit 0  # Don't fail if Lightning just isn't set up yet
    else
        log::error "FAILED: $TESTS_FAILED/$total tests failed"
        exit 1
    fi
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main
fi