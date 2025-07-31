#!/usr/bin/env bash
# PostgreSQL Common Utilities
# Shared functions used across PostgreSQL management scripts

#######################################
# Check if PostgreSQL instance container exists
# Arguments:
#   $1 - instance name
# Returns: 0 if exists, 1 if not
#######################################
postgres::common::container_exists() {
    local instance_name="${1:-main}"
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    docker ps -a --format "{{.Names}}" | grep -q "^${container_name}$"
}

#######################################
# Show helpful error when instance not found
# Arguments:
#   $1 - instance name that was not found
#######################################
postgres::common::show_instance_not_found_error() {
    local instance_name="$1"
    log::warn "PostgreSQL instance '$instance_name' does not exist"
    local available_instances=($(postgres::common::list_instances 2>/dev/null))
    if [[ ${#available_instances[@]} -gt 0 ]]; then
        log::info "Available instances: $(printf '%s, ' "${available_instances[@]}" | sed 's/, $//')"
    else
        log::info "No instances are currently available"
    fi
    log::info "Use './manage.sh --action list' to see all instances"
}

#######################################
# Check if PostgreSQL instance container is running
# Arguments:
#   $1 - instance name
# Returns: 0 if running, 1 if not
#######################################
postgres::common::is_running() {
    local instance_name="${1:-main}"
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    docker ps --format "{{.Names}}" | grep -q "^${container_name}$"
}

#######################################
# Check if a port is available
# Arguments:
#   $1 - Port number to check
# Returns: 0 if available, 1 if in use
#######################################
postgres::common::is_port_available() {
    local port=$1
    
    # Try multiple methods to ensure accurate port detection
    # Method 1: Use nc (netcat) - most reliable for checking if port accepts connections
    if command -v nc >/dev/null 2>&1; then
        if nc -z localhost ${port} 2>/dev/null; then
            return 1  # Port is in use
        fi
    fi
    
    # Method 2: Check with lsof (may require permissions)
    if command -v lsof >/dev/null 2>&1; then
        if lsof -Pi :${port} -sTCP:LISTEN >/dev/null 2>&1; then
            return 1  # Port is in use
        fi
    fi
    
    # Method 3: Use netstat as fallback
    if command -v netstat >/dev/null 2>&1; then
        if netstat -tuln 2>/dev/null | grep -q ":${port} "; then
            return 1  # Port is in use
        fi
    fi
    
    # Method 4: Try to bind to the port directly (most accurate but requires cleanup)
    if command -v timeout >/dev/null 2>&1; then
        if ! timeout 1 bash -c "exec 2>/dev/null; exec 3<>/dev/tcp/localhost/${port}" 2>/dev/null; then
            return 0  # Port is available (connection failed)
        else
            return 1  # Port is in use (connection succeeded)
        fi
    fi
    
    # If we get here, assume port is available
    return 0
}

#######################################
# Find available port in PostgreSQL range
# Returns: Available port number or empty if none available
#######################################
postgres::common::find_available_port() {
    for port in $(seq $POSTGRES_INSTANCE_PORT_RANGE_START $POSTGRES_INSTANCE_PORT_RANGE_END); do
        if postgres::common::is_port_available "$port"; then
            echo "$port"
            return 0
        fi
    done
    return 1
}

#######################################
# Create required directories for instance
# Arguments:
#   $1 - instance name
#######################################
postgres::common::create_directories() {
    local instance_name="${1:-main}"
    local instance_dir="${POSTGRES_INSTANCES_DIR}/${instance_name}"
    local data_dir="${instance_dir}/data"
    local config_dir="${instance_dir}/config"
    
    log::info "${MSG_CREATING_DIRECTORIES}"
    
    # Create instance directory
    if [[ ! -d "${instance_dir}" ]]; then
        mkdir -p "${instance_dir}"
        log::debug "Created instance directory: ${instance_dir}"
    fi
    
    # Create data directory
    if [[ ! -d "${data_dir}" ]]; then
        mkdir -p "${data_dir}"
        log::debug "Created data directory: ${data_dir}"
    fi
    
    # Create config directory
    if [[ ! -d "${config_dir}" ]]; then
        mkdir -p "${config_dir}"
        log::debug "Created config directory: ${config_dir}"
    fi
    
    # Set appropriate permissions
    chmod 700 "${data_dir}" "${config_dir}"
}

#######################################
# Check available disk space
# Returns: 0 if sufficient, 1 if not
#######################################
postgres::common::check_disk_space() {
    local instances_dir="${POSTGRES_INSTANCES_DIR}"
    local parent_dir=$(dirname "$instances_dir")
    
    # Create parent directory if it doesn't exist
    mkdir -p "$parent_dir"
    
    # Get available space in GB
    local available_gb=$(df -BG "$parent_dir" | awk 'NR==2 {print $4}' | sed 's/G//')
    local min_space_gb=2  # Minimum 2GB required
    
    if [[ $available_gb -lt $min_space_gb ]]; then
        log::error "${MSG_INSUFFICIENT_DISK}: ${available_gb}GB available, ${min_space_gb}GB required"
        return 1
    fi
    
    return 0
}

#######################################
# Generate secure password
# Returns: Secure random password
#######################################
postgres::common::generate_password() {
    # Try openssl first
    if command -v openssl >/dev/null 2>&1; then
        openssl rand -base64 32 | tr -d "=+/" | cut -c1-25
    # Fallback to /dev/urandom
    elif [[ -c /dev/urandom ]]; then
        tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 25
    else
        # Last resort - use date and random
        echo "$(date +%s)$(( RANDOM * RANDOM ))" | sha256sum | cut -c1-25
    fi
}

#######################################
# Wait for PostgreSQL instance to be ready
# Arguments:
#   $1 - instance name
#   $2 - timeout in seconds (default: 30)
# Returns: 0 if ready, 1 if timeout
#######################################
postgres::common::wait_for_ready() {
    local instance_name="${1:-main}"
    local timeout="${2:-30}"
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    
    log::info "${MSG_WAITING_STARTUP}"
    
    local elapsed=0
    while [[ $elapsed -lt $timeout ]]; do
        # Check if container is still running
        if ! postgres::common::is_running "$instance_name"; then
            log::error "Container stopped unexpectedly"
            return 1
        fi
        
        # Try to connect to PostgreSQL
        if docker exec "$container_name" pg_isready -U "${POSTGRES_DEFAULT_USER}" >/dev/null 2>&1; then
            log::debug "PostgreSQL instance $instance_name is ready"
            return 0
        fi
        
        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    
    echo
    log::error "PostgreSQL instance $instance_name failed to start within ${timeout}s"
    return 1
}

#######################################
# Perform health check on PostgreSQL instance
# Arguments:
#   $1 - instance name
# Returns: 0 if healthy, 1 if not
#######################################
postgres::common::health_check() {
    local instance_name="${1:-main}"
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    
    # Check if container is running
    if ! postgres::common::is_running "$instance_name"; then
        return 1
    fi
    
    # Check if PostgreSQL is accepting connections (with timeout)
    if timeout 5 docker exec "$container_name" pg_isready -U "${POSTGRES_DEFAULT_USER}" >/dev/null 2>&1; then
        return 0
    fi
    
    return 1
}

#######################################
# Get instance configuration
# Arguments:
#   $1 - instance name
#   $2 - configuration key
# Returns: Configuration value
#######################################
postgres::common::get_instance_config() {
    local instance_name="${1:-main}"
    local config_key="$2"
    local config_file="${POSTGRES_INSTANCES_DIR}/${instance_name}/config/instance.conf"
    
    if [[ -f "$config_file" ]]; then
        grep "^${config_key}=" "$config_file" 2>/dev/null | cut -d'=' -f2- | tr -d '"'
    fi
}

#######################################
# Set instance configuration
# Arguments:
#   $1 - instance name
#   $2 - configuration key
#   $3 - configuration value
#######################################
postgres::common::set_instance_config() {
    local instance_name="${1:-main}"
    local config_key="$2"
    local config_value="$3"
    local config_file="${POSTGRES_INSTANCES_DIR}/${instance_name}/config/instance.conf"
    local config_dir="$(dirname "$config_file")"
    
    # Create config directory if it doesn't exist
    mkdir -p "$config_dir"
    
    # Remove existing key if present
    if [[ -f "$config_file" ]]; then
        grep -v "^${config_key}=" "$config_file" > "${config_file}.tmp" 2>/dev/null || true
        mv "${config_file}.tmp" "$config_file"
    fi
    
    # Add new key-value pair
    echo "${config_key}=\"${config_value}\"" >> "$config_file"
}

#######################################
# Get all PostgreSQL instances
# Returns: List of instance names
#######################################
postgres::common::list_instances() {
    if [[ -d "$POSTGRES_INSTANCES_DIR" ]]; then
        find "$POSTGRES_INSTANCES_DIR" -maxdepth 1 -type d -not -path "$POSTGRES_INSTANCES_DIR" -printf '%f\n' 2>/dev/null | sort
    fi
}

#######################################
# Count running instances
# Returns: Number of running instances
#######################################
postgres::common::count_running_instances() {
    local count=0
    local instances=($(postgres::common::list_instances))
    
    for instance in "${instances[@]}"; do
        if postgres::common::is_running "$instance"; then
            ((count++))
        fi
    done
    
    echo "$count"
}