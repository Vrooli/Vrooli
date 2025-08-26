#!/usr/bin/env bash
# BTCPay Server Common Functions

set -euo pipefail
export BTCPAY_BASE_URL="${BTCPAY_PROTOCOL}://${BTCPAY_HOST}"

# Check if BTCPay is installed
btcpay::is_installed() {
    docker image inspect "${BTCPAY_IMAGE}" &>/dev/null
}

# Check if BTCPay is running
btcpay::is_running() {
    docker ps --filter "name=^${BTCPAY_CONTAINER_NAME}$" --format "{{.Names}}" | grep -q "^${BTCPAY_CONTAINER_NAME}$"
}

# Get BTCPay container health
btcpay::get_health() {
    if ! btcpay::is_running; then
        echo "not_running"
        return 1
    fi
    
    # Check if API responds
    if curl -sf "${BTCPAY_BASE_URL}/api/v1/health" &>/dev/null; then
        echo "healthy"
    else
        echo "unhealthy"
    fi
}

# Get BTCPay version
btcpay::get_version() {
    if btcpay::is_running; then
        docker exec "${BTCPAY_CONTAINER_NAME}" dotnet BTCPayServer.dll --version 2>/dev/null | head -1 || echo "unknown"
    else
        echo "not_running"
    fi
}