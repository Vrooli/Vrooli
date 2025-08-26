#!/usr/bin/env bash
# BTCPay Docker Operations

btcpay::docker::start() {
    log::info "Starting BTCPay Server containers..."
    
    # Start PostgreSQL first
    if docker ps -a --format '{{.Names}}' | grep -q "^${BTCPAY_POSTGRES_CONTAINER}$"; then
        docker start "${BTCPAY_POSTGRES_CONTAINER}" || return 1
        sleep 3
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
            -e BTCPAY_POSTGRES="User ID=${BTCPAY_POSTGRES_USER};Password=${BTCPAY_POSTGRES_PASSWORD};Host=${BTCPAY_POSTGRES_CONTAINER};Port=5432;Database=${BTCPAY_POSTGRES_DB};" \
            -e BTCPAY_ROOTPATH="/" \
            -e BTCPAY_PORT=49392 \
            "${BTCPAY_IMAGE}" || return 1
    fi
    
    log::success "BTCPay Server started"
}

btcpay::docker::stop() {
    log::info "Stopping BTCPay Server containers..."
    docker stop "${BTCPAY_CONTAINER_NAME}" 2>/dev/null || true
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