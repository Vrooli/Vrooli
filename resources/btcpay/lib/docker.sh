#!/usr/bin/env bash
# BTCPay Docker Operations

btcpay::docker::start() {
    log::info "Starting BTCPay Server containers..."
    
    # Check for multi-currency configuration
    local chains="btc"
    if [[ -f "${BTCPAY_CONFIG_DIR}/multicurrency.json" ]]; then
        local configured_chains=$(jq -r '.chains // "btc"' "${BTCPAY_CONFIG_DIR}/multicurrency.json")
        if [[ -n "$configured_chains" ]]; then
            chains="$configured_chains"
            log::info "Using multi-currency configuration: ${chains}"
        fi
    fi
    
    # Start PostgreSQL first
    if docker ps -a --format '{{.Names}}' | grep -q "^${BTCPAY_POSTGRES_CONTAINER}$"; then
        docker start "${BTCPAY_POSTGRES_CONTAINER}" || return 1
    else
        log::warning "PostgreSQL container not found, creating..."
        docker run -d \
            --name "${BTCPAY_POSTGRES_CONTAINER}" \
            --network "${BTCPAY_NETWORK}" \
            -e POSTGRES_USER="${BTCPAY_POSTGRES_USER}" \
            -e POSTGRES_PASSWORD="${BTCPAY_POSTGRES_PASSWORD}" \
            -e POSTGRES_DB="${BTCPAY_POSTGRES_DB}" \
            -v "${BTCPAY_POSTGRES_DATA}:/var/lib/postgresql/data" \
            "${BTCPAY_POSTGRES_IMAGE}" || return 1
    fi
    
    # Wait for PostgreSQL to be ready
    log::info "Waiting for PostgreSQL to be ready..."
    local retries=30
    while [[ $retries -gt 0 ]]; do
        if docker exec "${BTCPAY_POSTGRES_CONTAINER}" pg_isready -U "${BTCPAY_POSTGRES_USER}" &>/dev/null; then
            log::success "PostgreSQL is ready"
            break
        fi
        ((retries--))
        sleep 1
    done
    
    if [[ $retries -eq 0 ]]; then
        log::error "PostgreSQL failed to start"
        return 1
    fi
    
    # Start NBXplorer for blockchain synchronization
    if docker ps -a --format '{{.Names}}' | grep -q "^${BTCPAY_NBXPLORER_CONTAINER}$"; then
        docker start "${BTCPAY_NBXPLORER_CONTAINER}" || return 1
    else
        log::warning "NBXplorer container not found, creating..."
        docker run -d \
            --name "${BTCPAY_NBXPLORER_CONTAINER}" \
            --network "${BTCPAY_NETWORK}" \
            -p "24444:32838" \
            -v "${BTCPAY_NBXPLORER_DATA}:/datadir" \
            -e NBXPLORER_NETWORK="regtest" \
            -e NBXPLORER_BIND="0.0.0.0:32838" \
            -e NBXPLORER_CHAINS="${chains}" \
            -e NBXPLORER_SIGNALFILESDIR="/datadir" \
            -e NBXPLORER_POSTGRES="Server=${BTCPAY_POSTGRES_CONTAINER};Port=5432;Database=${BTCPAY_POSTGRES_DB};User Id=${BTCPAY_POSTGRES_USER};Password=${BTCPAY_POSTGRES_PASSWORD};" \
            "${BTCPAY_NBXPLORER_IMAGE}" || return 1
    fi
    
    # Wait for NBXplorer to be ready
    log::info "Waiting for NBXplorer to be ready (may take up to 60 seconds)..."
    retries=30
    while [[ $retries -gt 0 ]]; do
        # Check if NBXplorer is responding to health check
        if docker exec "${BTCPAY_NBXPLORER_CONTAINER}" test -f /datadir/RegTest/settings.config 2>/dev/null; then
            log::success "NBXplorer is ready"
            break
        fi
        ((retries--))
        sleep 2
    done
    
    # Give NBXplorer a bit more time to fully initialize
    sleep 3
    
    # Start BTCPay Server
    if docker ps -a --format '{{.Names}}' | grep -q "^${BTCPAY_CONTAINER_NAME}$"; then
        docker start "${BTCPAY_CONTAINER_NAME}" || return 1
    else
        log::warning "BTCPay container not found, creating..."
        docker run -d \
            --name "${BTCPAY_CONTAINER_NAME}" \
            --network "${BTCPAY_NETWORK}" \
            -p "${BTCPAY_PORT}:49392" \
            -v "${BTCPAY_DATA_DIR}:/datadir" \
            -v "${BTCPAY_CONFIG_DIR}:/root/.btcpayserver" \
            -e BTCPAY_POSTGRES="Server=${BTCPAY_POSTGRES_CONTAINER};Port=5432;Database=${BTCPAY_POSTGRES_DB};User Id=${BTCPAY_POSTGRES_USER};Password=${BTCPAY_POSTGRES_PASSWORD};" \
            -e BTCPAY_EXPLORERURL="http://${BTCPAY_NBXPLORER_CONTAINER}:32838/" \
            -e BTCPAY_CHAINS="${chains}" \
            -e BTCPAY_ROOTPATH="/" \
            -e BTCPAY_PORT=49392 \
            "${BTCPAY_IMAGE}" || return 1
    fi
    
    log::success "BTCPay Server started with NBXplorer blockchain support"
}

btcpay::docker::stop() {
    log::info "Stopping BTCPay Server containers..."
    docker stop "${BTCPAY_CONTAINER_NAME}" 2>/dev/null || true
    docker stop "${BTCPAY_NBXPLORER_CONTAINER}" 2>/dev/null || true
    docker stop "${BTCPAY_POSTGRES_CONTAINER}" 2>/dev/null || true
    log::success "BTCPay Server stopped"
}

btcpay::docker::restart() {
    btcpay::docker::stop
    sleep 2
    btcpay::docker::start
}

btcpay::docker::logs() {
    local follow="${1:-false}"
    local tail_lines="${2:-50}"
    
    if [[ "$follow" == "true" ]]; then
        docker logs -f "${BTCPAY_CONTAINER_NAME}"
    else
        docker logs --tail "$tail_lines" "${BTCPAY_CONTAINER_NAME}"
    fi
}

export -f btcpay::docker::start
export -f btcpay::docker::stop
export -f btcpay::docker::restart
export -f btcpay::docker::logs