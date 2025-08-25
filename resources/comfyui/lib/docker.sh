#!/usr/bin/env bash
# ComfyUI Docker Container Management - Simplified with docker-resource-utils.sh
# Handles container lifecycle, configuration, and Docker-specific operations

# Source var.sh to get proper directory variables
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
_COMFYUI_DOCKER_DIR="${APP_ROOT}/resources/comfyui/lib"
# shellcheck disable=SC1091
source "${_COMFYUI_DOCKER_DIR}/../../../lib/utils/var.sh"

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
# Start ComfyUI container with appropriate configuration
#######################################
comfyui::docker::start_container() {
    log::info "Starting ComfyUI container..."
    
    local port="${COMFYUI_CUSTOM_PORT:-$COMFYUI_DEFAULT_PORT}"
    local image="${COMFYUI_CUSTOM_IMAGE:-$COMFYUI_DEFAULT_IMAGE}"
    local network="${COMFYUI_CONTAINER_NAME}-network"
    
    # Pull image
    docker::pull_image "$image"
    
    # Build volumes string
    local volumes=""
    for volume in "${COMFYUI_VOLUMES[@]}"; do
        volumes+="$volume "
    done
    
    # Prepare environment variables
    local env_vars=()
    for env_var in "${COMFYUI_ENV_VARS[@]}"; do
        env_vars+=("$env_var")
    done
    
    # Build GPU-specific Docker options
    local docker_opts=()
    case "${GPU_TYPE}" in
        nvidia)
            if docker info 2>/dev/null | grep -E "(Runtimes:.*nvidia|Default Runtime: nvidia)" >/dev/null; then
                docker_opts+=("--gpus" "all")
                env_vars+=("NVIDIA_VISIBLE_DEVICES=all" "NVIDIA_DRIVER_CAPABILITIES=all")
                [[ -n "$COMFYUI_VRAM_LIMIT" ]] && env_vars+=("PYTORCH_CUDA_ALLOC_CONF=max_split_size_mb=$((COMFYUI_VRAM_LIMIT * 1024))")
            else
                log::warn "NVIDIA runtime not available, falling back to CPU mode"
                env_vars+=("CUDA_VISIBLE_DEVICES=")
            fi
            ;;
        amd)
            docker_opts+=("--device=/dev/kfd" "--device=/dev/dri" "--group-add" "video")
            env_vars+=("HSA_OVERRIDE_GFX_VERSION=10.3.0" "PYTORCH_HIP_ALLOC_CONF=max_split_size_mb=512")
            ;;
        *)
            env_vars+=("CUDA_VISIBLE_DEVICES=")
            ;;
    esac
    
    # Health check
    local health_cmd="curl -f http://localhost:8188${COMFYUI_HEALTH_ENDPOINT} || exit 1"
    
    # Use advanced creation
    docker_resource::create_service_advanced \
        "$COMFYUI_CONTAINER_NAME" \
        "$image" \
        "${port}:8188" \
        "$network" \
        "$volumes" \
        "env_vars" \
        "docker_opts" \
        "$health_cmd" \
        ""
    
    if [[ $? -eq 0 ]]; then
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

comfyui::docker::stop() {
    docker::container_exists "$COMFYUI_CONTAINER_NAME" || { log::info "ComfyUI container does not exist"; return 0; }
    docker::is_running "$COMFYUI_CONTAINER_NAME" || { log::info "ComfyUI is not running"; return 0; }
    
    docker::stop_container "$COMFYUI_CONTAINER_NAME"
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
    local lines="${1:-100}"
    local follow="${2:-true}"
    docker_resource::show_logs_with_follow "$COMFYUI_CONTAINER_NAME" "$lines" "$follow"
}

comfyui::docker::remove_container() {
    docker::remove_container "$COMFYUI_CONTAINER_NAME" "true"
}

#######################################
# Pull ComfyUI Docker image
#######################################
comfyui::docker::pull_image() {
    local image="${COMFYUI_CUSTOM_IMAGE:-$COMFYUI_DEFAULT_IMAGE}"
    log::info "Pulling ComfyUI Docker image: $image"
    docker::pull_image "$image"
}

comfyui::docker::exec() {
    docker_resource::exec "$COMFYUI_CONTAINER_NAME" "$@"
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