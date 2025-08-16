#!/usr/bin/env bash
# ComfyUI Docker Container Management - Simplified with docker-resource-utils.sh
# Handles container lifecycle, configuration, and Docker-specific operations

# Source var.sh to get proper directory variables
_COMFYUI_DOCKER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${_COMFYUI_DOCKER_DIR}/../../../../lib/utils/var.sh"

# Source shared libraries
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-resource-utils.sh"

#######################################
# Simple health check for ComfyUI using the system_stats endpoint
#######################################
comfyui::docker::is_healthy() {
    curl -s -f "http://localhost:${COMFYUI_CUSTOM_PORT:-$COMFYUI_DEFAULT_PORT}${COMFYUI_HEALTH_ENDPOINT}" >/dev/null 2>&1
}

#######################################
# Wait for ComfyUI to be healthy with timeout
#######################################
comfyui::docker::_wait_for_healthy() {
    local max_wait=${1:-60} waited=0
    log::info "Waiting for ComfyUI to be ready..."
    
    while [[ $waited -lt $max_wait ]]; do
        comfyui::docker::is_healthy && return 0
        sleep ${COMFYUI_RESTART_WAIT:-2}
        waited=$((waited + 2))
    done
    return 1
}

#######################################
# Build GPU-specific arguments for ComfyUI
#######################################
comfyui::docker::_get_gpu_args() {
    case "${GPU_TYPE}" in
        nvidia)
            if docker info 2>/dev/null | grep -E "(Runtimes:.*nvidia|Default Runtime: nvidia)" >/dev/null; then
                echo "--gpus all -e NVIDIA_VISIBLE_DEVICES=all -e NVIDIA_DRIVER_CAPABILITIES=all"
                [[ -n "$COMFYUI_VRAM_LIMIT" ]] && echo "-e PYTORCH_CUDA_ALLOC_CONF=max_split_size_mb=$((COMFYUI_VRAM_LIMIT * 1024))"
            else
                log::warn "NVIDIA runtime not available, falling back to CPU mode"
                echo "-e CUDA_VISIBLE_DEVICES="
            fi
            ;;
        amd)
            echo "--device=/dev/kfd --device=/dev/dri --group-add video -e HSA_OVERRIDE_GFX_VERSION=10.3.0 -e PYTORCH_HIP_ALLOC_CONF=max_split_size_mb=512"
            ;;
        *)
            echo "-e CUDA_VISIBLE_DEVICES="
            ;;
    esac
}

#######################################
# Start ComfyUI container with appropriate configuration
#######################################
comfyui::docker::start_container() {
    log::info "Starting ComfyUI container..."
    
    local port="${COMFYUI_CUSTOM_PORT:-$COMFYUI_DEFAULT_PORT}"
    local image="${COMFYUI_CUSTOM_IMAGE:-$COMFYUI_DEFAULT_IMAGE}"
    local network="${COMFYUI_CONTAINER_NAME}-network"
    
    # Build volumes string
    local volumes=""
    for volume in "${COMFYUI_VOLUMES[@]}"; do
        volumes+="$volume "
    done
    
    # Build GPU arguments
    local gpu_args
    gpu_args=$(comfyui::docker::_get_gpu_args)
    
    # Create container with GPU support using direct docker command
    # This preserves ComfyUI's specific GPU requirements while simplifying the structure
    local cmd=(docker run -d --name "$COMFYUI_CONTAINER_NAME")
    cmd+=(--restart unless-stopped)
    cmd+=(-p "${port}:8188")
    cmd+=(--network "$network")
    
    # Add GPU args if any
    [[ -n "$gpu_args" ]] && cmd+=($gpu_args)
    
    # Add volumes
    for volume in ${volumes}; do
        cmd+=(-v "$volume")
    done
    
    # Add environment variables
    for env_var in "${COMFYUI_ENV_VARS[@]}"; do
        cmd+=(-e "$env_var")
    done
    
    # Add health check
    cmd+=(--health-cmd "curl -f http://localhost:8188${COMFYUI_HEALTH_ENDPOINT} || exit 1")
    cmd+=(--health-interval 30s --health-timeout 5s --health-retries 3)
    
    cmd+=("$image")
    
    # Create network first
    docker::create_network "$network"
    
    log::debug "Docker command: ${cmd[*]}"
    
    if "${cmd[@]}" >/dev/null 2>&1; then
        log::success "Container started successfully"
        
        # Wait for health check
        if comfyui::docker::_wait_for_healthy 120; then
            log::success "ComfyUI is ready!"
            echo
            log::header "🚀 ComfyUI Access Information"
            log::info "Web UI: http://localhost:$port"
            log::info "API Endpoint: http://localhost:$port/api"
            echo
            return 0
        else
            log::error "ComfyUI failed to become ready within 120 seconds"
            return 1
        fi
    else
        log::error "Failed to start container"
        return 1
    fi
}

#######################################
# Stop ComfyUI container
#######################################
comfyui::docker::stop() {
    docker::container_exists "$COMFYUI_CONTAINER_NAME" || { log::info "ComfyUI container does not exist"; return 0; }
    docker::is_running "$COMFYUI_CONTAINER_NAME" || { log::info "ComfyUI is not running"; return 0; }
    
    log::info "Stopping ComfyUI..."
    docker::stop_container "$COMFYUI_CONTAINER_NAME" && log::success "ComfyUI stopped successfully"
}

#######################################
# Start existing ComfyUI container
#######################################
comfyui::docker::start() {
    docker::container_exists "$COMFYUI_CONTAINER_NAME" || { log::error "ComfyUI container does not exist. Run install first."; return 1; }
    docker::is_running "$COMFYUI_CONTAINER_NAME" && { log::info "ComfyUI is already running"; return 0; }
    
    log::info "Starting ComfyUI..."
    
    if docker start "$COMFYUI_CONTAINER_NAME"; then
        # Wait for health check
        if comfyui::docker::_wait_for_healthy 60; then
            log::success "ComfyUI started successfully"
            log::info "Access ComfyUI at: http://localhost:${COMFYUI_CUSTOM_PORT:-$COMFYUI_DEFAULT_PORT}"
            return 0
        else
            log::error "ComfyUI started but is not responding"
            return 1
        fi
    else
        log::error "Failed to start ComfyUI"
        return 1
    fi
}

#######################################
# Restart ComfyUI container
#######################################
comfyui::docker::restart() {
    log::info "Restarting ComfyUI..."
    
    if docker::restart_container "$COMFYUI_CONTAINER_NAME"; then
        # Wait for health check
        if comfyui::docker::_wait_for_healthy 60; then
            log::success "ComfyUI restarted successfully"
            return 0
        else
            log::error "ComfyUI restarted but is not responding"
            return 1
        fi
    else
        log::error "Failed to restart ComfyUI"
        return 1
    fi
}

#######################################
# Show ComfyUI container logs
#######################################
comfyui::docker::logs() {
    docker::container_exists "$COMFYUI_CONTAINER_NAME" || { log::error "ComfyUI container does not exist"; return 1; }
    
    log::info "Showing ComfyUI logs (press Ctrl+C to exit)..."
    echo
    docker logs -f --tail 100 "$COMFYUI_CONTAINER_NAME"
}

#######################################
# Remove ComfyUI container
#######################################
comfyui::docker::remove_container() {
    docker::container_exists "$COMFYUI_CONTAINER_NAME" || { log::debug "Container does not exist, nothing to remove"; return 0; }
    
    log::info "Removing ComfyUI container..."
    if docker::is_running "$COMFYUI_CONTAINER_NAME"; then
        docker stop "$COMFYUI_CONTAINER_NAME" || true
    fi
    docker rm -f "$COMFYUI_CONTAINER_NAME" && log::success "Container removed successfully"
}

#######################################
# Pull ComfyUI Docker image
#######################################
comfyui::docker::pull_image() {
    local image="${COMFYUI_CUSTOM_IMAGE:-$COMFYUI_DEFAULT_IMAGE}"
    
    log::info "Pulling ComfyUI Docker image: $image"
    log::info "This may take several minutes depending on your internet connection..."
    
    if docker pull "$image"; then
        log::success "Successfully pulled ComfyUI image"
        log::info "Image details:"
        docker images "$image" --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
    else
        log::error "Failed to pull ComfyUI image. Check your internet connection and Docker Hub access."
        return 1
    fi
}

#######################################
# Execute command in ComfyUI container
#######################################
comfyui::docker::exec() {
    local command="$1"
    shift
    
    docker::is_running "$COMFYUI_CONTAINER_NAME" || { log::error "ComfyUI container is not running"; return 1; }
    docker exec "$COMFYUI_CONTAINER_NAME" "$command" "$@"
}

#######################################
# Copy file to ComfyUI container
#######################################
comfyui::docker::copy_to_container() {
    local source="$1" dest="$2"
    
    docker::container_exists "$COMFYUI_CONTAINER_NAME" || { log::error "ComfyUI container does not exist"; return 1; }
    docker cp "$source" "${COMFYUI_CONTAINER_NAME}:${dest}"
}

#######################################
# Copy file from ComfyUI container
#######################################
comfyui::docker::copy_from_container() {
    local source="$1" dest="$2"
    
    docker::container_exists "$COMFYUI_CONTAINER_NAME" || { log::error "ComfyUI container does not exist"; return 1; }
    docker cp "${COMFYUI_CONTAINER_NAME}:${source}" "$dest"
}