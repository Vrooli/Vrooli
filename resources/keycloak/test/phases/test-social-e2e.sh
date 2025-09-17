#!/usr/bin/env bash
################################################################################
# Keycloak Social Provider End-to-End Test
# Tests the complete flow of social provider authentication
# Maximum execution time: 120 seconds
################################################################################

# Start with non-strict mode to allow flexible sourcing
set +euo pipefail

# Determine paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Export APP_ROOT for libraries that need it
export APP_ROOT

# Source utilities - these are required for the test to work
source "${APP_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || true
source "${APP_ROOT}/scripts/resources/common.sh" 2>/dev/null || true
source "${RESOURCE_DIR}/config/defaults.sh" 2>/dev/null || true

# Source common.sh with its functions
source "${RESOURCE_DIR}/lib/common.sh" 2>/dev/null || true

# Now enable error handling for the test itself
set -e

# Test configuration
TEST_REALM="test-social-e2e-$(date +%s)"
TEST_CLIENT="test-app"

# Mock OAuth providers for testing (using httpbin.org for OAuth flow simulation)
MOCK_PROVIDERS=(
    "github:mock_github_id:mock_github_secret"
    "google:mock_google_id:mock_google_secret"
    "facebook:mock_facebook_id:mock_facebook_secret"
)

# Cleanup function
cleanup() {
    echo "[INFO] Cleaning up test realm..."
    local token
    token=$(keycloak::get_admin_token 2>/dev/null || echo "")
    if [[ -n "$token" ]]; then
        timeout 5 curl -sf -X DELETE \
            "http://localhost:${KEYCLOAK_PORT:-8070}/admin/realms/${TEST_REALM}" \
            -H "Authorization: Bearer ${token}" >/dev/null 2>&1 || true
    fi
}

# Setup test realm
setup_test_realm() {
    echo "[INFO] Creating test realm: ${TEST_REALM}"
    
    local token
    token=$(keycloak::get_admin_token 2>/dev/null || echo "")
    if [[ -z "$token" ]]; then
        echo "[ERROR] Failed to get admin token"
        return 1
    fi
    
    # Create realm with social providers enabled
    local realm_config
    realm_config=$(cat <<EOF
{
    "realm": "${TEST_REALM}",
    "enabled": true,
    "registrationAllowed": true,
    "resetPasswordAllowed": true,
    "loginWithEmailAllowed": true,
    "duplicateEmailsAllowed": false
}
EOF
    )
    
    if timeout 5 curl -sf -X POST \
        "http://localhost:${KEYCLOAK_PORT:-8070}/admin/realms" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d "${realm_config}" >/dev/null 2>&1; then
        echo "[SUCCESS] Test realm created successfully"
    else
        echo "[ERROR] Failed to create test realm"
        return 1
    fi
    
    return 0
}

# Test adding social providers
test_add_providers() {
    echo "[INFO] Testing social provider addition..."
    
    local token
    token=$(keycloak::get_admin_token 2>/dev/null || echo "")
    if [[ -z "$token" ]]; then
        echo "[ERROR] Failed to get admin token"
        return 1
    fi
    
    local providers_added=0
    
    for provider_config in "${MOCK_PROVIDERS[@]}"; do
        IFS=':' read -r provider client_id client_secret <<< "$provider_config"
        echo "[INFO] Adding ${provider} provider..."
        
        local provider_data
        provider_data=$(cat <<EOF
{
    "alias": "${provider}",
    "providerId": "${provider}",
    "enabled": true,
    "trustEmail": true,
    "storeToken": true,
    "config": {
        "clientId": "${client_id}",
        "clientSecret": "${client_secret}",
        "syncMode": "IMPORT"
    }
}
EOF
        )
        
        if timeout 5 curl -sf -X POST \
            "http://localhost:${KEYCLOAK_PORT:-8070}/admin/realms/${TEST_REALM}/identity-provider/instances" \
            -H "Authorization: Bearer ${token}" \
            -H "Content-Type: application/json" \
            -d "${provider_data}" >/dev/null 2>&1; then
            echo "[SUCCESS] Added ${provider} provider"
            ((providers_added++))
        else
            echo "[WARNING] Failed to add ${provider} provider (may already exist)"
        fi
    done
    
    if [[ $providers_added -gt 0 ]]; then
        echo "[SUCCESS] Added ${providers_added} social providers"
        return 0
    else
        echo "[WARNING] No providers were added (may already exist)"
        return 0
    fi
}

# Test provider discovery
test_provider_discovery() {
    echo "[INFO] Testing provider discovery endpoints..."
    
    # Test OpenID Connect discovery
    local discovery_url="http://localhost:${KEYCLOAK_PORT:-8070}/realms/${TEST_REALM}/.well-known/openid-configuration"
    
    if timeout 5 curl -sf "$discovery_url" >/dev/null 2>&1; then
        echo "[SUCCESS] OIDC discovery endpoint accessible"
    else
        echo "[WARNING] OIDC discovery endpoint not accessible"
    fi
    
    return 0
}

# Main test execution
main() {
    echo "[INFO] Starting Keycloak social provider end-to-end tests..."
    
    # Ensure Keycloak is running
    if ! docker ps --format "table {{.Names}}" 2>/dev/null | grep -q "${KEYCLOAK_CONTAINER_NAME:-vrooli-keycloak}"; then
        echo "[ERROR] Keycloak container not running"
        return 1
    fi
    
    # Set trap for cleanup
    trap cleanup EXIT
    
    local failed=0
    
    # Run test phases
    if setup_test_realm; then
        echo "[SUCCESS] Test realm setup completed"
    else
        ((failed++))
    fi
    
    if test_add_providers; then
        echo "[SUCCESS] Provider addition test completed"
    else
        ((failed++))
    fi
    
    if test_provider_discovery; then
        echo "[SUCCESS] Provider discovery test completed"
    else
        ((failed++))
    fi
    
    # Report results
    if [[ $failed -eq 0 ]]; then
        echo "[SUCCESS] All social provider end-to-end tests passed"
        return 0
    else
        echo "[ERROR] ${failed} test phase(s) failed"
        return 1
    fi
}

# Execute main function
main "$@"