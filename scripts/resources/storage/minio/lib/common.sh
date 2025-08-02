#!/usr/bin/env bash
# MinIO Common Utilities
# Shared functions used across MinIO management scripts

#######################################
# Check if MinIO container exists
# Returns: 0 if exists, 1 if not
#######################################
minio::common::container_exists() {
    docker ps -a --format "table {{.Names}}" | grep -q "^${MINIO_CONTAINER_NAME}$"
}

#######################################
# Check if MinIO container is running
# Returns: 0 if running, 1 if not
#######################################
minio::common::is_running() {
    docker ps --format "table {{.Names}}" | grep -q "^${MINIO_CONTAINER_NAME}$"
}

#######################################
# Check if a port is available
# Arguments:
#   $1 - Port number to check
# Returns: 0 if available, 1 if in use
#######################################
minio::common::is_port_available() {
    local port=$1
    
    if command -v lsof >/dev/null 2>&1; then
        ! lsof -Pi :${port} -sTCP:LISTEN -t >/dev/null 2>&1
    else
        ! netstat -tuln 2>/dev/null | grep -q ":${port} "
    fi
}

#######################################
# Check if both MinIO ports are available
# Returns: 0 if both available, 1 if any in use
#######################################
minio::common::check_ports() {
    local api_port_available=true
    local console_port_available=true
    
    if ! minio::common::is_port_available "${MINIO_PORT}"; then
        api_port_available=false
        log::warn "Port ${MINIO_PORT} is already in use"
    fi
    
    if ! minio::common::is_port_available "${MINIO_CONSOLE_PORT}"; then
        console_port_available=false
        log::warn "Port ${MINIO_CONSOLE_PORT} is already in use"
    fi
    
    if [[ "$api_port_available" == "false" || "$console_port_available" == "false" ]]; then
        log::info "${MSG_HELP_PORT_CONFLICT}"
        return 1
    fi
    
    return 0
}

#######################################
# Create required directories
#######################################
minio::common::create_directories() {
    log::info "${MSG_CREATING_DIRECTORIES}"
    
    # Create data directory
    if [[ ! -d "${MINIO_DATA_DIR}" ]]; then
        mkdir -p "${MINIO_DATA_DIR}"
        log::debug "Created data directory: ${MINIO_DATA_DIR}"
    fi
    
    # Create config directory
    if [[ ! -d "${MINIO_CONFIG_DIR}" ]]; then
        mkdir -p "${MINIO_CONFIG_DIR}"
        log::debug "Created config directory: ${MINIO_CONFIG_DIR}"
    fi
    
    # Fix Docker volume permissions if setup was run with sudo
    docker::fix_volume_permissions "${MINIO_DATA_DIR}" 2>/dev/null || {
        log::debug "Could not fix Docker volume permissions for ${MINIO_DATA_DIR}, continuing..."
    }
    docker::fix_volume_permissions "${MINIO_CONFIG_DIR}" 2>/dev/null || {
        log::debug "Could not fix Docker volume permissions for ${MINIO_CONFIG_DIR}, continuing..."
    }
    
    # Set appropriate permissions
    chmod 700 "${MINIO_DATA_DIR}" "${MINIO_CONFIG_DIR}"
}

#######################################
# Check available disk space
# Returns: 0 if sufficient, 1 if not
#######################################
minio::common::check_disk_space() {
    local data_dir="${MINIO_DATA_DIR}"
    local parent_dir=$(dirname "$data_dir")
    
    # Create parent directory if it doesn't exist
    mkdir -p "$parent_dir"
    
    # Get available space in GB
    local available_gb=$(df -BG "$parent_dir" | awk 'NR==2 {print $4}' | sed 's/G//')
    
    if [[ $available_gb -lt $MINIO_MIN_DISK_SPACE_GB ]]; then
        log::error "${MSG_INSUFFICIENT_DISK}: ${available_gb}GB available, ${MINIO_MIN_DISK_SPACE_GB}GB required"
        return 1
    fi
    
    if [[ $available_gb -lt 10 ]]; then
        log::warn "${MSG_LOW_DISK_SPACE}: ${available_gb}GB available"
    fi
    
    log::debug "Disk space check passed: ${available_gb}GB available"
    return 0
}

#######################################
# Generate secure credentials
# Sets MINIO_ROOT_USER and MINIO_ROOT_PASSWORD if not already set
#######################################
minio::common::generate_credentials() {
    local creds_file="${MINIO_CONFIG_DIR}/credentials"
    
    # If credentials file exists and environment variables not set, load from file
    if [[ -f "$creds_file" && -z "${MINIO_CUSTOM_ROOT_USER:-}" && -z "${MINIO_CUSTOM_ROOT_PASSWORD:-}" ]]; then
        log::debug "Loading existing credentials from file"
        source "$creds_file"
        export MINIO_ROOT_USER MINIO_ROOT_PASSWORD
        return 0
    fi
    
    # Use custom credentials if provided
    if [[ -n "${MINIO_CUSTOM_ROOT_USER:-}" || -n "${MINIO_CUSTOM_ROOT_PASSWORD:-}" ]]; then
        log::debug "Using custom credentials"
        return 0
    fi
    
    # Generate new secure credentials
    if [[ "$MINIO_ROOT_USER" == "minioadmin" && "$MINIO_ROOT_PASSWORD" == "minioadmin" ]]; then
        log::warn "${MSG_USING_DEFAULT_CREDS}"
        
        # Generate secure username (8-16 alphanumeric characters)
        local new_user="minio$(openssl rand -hex 4)"
        
        # Generate secure password (16-32 characters with special chars)
        local new_password=$(openssl rand -base64 24 | tr -d "=+/" | cut -c1-20)
        
        # Override the defaults
        MINIO_ROOT_USER="$new_user"
        MINIO_ROOT_PASSWORD="$new_password"
        export MINIO_ROOT_USER MINIO_ROOT_PASSWORD
        
        # Save to file with restricted permissions
        cat > "$creds_file" << EOF
# MinIO Credentials - Generated on $(date)
# Keep this file secure!
export MINIO_ROOT_USER="${MINIO_ROOT_USER}"
export MINIO_ROOT_PASSWORD="${MINIO_ROOT_PASSWORD}"
EOF
        chmod 600 "$creds_file"
        
        log::success "${MSG_CREDENTIALS_GENERATED}"
        log::info "Credentials saved to: $creds_file"
    fi
}

#######################################
# Wait for MinIO to be ready
# Arguments:
#   $1 - Max wait time in seconds (optional, default: MINIO_STARTUP_MAX_WAIT)
# Returns: 0 if ready, 1 if timeout
#######################################
minio::common::wait_for_ready() {
    local max_wait=${1:-$MINIO_STARTUP_MAX_WAIT}
    local waited=0
    
    log::info "${MSG_WAITING_STARTUP}"
    
    while [[ $waited -lt $max_wait ]]; do
        if minio::common::health_check; then
            log::debug "MinIO is ready after ${waited} seconds"
            return 0
        fi
        
        sleep $MINIO_STARTUP_WAIT_INTERVAL
        waited=$((waited + MINIO_STARTUP_WAIT_INTERVAL))
        
        # Show progress
        if [[ $((waited % 10)) -eq 0 ]]; then
            log::debug "Still waiting... (${waited}/${max_wait}s)"
        fi
    done
    
    log::error "MinIO failed to start within ${max_wait} seconds"
    return 1
}

#######################################
# Perform health check on MinIO
# Returns: 0 if healthy, 1 if not
#######################################
minio::common::health_check() {
    local health_endpoint="${MINIO_BASE_URL}/minio/health/live"
    
    # First check if container is running
    if ! minio::common::is_running; then
        return 1
    fi
    
    # Then check API health
    if curl -sf "${health_endpoint}" \
        --max-time "${MINIO_API_TIMEOUT}" \
        >/dev/null 2>&1; then
        return 0
    fi
    
    return 1
}

#######################################
# Get MinIO container logs
# Arguments:
#   $1 - Number of lines to show (optional, default: 50)
#######################################
minio::common::show_logs() {
    local lines=${1:-50}
    
    if minio::common::container_exists; then
        docker logs --tail "$lines" "$MINIO_CONTAINER_NAME"
    else
        log::warn "MinIO container does not exist"
    fi
}

#######################################
# Clean up MinIO resources
# Used during uninstallation
#######################################
minio::common::cleanup() {
    log::info "Cleaning up MinIO resources..."
    
    # Remove container if exists
    if minio::common::container_exists; then
        docker rm -f "$MINIO_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    # Remove network if exists and no other containers are using it
    if docker network ls | grep -q "$MINIO_NETWORK_NAME"; then
        docker network rm "$MINIO_NETWORK_NAME" >/dev/null 2>&1 || true
    fi
    
    # Note: We don't remove data directories by default to preserve user data
    log::info "Data directories preserved at: ${MINIO_DATA_DIR}"
}

#######################################
# Format bytes to human readable
# Arguments:
#   $1 - Bytes
# Output: Human readable size
#######################################
minio::common::format_bytes() {
    local bytes=$1
    local units=("B" "KB" "MB" "GB" "TB")
    local unit=0
    
    while [[ $bytes -gt 1024 && $unit -lt 4 ]]; do
        bytes=$((bytes / 1024))
        unit=$((unit + 1))
    done
    
    echo "${bytes}${units[$unit]}"
}