#!/usr/bin/env bash
# Browserless Docker Operations - Minimal wrapper
# Delegates to docker-utils.sh for all operations

#######################################
# Build Docker run command for Browserless
# This is the only Browserless-specific Docker logic
#######################################
browserless::build_docker_run_args() {
    local max_browsers="${1:-$MAX_BROWSERS}"
    local timeout="${2:-$TIMEOUT}"
    local headless="${3:-$HEADLESS}"
    
    local args=""
    args+=" --name $BROWSERLESS_CONTAINER_NAME"
    args+=" --network $BROWSERLESS_NETWORK_NAME"
    args+=" -p ${BROWSERLESS_PORT}:3000"
    args+=" -v ${BROWSERLESS_DATA_DIR}:/workspace"
    args+=" --restart unless-stopped"
    args+=" --shm-size=${BROWSERLESS_DOCKER_SHM_SIZE}"
    
    # Environment variables
    args+=" -e CONCURRENT=${max_browsers}"
    args+=" -e TIMEOUT=${timeout}"
    args+=" -e ENABLE_DEBUGGER=false"
    args+=" -e PREBOOT_CHROME=true"
    args+=" -e KEEP_ALIVE=true"
    args+=" -e PRE_REQUEST_HEALTH_CHECK=false"
    args+=" -e FUNCTION_ENABLE_INCOGNITO_MODE=true"
    args+=" -e WORKSPACE_DELETE_EXPIRED=true"
    args+=" -e WORKSPACE_EXPIRE_DAYS=7"
    
    # Security settings
    args+=" --cap-add=${BROWSERLESS_DOCKER_CAPS}"
    args+=" --security-opt seccomp=${BROWSERLESS_DOCKER_SECCOMP}"
    
    echo "$args"
}

#######################################
# Legacy compatibility functions
# These delegate to the new framework functions
#######################################

browserless::container_exists() {
    docker::container_exists "$BROWSERLESS_CONTAINER_NAME"
}

browserless::is_running() {
    docker::is_running "$BROWSERLESS_CONTAINER_NAME"
}

browserless::docker_run() {
    local args
    args=$(browserless::build_docker_run_args "$@")
    docker::run_container "$BROWSERLESS_IMAGE" "$args"
}

browserless::docker_stop() {
    docker::stop_container "$BROWSERLESS_CONTAINER_NAME"
}

browserless::docker_start() {
    docker::start_container "$BROWSERLESS_CONTAINER_NAME"
}

browserless::docker_restart() {
    docker::restart_container "$BROWSERLESS_CONTAINER_NAME"
}

browserless::docker_logs() {
    local lines="${1:-50}"
    docker::show_logs "$BROWSERLESS_CONTAINER_NAME" "$lines"
}

browserless::docker_remove() {
    docker::remove_container "$BROWSERLESS_CONTAINER_NAME"
}

browserless::create_network() {
    docker::create_network "$BROWSERLESS_NETWORK_NAME"
}

browserless::remove_network() {
    docker::remove_network "$BROWSERLESS_NETWORK_NAME"
}

browserless::show_resource_usage() {
    docker::show_container_stats "$BROWSERLESS_CONTAINER_NAME"
}