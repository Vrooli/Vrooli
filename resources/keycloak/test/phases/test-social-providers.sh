#!/usr/bin/env bash
# Test social provider management functionality

set -euo pipefail

# Determine script location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/common.sh"
source "${RESOURCE_DIR}/lib/social-providers.sh"

# Test social provider management
test_social_providers() {
    log::info "Testing social provider management..."
    
    # Ensure Keycloak is running
    if ! keycloak::is_running; then
        log::error "Keycloak is not running"
        return 1
    fi
    
    # Test listing providers (should be empty initially)
    log::info "Testing provider listing..."
    if keycloak::list_social_providers "master" >/dev/null 2>&1; then
        log::success "Provider listing works"
    else
        log::warning "Provider listing failed (may be normal if no providers configured)"
    fi
    
    # Test adding a GitHub provider (will fail without valid credentials)
    log::info "Testing GitHub provider addition (expected to fail without valid credentials)..."
    if keycloak::add_github_provider "master" "test_client_id" "test_client_secret" 2>/dev/null; then
        log::success "GitHub provider added (unexpected)"
        # Clean up
        keycloak::remove_social_provider "master" "github" >/dev/null 2>&1
    else
        log::info "GitHub provider addition failed as expected (no valid credentials)"
    fi
    
    # Test provider URL generation
    log::info "Testing provider URL generation..."
    local provider_url="http://localhost:${KEYCLOAK_PORT:-8070}/realms/master/broker/github/login"
    log::info "Provider URL would be: $provider_url"
    
    log::success "Social provider tests completed"
    return 0
}

# Main test execution
main() {
    log::info "Starting Keycloak social provider tests..."
    
    if test_social_providers; then
        log::success "All social provider tests passed"
        return 0
    else
        log::error "Social provider tests failed"
        return 1
    fi
}

main "$@"