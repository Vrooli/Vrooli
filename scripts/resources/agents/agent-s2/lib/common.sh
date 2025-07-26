#!/usr/bin/env bash
# Agent S2 Common Functions
# Shared utilities for Agent S2 management

#######################################
# Check if Agent S2 container exists
# Returns: 0 if exists, 1 otherwise
#######################################
agents2::container_exists() {
    docker ps -a --format "{{.Names}}" | grep -q "^${AGENTS2_CONTAINER_NAME}$"
}

#######################################
# Check if Agent S2 is running
# Returns: 0 if running, 1 otherwise
#######################################
agents2::is_running() {
    docker ps --format "{{.Names}}" | grep -q "^${AGENTS2_CONTAINER_NAME}$"
}

#######################################
# Check if Agent S2 image exists
# Returns: 0 if exists, 1 otherwise
#######################################
agents2::image_exists() {
    docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${AGENTS2_IMAGE_NAME}$"
}

#######################################
# Check if Agent S2 is healthy
# Returns: 0 if healthy, 1 otherwise
#######################################
agents2::is_healthy() {
    local max_attempts=3
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if curl -f -s --max-time "$AGENTS2_API_TIMEOUT" "${AGENTS2_BASE_URL}/health" >/dev/null 2>&1; then
            return 0
        fi
        ((attempt++))
        sleep 1
    done
    
    return 1
}

#######################################
# Get container health status
# Returns: health status string
#######################################
agents2::get_health_status() {
    if agents2::container_exists; then
        docker container inspect "$AGENTS2_CONTAINER_NAME" --format='{{.State.Health.Status}}' 2>/dev/null || echo "unknown"
    else
        echo "not_exists"
    fi
}

#######################################
# Create Agent S2 data directories
# Returns: 0 if successful, 1 if failed
#######################################
agents2::create_directories() {
    log::info "$MSG_CREATING_DIRS"
    
    # Create main data directory
    if ! mkdir -p "$AGENTS2_DATA_DIR"; then
        log::error "$MSG_CREATE_DIRS_FAILED: $AGENTS2_DATA_DIR"
        return 1
    fi
    
    # Create subdirectories
    local dirs=(
        "$AGENTS2_DATA_DIR/config"
        "$AGENTS2_DATA_DIR/logs"
        "$AGENTS2_DATA_DIR/cache"
        "$AGENTS2_DATA_DIR/models"
    )
    
    for dir in "${dirs[@]}"; do
        if ! mkdir -p "$dir"; then
            log::error "$MSG_CREATE_DIRS_FAILED: $dir"
            return 1
        fi
    done
    
    # Set permissions
    chmod -R 755 "$AGENTS2_DATA_DIR"
    
    log::success "$MSG_DIRECTORIES_CREATED"
    return 0
}

#######################################
# Save environment configuration
# Returns: 0 if successful, 1 if failed
#######################################
agents2::save_env_config() {
    local env_file="$AGENTS2_DATA_DIR/config/.env"
    
    cat > "$env_file" << EOF
# Agent S2 Configuration
# Generated on $(date)

# API Configuration
AGENTS2_API_HOST=0.0.0.0
AGENTS2_API_PORT=4113

# LLM Configuration  
AGENTS2_LLM_PROVIDER=$AGENTS2_LLM_PROVIDER
AGENTS2_LLM_MODEL=$AGENTS2_LLM_MODEL
AGENTS2_API_KEY=$AGENTS2_API_KEY

# Display Configuration
DISPLAY=$AGENTS2_DISPLAY
AGENTS2_SCREEN_RESOLUTION=$AGENTS2_SCREEN_RESOLUTION

# VNC Configuration
AGENTS2_VNC_PASSWORD=$AGENTS2_VNC_PASSWORD

# Security
AGENTS2_ENABLE_HOST_DISPLAY=$AGENTS2_ENABLE_HOST_DISPLAY
EOF
    
    chmod 600 "$env_file"
    return 0
}

#######################################
# Wait for service to be ready
# Arguments:
#   $1 - max wait time in seconds (optional, default: AGENTS2_STARTUP_MAX_WAIT)
# Returns: 0 if ready, 1 if timeout
#######################################
agents2::wait_for_ready() {
    local max_wait="${1:-$AGENTS2_STARTUP_MAX_WAIT}"
    local elapsed=0
    
    log::info "$MSG_WAITING_READY"
    
    while [[ $elapsed -lt $max_wait ]]; do
        if agents2::is_healthy; then
            return 0
        fi
        
        sleep "$AGENTS2_STARTUP_WAIT_INTERVAL"
        elapsed=$((elapsed + AGENTS2_STARTUP_WAIT_INTERVAL))
        
        # Show progress
        if [[ $((elapsed % 10)) -eq 0 ]]; then
            log::info "Still waiting... (${elapsed}s / ${max_wait}s)"
        fi
    done
    
    return 1
}

#######################################
# Get service information
# Returns: JSON string with service info
#######################################
agents2::get_info() {
    local status="not_installed"
    local health="unknown"
    local api_url="$AGENTS2_BASE_URL"
    local vnc_url="$AGENTS2_VNC_URL"
    
    if agents2::container_exists; then
        if agents2::is_running; then
            status="running"
            health=$(agents2::get_health_status)
        else
            status="stopped"
        fi
    fi
    
    cat << EOF
{
    "status": "$status",
    "health": "$health",
    "api_url": "$api_url",
    "vnc_url": "$vnc_url",
    "container": "$AGENTS2_CONTAINER_NAME",
    "image": "$AGENTS2_IMAGE_NAME",
    "port": $AGENTS2_PORT,
    "vnc_port": $AGENTS2_VNC_PORT
}
EOF
}