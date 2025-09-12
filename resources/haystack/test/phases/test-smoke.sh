#!/usr/bin/env bash
################################################################################
# Haystack Smoke Tests - v2.0 Universal Contract Compliant
# 
# Quick validation that Haystack is installed and can start
# Must complete in <30 seconds per universal.yaml
################################################################################

set -euo pipefail

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
HAYSTACK_CLI="${APP_ROOT}/resources/haystack/cli.sh"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/resources/port-registry.sh"

# Get Haystack port
HAYSTACK_PORT=$(resources::get_port "haystack")

# Test functions
test_cli_available() {
    log::test "CLI is available"
    if [[ -x "${HAYSTACK_CLI}" ]]; then
        log::success "CLI found and executable"
        return 0
    else
        log::error "CLI not found or not executable at ${HAYSTACK_CLI}"
        return 1
    fi
}

test_help_command() {
    log::test "Help command works"
    if "${HAYSTACK_CLI}" help &>/dev/null; then
        log::success "Help command successful"
        return 0
    else
        log::error "Help command failed"
        return 1
    fi
}

test_can_start() {
    log::test "Service can start"
    
    # Check if already running
    if timeout 5 curl -sf "http://localhost:${HAYSTACK_PORT}/health" &>/dev/null; then
        log::info "Service already running"
        log::success "Service is available"
        return 0
    fi
    
    # Try to start
    if "${HAYSTACK_CLI}" manage start --wait &>/dev/null; then
        log::success "Service started successfully"
        return 0
    else
        log::error "Failed to start service"
        return 1
    fi
}

test_health_check() {
    log::test "Health check responds"
    
    if timeout 5 curl -sf "http://localhost:${HAYSTACK_PORT}/health" &>/dev/null; then
        log::success "Health check successful"
        return 0
    else
        log::error "Health check failed"
        return 1
    fi
}

test_can_stop() {
    log::test "Service can stop"
    
    if "${HAYSTACK_CLI}" manage stop &>/dev/null; then
        log::success "Service stopped successfully"
        return 0
    else
        log::error "Failed to stop service"
        return 1
    fi
}

# Main test execution
main() {
    local failed=0
    
    log::header "Haystack Smoke Tests"
    
    # Run tests
    test_cli_available || ((failed++))
    test_help_command || ((failed++))
    test_can_start || ((failed++))
    test_health_check || ((failed++))
    test_can_stop || ((failed++))
    
    # Report results
    if [[ ${failed} -eq 0 ]]; then
        log::success "All smoke tests passed"
        exit 0
    else
        log::error "${failed} smoke tests failed"
        exit 1
    fi
}

# Run with timeout (30 seconds per universal.yaml)
timeout 30 bash -c "$(declare -f main test_cli_available test_help_command test_can_start test_health_check test_can_stop); main" || {
    log::error "Smoke tests exceeded 30 second timeout"
    exit 1
}