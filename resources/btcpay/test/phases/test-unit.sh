#!/usr/bin/env bash
################################################################################
# BTCPay Unit Tests - Library function validation
# Tests individual functions in isolation
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

# Test: Configuration defaults
test_config_defaults() {
    log::info "Testing: Configuration defaults..."
    
    local failed=0
    
    # Check required variables are set
    [[ -n "${BTCPAY_CONTAINER_NAME}" ]] || { log::error "BTCPAY_CONTAINER_NAME not set"; ((failed++)); }
    [[ -n "${BTCPAY_PORT}" ]] || { log::error "BTCPAY_PORT not set"; ((failed++)); }
    [[ -n "${BTCPAY_IMAGE}" ]] || { log::error "BTCPAY_IMAGE not set"; ((failed++)); }
    [[ -n "${BTCPAY_POSTGRES_USER}" ]] || { log::error "BTCPAY_POSTGRES_USER not set"; ((failed++)); }
    [[ -n "${BTCPAY_BASE_URL}" ]] || { log::error "BTCPAY_BASE_URL not set"; ((failed++)); }
    
    if [[ $failed -eq 0 ]]; then
        log::success "All configuration defaults are set"
        return 0
    else
        log::error "${failed} configuration variables missing"
        return 1
    fi
}

# Test: Common function exports
test_function_exports() {
    log::info "Testing: Function exports..."
    
    local failed=0
    local functions=(
        "btcpay::is_installed"
        "btcpay::is_running"
        "btcpay::get_health"
        "btcpay::get_version"
    )
    
    for func in "${functions[@]}"; do
        if type -t "$func" &>/dev/null; then
            log::success "Function exported: $func"
        else
            log::error "Function not exported: $func"
            ((failed++))
        fi
    done
    
    if [[ $failed -eq 0 ]]; then
        return 0
    else
        log::error "${failed} functions not properly exported"
        return 1
    fi
}

# Test: URL construction
test_url_construction() {
    log::info "Testing: URL construction..."
    
    if [[ "${BTCPAY_BASE_URL}" == "${BTCPAY_PROTOCOL}://${BTCPAY_HOST}" ]]; then
        log::success "Base URL correctly constructed: ${BTCPAY_BASE_URL}"
        return 0
    else
        log::error "URL construction mismatch"
        log::error "Expected: ${BTCPAY_PROTOCOL}://${BTCPAY_HOST}"
        log::error "Got: ${BTCPAY_BASE_URL}"
        return 1
    fi
}

# Test: Port validation
test_port_validation() {
    log::info "Testing: Port validation..."
    
    if [[ "${BTCPAY_PORT}" =~ ^[0-9]+$ ]] && [[ "${BTCPAY_PORT}" -gt 1024 ]] && [[ "${BTCPAY_PORT}" -lt 65535 ]]; then
        log::success "Port ${BTCPAY_PORT} is valid"
        return 0
    else
        log::error "Invalid port: ${BTCPAY_PORT}"
        return 1
    fi
}

# Test: Directory paths
test_directory_paths() {
    log::info "Testing: Directory paths..."
    
    local failed=0
    
    # Check path variables are set
    [[ -n "${BTCPAY_DATA_DIR}" ]] || { log::error "BTCPAY_DATA_DIR not set"; ((failed++)); }
    [[ -n "${BTCPAY_CONFIG_DIR}" ]] || { log::error "BTCPAY_CONFIG_DIR not set"; ((failed++)); }
    [[ -n "${BTCPAY_POSTGRES_DATA}" ]] || { log::error "BTCPAY_POSTGRES_DATA not set"; ((failed++)); }
    [[ -n "${BTCPAY_LOGS_DIR}" ]] || { log::error "BTCPAY_LOGS_DIR not set"; ((failed++)); }
    
    # Validate paths are absolute
    for path_var in BTCPAY_DATA_DIR BTCPAY_CONFIG_DIR BTCPAY_POSTGRES_DATA BTCPAY_LOGS_DIR; do
        local path="${!path_var}"
        if [[ "${path:0:1}" == "/" ]]; then
            log::success "${path_var} is absolute: ${path}"
        else
            log::error "${path_var} is not absolute: ${path}"
            ((failed++))
        fi
    done
    
    if [[ $failed -eq 0 ]]; then
        return 0
    else
        log::error "${failed} directory path issues"
        return 1
    fi
}

# Test: Docker image format
test_docker_image_format() {
    log::info "Testing: Docker image format..."
    
    if [[ "${BTCPAY_IMAGE}" =~ ^[a-z0-9]+(/[a-z0-9]+)?:[a-z0-9\.\-]+$ ]]; then
        log::success "Docker image format valid: ${BTCPAY_IMAGE}"
        return 0
    else
        log::error "Invalid Docker image format: ${BTCPAY_IMAGE}"
        return 1
    fi
}

# Test: Network name validation
test_network_name() {
    log::info "Testing: Network name validation..."
    
    if [[ -n "${BTCPAY_NETWORK}" ]] && [[ "${BTCPAY_NETWORK}" =~ ^[a-zA-Z0-9][a-zA-Z0-9_.-]*$ ]]; then
        log::success "Network name valid: ${BTCPAY_NETWORK}"
        return 0
    else
        log::error "Invalid network name: ${BTCPAY_NETWORK}"
        return 1
    fi
}

# Main test execution
main() {
    local failed=0
    local total=7
    
    log::info "========================================="
    log::info "BTCPay Unit Tests"
    log::info "========================================="
    
    # Run each test
    test_config_defaults || ((failed++))
    test_function_exports || ((failed++))
    test_url_construction || ((failed++))
    test_port_validation || ((failed++))
    test_directory_paths || ((failed++))
    test_docker_image_format || ((failed++))
    test_network_name || ((failed++))
    
    # Report results
    log::info "========================================="
    if [[ $failed -eq 0 ]]; then
        log::success "PASSED: All ${total} unit tests passed"
        exit 0
    else
        log::error "FAILED: ${failed}/${total} tests failed"
        exit 1
    fi
}

# Execute tests
main "$@"