#!/usr/bin/env bash
# Node-RED Docker Operations
# Streamlined Docker wrapper delegating to docker-utils framework

NODE_RED_DOCKER_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${NODE_RED_DOCKER_LIB_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-utils.sh"

#######################################
# Build custom Node-RED Docker image
# Returns: 0 on success, 1 on failure
#######################################
node_red::build_custom_image() {
    local docker_dir="${NODE_RED_SCRIPT_DIR}/docker"
    
    if [[ ! -f "$docker_dir/Dockerfile" ]]; then
        log::error "Dockerfile not found at: $docker_dir/Dockerfile"
        return 1
    fi
    
    log::info "Building custom Node-RED image..."
    docker::build_image "$NODE_RED_CUSTOM_IMAGE" "$docker_dir"
}

#######################################
# Remove Node-RED Docker image
# Returns: 0 on success, 1 on failure
#######################################
node_red::remove_image() {
    local image="${1:-$NODE_RED_CUSTOM_IMAGE}"
    
    log::info "Removing Node-RED image: $image"
    docker::remove_image "$image"
}

#######################################
# Pull latest Node-RED official image
# Returns: 0 on success, 1 on failure
#######################################
node_red::pull_image() {
    local image="${1:-$NODE_RED_IMAGE}"
    
    log::info "Pulling Node-RED image: $image"
    docker::pull_image "$image"
}

#######################################
# Clean up Node-RED Docker resources
# Removes stopped containers, unused networks, and dangling images
# Returns: 0 on success, 1 on failure
#######################################
node_red::cleanup_docker() {
    log::info "Cleaning up Node-RED Docker resources..."
    
    # Remove stopped Node-RED containers
    local stopped_containers
    stopped_containers=$(docker ps -a -f "name=$NODE_RED_CONTAINER_NAME" -f "status=exited" -q)
    if [[ -n "$stopped_containers" ]]; then
        log::info "Removing stopped Node-RED containers..."
        docker rm $stopped_containers
    fi
    
    # Remove Node-RED networks that are not in use
    if docker network exists "$NODE_RED_NETWORK_NAME"; then
        local connected_containers
        connected_containers=$(docker network inspect "$NODE_RED_NETWORK_NAME" --format '{{len .Containers}}' 2>/dev/null || echo "0")
        if [[ "$connected_containers" == "0" ]]; then
            log::info "Removing unused Node-RED network..."
            docker network rm "$NODE_RED_NETWORK_NAME"
        fi
    fi
    
    # Remove dangling Node-RED images
    local dangling_images
    dangling_images=$(docker images -f "dangling=true" -f "label=org.opencontainers.image.title=Node-RED" -q)
    if [[ -n "$dangling_images" ]]; then
        log::info "Removing dangling Node-RED images..."
        docker rmi $dangling_images
    fi
    
    log::success "Docker cleanup completed"
}