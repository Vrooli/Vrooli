#!/usr/bin/env bash
################################################################################
# Keycloak Integration Test - End-to-End Functionality
# Maximum execution time: 120 seconds
################################################################################

set -euo pipefail

# Determine paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${RESOURCE_DIR}/config/defaults.sh"

# Get port - use default if not set
KEYCLOAK_PORT="${KEYCLOAK_PORT:-8070}"

# Test data
TEST_REALM="test-realm-$$"
TEST_USER="test-user-$$"
TEST_CLIENT="test-client-$$"

# Helper functions
get_admin_token() {
    local token_url="http://localhost:${KEYCLOAK_PORT}/realms/master/protocol/openid-connect/token"
    
    local response=$(timeout 5 curl -sf -X POST "${token_url}" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "username=${KEYCLOAK_ADMIN_USER}" \
        -d "password=${KEYCLOAK_ADMIN_PASSWORD}" \
        -d "grant_type=password" \
        -d "client_id=admin-cli" 2>/dev/null || echo "{}")
    
    echo "$response" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4
}

# Test functions
test_admin_authentication() {
    log::info "Testing admin authentication..."
    
    local token=$(get_admin_token)
    
    if [[ -n "$token" ]]; then
        log::success "Admin authentication successful"
        return 0
    else
        log::error "Admin authentication failed"
        return 1
    fi
}

test_realm_creation() {
    log::info "Testing realm creation..."
    
    local token=$(get_admin_token)
    if [[ -z "$token" ]]; then
        log::error "Cannot test realm creation without admin token"
        return 1
    fi
    
    local realm_json='{
        "realm": "'${TEST_REALM}'",
        "enabled": true,
        "sslRequired": "none"
    }'
    
    local response_code=$(timeout 5 curl -sf -o /dev/null -w "%{http_code}" \
        -X POST "http://localhost:${KEYCLOAK_PORT}/admin/realms" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d "${realm_json}" 2>/dev/null || echo "000")
    
    if [[ "$response_code" == "201" ]]; then
        log::success "Realm creation successful"
        return 0
    else
        log::error "Realm creation failed (HTTP ${response_code})"
        return 1
    fi
}

test_realm_listing() {
    log::info "Testing realm listing..."
    
    local token=$(get_admin_token)
    if [[ -z "$token" ]]; then
        log::error "Cannot test realm listing without admin token"
        return 1
    fi
    
    local realms=$(timeout 5 curl -sf \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms" \
        -H "Authorization: Bearer ${token}" 2>/dev/null || echo "[]")
    
    if echo "$realms" | grep -q "master"; then
        log::success "Realm listing successful"
        return 0
    else
        log::error "Realm listing failed"
        return 1
    fi
}

test_openid_configuration() {
    log::info "Testing OpenID Connect discovery..."
    
    local oidc_url="http://localhost:${KEYCLOAK_PORT}/realms/master/.well-known/openid-configuration"
    
    local config=$(timeout 5 curl -sf "${oidc_url}" 2>/dev/null || echo "{}")
    
    if echo "$config" | grep -q "issuer"; then
        log::success "OIDC discovery endpoint working"
        return 0
    else
        log::error "OIDC discovery endpoint failed"
        return 1
    fi
}

test_cli_commands() {
    log::info "Testing CLI command structure..."
    
    # Test help command
    if "${RESOURCE_DIR}/cli.sh" help > /dev/null 2>&1; then
        log::success "CLI help command works"
    else
        log::error "CLI help command failed"
        return 1
    fi
    
    # Test status command
    if "${RESOURCE_DIR}/cli.sh" status > /dev/null 2>&1; then
        log::success "CLI status command works"
    else
        log::warning "CLI status command failed (container may not be running)"
    fi
    
    return 0
}

cleanup_test_data() {
    log::info "Cleaning up test data..."
    
    local token=$(get_admin_token)
    if [[ -n "$token" ]] && [[ -n "$TEST_REALM" ]]; then
        timeout 5 curl -sf -X DELETE \
            "http://localhost:${KEYCLOAK_PORT}/admin/realms/${TEST_REALM}" \
            -H "Authorization: Bearer ${token}" 2>/dev/null || true
    fi
}

# Main test execution
main() {
    log::info "Starting Keycloak integration tests..."
    
    local failed=0
    
    # Check if container is running first
    if ! docker ps --format "table {{.Names}}" | grep -q "${KEYCLOAK_CONTAINER_NAME}"; then
        log::warning "Keycloak container not running, skipping integration tests"
        return 2  # Return 2 for "tests not available"
    fi
    
    # Run tests
    test_admin_authentication || ((failed++))
    test_realm_creation || ((failed++))
    test_realm_listing || ((failed++))
    test_openid_configuration || ((failed++))
    test_cli_commands || ((failed++))
    
    # Cleanup
    cleanup_test_data
    
    # Report results
    if [[ $failed -eq 0 ]]; then
        log::success "All integration tests passed"
        return 0
    else
        log::error "${failed} integration test(s) failed"
        return 1
    fi
}

# Execute main function
main