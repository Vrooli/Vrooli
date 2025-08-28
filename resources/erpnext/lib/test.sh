#!/bin/bash
# ERPNext Test Functions - Health checks and validation only

# Smoke test - basic health check
erpnext::test::smoke() {
    log::info "Running ERPNext smoke test..."
    
    # Test if containers are running
    if ! erpnext::is_running; then
        log::error "ERPNext containers are not running"
        return 1
    fi
    
    # Test if ERPNext is healthy
    if ! erpnext::is_healthy; then
        log::error "ERPNext is not responding to health checks"
        return 1
    fi
    
    # Test if port is accessible
    if command -v curl >/dev/null; then
        if ! curl -f -s "http://localhost:${ERPNEXT_PORT}" >/dev/null; then
            log::error "ERPNext web interface is not accessible on port ${ERPNEXT_PORT}"
            return 1
        fi
    fi
    
    log::success "ERPNext smoke test passed"
    return 0
}

# Integration test - comprehensive health validation
erpnext::test::integration() {
    log::info "Running ERPNext integration test..."
    
    # Run smoke test first
    erpnext::test::smoke || return 1
    
    # Test database connectivity
    if ! docker exec erpnext-app bench --site "${ERPNEXT_SITE_NAME}" mariadb --execute "SELECT 1" >/dev/null 2>&1; then
        log::error "Database connectivity test failed"
        return 1
    fi
    
    # Test Redis connectivity
    if ! docker exec erpnext-redis redis-cli ping >/dev/null 2>&1; then
        log::error "Redis connectivity test failed"
        return 1
    fi
    
    # Test if apps are loaded correctly
    if ! docker exec erpnext-app bench version >/dev/null 2>&1; then
        log::error "ERPNext apps version check failed"
        return 1
    fi
    
    log::success "ERPNext integration test passed"
    return 0
}

# Unit test - individual component validation (optional)
erpnext::test::unit() {
    log::info "Running ERPNext unit tests..."
    
    # Test Docker image availability
    if ! docker images | grep -q "frappe/erpnext"; then
        log::error "ERPNext Docker image not found"
        return 1
    fi
    
    # Test configuration files
    if [ ! -f "${ERPNEXT_RESOURCE_DIR}/config/defaults.sh" ]; then
        log::error "ERPNext configuration file not found"
        return 1
    fi
    
    # Test data directory structure
    local data_dir="${HOME}/.erpnext"
    if [ ! -d "$data_dir" ]; then
        log::warn "ERPNext data directory not found (expected after installation)"
    fi
    
    log::success "ERPNext unit tests passed"
    return 0
}

# Performance test - validate ERPNext performance (custom test phase)
erpnext::test::performance() {
    log::info "Running ERPNext performance test..."
    
    if ! erpnext::is_running; then
        log::error "ERPNext must be running for performance tests"
        return 1
    fi
    
    # Test response time
    if command -v curl >/dev/null; then
        local response_time
        response_time=$(curl -o /dev/null -s -w '%{time_total}' "http://localhost:${ERPNEXT_PORT}")
        
        # Convert to milliseconds for comparison
        local response_ms
        response_ms=$(echo "$response_time * 1000" | bc -l 2>/dev/null || echo "1000")
        
        if (( $(echo "$response_ms > 5000" | bc -l 2>/dev/null || echo 1) )); then
            log::warn "ERPNext response time is slow: ${response_ms}ms"
        else
            log::success "ERPNext response time acceptable: ${response_ms}ms"
        fi
    fi
    
    # Test memory usage
    local memory_usage
    memory_usage=$(docker stats erpnext-app --no-stream --format "table {{.MemUsage}}" 2>/dev/null | tail -n 1 | cut -d'/' -f1 | sed 's/[^0-9.]//g' || echo "0")
    
    if [ -n "$memory_usage" ] && (( $(echo "$memory_usage > 1000" | bc -l 2>/dev/null || echo 0) )); then
        log::warn "ERPNext memory usage is high: ${memory_usage}MB"
    else
        log::success "ERPNext memory usage acceptable: ${memory_usage}MB"
    fi
    
    log::success "ERPNext performance test completed"
    return 0
}