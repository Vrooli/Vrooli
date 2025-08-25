#!/bin/bash

# BTCPay Server Start Functions

set -euo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
BTCPAY_START_DIR="${APP_ROOT}/resources/btcpay/lib"

# Source common functions
source "${BTCPAY_START_DIR}/common.sh"

# Main start function
btcpay::start() {
    log::header "Starting BTCPay Server"
    
    # Check if already running
    if btcpay::is_running; then
        log::info "BTCPay Server is already running"
        return 0
    fi
    
    # Check if installed
    if ! btcpay::is_installed; then
        log::error "BTCPay Server is not installed. Run 'install' first."
        return 1
    fi
    
    # Ensure directories exist
    mkdir -p "${BTCPAY_CONFIG_DIR}"
    mkdir -p "${BTCPAY_POSTGRES_DATA}"
    mkdir -p "${BTCPAY_LOGS_DIR}"
    
    # Ensure network exists
    docker network create "${BTCPAY_NETWORK}" 2>/dev/null || true
    
    # Start PostgreSQL if not running
    if ! docker ps --filter "name=${BTCPAY_POSTGRES_CONTAINER}" --format "{{.Names}}" | grep -q "${BTCPAY_POSTGRES_CONTAINER}"; then
        log::info "Starting PostgreSQL database..."
        docker run -d \
            --name "${BTCPAY_POSTGRES_CONTAINER}" \
            --network "${BTCPAY_NETWORK}" \
            -e POSTGRES_USER=btcpay \
            -e POSTGRES_PASSWORD=btcpay123 \
            -e POSTGRES_DB=btcpayserver \
            -v "${BTCPAY_POSTGRES_DATA}:/var/lib/postgresql/data" \
            --restart unless-stopped \
            "${BTCPAY_POSTGRES_IMAGE}"
        
        # Wait for PostgreSQL to be ready
        log::info "Waiting for PostgreSQL to be ready..."
        sleep 5
    fi
    
    # Start BTCPay Server
    log::info "Starting BTCPay Server container..."
    docker run -d \
        --name "${BTCPAY_CONTAINER_NAME}" \
        --network "${BTCPAY_NETWORK}" \
        -p "${BTCPAY_PORT}:${BTCPAY_PORT}" \
        -v "${BTCPAY_CONFIG_DIR}:/datadir" \
        -v "${BTCPAY_LOGS_DIR}:/logs" \
        -e BTCPAY_POSTGRES="Host=${BTCPAY_POSTGRES_CONTAINER};Database=btcpayserver;Username=btcpay;Password=btcpay123" \
        -e BTCPAY_BIND="0.0.0.0:${BTCPAY_PORT}" \
        -e BTCPAY_ROOTPATH="/" \
        -e BTCPAY_PROTOCOL="http" \
        -e BTCPAY_DEBUGLOG="debug.log" \
        --restart unless-stopped \
        "${BTCPAY_IMAGE}"
    
    # Wait for startup
    log::info "Waiting for BTCPay Server to start..."
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if btcpay::is_running && curl -sf "${BTCPAY_BASE_URL}/api/v1/health" &>/dev/null; then
            log::success "BTCPay Server started successfully"
            log::info "Access BTCPay at: ${BTCPAY_BASE_URL}"
            return 0
        fi
        sleep 2
        ((attempt++))
    done
    
    log::error "BTCPay Server failed to start properly"
    docker logs "${BTCPAY_CONTAINER_NAME}" --tail 20
    return 1
}

# Export function
export -f btcpay::start