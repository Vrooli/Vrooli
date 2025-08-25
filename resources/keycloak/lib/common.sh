#!/usr/bin/env bash
set -euo pipefail

# Define Keycloak lib directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
KEYCLOAK_LIB_DIR="${APP_ROOT}/resources/keycloak/lib"

# Source the shared var utility FIRST
source "${KEYCLOAK_LIB_DIR}/../../../../lib/utils/var.sh"

# Set up paths
KEYCLOAK_BASE_DIR="${KEYCLOAK_LIB_DIR}/.."
KEYCLOAK_CONFIG_DIR="${KEYCLOAK_BASE_DIR}/config"
KEYCLOAK_DATA_DIR="${HOME}/.vrooli/keycloak"
KEYCLOAK_LOG_DIR="${KEYCLOAK_DATA_DIR}/logs"
KEYCLOAK_LOG_FILE="${KEYCLOAK_LOG_DIR}/keycloak.log"
KEYCLOAK_PID_FILE="${KEYCLOAK_DATA_DIR}/keycloak.pid"
KEYCLOAK_REALMS_DIR="${KEYCLOAK_DATA_DIR}/realms"

# Source default configuration
source "${KEYCLOAK_CONFIG_DIR}/defaults.sh"

# Source port registry
source "${KEYCLOAK_LIB_DIR}/../../../../resources/port_registry.sh"

# Get port for Keycloak
keycloak::get_port() {
    ports::get_resource_port "keycloak"
}

# Check if Keycloak is installed
keycloak::is_installed() {
    command -v docker &>/dev/null
}

# Check if Keycloak container exists
keycloak::container_exists() {
    docker ps -a --format "{{.Names}}" | grep -q "^${KEYCLOAK_CONTAINER_NAME}$" 2>/dev/null
}

# Check if Keycloak is running
keycloak::is_running() {
    if keycloak::container_exists; then
        local status
        status=$(docker inspect --format='{{.State.Status}}' "${KEYCLOAK_CONTAINER_NAME}" 2>/dev/null || echo "stopped")
        [[ "${status}" == "running" ]]
    else
        return 1
    fi
}

# Get container IP address
keycloak::get_container_ip() {
    if keycloak::container_exists; then
        docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${KEYCLOAK_CONTAINER_NAME}" 2>/dev/null || echo ""
    else
        echo ""
    fi
}

# Wait for Keycloak to be ready
keycloak::wait_for_ready() {
    local max_attempts="${1:-60}"
    local port
    port=$(keycloak::get_port)
    
    local attempt=0
    while [[ ${attempt} -lt ${max_attempts} ]]; do
        # Try the realms endpoint as health check with timeout
        if timeout 5 curl -sf "http://localhost:${port}/realms/master" >/dev/null 2>&1; then
            return 0
        fi
        sleep 2
        ((attempt++))
    done
    
    return 1
}

# Get admin access token
keycloak::get_admin_token() {
    local port
    port=$(keycloak::get_port)
    
    timeout 10 curl -sf -X POST "http://localhost:${port}/realms/master/protocol/openid-connect/token" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "username=${KEYCLOAK_ADMIN_USER}" \
        -d "password=${KEYCLOAK_ADMIN_PASSWORD}" \
        -d "grant_type=password" \
        -d "client_id=admin-cli" \
        | jq -r '.access_token // empty' 2>/dev/null || echo ""
}