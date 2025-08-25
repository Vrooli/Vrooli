#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
KEYCLOAK_LIB_DIR="${APP_ROOT}/resources/keycloak/lib"

# Dependencies are expected to be sourced by caller

# Start Keycloak
keycloak::start() {
    log::info "Starting Keycloak"
    
    # Check if already running
    if keycloak::is_running; then
        log::warning "Keycloak is already running"
        return 0
    fi
    
    # Install if needed
    if ! keycloak::is_installed; then
        log::info "Docker not available, installing..."
        source "${KEYCLOAK_LIB_DIR}/install.sh"
        keycloak::install || return 1
    fi
    
    local port
    port=$(keycloak::get_port)
    
    # Remove existing container if it exists but is stopped
    if keycloak::container_exists && ! keycloak::is_running; then
        log::info "Removing existing stopped container"
        docker rm "${KEYCLOAK_CONTAINER_NAME}" >/dev/null 2>&1 || true
    fi
    
    # Start the container
    log::info "Starting Keycloak container on port ${port}..."
    
    docker run -d \
        --name "${KEYCLOAK_CONTAINER_NAME}" \
        --network "${KEYCLOAK_NETWORK}" \
        -p "${port}:8080" \
        -e "KEYCLOAK_ADMIN=${KEYCLOAK_ADMIN_USER}" \
        -e "KEYCLOAK_ADMIN_PASSWORD=${KEYCLOAK_ADMIN_PASSWORD}" \
        -e "KC_DB=${KEYCLOAK_DB}" \
        -e "KC_HOSTNAME_STRICT=${KEYCLOAK_HOSTNAME_STRICT}" \
        -e "KC_HOSTNAME_STRICT_HTTPS=${KEYCLOAK_HOSTNAME_STRICT_HTTPS}" \
        -e "KC_HTTP_ENABLED=${KEYCLOAK_HTTP_ENABLED}" \
        -e "KC_HEALTH_ENABLED=${KEYCLOAK_HEALTH_ENABLED}" \
        -e "KC_METRICS_ENABLED=${KEYCLOAK_METRICS_ENABLED}" \
        -e "KC_FEATURES=${KEYCLOAK_FEATURES}" \
        -e "KC_LOG_LEVEL=${KEYCLOAK_LOG_LEVEL}" \
        -e "JAVA_OPTS=${KEYCLOAK_JVM_OPTS}" \
        -v "${KEYCLOAK_REALMS_DIR}:/opt/keycloak/data/import:ro" \
        "${KEYCLOAK_IMAGE}" \
        start-dev \
        --import-realm \
        >/dev/null 2>&1
    
    # Store container ID as PID for consistency with other resources
    docker inspect --format='{{.State.Pid}}' "${KEYCLOAK_CONTAINER_NAME}" > "${KEYCLOAK_PID_FILE}" 2>/dev/null || true
    
    # Wait for service to be ready
    log::info "Waiting for Keycloak to be ready..."
    if keycloak::wait_for_ready 60; then
        log::success "Keycloak started successfully on port ${port}"
        log::info "Admin Console: http://localhost:${port}/admin"
        log::info "Credentials: ${KEYCLOAK_ADMIN_USER}/${KEYCLOAK_ADMIN_PASSWORD}"
        return 0
    else
        log::error "Failed to start Keycloak service"
        keycloak::stop
        return 1
    fi
}

# Stop Keycloak
keycloak::stop() {
    log::info "Stopping Keycloak"
    
    if keycloak::container_exists; then
        log::info "Stopping Keycloak container"
        docker stop "${KEYCLOAK_CONTAINER_NAME}" >/dev/null 2>&1 || true
        docker rm "${KEYCLOAK_CONTAINER_NAME}" >/dev/null 2>&1 || true
    fi
    
    # Clean up PID file
    if [[ -f "${KEYCLOAK_PID_FILE}" ]]; then
        rm -f "${KEYCLOAK_PID_FILE}"
    fi
    
    # Also kill any process on the port (failsafe)
    local port
    port=$(keycloak::get_port)
    local pids
    pids=$(lsof -ti:${port} 2>/dev/null || true)
    if [[ -n "${pids}" ]]; then
        echo "${pids}" | xargs kill -9 2>/dev/null || true
    fi
    
    log::success "Keycloak stopped"
    return 0
}

# Restart Keycloak
keycloak::restart() {
    log::info "Restarting Keycloak"
    keycloak::stop
    sleep 3
    keycloak::start
}