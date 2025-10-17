#!/bin/bash

# BTCPay Server Start Functions

set -euo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
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
    mkdir -p "${BTCPAY_NBXPLORER_DATA}"
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
            -e POSTGRES_PASSWORD="${BTCPAY_POSTGRES_PASSWORD}" \
            -e POSTGRES_DB=btcpayserver \
            -v "${BTCPAY_POSTGRES_DATA}:/var/lib/postgresql/data" \
            --restart unless-stopped \
            "${BTCPAY_POSTGRES_IMAGE}"
        
        # Wait for PostgreSQL to be ready
        log::info "Waiting for PostgreSQL to be ready..."
        local pg_max_attempts=30
        local pg_attempt=0
        while [ $pg_attempt -lt $pg_max_attempts ]; do
            if docker exec "${BTCPAY_POSTGRES_CONTAINER}" pg_isready -U btcpay -d btcpayserver &>/dev/null; then
                log::success "PostgreSQL is ready"
                break
            fi
            sleep 1
            ((pg_attempt++))
        done
        if [ $pg_attempt -ge $pg_max_attempts ]; then
            log::error "PostgreSQL failed to become ready"
            return 1
        fi
    fi
    
    # Start NBXplorer if not running
    if ! docker ps --filter "name=${BTCPAY_NBXPLORER_CONTAINER}" --format "{{.Names}}" | grep -q "${BTCPAY_NBXPLORER_CONTAINER}"; then
        log::info "Starting NBXplorer container..."
        docker run -d \
            --name "${BTCPAY_NBXPLORER_CONTAINER}" \
            --network "${BTCPAY_NETWORK}" \
            -v "${BTCPAY_NBXPLORER_DATA}:/datadir" \
            -e NBXPLORER_DATADIR="/datadir" \
            -e NBXPLORER_NETWORK="mainnet" \
            -e NBXPLORER_BIND="0.0.0.0:24444" \
            -e NBXPLORER_CHAINS="btc" \
            -e NBXPLORER_BTCRPCURL="http://127.0.0.1:8332/" \
            -e NBXPLORER_BTCNODEENDPOINT="127.0.0.1:8333" \
            --restart unless-stopped \
            "${BTCPAY_NBXPLORER_IMAGE}"
        
        # Wait a bit for NBXplorer to initialize
        log::info "Waiting for NBXplorer to initialize..."
        sleep 5
    fi
    
    # Start BTCPay Server
    log::info "Starting BTCPay Server container..."
    docker run -d \
        --name "${BTCPAY_CONTAINER_NAME}" \
        --network "${BTCPAY_NETWORK}" \
        -p "${BTCPAY_PORT}:49392" \
        -v "${BTCPAY_CONFIG_DIR}:/datadir" \
        -v "${BTCPAY_LOGS_DIR}:/logs" \
        -e BTCPAY_POSTGRES="Server=${BTCPAY_POSTGRES_CONTAINER};Port=5432;Database=btcpayserver;User Id=btcpay;Password=${BTCPAY_POSTGRES_PASSWORD}" \
        -e BTCPAY_BIND="0.0.0.0:49392" \
        -e BTCPAY_ROOTPATH="/" \
        -e BTCPAY_PROTOCOL="http" \
        -e BTCPAY_DEBUGLOG="debug.log" \
        -e BTCPAY_BTCEXPLORERURL="http://${BTCPAY_NBXPLORER_CONTAINER}:24444/" \
        -e BTCPAY_BTCEXPLORERCOOKIEFILE="" \
        --restart unless-stopped \
        "${BTCPAY_IMAGE}"
    
    # Wait for startup
    log::info "Waiting for BTCPay Server to start..."
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if btcpay::is_running && timeout 5 curl -sf "${BTCPAY_BASE_URL}/api/v1/health" &>/dev/null; then
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