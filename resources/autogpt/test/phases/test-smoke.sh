#!/usr/bin/env bash
################################################################################
# AutoGPT Smoke Tests - Quick health validation (<30s)
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${RESOURCE_DIR}/lib/common.sh"

# Test configuration
AUTOGPT_PORT="${AUTOGPT_PORT_API:-8080}"
MAX_WAIT=30

# Health check test
test_health_check() {
    log::info "Testing health check endpoint..."
    
    if ! autogpt_container_running; then
        log::warning "AutoGPT container not running - attempting to start"
        if ! vrooli resource autogpt manage start --wait; then
            log::error "Failed to start AutoGPT"
            return 1
        fi
    fi
    
    # Test health endpoint with timeout
    if timeout 5 curl -sf "http://localhost:${AUTOGPT_PORT}/health" > /dev/null 2>&1; then
        log::success "Health check passed"
        return 0
    else
        log::error "Health check failed"
        return 1
    fi
}

# Container status test
test_container_status() {
    log::info "Testing container status..."
    
    if autogpt_container_running; then
        log::success "Container is running"
        
        # Check container health status
        local health_status
        health_status=$(docker inspect --format='{{.State.Health.Status}}' "${AUTOGPT_CONTAINER_NAME}" 2>/dev/null || echo "none")
        
        if [[ "${health_status}" == "healthy" ]]; then
            log::success "Container is healthy"
        elif [[ "${health_status}" == "none" ]]; then
            log::warning "No health check configured in container"
        else
            log::warning "Container health status: ${health_status}"
        fi
        
        return 0
    else
        log::error "Container is not running"
        return 1
    fi
}

# Port availability test
test_port_availability() {
    log::info "Testing port availability..."
    
    if nc -z localhost "${AUTOGPT_PORT}" 2>/dev/null; then
        log::success "Port ${AUTOGPT_PORT} is accessible"
        return 0
    else
        log::error "Port ${AUTOGPT_PORT} is not accessible"
        return 1
    fi
}

# Basic API response test
test_api_response() {
    log::info "Testing API basic response..."
    
    # Test if API returns proper JSON
    local response
    if response=$(timeout 5 curl -sf "http://localhost:${AUTOGPT_PORT}/health" 2>/dev/null); then
        if echo "${response}" | jq -e . > /dev/null 2>&1; then
            log::success "API returns valid JSON"
            return 0
        else
            log::warning "API response is not valid JSON: ${response}"
            return 1
        fi
    else
        log::error "Failed to get API response"
        return 1
    fi
}

# CLI command availability test
test_cli_commands() {
    log::info "Testing CLI command availability..."
    
    local commands=("help" "status" "manage" "test" "content")
    local failed=0
    
    for cmd in "${commands[@]}"; do
        if vrooli resource autogpt "${cmd}" --help > /dev/null 2>&1; then
            log::success "Command '${cmd}' is available"
        else
            log::error "Command '${cmd}' is not available"
            ((failed++))
        fi
    done
    
    return ${failed}
}

# Main test execution
main() {
    log::header "AutoGPT Smoke Tests"
    
    local failed=0
    
    # Run tests
    test_health_check || ((failed++))
    test_container_status || ((failed++))
    test_port_availability || ((failed++))
    test_api_response || ((failed++))
    test_cli_commands || ((failed++))
    
    # Summary
    if [[ ${failed} -eq 0 ]]; then
        log::success "All smoke tests passed"
    else
        log::error "${failed} smoke tests failed"
    fi
    
    return ${failed}
}

# Execute
main "$@"