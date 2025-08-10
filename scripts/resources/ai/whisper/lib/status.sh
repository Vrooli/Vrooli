#!/usr/bin/env bash
# Whisper Status Functions
# Functions for checking and reporting Whisper status

#######################################
# Show comprehensive Whisper status
# Returns: 0 if successful, 1 otherwise
#######################################
status::show_status() {
    log::header "🎤 Whisper Status"
    
    local overall_status="healthy"
    
    # Check Docker availability
    echo "Docker Status:"
    if status::check_docker; then
        log::success "  ✅ Docker is available and configured"
    else
        log::error "  ❌ Docker is not available"
        overall_status="unhealthy"
    fi
    
    echo
    
    # Check container status
    echo "Container Status:"
    if status::container_exists; then
        if status::is_running; then
            log::success "  ✅ Whisper container is running"
            
            # Get basic container info
            local created=$(docker inspect "$WHISPER_CONTAINER_NAME" --format '{{.Created}}' 2>/dev/null)
            local image=$(docker inspect "$WHISPER_CONTAINER_NAME" --format '{{.Config.Image}}' 2>/dev/null)
            log::info "     Container: $WHISPER_CONTAINER_NAME"
            log::info "     Image: $image"
            log::info "     Created: ${created:0:19}"
        else
            log::warn "  ⚠️  Whisper container exists but is not running"
            overall_status="degraded"
        fi
    else
        log::info "  ℹ️  Whisper container does not exist"
        overall_status="not_installed"
    fi
    
    echo
    
    # Check API status
    echo "API Status:"
    if status::is_running; then
        log::info "  🔍 Testing API endpoint: $WHISPER_BASE_URL"
        
        if status::is_healthy; then
            log::success "  ✅ Whisper API is responding"
            
            # Get model information if available
            local model_info
            model_info=$(status::get_current_model 2>/dev/null)
            if [[ -n "$model_info" ]]; then
                log::info "     Current model: $model_info"
            fi
        else
            log::error "  ❌ Whisper API is not responding"
            overall_status="unhealthy"
        fi
    else
        log::info "  ⏸️  API not available (container not running)"
    fi
    
    echo
    
    # Check port status
    echo "Network Status:"
    log::info "  Port: $WHISPER_PORT"
    if system::is_port_in_use "$WHISPER_PORT"; then
        if status::is_running; then
            log::success "  ✅ Port $WHISPER_PORT is in use by Whisper"
        else
            log::warn "  ⚠️  Port $WHISPER_PORT is in use by another service"
            overall_status="conflict"
        fi
    else
        log::info "  ℹ️  Port $WHISPER_PORT is available"
    fi
    
    echo
    
    # Check data directories
    echo "Storage Status:"
    if [[ -d "$WHISPER_DATA_DIR" ]]; then
        log::success "  ✅ Data directory exists: $WHISPER_DATA_DIR"
        
        # Check subdirectories
        for dir in "$WHISPER_MODELS_DIR" "$WHISPER_UPLOADS_DIR"; do
            if [[ -d "$dir" ]]; then
                local size=$(du -sh "$dir" 2>/dev/null | cut -f1)
                log::info "     $(basename "$dir"): $size"
            else
                log::info "     $(basename "$dir"): not created"
            fi
        done
    else
        log::info "  ℹ️  Data directory not created: $WHISPER_DATA_DIR"
    fi
    
    echo
    
    # Check GPU status if GPU is enabled
    if [[ "$WHISPER_GPU_ENABLED" == "yes" ]]; then
        echo "GPU Status:"
        if status::is_gpu_available; then
            log::success "  ✅ NVIDIA GPU is available"
            if system::is_command "nvidia-smi"; then
                local gpu_info=$(nvidia-smi --query-gpu=name,memory.used,memory.total --format=csv,noheader,nounits 2>/dev/null | head -1)
                if [[ -n "$gpu_info" ]]; then
                    log::info "     GPU: $gpu_info"
                fi
            fi
        else
            log::warn "  ⚠️  GPU not available, will use CPU"
        fi
        echo
    fi
    
    # Configuration summary
    echo "Configuration:"
    log::info "  Base URL: $WHISPER_BASE_URL"
    log::info "  Container: $WHISPER_CONTAINER_NAME"
    log::info "  Default Model: $WHISPER_DEFAULT_MODEL"
    log::info "  Data Directory: $WHISPER_DATA_DIR"
    log::info "  GPU Enabled: $WHISPER_GPU_ENABLED"
    
    echo
    
    # Overall status summary
    case "$overall_status" in
        "healthy")
            log::success "🎉 Overall Status: Healthy - Whisper is running and responding"
            return 0
            ;;
        "degraded")
            log::warn "⚠️  Overall Status: Degraded - Some issues detected"
            return 0
            ;;
        "conflict")
            log::warn "⚠️  Overall Status: Port Conflict - Port is in use by another service"
            return 1
            ;;
        "unhealthy")
            log::error "❌ Overall Status: Unhealthy - Whisper is not working properly"
            return 1
            ;;
        "not_installed")
            log::info "ℹ️  Overall Status: Not Installed - Run installation first"
            return 1
            ;;
        *)
            log::error "❓ Overall Status: Unknown"
            return 1
            ;;
    esac
}

#######################################
# Get current model from running container
# Outputs: model name
#######################################
status::get_current_model() {
    if ! status::is_running; then
        return 1
    fi
    
    # Try to get model from environment variable
    docker inspect "$WHISPER_CONTAINER_NAME" --format '{{range .Config.Env}}{{println .}}{{end}}' 2>/dev/null | \
        grep "ASR_MODEL=" | \
        cut -d= -f2
}

#######################################
# Show quick status (for use in other scripts)
# Outputs: status string
#######################################
status::quick_status() {
    if ! status::is_running; then
        echo "stopped"
        return 1
    fi
    
    if status::is_healthy; then
        echo "running"
        return 0
    else
        echo "unhealthy"
        return 1
    fi
}

#######################################
# Check if Whisper is ready for transcription
# Returns: 0 if ready, 1 otherwise
#######################################
status::is_ready() {
    if ! status::is_running; then
        return 1
    fi
    
    if ! status::is_healthy; then
        return 1
    fi
    
    # Additional check: ensure model is loaded
    # This might take time on first startup
    local attempts=0
    local max_attempts=5
    
    while [[ $attempts -lt $max_attempts ]]; do
        local response
        response=$(curl -s -o /dev/null -w "%{http_code}" \
            --max-time "$WHISPER_API_TIMEOUT" \
            "$WHISPER_BASE_URL/health" 2>/dev/null)
        
        if [[ "$response" == "200" ]]; then
            return 0
        fi
        
        ((attempts++))
        sleep 2
    done
    
    return 1
}

#######################################
# Show resource usage
# Returns: 0 if successful, 1 otherwise
#######################################
status::show_resource_usage() {
    if ! status::is_running; then
        log::error "${MSG_NOT_RUNNING}"
        return 1
    fi
    
    echo "Resource Usage:"
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" "$WHISPER_CONTAINER_NAME"
}

# Export functions for subshell availability
export -f status::show_status
export -f status::get_current_model
export -f status::quick_status
export -f status::is_ready
export -f status::show_resource_usage