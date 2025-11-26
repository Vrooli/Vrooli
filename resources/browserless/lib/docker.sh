#!/usr/bin/env bash
# Browserless Docker Management - Optimized for browser automation
# Uses docker-resource-utils.sh pattern for consistency

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
_BROWSERLESS_DOCKER_DIR="${APP_ROOT}/resources/browserless/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
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
        "MAX_CONCURRENT_SESSIONS=${max_browsers}"
        "MAX_QUEUE_LENGTH=${BROWSERLESS_MAX_QUEUE_LENGTH:-10}"
        "TIMEOUT=${timeout}"
        "CONNECTION_TIMEOUT=${BROWSERLESS_CONNECTION_TIMEOUT:-15000}"
        "WORKER_TIMEOUT=${BROWSERLESS_WORKER_TIMEOUT:-90000}"
        "CHROME_REFRESH_TIME=${BROWSERLESS_CHROME_REFRESH_MS:-600000}"
        "SOCKET_CLOSE_TIMEOUT=${BROWSERLESS_SOCKET_CLOSE_TIMEOUT:-5000}"
        "ENABLE_DEBUGGER=false"
        "PREBOOT_CHROME=false"
        "KEEP_ALIVE=false"
        "PRE_REQUEST_HEALTH_CHECK=false"
        "DEFAULT_LAUNCH_ARGS=${BROWSERLESS_DEFAULT_LAUNCH_ARGS:-"--no-sandbox --disable-dev-shm-usage --disable-gpu --disable-software-rasterizer --disable-dev-tools --disable-features=TranslateUI"}"
        "FUNCTION_ENABLE_INCOGNITO_MODE=true"
        "WORKSPACE_DELETE_EXPIRED=true"
        "WORKSPACE_EXPIRE_DAYS=7"
        "EXIT_ON_HEALTH_FAILURE=true"
        "HEALTH_CHECK_INTERVAL=10000"
    )
    
    # Browserless-specific Docker options for browser isolation
    local docker_opts=(
        "--shm-size=${BROWSERLESS_DOCKER_SHM_SIZE:-2gb}"
        "--cap-add=${BROWSERLESS_DOCKER_CAPS:-SYS_ADMIN}"
        "--security-opt" "seccomp=${BROWSERLESS_DOCKER_SECCOMP:-unconfined}"
        "--memory=${BROWSERLESS_DOCKER_MEMORY:-2g}"
        "--cpus=${BROWSERLESS_DOCKER_CPUS:-2}"
        "--pids-limit=${BROWSERLESS_DOCKER_PIDS_LIMIT:-512}"
    )
    
    # Port mappings and volumes - conditionally configure for host networking
    local port_mappings
    local volumes="${BROWSERLESS_DATA_DIR}:/workspace"
    local health_cmd
    
    if [[ "${BROWSERLESS_USE_HOST_NETWORK:-yes}" == "yes" ]]; then
        # Host networking: container uses host's network directly
        docker_opts+=("--network" "host")
        port_mappings=""  # No port mapping needed with host networking
        health_cmd="curl -f http://localhost:${BROWSERLESS_PORT}/pressure || exit 1"
        log::info "Using host networking for browserless - can access localhost services"
    else
        # Bridge networking: container uses Docker bridge network
        port_mappings="${BROWSERLESS_PORT}:3000"
        health_cmd="curl -f http://localhost:3000/pressure || exit 1"
        log::info "Using bridge networking for browserless - localhost services not accessible"
    fi
    
    # Use advanced creation function (network parameter is ignored when using --network host)
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
    local networking_mode="bridge"
    local localhost_access="No - use Docker bridge IP or container names"
    
    if [[ "${BROWSERLESS_USE_HOST_NETWORK:-yes}" == "yes" ]]; then
        networking_mode="host"
        localhost_access="Yes - can access localhost services"
    fi
    
    local additional_info=(
        "WebSocket: ws://localhost:${BROWSERLESS_PORT}"
        "Status: http://localhost:${BROWSERLESS_PORT}/introspection"
        "API Version: http://localhost:${BROWSERLESS_PORT}/json/version"
        "Concurrent browsers: ${BROWSERLESS_MAX_BROWSERS:-5}"
        "Network mode: ${networking_mode}"
        "Localhost access: ${localhost_access}"
    )
    
    docker_resource::show_connection_info "Browserless" "http://localhost:${BROWSERLESS_PORT}" "${additional_info[@]}"
}
