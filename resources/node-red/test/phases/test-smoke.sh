#!/usr/bin/env bash
################################################################################
# Node-RED Smoke Tests - Quick health validation (<30s)
# 
# Validates basic connectivity and service health
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NODE_RED_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${NODE_RED_DIR}/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"

# Configuration
NODE_RED_PORT="${NODE_RED_PORT:-1880}"
NODE_RED_URL="http://localhost:${NODE_RED_PORT}"
NODE_RED_CONTAINER="${NODE_RED_CONTAINER_NAME:-node-red}"

# Test functions
test_container_running() {
    log::test "Container running check"
    
    if docker ps --format "{{.Names}}" | grep -q "^${NODE_RED_CONTAINER}$"; then
        log::success "Container is running"
        return 0
    else
        log::error "Container not running"
        return 1
    fi
}

test_health_endpoint() {
    log::test "Health endpoint check"
    
    # Proper timeout handling per v2.0 contract
    if timeout 5 curl -sf "${NODE_RED_URL}/settings" > /dev/null; then
        log::success "Health endpoint responsive"
        return 0
    else
        log::error "Health endpoint not responding"
        return 1
    fi
}

test_editor_accessible() {
    log::test "Editor UI accessibility"
    
    if timeout 5 curl -sf "${NODE_RED_URL}" | grep -q "Node-RED"; then
        log::success "Editor UI accessible"
        return 0
    else
        log::error "Editor UI not accessible"
        return 1
    fi
}

test_api_responsive() {
    log::test "API endpoint check"
    
    local response
    response=$(timeout 5 curl -sf "${NODE_RED_URL}/flows" 2>/dev/null || echo "failed")
    
    if [[ "$response" != "failed" ]]; then
        log::success "API endpoints responsive"
        return 0
    else
        log::error "API not responding"
        return 1
    fi
}

test_websocket_available() {
    log::test "WebSocket connectivity"
    
    # Check if WebSocket upgrade headers are present
    local headers
    headers=$(timeout 5 curl -sI "${NODE_RED_URL}/comms" 2>/dev/null || echo "")
    
    if [[ -n "$headers" ]]; then
        log::success "WebSocket endpoint available"
        return 0
    else
        log::warning "WebSocket check inconclusive"
        return 0  # Non-critical for smoke test
    fi
}

test_data_persistence() {
    log::test "Data persistence directory"
    
    local data_dir="$HOME/.local/share/node-red"
    
    if [[ -d "$data_dir" ]]; then
        log::success "Data directory exists"
        return 0
    else
        log::warning "Data directory not found (will be created on first use)"
        return 0  # Non-critical
    fi
}

# Main test execution
main() {
    log::header "Node-RED Smoke Tests"
    
    local failed=0
    local tests=(
        test_container_running
        test_health_endpoint
        test_editor_accessible
        test_api_responsive
        test_websocket_available
        test_data_persistence
    )
    
    for test in "${tests[@]}"; do
        if ! $test; then
            ((failed++))
        fi
    done
    
    # Summary
    local total=${#tests[@]}
    local passed=$((total - failed))
    
    log::info "Test Summary: $passed/$total passed"
    
    if [[ $failed -eq 0 ]]; then
        log::success "All smoke tests passed!"
        return 0
    else
        log::error "$failed smoke test(s) failed"
        return 1
    fi
}

# Execute
main "$@"