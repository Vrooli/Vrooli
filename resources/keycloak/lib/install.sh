#!/usr/bin/env bash
set -euo pipefail

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
KEYCLOAK_LIB_DIR="${APP_ROOT}/resources/keycloak/lib"

# Dependencies are expected to be sourced by caller

# Install Keycloak
keycloak::install() {
    log::info "Installing Keycloak"
    
    # Check if Docker is available
    if ! command -v docker &>/dev/null; then
        log::error "Docker is required but not installed"
        return 1
    fi
    
    # Check if Docker daemon is running
    if ! docker info >/dev/null 2>&1; then
        log::error "Docker daemon is not running"
        return 1
    fi
    
    # Create directories
    mkdir -p "${KEYCLOAK_DATA_DIR}"
    mkdir -p "${KEYCLOAK_LOG_DIR}"
    mkdir -p "${KEYCLOAK_REALMS_DIR}"
    
    # Create Docker network if it doesn't exist
    if ! docker network ls --format "table {{.Name}}" | grep -q "^${KEYCLOAK_NETWORK}$"; then
        log::info "Creating Docker network: ${KEYCLOAK_NETWORK}"
        docker network create "${KEYCLOAK_NETWORK}" >/dev/null 2>&1 || {
            log::warning "Network ${KEYCLOAK_NETWORK} might already exist"
        }
    fi
    
    # Pull Keycloak image
    log::info "Pulling Keycloak Docker image: ${KEYCLOAK_IMAGE}"
    docker pull "${KEYCLOAK_IMAGE}"
    
    # Create realm export directory with proper permissions
    chmod 755 "${KEYCLOAK_REALMS_DIR}"
    
    # Register CLI with Vrooli
    if [[ -f "${var_SCRIPTS_RESOURCES_LIB_DIR}/install-resource-cli.sh" ]]; then
        "${var_SCRIPTS_RESOURCES_LIB_DIR}/install-resource-cli.sh" "${KEYCLOAK_LIB_DIR}/.." 2>/dev/null || true
    fi
    
    log::success "Keycloak installed successfully"
    return 0
}

# Uninstall Keycloak
keycloak::uninstall() {
    log::info "Uninstalling Keycloak"
    
    # Stop if running
    if keycloak::is_running; then
        keycloak::stop
    fi
    
    # Remove container if exists
    if keycloak::container_exists; then
        log::info "Removing Keycloak container"
        docker rm -f "${KEYCLOAK_CONTAINER_NAME}" >/dev/null 2>&1 || true
    fi
    
    # Remove Docker image
    if docker images --format "table {{.Repository}}:{{.Tag}}" | grep -q "^${KEYCLOAK_IMAGE}$"; then
        log::info "Removing Keycloak Docker image"
        docker rmi "${KEYCLOAK_IMAGE}" >/dev/null 2>&1 || true
    fi
    
    # Remove data directory
    if [[ -d "${KEYCLOAK_DATA_DIR}" ]]; then
        log::info "Removing Keycloak data directory"
        rm -rf "${KEYCLOAK_DATA_DIR}"
    fi
    
    # Remove network if no other containers are using it
    local network_containers
    network_containers=$(docker network inspect "${KEYCLOAK_NETWORK}" --format='{{len .Containers}}' 2>/dev/null || echo "0")
    if [[ "${network_containers}" == "0" ]]; then
        log::info "Removing Docker network: ${KEYCLOAK_NETWORK}"
        docker network rm "${KEYCLOAK_NETWORK}" >/dev/null 2>&1 || true
    fi
    
    log::success "Keycloak uninstalled successfully"
    return 0
}