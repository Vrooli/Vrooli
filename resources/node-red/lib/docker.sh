#!/usr/bin/env bash
# Node-RED Docker Management - Minimal Implementation
# Only essential functions needed for Node-RED operation

# Source var.sh to get proper directory variables
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
_NODE_RED_DOCKER_DIR="${APP_ROOT}/resources/node-red/lib"
# shellcheck disable=SC1091
source "${_NODE_RED_DOCKER_DIR}/../../../lib/utils/var.sh"

# Source shared libraries
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-resource-utils.sh"

# Create and start Node-RED container
node_red::docker::create_container() {
    # Ensure first-time setup is completed
    if node_red::is_first_run; then
        node_red::first_time_setup || return 1
    fi
    
    log::info "Starting Node-RED container..."
    
    # Pull image if needed
    log::info "Pulling Node-RED image..."
    docker::pull_image "$NODE_RED_IMAGE"
    
    # Select image (custom if available and BUILD_IMAGE is set)
    local image_to_use="$NODE_RED_IMAGE"
    if [[ "$BUILD_IMAGE" == "yes" ]] && docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${NODE_RED_CUSTOM_IMAGE}$"; then
        image_to_use="$NODE_RED_CUSTOM_IMAGE"
    fi
    
    # Prepare volumes
    local volumes="${NODE_RED_DATA_DIR}:/data"
    if [[ "$BUILD_IMAGE" == "yes" ]]; then
        volumes+=" /var/run/docker.sock:/var/run/docker.sock:rw"
        volumes+=" ${PWD}:/workspace:rw"
        volumes+=" $HOME:/host/home:rw"
    fi
    
    # Create container using simplified API
    if docker_resource::create_service_with_command \
        "$NODE_RED_CONTAINER_NAME" \
        "$image_to_use" \
        "$NODE_RED_PORT" \
        "1880" \
        "$NODE_RED_NETWORK_NAME" \
        "$volumes" \
        "curl -f http://localhost:1880 || exit 1" \
        "node-red"; then
        
        sleep "${NODE_RED_INITIALIZATION_WAIT:-3}"
        return 0
    else
        log::error "Failed to create Node-RED container"
        return 1
    fi
}

# Build custom Node-RED image if Dockerfile exists
node_red::build_custom_image() {
    local docker_dir="${NODE_RED_SCRIPT_DIR}/docker"
    
    if [[ ! -f "$docker_dir/Dockerfile" ]]; then
        log::error "Dockerfile not found at: $docker_dir/Dockerfile"
        return 1
    fi
    
    log::info "Building custom Node-RED image..."
    docker build -t "$NODE_RED_CUSTOM_IMAGE" "$docker_dir"
}

# Remove Node-RED container and clean up
# Arguments: $1 - remove data (yes/no)
node_red::docker::remove_container() {
    local remove_data="${1:-no}"
    
    # Stop and remove container
    docker::remove_container "$NODE_RED_CONTAINER_NAME" "true"
    
    # Remove network only if empty
    docker::cleanup_network_if_empty "$NODE_RED_NETWORK_NAME"
    
    # Handle data removal if requested
    if [[ "$remove_data" == "yes" ]]; then
        docker_resource::remove_data "Node-RED" "$NODE_RED_DATA_DIR" "yes"
    fi
    
    return 0
}

# Start Node-RED container
node_red::docker::start() {
    if ! docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        log::error "Node-RED container does not exist. Run install first."
        return 1
    fi
    
    log::info "Starting Node-RED container..."
    
    if docker::start_container "$NODE_RED_CONTAINER_NAME"; then
        if node_red::wait_for_ready; then
            log::success "Node-RED started successfully"
            log::info "Access URL: http://localhost:$NODE_RED_PORT"
            return 0
        else
            log::error "Node-RED failed to start properly"
            return 1
        fi
    else
        log::error "Failed to start Node-RED container"
        return 1
    fi
}

# Stop Node-RED container
node_red::docker::stop() {
    if ! docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        log::warn "Node-RED container does not exist"
        return 0
    fi
    
    log::info "Stopping Node-RED container..."
    
    if docker::stop_container "$NODE_RED_CONTAINER_NAME"; then
        log::success "Node-RED stopped successfully"
        return 0
    else
        log::error "Failed to stop Node-RED container"
        return 1
    fi
}

# Restart Node-RED container
node_red::docker::restart() {
    log::info "Restarting Node-RED container..."
    
    if docker::restart_container "$NODE_RED_CONTAINER_NAME"; then
        if node_red::wait_for_ready; then
            log::success "Node-RED restarted successfully"
            return 0
        else
            log::error "Node-RED failed to start after restart"
            return 1
        fi
    else
        log::error "Failed to restart Node-RED"
        return 1
    fi
}


