#!/usr/bin/env bash
# Browserless Docker Management - Optimized for browser automation
# Uses docker-resource-utils.sh pattern for consistency

_BROWSERLESS_DOCKER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${_BROWSERLESS_DOCKER_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-resource-utils.sh"

#######################################
# Create Browserless container with browser automation settings
# Returns: 0 on success, 1 on failure
#######################################
browserless::docker::create_container() {
    local max_browsers="${1:-${BROWSERLESS_MAX_BROWSERS:-5}}"
    local timeout="${2:-${BROWSERLESS_TIMEOUT:-30000}}"
    
    mkdir -p "$BROWSERLESS_DATA_DIR"
    log::info "Starting Browserless container..."
    
    # Validate required environment variables
    docker_resource::validate_env_vars "BROWSERLESS_CONTAINER_NAME" "BROWSERLESS_IMAGE" "BROWSERLESS_PORT" || return 1
    
    # Prepare environment variables array
    local env_vars=(
        "CONCURRENT=${max_browsers}"
        "TIMEOUT=${timeout}"
        "ENABLE_DEBUGGER=false"
        "PREBOOT_CHROME=true"
        "KEEP_ALIVE=true"
        "PRE_REQUEST_HEALTH_CHECK=false"
        "FUNCTION_ENABLE_INCOGNITO_MODE=true"
        "WORKSPACE_DELETE_EXPIRED=true"
        "WORKSPACE_EXPIRE_DAYS=7"
    )
    
    # Browserless-specific Docker options for browser isolation
    local docker_opts=(
        "--shm-size=${BROWSERLESS_DOCKER_SHM_SIZE:-2gb}"
        "--cap-add=${BROWSERLESS_DOCKER_CAPS:-SYS_ADMIN}"
        "--security-opt" "seccomp=${BROWSERLESS_DOCKER_SECCOMP:-unconfined}"
    )
    
    # Port mappings and volumes
    local port_mappings="${BROWSERLESS_PORT}:3000"
    local volumes="${BROWSERLESS_DATA_DIR}:/workspace"
    
    # Health check command
    local health_cmd="curl -f http://localhost:3000/introspection || exit 1"
    
    # Use advanced creation function
    if docker_resource::create_service_advanced \
        "$BROWSERLESS_CONTAINER_NAME" \
        "$BROWSERLESS_IMAGE" \
        "$port_mappings" \
        "$BROWSERLESS_NETWORK_NAME" \
        "$volumes" \
        "env_vars" \
        "docker_opts" \
        "$health_cmd" \
        ""; then
        
        sleep "${BROWSERLESS_INITIALIZATION_WAIT:-10}"
        return 0
    else
        log::error "Failed to create Browserless container"
        return 1
    fi
}

#######################################
# Start Browserless container
# Returns: 0 on success, 1 on failure
#######################################
browserless::docker::start() {
    docker::container_exists "$BROWSERLESS_CONTAINER_NAME" || { log::error "Container does not exist. Run install first."; return 1; }
    log::info "Starting Browserless container..."
    docker::start_container "$BROWSERLESS_CONTAINER_NAME" && browserless::wait_for_ready && \
    { log::success "Browserless started successfully"; browserless::docker::show_connection_info; }
}

#######################################
# Stop Browserless container
# Returns: 0 on success, 1 on failure
#######################################
browserless::docker::stop() {
    docker::stop_container "$BROWSERLESS_CONTAINER_NAME"
}

#######################################
# Restart Browserless container
# Returns: 0 on success, 1 on failure
#######################################
browserless::docker::restart() {
    docker::restart_container "$BROWSERLESS_CONTAINER_NAME"
}

#######################################
# Remove Browserless container
# Arguments: $1 - remove data (yes/no)
# Returns: 0 on success, 1 on failure
#######################################
browserless::docker::remove() {
    local remove_data="${1:-no}"
    docker::remove_container "$BROWSERLESS_CONTAINER_NAME" "true"
    
    if [[ "$remove_data" == "yes" ]]; then
        docker_resource::remove_data "Browserless" "$BROWSERLESS_DATA_DIR" "yes"
    fi
}

#######################################
# Show connection information
#######################################
browserless::docker::show_connection_info() {
    local additional_info=(
        "WebSocket: ws://localhost:${BROWSERLESS_PORT}"
        "Status: http://localhost:${BROWSERLESS_PORT}/introspection"
        "API Version: http://localhost:${BROWSERLESS_PORT}/json/version"
        "Concurrent browsers: ${BROWSERLESS_MAX_BROWSERS:-5}"
    )
    
    docker_resource::show_connection_info "Browserless" "http://localhost:${BROWSERLESS_PORT}" "${additional_info[@]}"
}