#!/usr/bin/env bash
# Generic Initialization Framework
# Provides first-run setup and configuration for all resources

# Source guard to prevent multiple sourcing
[[ -n "${_INIT_FRAMEWORK_SOURCED:-}" ]] && return 0
_INIT_FRAMEWORK_SOURCED=1

# Source required utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/log.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/docker-utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/wait-utils.sh" 2>/dev/null || true

#######################################
# Initialize a resource with standard setup flow
# Args: $1 - init_config (JSON configuration)
# Returns: 0 on success, 1 on failure
#
# Init Config Schema:
# {
#   "resource_name": "name",
#   "container_name": "container-name",
#   "data_dir": "/path/to/data",
#   "port": 1234,
#   "image": "docker-image:tag",
#   "env_vars": {
#     "KEY": "value"
#   },
#   "volumes": [
#     "/host/path:/container/path"
#   ],
#   "networks": ["network-name"],
#   "first_run_check": "function_to_check_if_first_run",
#   "setup_func": "function_to_run_on_first_setup",
#   "wait_for_ready": "function_to_wait_until_ready"
# }
#######################################
init::setup_resource() {
    local init_config="$1"
    
    local resource_name
    local container_name
    local data_dir
    local port
    local image
    local first_run_check
    local setup_func
    local wait_func
    
    resource_name=$(echo "$init_config" | jq -r '.resource_name // empty')
    container_name=$(echo "$init_config" | jq -r '.container_name // empty')
    data_dir=$(echo "$init_config" | jq -r '.data_dir // empty')
    port=$(echo "$init_config" | jq -r '.port // empty')
    image=$(echo "$init_config" | jq -r '.image // empty')
    first_run_check=$(echo "$init_config" | jq -r '.first_run_check // empty')
    setup_func=$(echo "$init_config" | jq -r '.setup_func // empty')
    wait_func=$(echo "$init_config" | jq -r '.wait_for_ready // empty')
    
    log::info "Initializing $resource_name..."
    
    # Create data directory if needed
    if [[ -n "$data_dir" ]] && [[ ! -d "$data_dir" ]]; then
        init::create_data_dir "$data_dir"
    fi
    
    # Check port availability
    if [[ -n "$port" ]] && ! docker::is_port_available "$port"; then
        log::error "Port $port is already in use"
        return 1
    fi
    
    # Create networks if specified
    local networks
    networks=$(echo "$init_config" | jq -r '.networks[]? // empty' 2>/dev/null)
    if [[ -n "$networks" ]]; then
        while IFS= read -r network; do
            [[ -n "$network" ]] && docker::create_network "$network"
        done <<< "$networks"
    fi
    
    # Pull image if needed
    if [[ -n "$image" ]] && ! init::pull_image "$image"; then
        log::error "Failed to pull image: $image"
        return 1
    fi
    
    # Create and start container
    if ! init::create_container "$init_config"; then
        log::error "Failed to create container"
        return 1
    fi
    
    # Wait for service to be ready
    if [[ -n "$wait_func" ]] && command -v "$wait_func" &>/dev/null; then
        log::info "Waiting for $resource_name to be ready..."
        "$wait_func"
    elif [[ -n "$port" ]]; then
        wait::for_port "$port" 60
    fi
    
    # Run first-time setup if needed
    local is_first_run=false
    if [[ -n "$first_run_check" ]] && command -v "$first_run_check" &>/dev/null; then
        "$first_run_check" && is_first_run=true
    fi
    
    if [[ "$is_first_run" == "true" ]] && [[ -n "$setup_func" ]] && command -v "$setup_func" &>/dev/null; then
        log::info "Running first-time setup..."
        "$setup_func"
    fi
    
    log::success "$resource_name initialized successfully"
    return 0
}

#######################################
# Create data directory with proper permissions
# Args: $1 - directory_path
#       $2 - owner (optional, default: 1000:1000)
# Returns: 0 on success, 1 on failure
#######################################
init::create_data_dir() {
    local dir="$1"
    local owner="${2:-1000:1000}"
    
    log::info "Creating data directory: $dir"
    
    if ! mkdir -p "$dir"; then
        log::error "Failed to create directory: $dir"
        return 1
    fi
    
    if [[ -n "$owner" ]] && [[ "$owner" != ":" ]]; then
        chown -R "$owner" "$dir" 2>/dev/null || true
    fi
    
    chmod 755 "$dir" 2>/dev/null || true
    return 0
}

#######################################
# Pull Docker image if not present
# Args: $1 - image_name
# Returns: 0 on success, 1 on failure
#######################################
init::pull_image() {
    local image="$1"
    
    # Check if image exists locally
    if docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${image}$"; then
        log::info "Image already present: $image"
        return 0
    fi
    
    log::info "Pulling image: $image"
    if ! docker pull "$image" 2>/dev/null; then
        log::error "Failed to pull image: $image"
        return 1
    fi
    
    return 0
}

#######################################
# Create Docker container from config
# Args: $1 - init_config (JSON)
# Returns: 0 on success, 1 on failure
#######################################
init::create_container() {
    local config="$1"
    
    local container_name
    local image
    local port
    
    container_name=$(echo "$config" | jq -r '.container_name // empty')
    image=$(echo "$config" | jq -r '.image // empty')
    port=$(echo "$config" | jq -r '.port // empty')
    
    # Check if container already exists
    if docker::container_exists "$container_name"; then
        log::info "Container already exists: $container_name"
        if ! docker::is_running "$container_name"; then
            docker start "$container_name" >/dev/null 2>&1
        fi
        return 0
    fi
    
    # Build docker run command
    local docker_cmd="docker run -d --name $container_name"
    
    # Add port mapping
    if [[ -n "$port" ]]; then
        docker_cmd+=" -p ${port}:${port}"
    fi
    
    # Add environment variables
    local env_vars
    env_vars=$(echo "$config" | jq -r '.env_vars | to_entries[]? | "-e \(.key)=\(.value)"' 2>/dev/null)
    if [[ -n "$env_vars" ]]; then
        while IFS= read -r env; do
            [[ -n "$env" ]] && docker_cmd+=" $env"
        done <<< "$env_vars"
    fi
    
    # Add volumes
    local volumes
    volumes=$(echo "$config" | jq -r '.volumes[]? // empty' 2>/dev/null)
    if [[ -n "$volumes" ]]; then
        while IFS= read -r volume; do
            [[ -n "$volume" ]] && docker_cmd+=" -v $volume"
        done <<< "$volumes"
    fi
    
    # Add networks - check for special "host" network mode
    local networks
    networks=$(echo "$config" | jq -r '.networks[]? // empty' 2>/dev/null)
    local use_host_network=false
    if [[ -n "$networks" ]]; then
        while IFS= read -r network; do
            if [[ "$network" == "host" ]]; then
                use_host_network=true
                docker_cmd+=" --network host"
                # Remove port mapping for host network mode
                docker_cmd="${docker_cmd//-p ${port}:${port}/}"
            elif [[ -n "$network" ]]; then
                docker_cmd+=" --network $network"
            fi
        done <<< "$networks"
    fi
    
    # Add host.docker.internal mapping for Linux (allows containers to reach host services)
    # This enables n8n to connect to services like Ollama running on the host
    # Skip this for host network mode as it's not needed
    if [[ "$use_host_network" != "true" ]]; then
        docker_cmd+=" --add-host host.docker.internal:host-gateway"
    fi
    
    # Add restart policy
    docker_cmd+=" --restart unless-stopped"
    
    # Add image
    docker_cmd+=" $image"
    
    log::info "Creating container: $container_name"
    if ! eval "$docker_cmd" >/dev/null 2>&1; then
        log::error "Failed to create container"
        return 1
    fi
    
    return 0
}

#######################################
# Setup database (generic for SQLite/PostgreSQL)
# Args: $1 - db_type (sqlite|postgres)
#       $2 - db_config (JSON)
# Returns: 0 on success, 1 on failure
#######################################
init::setup_database() {
    local db_type="$1"
    local db_config="$2"
    
    case "$db_type" in
        sqlite)
            init::setup_sqlite "$db_config"
            ;;
        postgres|postgresql)
            init::setup_postgres "$db_config"
            ;;
        *)
            log::error "Unknown database type: $db_type"
            return 1
            ;;
    esac
}

#######################################
# Setup SQLite database
# Args: $1 - config (JSON with path)
# Returns: 0 on success, 1 on failure
#######################################
init::setup_sqlite() {
    local config="$1"
    local db_path
    
    db_path=$(echo "$config" | jq -r '.path // empty')
    
    if [[ -z "$db_path" ]]; then
        log::error "No database path specified"
        return 1
    fi
    
    local db_dir
    db_dir=$(dirname "$db_path")
    
    if [[ ! -d "$db_dir" ]]; then
        mkdir -p "$db_dir" || return 1
    fi
    
    if [[ ! -f "$db_path" ]]; then
        log::info "Creating SQLite database: $db_path"
        touch "$db_path" || return 1
        chmod 644 "$db_path" 2>/dev/null || true
    fi
    
    return 0
}

#######################################
# Setup PostgreSQL connection
# Args: $1 - config (JSON with connection details)
# Returns: 0 on success, 1 on failure
#######################################
init::setup_postgres() {
    local config="$1"
    
    local host
    local port
    local database
    local user
    
    host=$(echo "$config" | jq -r '.host // "localhost"')
    port=$(echo "$config" | jq -r '.port // "5432"')
    database=$(echo "$config" | jq -r '.database // empty')
    user=$(echo "$config" | jq -r '.user // empty')
    
    log::info "Configuring PostgreSQL connection to $host:$port"
    
    # Test connection with pg_isready if available
    if command -v pg_isready &>/dev/null; then
        if ! pg_isready -h "$host" -p "$port" -U "$user" -d "$database" >/dev/null 2>&1; then
            log::warn "PostgreSQL not ready at $host:$port"
            return 1
        fi
    fi
    
    return 0
}