#!/usr/bin/env bash
################################################################################
# OBS Studio Smoke Tests - Quick Health Check (<30s)
#
# Validates basic OBS Studio functionality per v2.0 contract
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
RESOURCE_DIR="$(cd "${TEST_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Smoke test implementation
main() {
    local start_time=$(date +%s)
    
    log::header "🧪 OBS Studio Smoke Test"
    log::info "Quick health validation (<30s)"
    
    # Check if OBS Studio is installed
    log::info "Checking installation status..."
    if obs::is_installed; then
        log::success "✅ OBS Studio is installed"
    else
        log::error "❌ OBS Studio is not installed"
        exit 1
    fi
    
    # Check if service is running
    log::info "Checking service status..."
    if obs::is_running; then
        log::success "✅ OBS Studio is running"
    else
        log::warning "⚠️  OBS Studio is not running, attempting to start..."
        if obs::start; then
            log::success "✅ OBS Studio started successfully"
        else
            log::error "❌ Failed to start OBS Studio"
            exit 1
        fi
    fi
    
    # Health check with timeout
    log::info "Performing health check..."
    if timeout 5 curl -sf "http://localhost:${OBS_PORT:-4455}/health" &>/dev/null; then
        log::success "✅ Health endpoint responding"
    else
        # Try WebSocket connection as fallback
        if obs::websocket::test_connection; then
            log::success "✅ WebSocket connection successful"
        else
            log::error "❌ Health check failed - service not responding"
            exit 1
        fi
    fi
    
    # Check configuration
    log::info "Validating configuration..."
    if [[ -f "${RESOURCE_DIR}/config/runtime.json" ]]; then
        log::success "✅ Runtime configuration present"
    else
        log::error "❌ Runtime configuration missing"
        exit 1
    fi
    
    # Check critical directories
    log::info "Checking directories..."
    local config_dir="${OBS_CONFIG_DIR:-$HOME/.vrooli/obs-studio}"
    local recordings_dir="${OBS_RECORDINGS_DIR:-$HOME/Videos/obs-recordings}"
    
    if [[ -d "$config_dir" ]] && [[ -w "$config_dir" ]]; then
        log::success "✅ Config directory accessible"
    else
        log::warning "⚠️  Creating config directory..."
        mkdir -p "$config_dir"
    fi
    
    if [[ -d "$recordings_dir" ]] && [[ -w "$recordings_dir" ]]; then
        log::success "✅ Recordings directory accessible"
    else
        log::warning "⚠️  Creating recordings directory..."
        mkdir -p "$recordings_dir"
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Ensure we complete within 30 seconds
    if [[ $duration -gt 30 ]]; then
        log::warning "⚠️  Smoke test took ${duration}s (exceeds 30s limit)"
    else
        log::success "✅ Smoke test completed in ${duration}s"
    fi
    
    log::success "All smoke tests passed"
    exit 0
}

main "$@"