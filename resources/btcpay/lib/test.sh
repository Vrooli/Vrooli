#!/usr/bin/env bash
# BTCPay Test Operations

btcpay::test::smoke() {
    log::info "Running BTCPay smoke test..."
    
    # Check if installed
    if ! btcpay::is_installed; then
        log::error "BTCPay is not installed"
        return 1
    fi
    
    # Check if running
    if ! btcpay::is_running; then
        log::error "BTCPay is not running"
        return 1
    fi
    
    # Check health
    local health=$(btcpay::get_health)
    if [[ "$health" != "healthy" ]]; then
        log::error "BTCPay health check failed: $health"
        return 1
    fi
    
    log::success "BTCPay smoke test passed"
    return 0
}

btcpay::test::integration() {
    log::info "Running BTCPay integration tests..."
    
    # Run smoke test first
    btcpay::test::smoke || return 1
    
    # Test API endpoint
    log::info "Testing API endpoint..."
    if curl -sf "${BTCPAY_BASE_URL}/api/v1/health" &>/dev/null; then
        log::success "API endpoint accessible"
    else
        log::error "API endpoint not accessible"
        return 1
    fi
    
    # Test database connection
    log::info "Testing database connection..."
    if docker exec "${BTCPAY_POSTGRES_CONTAINER}" pg_isready -U "${BTCPAY_POSTGRES_USER}" &>/dev/null; then
        log::success "Database connection successful"
    else
        log::error "Database connection failed"
        return 1
    fi
    
    log::success "BTCPay integration tests passed"
    return 0
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
export -f btcpay::test::performance