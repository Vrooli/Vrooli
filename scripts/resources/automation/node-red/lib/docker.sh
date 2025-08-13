#!/usr/bin/env bash
# Node-RED Docker Operations - Minimal wrapper for docker-utils

NODE_RED_DOCKER_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${NODE_RED_DOCKER_LIB_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-utils.sh"

# Build custom Node-RED image if Dockerfile exists
node_red::build_custom_image() {
    local docker_dir="${NODE_RED_SCRIPT_DIR}/docker"
    
    if [[ ! -f "$docker_dir/Dockerfile" ]]; then
        log::error "Dockerfile not found at: $docker_dir/Dockerfile"
        return 1
    fi
    
    log::info "Building custom Node-RED image..."
    docker::build_image "$NODE_RED_CUSTOM_IMAGE" "$docker_dir"
}

# Clean up Node-RED specific resources  
node_red::cleanup_docker() {
    log::info "Cleaning up Node-RED Docker resources..."
    
    # Use docker-utils for container cleanup
    docker::cleanup_stopped_containers "$NODE_RED_CONTAINER_NAME"
    
    # Use docker-utils for network cleanup
    docker::remove_network_if_empty "$NODE_RED_NETWORK_NAME"
    
    log::success "Docker cleanup completed"
}