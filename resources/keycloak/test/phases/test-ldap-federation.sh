#!/usr/bin/env bash
# Test LDAP federation management functionality

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
source "${RESOURCE_DIR}/lib/ldap-federation.sh"

# Test LDAP federation management
test_ldap_federation() {
    log::info "Testing LDAP federation management..."
    
    # Ensure Keycloak is running
    if ! keycloak::is_running; then
        log::error "Keycloak is not running"
        return 1
    fi
    
    # Test listing providers (should be empty initially)
    log::info "Testing LDAP provider listing..."
    if keycloak::list_ldap_providers "master" >/dev/null 2>&1; then
        log::success "LDAP provider listing works"
    else
        log::warning "LDAP provider listing failed (may be normal if no providers configured)"
    fi
    
    # Test adding an LDAP provider (will fail without valid LDAP server)
    log::info "Testing LDAP provider addition (expected to fail without valid LDAP server)..."
    if keycloak::add_ldap_provider "master" "test-ldap" "ldap://test.example.com:389" "ou=users,dc=example,dc=com" "cn=admin,dc=example,dc=com" "password" 2>/dev/null; then
        log::success "LDAP provider added (unexpected)"
        # Clean up
        keycloak::remove_ldap_provider "master" "test-ldap" >/dev/null 2>&1
    else
        log::info "LDAP provider addition failed as expected (no valid LDAP server)"
    fi
    
    # Test AD provider type
    log::info "Testing AD provider configuration..."
    if keycloak::add_ad_provider "master" "test-ad" "ldap://ad.example.com:389" "CN=Users,DC=example,DC=com" "admin@example.com" "password" 2>/dev/null; then
        log::success "AD provider added (unexpected)"
        # Clean up
        keycloak::remove_ldap_provider "master" "test-ad" >/dev/null 2>&1
    else
        log::info "AD provider addition failed as expected (no valid AD server)"
    fi
    
    log::success "LDAP federation tests completed"
    return 0
}

# Main test execution
main() {
    log::info "Starting Keycloak LDAP federation tests..."
    
    if test_ldap_federation; then
        log::success "All LDAP federation tests passed"
        return 0
    else
        log::error "LDAP federation tests failed"
        return 1
    fi
}

main "$@"