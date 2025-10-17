#!/usr/bin/env bash
# Whisper Common Utility Functions
# Shared utilities used across all modules

# Initialize APP_ROOT if not set
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source messages
source "${var_RESOURCES_DIR}/whisper/config/messages.sh"
messages::export_messages

#######################################
# Check if Docker is installed and configured
# Returns: 0 if installed, 1 otherwise
#######################################
common::check_docker() {
    if ! system::is_command "docker"; then
        log::error "${MSG_DOCKER_NOT_FOUND}"
        log::info "${MSG_DOCKER_INSTALL_HINT}"
        return 1
    fi
    
    # Check if Docker daemon is running
    if ! docker info >/dev/null 2>&1; then
        log::error "${MSG_DOCKER_NOT_RUNNING}"
        log::info "${MSG_DOCKER_START_HINT}"
        return 1
    fi
    
    # Check if user has permissions
    if ! docker ps >/dev/null 2>&1; then
        log::error "${MSG_DOCKER_NO_PERMISSIONS}"
        log::info "${MSG_DOCKER_PERMISSIONS_HINT}"
        log::info "${MSG_DOCKER_LOGOUT_HINT}"
        return 1
    fi
    
    return 0
}

#######################################
# Check if Whisper container exists
# Returns: 0 if exists, 1 otherwise
#######################################
common::container_exists() {
    docker ps -a --format '{{.Names}}' | grep -q "^${WHISPER_CONTAINER_NAME}$"
}

#######################################
# Check if Whisper is running
# Returns: 0 if running, 1 otherwise
#######################################
common::is_running() {
    docker ps --format '{{.Names}}' | grep -q "^${WHISPER_CONTAINER_NAME}$"
}

#######################################
# Check if port is available
# Arguments:
#   $1 - port number
# Returns: 0 if available, 1 if in use
#######################################
common::is_port_available() {
    local port="$1"
    
    if system::is_port_in_use "$port"; then
        log::error "${MSG_PORT_IN_USE}"
        return 1
    fi
    
    return 0
}

#######################################
# Validate model size
# Arguments:
#   $1 - model size
# Returns: 0 if valid, 1 otherwise
#######################################
common::validate_model() {
    local model="$1"
    local valid_models=("tiny" "base" "small" "medium" "large" "large-v2" "large-v3")
    
    for valid_model in "${valid_models[@]}"; do
        if [[ "$model" == "$valid_model" ]]; then
            return 0
        fi
    done
    
    log::error "${MSG_INVALID_MODEL}"
    log::info "Valid models: ${valid_models[*]}"
    return 1
}

#######################################
# Get model size in GB
# Arguments:
#   $1 - model name
# Outputs: model size
#######################################
common::get_model_size() {
    local model="$1"
    
    case "$model" in
        "tiny") echo "$WHISPER_MODEL_SIZE_TINY" ;;
        "base") echo "$WHISPER_MODEL_SIZE_BASE" ;;
        "small") echo "$WHISPER_MODEL_SIZE_SMALL" ;;
        "medium") echo "$WHISPER_MODEL_SIZE_MEDIUM" ;;
        "large"|"large-v2"|"large-v3") echo "$WHISPER_MODEL_SIZE_LARGE" ;;
        *) echo "unknown" ;;
    esac
}

#######################################
# Create Whisper data directories
# Returns: 0 if successful, 1 otherwise
#######################################
whisper::create_directories() {
    log::info "${MSG_CREATING_DIRS}"
    
    if ! mkdir -p "$WHISPER_DATA_DIR" "$WHISPER_MODELS_DIR" "$WHISPER_UPLOADS_DIR"; then
        log::error "${MSG_CREATE_DIRS_FAILED}"
        return 1
    fi
    
    log::debug "${MSG_DIRECTORIES_CREATED}"
    return 0
}

#######################################
# Wait for Whisper to be healthy
# Returns: 0 if healthy, 1 on timeout
#######################################
whisper::wait_for_health() {
    log::info "${MSG_WAITING_STARTUP}"
    
    local max_attempts=$((WHISPER_STARTUP_MAX_WAIT / WHISPER_STARTUP_WAIT_INTERVAL))
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if whisper::is_healthy; then
            log::debug "${MSG_HEALTHY}"
            return 0
        fi
        
        log::debug "Health check attempt $attempt/$max_attempts failed, waiting..."
        sleep "$WHISPER_STARTUP_WAIT_INTERVAL"
        ((attempt++))
    done
    
    log::error "${MSG_STARTUP_TIMEOUT}"
    return 1
}

#######################################
# Check if Whisper API is healthy
# Returns: 0 if healthy, 1 otherwise
#######################################
whisper::is_healthy() {
    local response
    # Try the ASR endpoint with a POST request to check if service is up
    response=$(curl -s -X POST "$WHISPER_BASE_URL/asr" \
        -H "Content-Type: application/json" \
        -d '{"test":"health"}' \
        --max-time "$WHISPER_API_TIMEOUT" 2>/dev/null | grep -c "field required" 2>/dev/null || echo "0")
    
    [[ "$response" == "1" ]]
}

#######################################
# Check if GPU is available
# Returns: 0 if available, 1 otherwise
#######################################
whisper::is_gpu_available() {
    if system::is_command "nvidia-smi"; then
        nvidia-smi >/dev/null 2>&1
        return $?
    fi
    
    return 1
}

#######################################
# Get appropriate Docker image based on GPU availability
# Outputs: Docker image name
#######################################
whisper::get_docker_image() {
    if [[ "$WHISPER_GPU_ENABLED" == "yes" ]] && whisper::is_gpu_available; then
        echo "$WHISPER_IMAGE"
    else
        if [[ "$WHISPER_GPU_ENABLED" == "yes" ]]; then
            log::warn "${MSG_GPU_NOT_AVAILABLE}"
        fi
        echo "$WHISPER_CPU_IMAGE"
    fi
}

#######################################
# Remove Whisper container and cleanup
# Returns: 0 if successful, 1 otherwise
#######################################
whisper::cleanup() {
    local success=true
    
    if common::container_exists; then
        log::info "${MSG_REMOVING_CONTAINER}"
        if ! docker rm -f "$WHISPER_CONTAINER_NAME" >/dev/null 2>&1; then
            log::warn "Failed to remove container"
            success=false
        fi
    fi
    
    $success
}

# Export functions for subshell availability
export -f common::check_docker
export -f common::container_exists
export -f common::is_running
export -f common::is_port_available
export -f common::validate_model
export -f common::get_model_size
export -f whisper::create_directories
export -f whisper::wait_for_health
export -f whisper::is_healthy
export -f whisper::is_gpu_available
export -f whisper::get_docker_image
export -f whisper::cleanup