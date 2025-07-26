#!/usr/bin/env bash
# n8n Common Utility Functions
# Shared utilities used across all modules

#######################################
# Check if Docker is installed
# Returns: 0 if installed, 1 otherwise
#######################################
n8n::check_docker() {
    if ! system::is_command "docker"; then
        log::error "Docker is not installed"
        log::info "Please install Docker first: https://docs.docker.com/get-docker/"
        return 1
    fi
    
    # Check if Docker daemon is running
    if ! docker info >/dev/null 2>&1; then
        log::error "Docker daemon is not running"
        log::info "Start Docker with: sudo systemctl start docker"
        return 1
    fi
    
    # Check if user has permissions
    if ! docker ps >/dev/null 2>&1; then
        log::error "Current user doesn't have Docker permissions"
        log::info "Add user to docker group: sudo usermod -aG docker $USER"
        log::info "Then log out and back in for changes to take effect"
        return 1
    fi
    
    return 0
}

#######################################
# Check if n8n container exists
# Returns: 0 if exists, 1 otherwise
#######################################
n8n::container_exists() {
    docker ps -a --format '{{.Names}}' | grep -q "^${N8N_CONTAINER_NAME}$"
}

#######################################
# Check if n8n is running
# Returns: 0 if running, 1 otherwise
#######################################
n8n::is_running() {
    docker ps --format '{{.Names}}' | grep -q "^${N8N_CONTAINER_NAME}$"
}

#######################################
# Check if n8n API is responsive
# Returns: 0 if responsive, 1 otherwise
#######################################
n8n::is_healthy() {
    # n8n uses /healthz endpoint for health checks
    if system::is_command "curl"; then
        # Try multiple times as n8n takes time to fully initialize
        local attempts=0
        while [ $attempts -lt 5 ]; do
            if curl -f -s --max-time 5 "$N8N_BASE_URL/healthz" >/dev/null 2>&1; then
                return 0
            fi
            attempts=$((attempts + 1))
            sleep 2
        done
    fi
    return 1
}

#######################################
# Generate secure random password
#######################################
n8n::generate_password() {
    # Use multiple sources for randomness
    if system::is_command "openssl"; then
        openssl rand -base64 32 | tr -d "=+/" | cut -c1-16
    elif [[ -r /dev/urandom ]]; then
        tr -dc 'A-Za-z0-9!@#$%^&*' < /dev/urandom | head -c 16
    else
        # Fallback to timestamp-based password
        echo "n8n$(date +%s)$RANDOM" | sha256sum | cut -c1-16
    fi
}

#######################################
# Create n8n data directory
#######################################
n8n::create_directories() {
    log::info "Creating n8n data directory..."
    
    mkdir -p "$N8N_DATA_DIR" || {
        log::error "Failed to create n8n data directory"
        return 1
    }
    
    # Add rollback action
    resources::add_rollback_action \
        "Remove n8n data directory" \
        "rm -rf $N8N_DATA_DIR 2>/dev/null || true" \
        10
    
    log::success "n8n directories created"
    return 0
}

#######################################
# Create Docker network for n8n
#######################################
n8n::create_network() {
    if ! docker network ls | grep -q "$N8N_NETWORK_NAME"; then
        log::info "Creating Docker network for n8n..."
        
        if docker network create "$N8N_NETWORK_NAME" >/dev/null 2>&1; then
            log::success "Docker network created"
            
            # Add rollback action
            resources::add_rollback_action \
                "Remove Docker network" \
                "docker network rm $N8N_NETWORK_NAME 2>/dev/null || true" \
                5
        else
            log::warn "Failed to create Docker network (may already exist)"
        fi
    fi
}

#######################################
# Check if port is available
# Args: port number
# Returns: 0 if available, 1 if in use
#######################################
n8n::is_port_available() {
    local port=$1
    
    if system::is_command "lsof"; then
        ! lsof -i :"$port" >/dev/null 2>&1
    elif system::is_command "netstat"; then
        ! netstat -tln | grep -q ":$port "
    else
        # If we can't check, assume it's available
        return 0
    fi
}

#######################################
# Wait for service to be ready
# Args: max_attempts (optional, default 30)
# Returns: 0 if ready, 1 if timeout
#######################################
n8n::wait_for_ready() {
    local max_attempts=${1:-$N8N_HEALTH_CHECK_MAX_ATTEMPTS}
    local attempt=0
    
    log::info "Waiting for n8n to be ready..."
    
    while [ $attempt -lt $max_attempts ]; do
        if n8n::is_healthy; then
            log::success "n8n is ready!"
            return 0
        fi
        
        attempt=$((attempt + 1))
        echo -n "."
        sleep $N8N_HEALTH_CHECK_INTERVAL
    done
    
    echo
    log::error "n8n failed to become ready after $((max_attempts * N8N_HEALTH_CHECK_INTERVAL)) seconds"
    return 1
}

#######################################
# Get container environment variable
# Args: variable_name
# Returns: variable value or empty string
#######################################
n8n::get_container_env() {
    local var_name=$1
    
    if n8n::is_running; then
        docker exec "$N8N_CONTAINER_NAME" env | grep "^${var_name}=" | cut -d'=' -f2-
    fi
}

#######################################
# Check if basic auth is enabled
# Returns: 0 if enabled, 1 if disabled
#######################################
n8n::is_basic_auth_enabled() {
    local auth_active=$(n8n::get_container_env "N8N_BASIC_AUTH_ACTIVE")
    [[ "$auth_active" == "true" ]]
}

#######################################
# Validate workflow ID format
# Args: workflow_id
# Returns: 0 if valid, 1 if invalid
#######################################
n8n::validate_workflow_id() {
    local workflow_id=$1
    
    # n8n workflow IDs are typically numeric
    if [[ ! "$workflow_id" =~ ^[0-9]+$ ]]; then
        log::error "Invalid workflow ID format: $workflow_id"
        log::info "Workflow IDs should be numeric (e.g., 1, 42, 123)"
        return 1
    fi
    
    return 0
}

#######################################
# Check for required commands
# Returns: 0 if all present, 1 if any missing
#######################################
n8n::check_requirements() {
    local missing=0
    
    # Required commands
    local required_commands=("docker" "curl")
    
    for cmd in "${required_commands[@]}"; do
        if ! system::is_command "$cmd"; then
            log::error "Required command not found: $cmd"
            missing=1
        fi
    done
    
    # Optional but recommended commands
    local optional_commands=("jq" "openssl")
    
    for cmd in "${optional_commands[@]}"; do
        if ! system::is_command "$cmd"; then
            log::warn "Optional command not found: $cmd"
            log::info "Some features may be limited without $cmd"
        fi
    done
    
    return $missing
}