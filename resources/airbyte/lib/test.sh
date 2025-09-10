#!/bin/bash
# Airbyte Test Library

set -euo pipefail

# Test: smoke (quick health check)
test_smoke() {
    log_info "Running smoke tests..."
    
    # Check if services are running
    if ! docker ps | grep -q "airbyte-server"; then
        log_error "Airbyte server is not running"
        return 1
    fi
    
    # Check health endpoint
    if ! timeout 5 curl -sf "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/health" > /dev/null; then
        log_error "Health check failed"
        return 1
    fi
    
    log_info "Smoke tests passed"
    return 0
}

# Test: integration
test_integration() {
    log_info "Running integration tests..."
    
    # Test API endpoints
    log_info "Testing source definitions endpoint..."
    if ! curl -sf "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/source_definitions/list" > /dev/null; then
        log_error "Failed to list source definitions"
        return 1
    fi
    
    log_info "Testing destination definitions endpoint..."
    if ! curl -sf "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/destination_definitions/list" > /dev/null; then
        log_error "Failed to list destination definitions"
        return 1
    fi
    
    log_info "Testing workspace endpoint..."
    if ! curl -sf "http://localhost:${AIRBYTE_SERVER_PORT}/api/v1/workspaces/list" > /dev/null; then
        log_error "Failed to list workspaces"
        return 1
    fi
    
    # Test webapp accessibility
    log_info "Testing webapp..."
    if ! timeout 5 curl -sf "http://localhost:${AIRBYTE_WEBAPP_PORT}" > /dev/null; then
        log_error "Webapp is not accessible"
        return 1
    fi
    
    log_info "Integration tests passed"
    return 0
}

# Test: unit
test_unit() {
    log_info "Running unit tests..."
    
    # Test configuration loading
    if [[ -z "${AIRBYTE_VERSION:-}" ]]; then
        log_error "AIRBYTE_VERSION not set"
        return 1
    fi
    
    if [[ -z "${AIRBYTE_WEBAPP_PORT:-}" ]]; then
        log_error "AIRBYTE_WEBAPP_PORT not set"
        return 1
    fi
    
    # Test directory structure (resource dir, not data dir which is created on install)
    if [[ ! -d "${RESOURCE_DIR}/lib" ]]; then
        log_error "lib directory not found"
        return 1
    fi
    
    if [[ ! -d "${RESOURCE_DIR}/config" ]]; then
        log_error "config directory not found"
        return 1
    fi
    
    log_info "Unit tests passed"
    return 0
}

# Test: all
test_all() {
    local failed=0
    
    test_smoke || failed=$((failed + 1))
    test_integration || failed=$((failed + 1))
    test_unit || failed=$((failed + 1))
    
    if [[ $failed -gt 0 ]]; then
        log_error "$failed test suite(s) failed"
        return 1
    fi
    
    log_info "All tests passed"
    return 0
}