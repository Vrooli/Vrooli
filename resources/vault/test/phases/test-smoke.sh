#!/usr/bin/env bash
# Vault Resource Smoke Test - Quick health validation
# Tests that Vault service is running and responsive
# Max duration: 30 seconds per v2.0 contract

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
VAULT_CLI_DIR="${APP_ROOT}/resources/vault"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/common.sh"
# shellcheck disable=SC1091
source "${VAULT_CLI_DIR}/config/defaults.sh"
# Ensure configuration is exported
vault::export_config
# shellcheck disable=SC1091
source "${VAULT_CLI_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${VAULT_CLI_DIR}/lib/docker.sh"

# Vault Resource Smoke Test
vault::test::smoke() {
    log::info "Running Vault resource smoke test..."
    
    local overall_status=0
    local verbose="${VAULT_TEST_VERBOSE:-false}"
    
    # Test 1: Docker container exists
    log::info "1/5 Testing Docker container existence..."
    if vault::is_installed; then
        log::success "✓ Vault container exists"
        if [[ "$verbose" == "true" ]]; then
            local container_image
            container_image=$(docker inspect --format='{{.Config.Image}}' "$VAULT_CONTAINER_NAME" 2>/dev/null)
            log::info "  Image: $container_image"
        fi
    else
        log::error "✗ Vault container not found"
        overall_status=1
    fi
    
    # Test 2: Container is running
    log::info "2/5 Testing container status..."
    if vault::is_running; then
        log::success "✓ Vault container is running"
        if [[ "$verbose" == "true" ]]; then
            local container_status
            container_status=$(docker inspect --format='{{.State.Status}}' "$VAULT_CONTAINER_NAME" 2>/dev/null)
            log::info "  Status: $container_status"
        fi
    else
        log::error "✗ Vault container is not running"
        overall_status=1
    fi
    
    # Test 3: API health endpoint
    log::info "3/5 Testing API health endpoint..."
    if curl -sf "${VAULT_BASE_URL}/v1/sys/health" >/dev/null 2>&1; then
        log::success "✓ Health endpoint responding"
        if [[ "$verbose" == "true" ]]; then
            local health_response
            health_response=$(curl -sf "${VAULT_BASE_URL}/v1/sys/health" 2>/dev/null | jq -c '{initialized, sealed, version}' 2>/dev/null || echo "N/A")
            log::info "  Response: $health_response"
        fi
    else
        log::error "✗ Health endpoint not responding"
        overall_status=1
    fi
    
    # Test 4: Port accessibility
    log::info "4/5 Testing port accessibility..."
    if nc -z localhost "${VAULT_PORT}" 2>/dev/null; then
        log::success "✓ Port ${VAULT_PORT} is accessible"
    else
        log::error "✗ Port ${VAULT_PORT} is not accessible"
        overall_status=1
    fi
    
    # Test 5: Basic Vault status
    log::info "5/5 Testing Vault status..."
    local vault_status
    vault_status=$(vault::get_status)
    case "$vault_status" in
        healthy)
            log::success "✓ Vault is healthy and operational"
            ;;
        sealed)
            log::success "✓ Vault is running but sealed (normal for production)"
            ;;
        unhealthy)
            log::warn "⚠ Vault is running but unhealthy"
            overall_status=1
            ;;
        stopped)
            log::error "✗ Vault is stopped"
            overall_status=1
            ;;
        not_installed)
            log::error "✗ Vault is not installed"
            overall_status=1
            ;;
        *)
            log::warn "⚠ Unknown Vault status: $vault_status"
            overall_status=1
            ;;
    esac
    
    echo ""
    if [[ $overall_status -eq 0 ]]; then
        log::success "🎉 Vault resource smoke test PASSED"
        echo "Vault service is healthy and ready for secret management"
    else
        log::error "💥 Vault resource smoke test FAILED"
        echo "Vault service has issues that need to be resolved"
    fi
    
    return $overall_status
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    vault::test::smoke "$@"
fi