#!/usr/bin/env bash
# Node-RED Docker Operations
# Minimal Docker wrapper delegating to docker-utils framework

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
# Get Node-RED image information
# Arguments:
#   $1 - image name (optional, default: NODE_RED_IMAGE)
# Returns: JSON image information
#######################################
node_red::get_image_info() {
    local image="${1:-$NODE_RED_IMAGE}"
    
    docker::get_image_info "$image"
}

#######################################
# List all Node-RED related images
# Returns: List of Node-RED images
#######################################
node_red::list_images() {
    echo "Node-RED Docker Images:"
    echo "======================"
    
    # Official image
    if docker::image_exists "$NODE_RED_IMAGE"; then
        echo "✓ Official: $NODE_RED_IMAGE"
        local size
        size=$(docker images --format "{{.Size}}" "$NODE_RED_IMAGE" | head -1)
        echo "  Size: $size"
    else
        echo "✗ Official: $NODE_RED_IMAGE (not pulled)"
    fi
    
    # Custom image
    if docker::image_exists "$NODE_RED_CUSTOM_IMAGE"; then
        echo "✓ Custom: $NODE_RED_CUSTOM_IMAGE"
        local size
        size=$(docker images --format "{{.Size}}" "$NODE_RED_CUSTOM_IMAGE" | head -1)
        echo "  Size: $size"
    else
        echo "✗ Custom: $NODE_RED_CUSTOM_IMAGE (not built)"
    fi
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

#######################################
# Export Node-RED container as image
# Arguments:
#   $1 - export name (optional, default: node-red-export-TIMESTAMP)
# Returns: 0 on success, 1 on failure
#######################################
node_red::export_container() {
    local export_name="${1:-node-red-export-$(date +%Y%m%d-%H%M%S)}"
    
    if ! docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        log::error "Node-RED container does not exist"
        return 1
    fi
    
    log::info "Exporting Node-RED container as: $export_name"
    docker commit "$NODE_RED_CONTAINER_NAME" "$export_name"
    
    if docker::image_exists "$export_name"; then
        log::success "Container exported as image: $export_name"
        return 0
    else
        log::error "Failed to export container"
        return 1
    fi
}

#######################################
# Get Node-RED container logs with filtering
# Arguments:
#   $1 - log level filter (optional: error, warn, info, debug)
#   $2 - lines to show (optional, default: 100)
# Returns: Filtered log output
#######################################
node_red::get_filtered_logs() {
    local level_filter="${1:-}"
    local lines="${2:-100}"
    
    if ! docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        log::error "Node-RED container does not exist"
        return 1
    fi
    
    local logs
    logs=$(docker logs --tail "$lines" "$NODE_RED_CONTAINER_NAME" 2>&1)
    
    if [[ -n "$level_filter" ]]; then
        case "$level_filter" in
            "error")
                echo "$logs" | grep -i "error\|fatal"
                ;;
            "warn")
                echo "$logs" | grep -i "warn\|warning"
                ;;
            "info")
                echo "$logs" | grep -i "info"
                ;;
            "debug")
                echo "$logs" | grep -i "debug\|trace"
                ;;
            *)
                echo "$logs"
                ;;
        esac
    else
        echo "$logs"
    fi
}