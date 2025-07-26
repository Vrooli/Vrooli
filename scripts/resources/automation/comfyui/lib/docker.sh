#!/usr/bin/env bash
# ComfyUI Docker Container Management
# Handles container lifecycle, configuration, and Docker-specific operations

#######################################
# Start ComfyUI container with appropriate configuration
#######################################
comfyui::start_container() {
    log::info "Starting ComfyUI container..."
    
    local port="${COMFYUI_CUSTOM_PORT:-$COMFYUI_DEFAULT_PORT}"
    local image="${COMFYUI_CUSTOM_IMAGE:-$COMFYUI_DEFAULT_IMAGE}"
    
    # Build Docker run arguments
    local docker_args=(
        "run"
        "-d"
        "--name" "$COMFYUI_CONTAINER_NAME"
        "-p" "${port}:${COMFYUI_PROXY_PORT}"
        "-p" "${COMFYUI_DIRECT_PORT}:8188"
        "--restart" "unless-stopped"
    )
    
    # Add GPU support based on detected/specified type
    case "$GPU_TYPE" in
        nvidia)
            log::info "Configuring container for NVIDIA GPU..."
            
            # Check if NVIDIA runtime is available
            if docker::run info 2>/dev/null | grep -E "(Runtimes:.*nvidia|Default Runtime: nvidia)" >/dev/null; then
                docker_args+=("--gpus" "all")
                
                # Add NVIDIA-specific environment variables
                docker_args+=("-e" "NVIDIA_VISIBLE_DEVICES=all")
                docker_args+=("-e" "NVIDIA_DRIVER_CAPABILITIES=all")
                
                # Add VRAM limit if specified
                if [[ -n "$COMFYUI_VRAM_LIMIT" ]]; then
                    log::info "Setting VRAM limit to ${COMFYUI_VRAM_LIMIT}GB"
                    docker_args+=("-e" "PYTORCH_CUDA_ALLOC_CONF=max_split_size_mb=$((COMFYUI_VRAM_LIMIT * 1024))")
                fi
            else
                log::warn "NVIDIA runtime not available, falling back to CPU mode"
                GPU_TYPE="cpu"
            fi
            ;;
        amd)
            log::info "Configuring container for AMD GPU..."
            
            # AMD GPU support requires specific devices and environment
            docker_args+=(
                "--device=/dev/kfd"
                "--device=/dev/dri"
                "--group-add" "video"
                "-e" "HSA_OVERRIDE_GFX_VERSION=10.3.0"
                "-e" "PYTORCH_HIP_ALLOC_CONF=max_split_size_mb=512"
            )
            ;;
        cpu|*)
            log::info "Configuring container for CPU mode..."
            docker_args+=("-e" "CUDA_VISIBLE_DEVICES=")
            ;;
    esac
    
    # Add volumes
    for volume in "${COMFYUI_VOLUMES[@]}"; do
        docker_args+=("-v" "$volume")
    done
    
    # Add environment variables
    for env_var in "${COMFYUI_ENV_VARS[@]}"; do
        docker_args+=("-e" "$env_var")
    done
    
    # Add the image
    docker_args+=("$image")
    
    # Execute Docker run
    log::info "Launching ComfyUI container..."
    log::debug "Docker command: docker ${docker_args[*]}"
    
    if docker::run "${docker_args[@]}"; then
        log::success "Container started successfully"
        
        # Add rollback action
        resources::add_rollback_action \
            "Stop and remove ComfyUI container" \
            "docker::run stop $COMFYUI_CONTAINER_NAME && docker::run rm $COMFYUI_CONTAINER_NAME" \
            10
        
        # Wait for container to be ready
        log::info "Waiting for ComfyUI to initialize..."
        
        local max_wait=120  # 2 minutes
        local waited=0
        
        while [[ $waited -lt $max_wait ]]; do
            if comfyui::is_healthy; then
                log::success "ComfyUI is ready!"
                
                # Show access information
                echo
                log::header "🚀 ComfyUI Access Information"
                log::info "Web UI: http://localhost:$port"
                log::info "API Endpoint: http://localhost:$port/api"
                log::info "Direct ComfyUI: http://localhost:$COMFYUI_DIRECT_PORT"
                echo
                
                return 0
            fi
            
            sleep 2
            waited=$((waited + 2))
            
            # Check if container is still running
            if ! comfyui::is_running; then
                log::error "Container stopped unexpectedly"
                log::info "Checking container logs..."
                docker::run logs --tail 50 "$COMFYUI_CONTAINER_NAME"
                return 1
            fi
        done
        
        log::error "ComfyUI failed to become ready within ${max_wait} seconds"
        log::info "Container is running but API is not responding"
        log::info "Check logs with: $0 --action logs"
        return 1
        
    else
        log::error "Failed to start container"
        return 1
    fi
}

#######################################
# Stop ComfyUI container
#######################################
comfyui::stop() {
    if ! comfyui::container_exists; then
        log::info "ComfyUI container does not exist"
        return 0
    fi
    
    if ! comfyui::is_running; then
        log::info "ComfyUI is not running"
        return 0
    fi
    
    log::info "Stopping ComfyUI..."
    
    if docker::run stop "$COMFYUI_CONTAINER_NAME"; then
        log::success "ComfyUI stopped successfully"
        return 0
    else
        log::error "Failed to stop ComfyUI"
        return 1
    fi
}

#######################################
# Start existing ComfyUI container
#######################################
comfyui::start() {
    if ! comfyui::container_exists; then
        log::error "ComfyUI container does not exist"
        log::info "Run '$0 --action install' first"
        return 1
    fi
    
    if comfyui::is_running; then
        log::info "ComfyUI is already running"
        return 0
    fi
    
    log::info "Starting ComfyUI..."
    
    if docker::run start "$COMFYUI_CONTAINER_NAME"; then
        # Wait for it to be ready
        log::info "Waiting for ComfyUI to be ready..."
        
        if comfyui::is_healthy; then
            log::success "ComfyUI started successfully"
            
            local port="${COMFYUI_CUSTOM_PORT:-$COMFYUI_DEFAULT_PORT}"
            log::info "Access ComfyUI at: http://localhost:$port"
            
            return 0
        else
            log::error "ComfyUI started but is not responding"
            log::info "Check logs with: $0 --action logs"
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
comfyui::restart() {
    log::info "Restarting ComfyUI..."
    
    if comfyui::stop; then
        sleep 2
        if comfyui::start; then
            log::success "ComfyUI restarted successfully"
            return 0
        fi
    fi
    
    log::error "Failed to restart ComfyUI"
    return 1
}

#######################################
# Show ComfyUI container logs
#######################################
comfyui::logs() {
    if ! comfyui::container_exists; then
        log::error "ComfyUI container does not exist"
        return 1
    fi
    
    log::info "Showing ComfyUI logs (press Ctrl+C to exit)..."
    echo
    
    # Show last 100 lines and follow
    docker::run logs -f --tail 100 "$COMFYUI_CONTAINER_NAME"
}

#######################################
# Remove ComfyUI container
#######################################
comfyui::remove_container() {
    if ! comfyui::container_exists; then
        log::debug "Container does not exist, nothing to remove"
        return 0
    fi
    
    # Stop container if running
    if comfyui::is_running; then
        log::info "Stopping ComfyUI container..."
        docker::run stop "$COMFYUI_CONTAINER_NAME" || {
            log::warn "Failed to stop container gracefully"
        }
    fi
    
    # Remove container
    log::info "Removing ComfyUI container..."
    if docker::run rm -f "$COMFYUI_CONTAINER_NAME"; then
        log::success "Container removed successfully"
        return 0
    else
        log::error "Failed to remove container"
        return 1
    fi
}

#######################################
# Pull ComfyUI Docker image
#######################################
comfyui::pull_image() {
    local image="${COMFYUI_CUSTOM_IMAGE:-$COMFYUI_DEFAULT_IMAGE}"
    
    log::info "Pulling ComfyUI Docker image: $image"
    log::info "This may take several minutes depending on your internet connection..."
    
    if docker::run pull "$image"; then
        log::success "Successfully pulled ComfyUI image"
        
        # Show image information
        log::info "Image details:"
        docker::run images "$image" --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
        
        return 0
    else
        log::error "Failed to pull ComfyUI image"
        log::info "Check your internet connection and Docker Hub access"
        return 1
    fi
}

#######################################
# Execute command in ComfyUI container
#######################################
comfyui::exec() {
    local command="$1"
    shift
    local args=("$@")
    
    if ! comfyui::is_running; then
        log::error "ComfyUI container is not running"
        return 1
    fi
    
    docker::run exec "$COMFYUI_CONTAINER_NAME" "$command" "${args[@]}"
}

#######################################
# Copy file to ComfyUI container
#######################################
comfyui::copy_to_container() {
    local source="$1"
    local dest="$2"
    
    if ! comfyui::container_exists; then
        log::error "ComfyUI container does not exist"
        return 1
    fi
    
    docker::run cp "$source" "${COMFYUI_CONTAINER_NAME}:${dest}"
}

#######################################
# Copy file from ComfyUI container
#######################################
comfyui::copy_from_container() {
    local source="$1"
    local dest="$2"
    
    if ! comfyui::container_exists; then
        log::error "ComfyUI container does not exist"
        return 1
    fi
    
    docker::run cp "${COMFYUI_CONTAINER_NAME}:${source}" "$dest"
}