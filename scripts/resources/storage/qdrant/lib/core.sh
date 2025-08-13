#!/usr/bin/env bash
# Qdrant Core Functions - Consolidated Qdrant-specific logic
# Combines docker.sh, api.sh, and status.sh functionality
# All generic operations delegated to shared frameworks

# Source shared libraries
QDRANT_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-utils.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/http-utils.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/status-engine.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/health-framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/backup-framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/init-framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/wait-utils.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"

#######################################
# Qdrant Configuration Constants
# These are set in config/defaults.sh as readonly
# Only set non-readonly variables here
#######################################
: "${QDRANT_LOG_LEVEL:=INFO}"
: "${QDRANT_WEB_UI_ENABLED:=true}"

#######################################
# Get Qdrant initialization configuration
# Returns: JSON configuration for init framework
#######################################
qdrant::get_init_config() {
    # Build environment variables
    local env_config='{
        "QDRANT__LOG_LEVEL": "'$QDRANT_LOG_LEVEL'",
        "QDRANT__SERVICE__HTTP_PORT": "'$QDRANT_PORT'",
        "QDRANT__SERVICE__GRPC_PORT": "'$QDRANT_GRPC_PORT'",
        "QDRANT__WEB_UI": "'$QDRANT_WEB_UI_ENABLED'",
        "QDRANT__STORAGE__OPTIMIZED_VECTOR_STORAGE": "true"
    }'
    
    # Add basic performance settings (avoiding duplicates)
    env_config=$(echo "$env_config" | jq '. += {
        "QDRANT__STORAGE__OPTIMIZERS__DELETED_THRESHOLD": 0.2,
        "QDRANT__STORAGE__OPTIMIZERS__VACUUM_MIN_VECTOR_NUMBER": 1000,
        "QDRANT__STORAGE__OPTIMIZERS__DEFAULT_SEGMENT_NUMBER": 0
    }')
    
    # Add API key authentication if provided
    if [[ -n "${QDRANT_API_KEY:-}" ]]; then
        env_config=$(echo "$env_config" | jq '. += {
            "QDRANT__SERVICE__API_KEY": "'$QDRANT_API_KEY'"
        }')
    fi
    
    # Add resource limits
    if [[ "$QDRANT_MAX_WORKERS" -gt 0 ]]; then
        env_config=$(echo "$env_config" | jq '. += {
            "QDRANT__STORAGE__OPTIMIZERS__MAX_OPTIMIZATION_THREADS": "'$QDRANT_MAX_WORKERS'"
        }')
    fi
    
    # Build volumes array
    local volumes_array="[\"${QDRANT_DATA_DIR}:/qdrant/storage:z\""
    if [[ -d "$QDRANT_CONFIG_DIR" ]]; then
        volumes_array+=",\"${QDRANT_CONFIG_DIR}:/qdrant/config:ro\""
    fi
    if [[ -d "$QDRANT_SNAPSHOTS_DIR" ]]; then
        volumes_array+=",\"${QDRANT_SNAPSHOTS_DIR}:/qdrant/snapshots:z\""
    fi
    volumes_array+="]"
    
    # Build init config
    local config='{
        "resource_name": "qdrant",
        "container_name": "'$QDRANT_CONTAINER_NAME'",
        "data_dir": "'$QDRANT_DATA_DIR'",
        "port": '$QDRANT_PORT',
        "secondary_ports": ['$QDRANT_GRPC_PORT'],
        "image": "'$QDRANT_IMAGE'",
        "env_vars": '$env_config',
        "volumes": '$volumes_array',
        "networks": ["'$QDRANT_NETWORK_NAME'"],
        "healthcheck": {
            "test": ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:'$QDRANT_PORT'/health"],
            "interval": "30s",
            "timeout": "10s",
            "retries": 3,
            "start_period": "40s"
        },
        "restart_policy": "unless-stopped",
        "first_run_check": "qdrant::is_first_run",
        "setup_func": "qdrant::first_time_setup",
        "wait_for_ready": "qdrant::wait_for_ready"
    }'
    
    echo "$config"
}

#######################################
# Install Qdrant using init framework
# Arguments: None
# Returns: 0 on success, 1 on failure
#######################################
qdrant::install() {
    local init_config
    init_config=$(qdrant::get_init_config)
    
    log::info "Installing Qdrant vector database..."
    
    if ! init::setup_resource "$init_config"; then
        log::error "Failed to install Qdrant"
        return 1
    fi
    
    log::success "Qdrant installation completed successfully!"
    qdrant::display_connection_info
    return 0
}

#######################################
# Check if this is a first-time installation
# Returns: 0 if first run, 1 if already installed
#######################################
qdrant::is_first_run() {
    # Check if data directory exists and is populated
    if [[ -d "$QDRANT_DATA_DIR" ]] && [[ -n "$(ls -A "$QDRANT_DATA_DIR" 2>/dev/null)" ]]; then
        return 1  # Not first run
    fi
    return 0  # First run
}

#######################################
# First-time setup after installation
# Returns: 0 on success, 1 on failure
#######################################
qdrant::first_time_setup() {
    log::info "Performing first-time Qdrant setup..."
    
    # Create necessary directories
    mkdir -p "$QDRANT_DATA_DIR" "$QDRANT_CONFIG_DIR" "$QDRANT_SNAPSHOTS_DIR"
    
    # Wait for Qdrant to be ready
    if ! qdrant::wait_for_ready; then
        log::error "Qdrant failed to start properly"
        return 1
    fi
    
    # Create default collections if configured
    if [[ "${#QDRANT_DEFAULT_COLLECTIONS[@]}" -gt 0 ]]; then
        log::info "Creating default collections..."
        qdrant::create_default_collections
    fi
    
    log::success "First-time setup completed"
    return 0
}

#######################################
# Wait for Qdrant to be ready
# Returns: 0 when ready, 1 on timeout
#######################################
qdrant::wait_for_ready() {
    local max_attempts=$((QDRANT_STARTUP_MAX_WAIT / QDRANT_STARTUP_WAIT_INTERVAL))
    local attempt=0
    
    log::info "Waiting for Qdrant to be ready..."
    
    while [[ $attempt -lt $max_attempts ]]; do
        if qdrant::check_basic_health >/dev/null 2>&1; then
            log::success "Qdrant is ready!"
            return 0
        fi
        
        sleep "$QDRANT_STARTUP_WAIT_INTERVAL"
        ((attempt++))
    done
    
    log::error "Timeout waiting for Qdrant to be ready"
    return 1
}

#######################################
# Create default collections
# Returns: 0 on success, 1 on failure
#######################################
qdrant::create_default_collections() {
    local collection_config
    local collection_name
    local vector_size
    local distance_metric
    
    for collection_config in "${QDRANT_COLLECTION_CONFIGS[@]}"; do
        IFS=':' read -r collection_name vector_size distance_metric <<< "$collection_config"
        
        log::info "Creating collection: $collection_name (size: $vector_size, distance: $distance_metric)"
        
        if ! qdrant::collections::create "$collection_name" "$vector_size" "$distance_metric"; then
            log::warn "Failed to create collection: $collection_name"
        fi
    done
}

#######################################
# Display connection information
# Returns: 0 on success
#######################################
qdrant::display_connection_info() {
    echo
    log::info "🔗 Qdrant Connection Information:"
    echo "   REST API:  $QDRANT_BASE_URL"
    echo "   gRPC API:  $QDRANT_GRPC_URL"
    echo "   Web UI:    $QDRANT_BASE_URL/dashboard"
    if [[ -n "${QDRANT_API_KEY:-}" ]]; then
        echo "   API Key:   **************** (configured)"
    else
        echo "   API Key:   None (open access)"
    fi
    echo "   Container: $QDRANT_CONTAINER_NAME"
    echo "   Data Dir:  $QDRANT_DATA_DIR"
    echo
}

#######################################
# Uninstall Qdrant
# Arguments:
#   $1 - preserve_data (boolean): whether to preserve data
# Returns: 0 on success, 1 on failure
#######################################
qdrant::uninstall() {
    local preserve_data="${1:-true}"
    
    log::info "Uninstalling Qdrant..."
    
    # Stop and remove container
    if docker::container_exists "$QDRANT_CONTAINER_NAME"; then
        docker::stop_container "$QDRANT_CONTAINER_NAME"
        docker::remove_container "$QDRANT_CONTAINER_NAME"
    fi
    
    # Remove network if it exists and no other containers are using it
    if docker::network_exists "$QDRANT_NETWORK_NAME"; then
        docker::remove_network_if_unused "$QDRANT_NETWORK_NAME"
    fi
    
    # Handle data removal
    if [[ "$preserve_data" != "true" ]]; then
        log::info "Removing Qdrant data..."
        rm -rf "$QDRANT_DATA_DIR" "$QDRANT_CONFIG_DIR" "$QDRANT_SNAPSHOTS_DIR"
    else
        log::info "Preserving Qdrant data in $QDRANT_DATA_DIR"
    fi
    
    log::success "Qdrant uninstalled successfully"
    return 0
}

#######################################
# Get Qdrant information
# Returns: 0 on success
#######################################
qdrant::info() {
    if ! qdrant::common::is_running; then
        log::error "Qdrant is not running"
        return 1
    fi
    
    echo "📊 Qdrant Information:"
    echo
    
    # Basic info
    qdrant::display_connection_info
    
    # Version info
    local version_info
    if version_info=$(qdrant::api::get_version 2>/dev/null); then
        echo "   Version:   $version_info"
    fi
    
    # Collections count
    local collections_count
    if collections_count=$(qdrant::collections::count 2>/dev/null); then
        echo "   Collections: $collections_count"
    fi
    
    # Storage usage
    local disk_usage
    if [[ -d "$QDRANT_DATA_DIR" ]]; then
        disk_usage=$(du -sh "$QDRANT_DATA_DIR" 2>/dev/null | cut -f1)
        echo "   Storage:   $disk_usage"
    fi
    
    echo
    return 0
}

#######################################
# Test Qdrant functionality
# Returns: 0 on success, 1 on failure
#######################################
qdrant::test() {
    log::info "Testing Qdrant functionality..."
    
    # Test basic health
    if ! qdrant::check_basic_health; then
        log::error "Basic health check failed"
        return 1
    fi
    
    # Test API functionality
    if ! qdrant::check_api_functionality; then
        log::error "API functionality test failed"
        return 1
    fi
    
    log::success "All Qdrant tests passed!"
    return 0
}

#######################################
# Check basic Qdrant health
# Returns: 0 if healthy, 1 if not
#######################################
qdrant::check_basic_health() {
    if ! docker::is_running "$QDRANT_CONTAINER_NAME"; then
        return 1
    fi
    
    # Check root endpoint (Qdrant doesn't have a /health endpoint)
    # health::check_endpoint expects timeout as second parameter
    local health_url="$QDRANT_BASE_URL/"
    if ! health::check_endpoint "$health_url" "$QDRANT_API_TIMEOUT"; then
        return 1
    fi
    
    return 0
}

#######################################
# Check advanced Qdrant API functionality
# Returns: 0 if functional, 1 if not
#######################################
qdrant::check_api_functionality() {
    local test_collection="test_health_check"
    
    # Test collection operations
    if ! qdrant::collections::create "$test_collection" "128" "Cosine" >/dev/null 2>&1; then
        return 1
    fi
    
    # Test collection info retrieval
    if ! qdrant::collections::info "$test_collection" >/dev/null 2>&1; then
        qdrant::collections::delete "$test_collection" "yes" >/dev/null 2>&1
        return 1
    fi
    
    # Cleanup test collection
    qdrant::collections::delete "$test_collection" "yes" >/dev/null 2>&1
    
    return 0
}

#######################################
# Common function: Check if Qdrant is running
# Returns: 0 if running, 1 if not
#######################################
qdrant::common::is_running() {
    docker::is_running "$QDRANT_CONTAINER_NAME"
}

#######################################
# Common function: Show logs
# Arguments:
#   $1 - lines (optional): number of lines to show
# Returns: 0 on success
#######################################
qdrant::common::show_logs() {
    local lines="${1:-50}"
    
    if ! qdrant::common::is_running; then
        log::error "Qdrant container is not running"
        return 1
    fi
    
    docker logs --tail "$lines" --follow "$QDRANT_CONTAINER_NAME"
}

#######################################
# Inject data into Qdrant from configuration
# Uses the lightweight inject.sh adapter
# Arguments:
#   injection_config - JSON configuration for injection
# Returns: 0 on success, 1 on failure
#######################################
qdrant::inject() {
    local injection_config="${INJECTION_CONFIG:-}"
    
    if [[ -z "$injection_config" ]]; then
        log::error "Missing required --injection-config parameter"
        log::info "Usage: manage.sh --action inject --injection-config '{...}'"
        return 1
    fi
    
    # Execute injection using lightweight adapter
    "${QDRANT_LIB_DIR}/inject.sh" --inject --config "$injection_config"
}

#######################################
# Validate injection configuration
# Validates without performing actual injection
# Arguments:
#   injection_config - JSON configuration to validate
# Returns: 0 if valid, 1 if invalid
#######################################
qdrant::validate_injection() {
    local injection_config="${INJECTION_CONFIG:-}"
    
    if [[ -z "$injection_config" ]]; then
        log::error "Missing required --injection-config parameter"
        log::info "Usage: manage.sh --action validate-injection --injection-config '{...}'"
        return 1
    fi
    
    # Validate using lightweight adapter
    "${QDRANT_LIB_DIR}/inject.sh" --validate --config "$injection_config"
}

#######################################
# DOCKER MANAGEMENT (from docker.sh)
#######################################

qdrant::docker::start() {
    if ! docker::check_daemon; then
        log::error "Docker daemon is not running"
        return 1
    fi
    
    if docker::is_running "$QDRANT_CONTAINER_NAME"; then
        log::info "Qdrant is already running"
        return 0
    fi
    
    log::info "Starting Qdrant container..."
    if docker::start_container "$QDRANT_CONTAINER_NAME"; then
        log::success "Qdrant started successfully"
        return 0
    else
        log::error "Failed to start Qdrant"
        return 1
    fi
}

qdrant::docker::stop() {
    if ! docker::container_exists "$QDRANT_CONTAINER_NAME"; then
        log::info "Qdrant container does not exist"
        return 0
    fi
    
    if ! docker::is_running "$QDRANT_CONTAINER_NAME"; then
        log::info "Qdrant is already stopped"
        return 0
    fi
    
    log::info "Stopping Qdrant container..."
    if docker::stop_container "$QDRANT_CONTAINER_NAME"; then
        log::success "Qdrant stopped successfully"
        return 0
    else
        log::error "Failed to stop Qdrant"
        return 1
    fi
}

qdrant::docker::restart() {
    log::info "Restarting Qdrant..."
    if ! qdrant::docker::stop; then
        return 1
    fi
    sleep 2
    if ! qdrant::docker::start; then
        return 1
    fi
    if ! qdrant::wait_for_ready; then
        log::error "Qdrant failed to start properly after restart"
        return 1
    fi
    log::success "Qdrant restarted successfully"
    return 0
}

qdrant::docker::remove() {
    if ! docker::container_exists "$QDRANT_CONTAINER_NAME"; then
        log::info "Qdrant container does not exist"
        return 0
    fi
    
    if docker::is_running "$QDRANT_CONTAINER_NAME"; then
        qdrant::docker::stop
    fi
    
    log::info "Removing Qdrant container..."
    if docker::remove_container "$QDRANT_CONTAINER_NAME"; then
        log::success "Qdrant container removed successfully"
        return 0
    else
        log::error "Failed to remove Qdrant container"
        return 1
    fi
}

#######################################
# API OPERATIONS (from api.sh)
#######################################

qdrant::api::request() {
    local method="$1"
    local endpoint="$2"
    local body="${3:-}"
    
    local url="${QDRANT_BASE_URL}${endpoint}"
    local headers="Content-Type: application/json"
    
    # Add authentication header if API key is configured
    if [[ -n "${QDRANT_API_KEY:-}" ]]; then
        headers="${headers}\napi-key: ${QDRANT_API_KEY}"
    fi
    
    # Use http-utils framework for the request
    if [[ -n "$body" ]]; then
        http::request "$method" "$url" "$body" "$headers"
    else
        http::request "$method" "$url" "" "$headers"
    fi
}

qdrant::api::health_check() {
    local response
    response=$(qdrant::api::request "GET" "/" 2>/dev/null)
    
    if [[ $? -eq 0 && -n "$response" ]]; then
        # Check if response contains version information
        if echo "$response" | jq -e '.version' >/dev/null 2>&1; then
            return 0
        fi
    fi
    
    return 1
}

qdrant::api::get_version() {
    local response
    response=$(qdrant::api::request "GET" "/" 2>/dev/null)
    
    if [[ $? -eq 0 && -n "$response" ]]; then
        echo "$response" | jq -r '.version // "unknown"' 2>/dev/null || echo "unknown"
        return 0
    else
        echo "unknown"
        return 1
    fi
}

qdrant::api::get_cluster_info() {
    qdrant::api::request "GET" "/cluster"
}

qdrant::api::get_telemetry() {
    qdrant::api::request "GET" "/telemetry"
}

# Additional API functions consolidated from api.sh
qdrant::api::test() {
    local response
    response=$(qdrant::api::request "GET" "/" 2>/dev/null || true)
    
    if echo "$response" | grep -q "qdrant\|version"; then
        log::success "Qdrant API is accessible"
        return 0
    else
        log::error "Qdrant API connection failed"
        return 1
    fi
}

qdrant::api::list_collections() {
    local response
    response=$(qdrant::api::request "GET" "/collections" 2>/dev/null || true)
    
    if [[ -n "$response" ]] && echo "$response" | jq -e '.result' >/dev/null 2>&1; then
        echo "$response"
        return 0
    fi
    
    log::error "Failed to list collections"
    return 1
}

qdrant::api::get_collection() {
    local collection="${1:-}"
    
    if [[ -z "$collection" ]]; then
        log::error "Collection name is required"
        return 1
    fi
    
    local response
    response=$(qdrant::api::request "GET" "/collections/${collection}" 2>/dev/null || true)
    
    if echo "$response" | grep -q '"status":"ok"\|"result"'; then
        echo "$response"
        return 0
    elif echo "$response" | grep -q "not found"; then
        log::error "Collection '$collection' not found"
        return 1
    else
        log::error "Failed to get collection info: $collection"
        return 1
    fi
}

qdrant::api::create_collection() {
    local collection="${1:-}"
    local vector_size="${2:-1536}"
    local distance="${3:-Cosine}"
    
    if [[ -z "$collection" ]]; then
        log::error "Collection name is required"
        return 1
    fi
    
    local config='{"vectors":{"size":'$vector_size',"distance":"'$distance'"}}'
    
    local response
    response=$(qdrant::api::request "PUT" "/collections/${collection}" "$config" 2>/dev/null || true)
    
    if echo "$response" | grep -q '"status":"ok"\|"result"'; then
        echo "$response"
        return 0
    else
        log::error "Failed to create collection: $collection"
        return 1
    fi
}

qdrant::api::delete_collection() {
    local collection="${1:-}"
    
    if [[ -z "$collection" ]]; then
        log::error "Collection name is required"
        return 1
    fi
    
    local response
    response=$(qdrant::api::request "DELETE" "/collections/${collection}" 2>/dev/null || true)
    echo "$response"
    return 0
}

qdrant::api::create_snapshot() {
    local collection="${1:-}"
    
    if [[ -z "$collection" ]]; then
        log::error "Collection name is required"
        return 1
    fi
    
    qdrant::api::request "POST" "/collections/${collection}/snapshots"
}

qdrant::api::list_snapshots() {
    local collection="${1:-}"
    
    if [[ -z "$collection" ]]; then
        log::error "Collection name is required"
        return 1
    fi
    
    qdrant::api::request "GET" "/collections/${collection}/snapshots"
}

#######################################
# ESSENTIAL COMMON FUNCTIONS (from common.sh)
#######################################

qdrant::common::container_exists() {
    docker::container_exists "$QDRANT_CONTAINER_NAME"
}

qdrant::common::is_port_available() {
    local port=$1
    
    if command -v lsof >/dev/null 2>&1; then
        ! lsof -Pi :"${port}" -sTCP:LISTEN -t >/dev/null 2>&1
    else
        ! netstat -tuln 2>/dev/null | grep -q ":${port} "
    fi
}

qdrant::common::check_ports() {
    local rest_port_available=true
    local grpc_port_available=true
    
    if ! qdrant::common::is_port_available "${QDRANT_PORT}"; then
        rest_port_available=false
        log::warn "Port ${QDRANT_PORT} is already in use"
    fi
    
    if ! qdrant::common::is_port_available "${QDRANT_GRPC_PORT}"; then
        grpc_port_available=false
        log::warn "gRPC port ${QDRANT_GRPC_PORT} is already in use"
    fi
    
    if [[ "$rest_port_available" == "true" && "$grpc_port_available" == "true" ]]; then
        return 0
    else
        return 1
    fi
}

qdrant::common::check_disk_space() {
    local data_dir="${QDRANT_DATA_DIR}"
    local min_space_gb="${QDRANT_MIN_DISK_SPACE_GB}"
    
    mkdir -p "$data_dir"
    
    # Check available space
    local available_kb
    available_kb=$(df "$data_dir" | awk 'NR==2 {print $4}')
    local available_gb=$((available_kb / 1024 / 1024))
    
    if [[ $available_gb -lt $min_space_gb ]]; then
        log::warn "Low disk space: ${available_gb}GB available (minimum: ${min_space_gb}GB)"
        return 1
    fi
    
    return 0
}

qdrant::common::cleanup() {
    trash::safe_remove /tmp/qdrant_*.json --temp 2>/dev/null || true
    trash::safe_remove /tmp/qdrant_*.tmp --temp 2>/dev/null || true
}

qdrant::common::get_process_info() {
    if ! qdrant::common::is_running; then
        echo "Qdrant container is not running"
        return 1
    fi
    
    echo "Container Information:"
    docker inspect "${QDRANT_CONTAINER_NAME}" --format='{{json .State}}' | jq '.'
    echo
    echo "Resource Usage:"
    docker stats "${QDRANT_CONTAINER_NAME}" --no-stream --format="table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}"
}

#######################################
# HEALTH FRAMEWORK CONFIGURATION
#######################################

qdrant::get_health_config() {
    echo '{
        "container_name": "'$QDRANT_CONTAINER_NAME'",
        "service_name": "Qdrant",
        "checks": {
            "basic": "qdrant::check_basic_health",
            "advanced": "qdrant::check_api_functionality"
        },
        "endpoints": [
            {
                "name": "Health Endpoint",
                "url": "'$QDRANT_BASE_URL'/",
                "expected_status": 200,
                "timeout": '$QDRANT_API_TIMEOUT'
            },
            {
                "name": "Collections API",
                "url": "'$QDRANT_BASE_URL'/collections",
                "expected_status": 200,
                "timeout": '$QDRANT_API_TIMEOUT'
            }
        ],
        "monitoring": {
            "interval": '$QDRANT_HEALTH_CHECK_INTERVAL',
            "max_attempts": '$QDRANT_HEALTH_CHECK_MAX_ATTEMPTS'
        }
    }'
}

qdrant::health() {
    if ! docker::check_daemon; then
        return 1
    fi
    
    local config
    config=$(qdrant::get_health_config)
    health::tiered_check "$config"
}

qdrant::diagnose() {
    local config
    config=$(qdrant::get_health_config)
    health::diagnose "$config" "qdrant::display_additional_diagnostics"
}

qdrant::display_additional_diagnostics() {
    echo
    echo "🔍 Qdrant-Specific Diagnostics:"
    
    # Check data directory
    if [[ -d "$QDRANT_DATA_DIR" ]]; then
        local storage_size
        storage_size=$(du -sh "$QDRANT_DATA_DIR" 2>/dev/null | cut -f1)
        echo "   Storage Usage: $storage_size"
        
        local available_space
        available_space=$(df -h "$QDRANT_DATA_DIR" 2>/dev/null | awk 'NR==2{print $4}')
        echo "   Available Space: $available_space"
    else
        echo "   ⚠️  Data directory not found: $QDRANT_DATA_DIR"
    fi
    
    # Check configuration
    if [[ -n "${QDRANT_API_KEY:-}" ]]; then
        echo "   Authentication: API Key configured"
    else
        echo "   Authentication: Open access (no API key)"
    fi
    
    # Check ports
    echo "   REST API Port: $QDRANT_PORT"
    echo "   gRPC Port: $QDRANT_GRPC_PORT"
    
    # Check collections if Qdrant is running
    if qdrant::common::is_running; then
        local collections_count
        if collections_count=$(qdrant::collections::count 2>/dev/null); then
            echo "   Collections: $collections_count"
        else
            echo "   Collections: Unable to retrieve count"
        fi
    fi
    
    return 0
}

qdrant::monitor() {
    local interval="${1:-$QDRANT_HEALTH_CHECK_INTERVAL}"
    local config
    config=$(qdrant::get_health_config)
    config=$(echo "$config" | jq '.monitoring.interval = '$interval'')
    health::monitor "$config"
}

#######################################
# STATUS ENGINE CONFIGURATION
#######################################

qdrant::get_status_config() {
    echo '{
        "resource": {
            "name": "qdrant",
            "category": "storage",
            "description": "Qdrant Vector Database",
            "port": '$QDRANT_PORT',
            "container_name": "'$QDRANT_CONTAINER_NAME'",
            "data_dir": "'$QDRANT_DATA_DIR'"
        },
        "endpoints": {
            "ui": "'$QDRANT_BASE_URL'",
            "api": "'$QDRANT_BASE_URL'/collections",
            "health": "'$QDRANT_BASE_URL'/"
        },
        "additional_endpoints": [
            {
                "name": "REST API",
                "url": "'$QDRANT_BASE_URL'",
                "port": '$QDRANT_PORT'
            },
            {
                "name": "gRPC API", 
                "url": "'$QDRANT_GRPC_URL'",
                "port": '$QDRANT_GRPC_PORT'
            },
            {
                "name": "Web Dashboard",
                "url": "'$QDRANT_BASE_URL'/dashboard"
            }
        ],
        "health_tiers": {
            "healthy": "All collections accessible, API responsive",
            "degraded": "Some collections may be unavailable",
            "unhealthy": "Qdrant service not responding"
        }
    }'
}

qdrant::status::check() {
    local detailed="${1:-false}"
    
    if ! docker::check_daemon; then
        return 1
    fi
    
    local config
    config=$(qdrant::get_status_config)
    status::display_unified_status "$config" "qdrant::display_additional_info" "$detailed"
}

qdrant::display_additional_info() {
    if ! qdrant::common::is_running; then
        return 0
    fi
    
    echo
    echo "📊 Qdrant Statistics:"
    
    # Collections information
    local collections_count
    if collections_count=$(qdrant::collections::count 2>/dev/null); then
        echo "   Collections: $collections_count"
        
        # Show collection details if there are collections
        if [[ "$collections_count" -gt 0 ]]; then
            echo "   Collection Details:"
            qdrant::collections::list_brief 2>/dev/null | while read -r line; do
                echo "     • $line"
            done
        fi
    else
        echo "   Collections: Unable to retrieve"
    fi
    
    # Storage information
    if [[ -d "$QDRANT_DATA_DIR" ]]; then
        local storage_size
        storage_size=$(du -sh "$QDRANT_DATA_DIR" 2>/dev/null | cut -f1)
        echo "   Storage Used: $storage_size"
        
        local available_space
        available_space=$(df -h "$QDRANT_DATA_DIR" 2>/dev/null | awk 'NR==2{print $4}')
        echo "   Available Space: $available_space"
    fi
    
    # API information
    if [[ -n "${QDRANT_API_KEY:-}" ]]; then
        echo "   Authentication: API Key configured"
    else
        echo "   Authentication: Open access"
    fi
    
    # Performance metrics
    local cluster_info
    if cluster_info=$(qdrant::api::request "GET" "/cluster" 2>/dev/null); then
        local peer_count
        peer_count=$(echo "$cluster_info" | jq -r '.result.peers | length' 2>/dev/null)
        if [[ "$peer_count" =~ ^[0-9]+$ ]]; then
            echo "   Cluster Peers: $peer_count"
        fi
    fi
    
    echo
    return 0
}

#######################################
# BACKUP AND RECOVERY (from docker.sh)
#######################################

qdrant::create_backup() {
    local label="${1:-auto}"
    
    if ! qdrant::common::is_running; then
        log::error "Qdrant must be running to create backup"
        return 1
    fi
    
    log::info "Creating Qdrant backup..."
    
    local temp_dir
    temp_dir=$(mktemp -d)
    
    # Create snapshots directory in temp
    mkdir -p "$temp_dir/snapshots"
    mkdir -p "$temp_dir/config"
    mkdir -p "$temp_dir/metadata"
    
    # Backup all collections as snapshots
    if ! qdrant::backup_collections_data "$temp_dir/snapshots"; then
        log::error "Failed to backup collection data"
        rm -rf "$temp_dir"
        return 1
    fi
    
    # Backup configuration files
    if [[ -d "$QDRANT_CONFIG_DIR" ]]; then
        cp -r "$QDRANT_CONFIG_DIR"/* "$temp_dir/config/" 2>/dev/null || true
    fi
    
    # Create backup metadata
    qdrant::create_backup_metadata "$temp_dir/metadata"
    
    # Use backup framework to store
    local backup_path
    if backup_path=$(backup::store "qdrant" "$temp_dir" "$label"); then
        log::success "Backup created: $(basename "$backup_path")"
        rm -rf "$temp_dir"
        return 0
    else
        log::error "Failed to store backup"
        rm -rf "$temp_dir"
        return 1
    fi
}

qdrant::backup_collections_data() {
    local target_dir="$1"
    local success=true
    
    # Get list of collections
    local collections
    if ! collections=$(qdrant::collections::list_simple 2>/dev/null); then
        log::warn "Unable to retrieve collections list"
        return 0
    fi
    
    # Create snapshot for each collection
    while IFS= read -r collection; do
        if [[ -n "$collection" ]]; then
            log::info "Creating snapshot for collection: $collection"
            
            if qdrant::snapshots::create_collection_snapshot "$collection" "$target_dir"; then
                log::debug "Snapshot created for $collection"
            else
                log::warn "Failed to create snapshot for $collection"
                success=false
            fi
        fi
    done <<< "$collections"
    
    if [[ "$success" == "true" ]]; then
        return 0
    else
        return 1
    fi
}

qdrant::create_backup_metadata() {
    local metadata_dir="$1"
    
    # Create backup info file
    cat > "$metadata_dir/backup_info.json" << EOF
{
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "qdrant_version": "$(qdrant::api::get_version 2>/dev/null || echo 'unknown')",
    "container_name": "$QDRANT_CONTAINER_NAME",
    "collections_count": $(qdrant::collections::count 2>/dev/null || echo 0),
    "backup_type": "full",
    "data_dir": "$QDRANT_DATA_DIR",
    "port": $QDRANT_PORT,
    "grpc_port": $QDRANT_GRPC_PORT
}
EOF
    
    # Create collections manifest
    local collections_list
    if collections_list=$(qdrant::collections::list 2>/dev/null); then
        echo "$collections_list" > "$metadata_dir/collections.txt"
    fi
    
    # Create configuration summary
    cat > "$metadata_dir/config_summary.txt" << EOF
Qdrant Configuration Summary
============================
Container: $QDRANT_CONTAINER_NAME
REST Port: $QDRANT_PORT
gRPC Port: $QDRANT_GRPC_PORT
Data Directory: $QDRANT_DATA_DIR
Image: $QDRANT_IMAGE
API Key: $(if [[ -n "${QDRANT_API_KEY:-}" ]]; then echo "Configured"; else echo "None"; fi)
EOF
    
    return 0
}

qdrant::list_backups() {
    backup::list "qdrant"
}

qdrant::backup_info() {
    local backup_name="$1"
    
    if [[ -z "$backup_name" ]]; then
        log::error "Backup name is required"
        return 1
    fi
    
    backup::info "qdrant" "$backup_name"
}

qdrant::recover() {
    local backup_name="$1"
    
    if [[ -z "$backup_name" ]]; then
        log::error "Backup name is required"
        return 1
    fi
    
    log::info "Starting Qdrant recovery from backup: $backup_name"
    
    # Stop Qdrant if running
    if qdrant::common::is_running; then
        log::info "Stopping Qdrant for recovery..."
        qdrant::docker::stop
    fi
    
    # Extract backup
    local restore_dir
    if ! restore_dir=$(backup::extract "qdrant" "$backup_name"); then
        log::error "Failed to extract backup"
        return 1
    fi
    
    # Backup current data
    if [[ -d "$QDRANT_DATA_DIR" ]]; then
        local current_backup
        current_backup="${QDRANT_DATA_DIR}.backup.$(date +%Y%m%d_%H%M%S)"
        log::info "Backing up current data to: $current_backup"
        mv "$QDRANT_DATA_DIR" "$current_backup"
    fi
    
    # Restore data
    log::info "Restoring Qdrant data..."
    mkdir -p "$QDRANT_DATA_DIR"
    
    # Restore snapshots if they exist
    if [[ -d "$restore_dir/snapshots" ]]; then
        cp -r "$restore_dir/snapshots"/* "$QDRANT_SNAPSHOTS_DIR/" 2>/dev/null || true
    fi
    
    # Restore configuration if it exists
    if [[ -d "$restore_dir/config" ]]; then
        mkdir -p "$QDRANT_CONFIG_DIR"
        cp -r "$restore_dir/config"/* "$QDRANT_CONFIG_DIR/" 2>/dev/null || true
    fi
    
    # Start Qdrant
    log::info "Starting Qdrant after recovery..."
    if ! qdrant::docker::start; then
        log::error "Failed to start Qdrant after recovery"
        return 1
    fi
    
    # Wait for Qdrant to be ready
    if ! qdrant::wait_for_ready; then
        log::error "Qdrant failed to start properly after recovery"
        return 1
    fi
    
    # Restore collections from snapshots
    if [[ -d "$restore_dir/snapshots" ]] && [[ -n "$(ls -A "$restore_dir/snapshots" 2>/dev/null)" ]]; then
        log::info "Restoring collections from snapshots..."
        qdrant::restore_collections_from_snapshots "$restore_dir/snapshots"
    fi
    
    # Cleanup
    rm -rf "$restore_dir"
    
    log::success "Recovery completed successfully!"
    
    # Show status
    qdrant::status::check
    
    return 0
}

qdrant::restore_collections_from_snapshots() {
    local snapshots_dir="$1"
    
    # Process each snapshot file
    for snapshot_file in "$snapshots_dir"/*.snapshot; do
        if [[ -f "$snapshot_file" ]]; then
            local filename
            filename=$(basename "$snapshot_file")
            local collection_name="${filename%.snapshot}"
            
            log::info "Restoring collection: $collection_name"
            
            if qdrant::snapshots::restore_collection_snapshot "$collection_name" "$snapshot_file"; then
                log::success "Restored collection: $collection_name"
            else
                log::warn "Failed to restore collection: $collection_name"
            fi
        fi
    done
}

qdrant::cleanup_backups() {
    local days="${1:-30}"
    log::info "Cleaning up Qdrant backups older than $days days..."
    backup::cleanup "qdrant" "$days"
}