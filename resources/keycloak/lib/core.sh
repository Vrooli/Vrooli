#!/usr/bin/env bash
set -euo pipefail

# Keycloak Core Functionality
# Required by v2.0 contract - consolidates core resource operations

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
KEYCLOAK_ROOT="${APP_ROOT}/resources/keycloak"

# Source dependencies if not already loaded
if ! declare -f log::info &>/dev/null; then
    source "${APP_ROOT}/scripts/lib/utils/log.sh"
fi

if ! declare -f resources::get_port &>/dev/null; then
    source "${APP_ROOT}/scripts/resources/port_registry.sh"
fi

# Load configuration
source "${KEYCLOAK_ROOT}/config/defaults.sh"

# Get port from registry
KEYCLOAK_PORT="${KEYCLOAK_PORT:-$(resources::get_port keycloak)}"

# Core container operations
keycloak::is_running() {
    docker ps --format "table {{.Names}}" 2>/dev/null | grep -q "${KEYCLOAK_CONTAINER_NAME}"
}

keycloak::is_healthy() {
    local health_url="http://localhost:${KEYCLOAK_PORT}/realms/master/.well-known/openid-configuration"
    if timeout 5 curl -sf "${health_url}" > /dev/null 2>&1; then
        return 0
    fi
    return 1
}

keycloak::get_admin_token() {
    local username="${KEYCLOAK_ADMIN_USER:-admin}"
    local password="${KEYCLOAK_ADMIN_PASSWORD:-admin}"
    local token_url="http://localhost:${KEYCLOAK_PORT}/realms/master/protocol/openid-connect/token"
    local response
    
    response=$(curl -sf -X POST "${token_url}" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "username=${username}" \
        -d "password=${password}" \
        -d "grant_type=password" \
        -d "client_id=admin-cli" 2>/dev/null || echo "{}")
    
    echo "${response}" | jq -r '.access_token // empty'
}

keycloak::wait_for_ready() {
    local max_attempts="${1:-60}"
    local attempt=0
    
    log::info "Waiting for Keycloak to be ready..."
    
    while [ ${attempt} -lt ${max_attempts} ]; do
        if keycloak::is_healthy; then
            log::success "Keycloak is ready"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
    done
    
    log::error "Keycloak failed to become ready after ${max_attempts} attempts"
    return 1
}

# Core realm operations
keycloak::realm_exists() {
    local realm="${1}"
    local token
    
    token=$(keycloak::get_admin_token)
    if [ -z "${token}" ]; then
        return 1
    fi
    
    local response
    response=$(curl -sf -X GET "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}" \
        -H "Authorization: Bearer ${token}" 2>/dev/null || echo "")
    
    if [ -n "${response}" ]; then
        return 0
    fi
    return 1
}

keycloak::create_realm() {
    local realm="${1}"
    local token
    
    token=$(keycloak::get_admin_token)
    if [ -z "${token}" ]; then
        log::error "Failed to get admin token"
        return 1
    fi
    
    local realm_json='{
        "realm": "'"${realm}"'",
        "enabled": true,
        "sslRequired": "external",
        "registrationAllowed": false,
        "loginWithEmailAllowed": true,
        "duplicateEmailsAllowed": false,
        "resetPasswordAllowed": true,
        "editUsernameAllowed": false,
        "bruteForceProtected": true
    }'
    
    local response
    response=$(curl -sf -X POST "http://localhost:${KEYCLOAK_PORT}/admin/realms" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d "${realm_json}" 2>&1)
    
    local exit_code=$?
    if [ ${exit_code} -eq 0 ]; then
        log::success "Realm '${realm}' created successfully"
        return 0
    else
        log::error "Failed to create realm: ${response}"
        return 1
    fi
}

# Core user operations
keycloak::create_user() {
    local realm="${1}"
    local username="${2}"
    local password="${3}"
    local email="${4:-${username}@example.com}"
    local token
    
    token=$(keycloak::get_admin_token)
    if [ -z "${token}" ]; then
        log::error "Failed to get admin token"
        return 1
    fi
    
    # Create user
    local user_json='{
        "username": "'"${username}"'",
        "email": "'"${email}"'",
        "enabled": true,
        "emailVerified": true,
        "credentials": [{
            "type": "password",
            "value": "'"${password}"'",
            "temporary": false
        }]
    }'
    
    local response
    response=$(curl -sf -X POST "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}/users" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d "${user_json}" 2>&1)
    
    local exit_code=$?
    if [ ${exit_code} -eq 0 ]; then
        log::success "User '${username}' created in realm '${realm}'"
        return 0
    else
        log::error "Failed to create user: ${response}"
        return 1
    fi
}

# Core client operations
keycloak::create_client() {
    local realm="${1}"
    local client_id="${2}"
    local client_secret="${3:-}"
    local redirect_uri="${4:-http://localhost:3000/*}"
    local token
    
    token=$(keycloak::get_admin_token)
    if [ -z "${token}" ]; then
        log::error "Failed to get admin token"
        return 1
    fi
    
    local client_json
    if [ -n "${client_secret}" ]; then
        # Confidential client with secret
        client_json='{
            "clientId": "'"${client_id}"'",
            "secret": "'"${client_secret}"'",
            "enabled": true,
            "publicClient": false,
            "redirectUris": ["'"${redirect_uri}"'"],
            "webOrigins": ["*"],
            "standardFlowEnabled": true,
            "directAccessGrantsEnabled": true
        }'
    else
        # Public client without secret
        client_json='{
            "clientId": "'"${client_id}"'",
            "enabled": true,
            "publicClient": true,
            "redirectUris": ["'"${redirect_uri}"'"],
            "webOrigins": ["*"],
            "standardFlowEnabled": true,
            "directAccessGrantsEnabled": true
        }'
    fi
    
    local response
    response=$(curl -sf -X POST "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}/clients" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d "${client_json}" 2>&1)
    
    local exit_code=$?
    if [ ${exit_code} -eq 0 ]; then
        log::success "Client '${client_id}' created in realm '${realm}'"
        return 0
    else
        log::error "Failed to create client: ${response}"
        return 1
    fi
}

# Export all functions
export -f keycloak::is_running
export -f keycloak::is_healthy
export -f keycloak::get_admin_token
export -f keycloak::wait_for_ready
export -f keycloak::realm_exists
export -f keycloak::create_realm
export -f keycloak::create_user
export -f keycloak::create_client