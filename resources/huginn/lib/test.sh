#!/usr/bin/env bash
################################################################################
# Huginn Test Library - v2.0 Universal Contract Compliant
# 
# Test implementations for Huginn resource
################################################################################

set -euo pipefail

# Test entry point for CLI
huginn::test() {
    local test_type="${1:-all}"
    local test_runner="${HUGINN_CLI_DIR}/test/run-tests.sh"
    
    if [[ ! -f "${test_runner}" ]]; then
        log::error "Test runner not found at ${test_runner}"
        return 1
    fi
    
    # Delegate to test runner
    bash "${test_runner}" "${test_type}"
}

# Smoke test implementation
huginn::test_smoke() {
    huginn::test "smoke"
}

# Integration test implementation  
huginn::test_integration() {
    huginn::test "integration"
}

# Unit test implementation
huginn::test_unit() {
    huginn::test "unit"
}

# All tests implementation
huginn::test_all() {
    huginn::test "all"
}

# Health check for testing
huginn::health_check() {
    log::info "Running Huginn health check..."
    
    local health_url="${HUGINN_BASE_URL:-http://localhost:${HUGINN_PORT}}/health"
    local max_wait=5
    
    # First check if container is running
    if ! docker ps --filter "name=${CONTAINER_NAME}" --format "{{.Names}}" 2>/dev/null | grep -q "${CONTAINER_NAME}"; then
        log::warn "Huginn container is not running"
        return 1
    fi
    
    # Check health endpoint
    if timeout ${max_wait} curl -sf "${health_url}" &>/dev/null || \
       timeout ${max_wait} curl -sf "${HUGINN_BASE_URL}" &>/dev/null; then
        log::success "Health check passed"
        return 0
    else
        log::error "Health check failed - service not responding"
        return 1
    fi
}

# Validate test environment
huginn::validate_test_environment() {
    local errors=0
    
    # Check Docker
    if ! command -v docker &>/dev/null; then
        log::error "Docker is not installed"
        ((errors++))
    fi
    
    # Check curl
    if ! command -v curl &>/dev/null; then
        log::error "curl is not installed"
        ((errors++))
    fi
    
    # Check test directory structure
    if [[ ! -d "${HUGINN_CLI_DIR}/test/phases" ]]; then
        log::error "Test phases directory missing"
        ((errors++))
    fi
    
    return ${errors}
}

# Quick validation for CI/CD
huginn::quick_test() {
    log::info "Running quick validation tests..."
    
    # Just run smoke tests for quick validation
    huginn::test_smoke
}