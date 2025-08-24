#!/usr/bin/env bash
# QuestDB Docker Management - Ultra-Simplified
# Uses docker-resource-utils.sh for minimal boilerplate

# Source var.sh to get proper directory variables
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
QUESTDB_LIB_DIR="${APP_ROOT}/resources/questdb/lib"
# shellcheck disable=SC1091
source "${QUESTDB_LIB_DIR}/../../../../lib/utils/var.sh"

# Source shared libraries
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-resource-utils.sh"
# shellcheck disable=SC1091
source "${QUESTDB_LIB_DIR}/common.sh"

# Check if QuestDB container exists
questdb::docker::exists() {
    docker ps -a --format "{{.Names}}" | grep -q "^${QUESTDB_CONTAINER_NAME}$" 2>/dev/null
}

# Check if QuestDB container is running
questdb::docker::is_running() {
    docker ps --format "{{.Names}}" | grep -q "^${QUESTDB_CONTAINER_NAME}$" 2>/dev/null
}

# Create and start QuestDB container
questdb::docker::create_container() {
    # Ensure directories exist
    questdb::common::create_dirs || return 1
    
    # Pull image if needed
    log::info "Pulling QuestDB image..."
    docker::pull_image "$QUESTDB_IMAGE"
    
    log::info "Creating QuestDB time-series database container..."
    
    # Prepare port mappings for triple-port setup
    # Map external ports to internal QuestDB ports (HTTP:9000, PG:8812, ILP:9009)
    local port_mappings="${QUESTDB_HTTP_PORT}:9000 ${QUESTDB_PG_PORT}:8812 ${QUESTDB_ILP_PORT}:9009"
    
    # Prepare volumes
    local volumes="${QUESTDB_DATA_DIR}:/var/lib/questdb ${QUESTDB_CONFIG_DIR}:/questdb/conf ${QUESTDB_LOG_DIR}:/var/log/questdb"
    
    # Prepare environment variables
    local env_vars=(
        "QDB_PG_USER=${QUESTDB_PG_USER}"
        "QDB_PG_PASSWORD=${QUESTDB_PG_PASSWORD}"
    )
    
    # Prepare Docker options (ulimit for performance)
    local docker_opts=(
        "--ulimit" "nofile=65536:65536"
    )
    
    # Use advanced service creation
    if docker_resource::create_service_advanced \
        "$QUESTDB_CONTAINER_NAME" \
        "$QUESTDB_IMAGE" \
        "$port_mappings" \
        "$QUESTDB_NETWORK_NAME" \
        "$volumes" \
        "env_vars" \
        "docker_opts" \
        "curl -f 'http://localhost:9000/exec?query=SELECT%201&fmt=json' || exit 1" \
        ""; then
        
        # Wait for container to be ready
        if questdb::common::wait_for_ready; then
            log::success "QuestDB time-series database is ready"
            questdb::docker::show_connection_info
            return 0
        else
            log::error "QuestDB failed to become ready"
            return 1
        fi
    else
        log::error "Failed to create QuestDB container"
        return 1
    fi
}

# Start QuestDB container
questdb::docker::start() {
    if ! docker::container_exists "$QUESTDB_CONTAINER_NAME"; then
        log::error "QuestDB container does not exist. Run install first."
        return 1
    fi
    
    log::info "Starting QuestDB container..."
    
    if docker::start_container "$QUESTDB_CONTAINER_NAME"; then
        if questdb::common::wait_for_ready; then
            log::success "QuestDB time-series database is running"
            questdb::docker::show_connection_info
            return 0
        else
            log::error "QuestDB failed to start properly"
            return 1
        fi
    else
        log::error "Failed to start QuestDB container"
        return 1
    fi
}

# Stop QuestDB container
questdb::docker::stop() {
    if ! docker::container_exists "$QUESTDB_CONTAINER_NAME"; then
        log::warn "QuestDB container does not exist"
        return 0
    fi
    
    log::info "Stopping QuestDB container..."
    
    if docker::stop_container "$QUESTDB_CONTAINER_NAME"; then
        log::success "QuestDB stopped successfully"
        return 0
    else
        log::error "Failed to stop QuestDB"
        return 1
    fi
}

# Restart QuestDB container
questdb::docker::restart() {
    log::info "Restarting QuestDB container..."
    
    if docker::restart_container "$QUESTDB_CONTAINER_NAME"; then
        if questdb::common::wait_for_ready; then
            log::success "QuestDB restarted successfully"
            questdb::docker::show_connection_info
            return 0
        else
            log::error "QuestDB failed to restart properly"
            return 1
        fi
    else
        log::error "Failed to restart QuestDB"
        return 1
    fi
}

# Remove QuestDB container
# Arguments: $1 - remove data (yes/no)
questdb::docker::remove_container() {
    local remove_data="${1:-no}"
    
    # Stop and remove container
    docker::remove_container "$QUESTDB_CONTAINER_NAME" "true"
    
    # Remove data if requested
    if [[ "$remove_data" == "yes" ]]; then
        docker_resource::remove_data "QuestDB" "${QUESTDB_DATA_DIR} ${QUESTDB_CONFIG_DIR} ${QUESTDB_LOG_DIR}" "yes"
    fi
    
    return 0
}

#######################################
# Execute SQL query using PostgreSQL wire protocol
# Arguments: $@ - SQL query
# Returns: Query exit code
#######################################
questdb::docker::exec_sql() {
    local query="$*"
    docker_resource::exec "$QUESTDB_CONTAINER_NAME" psql -h localhost -p 8812 -U "${QUESTDB_PG_USER}" -d qdb -c "$query"
}


# Show connection information
questdb::docker::show_connection_info() {
    local additional_info=(
        "PostgreSQL Wire: ${QUESTDB_PG_URL}"
        "InfluxDB Line: localhost:${QUESTDB_ILP_PORT}"
        "Management UI: ${QUESTDB_BASE_URL} (web console)"
    )
    
    if [[ "${QUESTDB_HTTP_SECURITY_READONLY}" == "true" ]]; then
        additional_info+=("⚠️  HTTP API is in read-only mode")
    fi
    
    docker_resource::show_connection_info "QuestDB Time-Series Database" "${QUESTDB_BASE_URL}" "${additional_info[@]}"
}

#######################################
# Create client-specific QuestDB instance
# Arguments: $1 - client ID
# Returns: 0 on success, 1 on failure
#######################################
questdb::docker::create_client_instance() {
    local client_id="$1"
    
    if [[ -z "$client_id" ]]; then
        log::error "Missing required parameter: client_id"
        return 1
    fi
    
    log::info "Creating QuestDB client instance for: ${client_id}"
    
    local project_config_dir
    project_config_dir="$(secrets::get_project_config_dir)"
    
    # Allocate three ports for QuestDB atomically
    local ports_allocated
    ports_allocated=$(docker_resource::allocate_port_range 3 9020 9090)
    
    if [[ -z "$ports_allocated" ]]; then
        log::error "Could not allocate required ports for QuestDB client"
        return 1
    fi
    
    # Parse allocated ports
    local port_array=($ports_allocated)
    local client_http_port="${port_array[0]}"
    local client_pg_port="${port_array[1]}"
    local client_ilp_port="${port_array[2]}"
    
    # Create client directories
    local client_base_dir="${project_config_dir}/clients/${client_id}/questdb"
    mkdir -p "${client_base_dir}"/{data,config,logs}
    
    local client_container="questdb-client-${client_id}"
    local client_network="vrooli-${client_id}-network"
    
    # Prepare configuration for multi-port client
    local port_mappings="${client_http_port}:9000 ${client_pg_port}:8812 ${client_ilp_port}:9009"
    local volumes="${client_base_dir}/data:/var/lib/questdb ${client_base_dir}/config:/questdb/conf ${client_base_dir}/logs:/var/log/questdb"
    
    # Environment variables for client
    local client_env_vars=(
        "QDB_PG_USER=admin"
        "QDB_PG_PASSWORD=quest"
    )
    
    # Docker options for client
    local client_docker_opts=(
        "--ulimit" "nofile=65536:65536"
    )
    
    # Create container using advanced function
    if docker_resource::create_service_advanced \
        "$client_container" \
        "$QUESTDB_IMAGE" \
        "$port_mappings" \
        "$client_network" \
        "$volumes" \
        "client_env_vars" \
        "client_docker_opts" \
        "curl -f 'http://localhost:9000/exec?query=SELECT%201&fmt=json' || exit 1" \
        ""; then
        
        # Generate metadata JSON
        local ports_json='{
    "http": '${client_http_port}',
    "postgresql": '${client_pg_port}',
    "influxdb": '${client_ilp_port}'
  }'
        
        local metadata
        metadata=$(docker_resource::generate_client_metadata \
            "${client_id}" \
            "questdb" \
            "${client_container}" \
            "${client_network}" \
            "${ports_json}" \
            "http://localhost:${client_http_port}")
        
        # Save metadata with extended URLs using jq for safe JSON manipulation
        local client_config_file="${project_config_dir}/clients/${client_id}/questdb.json"
        
        # Create URLs JSON object
        local urls_json=$(jq -n \
            --arg http "http://localhost:${client_http_port}" \
            --arg postgresql "postgresql://admin:quest@localhost:${client_pg_port}/qdb" \
            --arg influxdb "localhost:${client_ilp_port}" \
            '{
                http: $http,
                postgresql: $postgresql,
                influxdb: $influxdb
            }')
        
        # Merge metadata with URLs using jq
        echo "$metadata" | jq --argjson urls "$urls_json" '. + {urls: $urls}' > "$client_config_file"
        
        log::success "QuestDB client instance created successfully"
        log::info "   Client: ${client_id}"
        log::info "   HTTP Port: ${client_http_port} | PostgreSQL Port: ${client_pg_port} | InfluxDB Port: ${client_ilp_port}"
        log::info "   Web Console: http://localhost:${client_http_port}"
        
        return 0
    else
        log::error "Failed to create QuestDB client instance"
        return 1
    fi
}

#######################################
# Destroy client-specific QuestDB instance
# Arguments: $1 - client ID
# Returns: 0 on success, 1 on failure
#######################################
questdb::docker::destroy_client_instance() {
    local client_id="$1"
    
    if [[ -z "$client_id" ]]; then
        log::error "Missing required parameter: client_id"
        return 1
    fi
    
    log::info "Destroying QuestDB client instance: ${client_id}"
    
    local project_config_dir
    project_config_dir="$(secrets::get_project_config_dir)"
    
    docker_resource::remove_client_instance \
        "questdb" \
        "$client_id" \
        "$project_config_dir" \
        "true"
    
    # Remove client configuration metadata safely
    trash::safe_remove "${project_config_dir}/clients/${client_id}/questdb.json" --no-confirm 2>/dev/null || true
    
    log::success "QuestDB client instance destroyed successfully"
    return 0
}