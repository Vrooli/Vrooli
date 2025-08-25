#!/bin/bash

# BTCPay Server Stop Functions

set -euo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
BTCPAY_STOP_DIR="${APP_ROOT}/resources/btcpay/lib"

# Source common functions
source "${BTCPAY_STOP_DIR}/common.sh"

# Main stop function
btcpay::stop() {
    log::header "Stopping BTCPay Server"
    
    if ! btcpay::is_running; then
        log::info "BTCPay Server is not running"
        return 0
    fi
    
    # Stop BTCPay container
    log::info "Stopping BTCPay Server container..."
    docker stop "${BTCPAY_CONTAINER_NAME}" &>/dev/null || true
    docker rm "${BTCPAY_CONTAINER_NAME}" &>/dev/null || true
    
    # Optionally stop PostgreSQL
    if docker ps --filter "name=${BTCPAY_POSTGRES_CONTAINER}" --format "{{.Names}}" | grep -q "${BTCPAY_POSTGRES_CONTAINER}"; then
        log::info "Stopping PostgreSQL container..."
        docker stop "${BTCPAY_POSTGRES_CONTAINER}" &>/dev/null || true
        docker rm "${BTCPAY_POSTGRES_CONTAINER}" &>/dev/null || true
    fi
    
    log::success "BTCPay Server stopped"
    return 0
}

# Export function
export -f btcpay::stop