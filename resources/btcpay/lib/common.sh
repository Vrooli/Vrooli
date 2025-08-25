#!/bin/bash

# BTCPay Server Common Functions and Variables

set -euo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
BTCPAY_COMMON_DIR="${APP_ROOT}/resources/btcpay/lib"

# Source shared utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${APP_ROOT}/scripts/lib/docker-utils.sh"

# BTCPay constants
export BTCPAY_CONTAINER_NAME="btcpay-server"
export BTCPAY_POSTGRES_CONTAINER="btcpay-postgres"
export BTCPAY_NETWORK="btcpay-network"
export BTCPAY_IMAGE="btcpayserver/btcpayserver:1.13.5"
export BTCPAY_POSTGRES_IMAGE="postgres:14-alpine"
export BTCPAY_PORT=23000
export BTCPAY_DATA_DIR="${HOME}/Vrooli/data/btcpay"
export BTCPAY_CONFIG_DIR="${BTCPAY_DATA_DIR}/config"
export BTCPAY_POSTGRES_DATA="${BTCPAY_DATA_DIR}/postgres"
export BTCPAY_LOGS_DIR="${BTCPAY_DATA_DIR}/logs"

# BTCPay configuration
export BTCPAY_HOST="localhost:${BTCPAY_PORT}"
export BTCPAY_PROTOCOL="http"
export BTCPAY_BASE_URL="${BTCPAY_PROTOCOL}://${BTCPAY_HOST}"

# Check if BTCPay is installed
btcpay::is_installed() {
    docker image inspect "${BTCPAY_IMAGE}" &>/dev/null
}

# Check if BTCPay is running
btcpay::is_running() {
    docker::container_running "${BTCPAY_CONTAINER_NAME}"
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