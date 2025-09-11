#!/usr/bin/env bash
################################################################################
# Gemini Resource Smoke Tests
# 
# Quick health check for Gemini API service (< 30s)
################################################################################

set -euo pipefail

# Determine paths
PHASES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "$PHASES_DIR/.." && pwd)"
RESOURCE_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="$(cd "$RESOURCE_DIR/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

# Source resource libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"

log::info "Starting Gemini smoke tests..."

# Test 1: Configuration loads
log::info "Test 1: Checking configuration..."
if [[ -n "$GEMINI_API_BASE" ]] && [[ -n "$GEMINI_DEFAULT_MODEL" ]]; then
    log::success "✓ Configuration loaded successfully"
else
    log::error "✗ Configuration not loaded properly"
    exit 1
fi

# Test 2: API initialization
log::info "Test 2: Initializing Gemini..."
if gemini::init; then
    log::success "✓ Gemini initialized successfully"
else
    log::error "✗ Failed to initialize Gemini"
    exit 1
fi

# Test 3: Health check (API connectivity)
log::info "Test 3: Testing API connectivity..."
if [[ "$GEMINI_API_KEY" == "placeholder-gemini-key" ]]; then
    log::warn "⚠ Using placeholder API key - skipping connectivity test"
    log::info "  To enable full testing, configure a real API key via:"
    log::info "  - Environment: export GEMINI_API_KEY='your-key'"
    log::info "  - Vault: vault kv put secret/resources/gemini/api/key gemini_api_key='your-key'"
else
    if timeout 5 gemini::test_connection; then
        log::success "✓ API connectivity verified"
    else
        log::error "✗ API connectivity test failed"
        exit 1
    fi
fi

# Test 4: Runtime configuration exists
log::info "Test 4: Checking runtime configuration..."
if [[ -f "${RESOURCE_DIR}/config/runtime.json" ]]; then
    # Validate key fields
    if jq -e '.startup_order' "${RESOURCE_DIR}/config/runtime.json" >/dev/null 2>&1; then
        log::success "✓ Runtime configuration valid"
    else
        log::error "✗ Runtime configuration missing required fields"
        exit 1
    fi
else
    log::error "✗ Runtime configuration file not found"
    exit 1
fi

# Test 5: Secrets configuration exists
log::info "Test 5: Checking secrets configuration..."
if [[ -f "${RESOURCE_DIR}/config/secrets.yaml" ]]; then
    log::success "✓ Secrets configuration exists"
else
    log::warn "⚠ Secrets configuration not found (optional)"
fi

log::success "✅ All smoke tests passed!"
exit 0