#!/usr/bin/env bash
# Qdrant Core Functions - Consolidated Qdrant-specific logic
# Combines docker.sh, api.sh, and status.sh functionality
# All generic operations delegated to shared frameworks

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
QDRANT_LIB_DIR="${APP_ROOT}/resources/qdrant/lib"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
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
# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/api-client.sh"

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
# Install Qdrant with comprehensive checks and setup
# Arguments: None
# Returns: 0 on success, 1 on failure
#######################################
qdrant::install() {
    echo "=== Installing Qdrant Vector Database ==="
    echo
    
    # Check prerequisites
    if ! qdrant::install::check_prerequisites; then
        return 1
    fi
    
    # Check if already installed
    if docker::container_exists "$QDRANT_CONTAINER_NAME"; then
        if docker::is_running "$QDRANT_CONTAINER_NAME"; then
            log::info "Qdrant is already installed and running"
            qdrant::display_connection_info
            return 0
        else
            log::info "Qdrant is installed but not running. Starting..."
            if qdrant::start; then
                qdrant::display_connection_info
                return 0
            else
                log::error "Failed to start existing Qdrant installation"
                return 1
            fi
        fi
    fi
    
    # Check port availability
    log::info "Checking port availability..."
    if ! qdrant::common::check_ports; then
        log::error "Required ports are not available"
        log::info "Ports needed: $QDRANT_PORT (HTTP), $QDRANT_GRPC_PORT (gRPC)"
        return 1
    fi
    
    # Check disk space
    if ! qdrant::common::check_disk_space; then
        log::error "Insufficient disk space for installation"
        return 1
    fi
    
    # Use init framework for installation
    local init_config
    init_config=$(qdrant::get_init_config)
    
    log::info "Installing Qdrant using framework..."
    
    if ! init::setup_resource "$init_config"; then
        log::error "Failed to install Qdrant"
        return 1
    fi
    
    # Wait for startup
    log::info "Waiting for Qdrant to start..."
    if ! qdrant::wait_for_ready 60; then
        log::error "Qdrant failed to start properly"
        return 1
    fi
    
    # Initialize default collections
    log::info "Creating default collections..."
    if ! qdrant::install::create_default_collections; then
        log::warn "Some default collections could not be created"
    fi
    
    # Update resource configuration
    if ! qdrant::install::update_resource_config; then
        log::warn "Failed to update resource configuration"
    fi
    
    # Success
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
    
    # Auto-install CLI if available
    # shellcheck disable=SC1091
    "${var_SCRIPTS_RESOURCES_LIB_DIR}/install-resource-cli.sh" "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)" 2>/dev/null || true
}

#######################################
# Uninstall Qdrant with comprehensive cleanup
# Arguments:
#   $1 - preserve_data (boolean): whether to preserve data
# Returns: 0 on success, 1 on failure
#######################################
qdrant::uninstall() {
    local preserve_data="${1:-true}"
    
    echo "=== Uninstalling Qdrant ==="
    echo
    
    if [[ "$preserve_data" == "false" ]]; then
        log::warn "This will permanently delete all Qdrant data!"
    fi
    
    # Stop container if running
    if docker::container_exists "$QDRANT_CONTAINER_NAME"; then
        if docker::is_running "$QDRANT_CONTAINER_NAME"; then
            log::info "Stopping Qdrant container..."
            docker::stop_container "$QDRANT_CONTAINER_NAME"
        fi
        
        log::info "Removing Qdrant container..."
        docker::remove_container "$QDRANT_CONTAINER_NAME"
    fi
    
    # Remove network if it exists and no other containers are using it
    if docker network ls | grep -q "$QDRANT_NETWORK_NAME" 2>/dev/null; then
        docker::remove_network_if_empty "$QDRANT_NETWORK_NAME"
    fi
    
    # Handle data removal
    if [[ "$preserve_data" == "false" ]]; then
        log::info "Removing Qdrant data..."
        rm -rf "$QDRANT_DATA_DIR" "$QDRANT_CONFIG_DIR" "$QDRANT_SNAPSHOTS_DIR"
    else
        log::info "Preserving Qdrant data in $QDRANT_DATA_DIR"
        log::info "Run with --remove-data yes to also remove data"
    fi
    
    # Remove from resource configuration
    qdrant::install::remove_from_resource_config
    
    log::success "Qdrant uninstalled successfully"
    return 0
}

#######################################
# Get Qdrant information
# Returns: 0 on success
#######################################
qdrant::info() {
    if ! docker::is_running "$QDRANT_CONTAINER_NAME"; then
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
# Common function: Show logs
# Arguments:
#   $1 - lines (optional): number of lines to show
# Returns: 0 on success
#######################################
qdrant::common::show_logs() {
    local lines="${1:-50}"
    
    if ! docker::is_running "$QDRANT_CONTAINER_NAME"; then
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
        log::info "Usage: resource-qdrant inject <file.json>"
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
        log::info "Usage: resource-qdrant inject <file.json> --validate"
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
# API OPERATIONS - Kept for backwards compatibility only
# New code should use qdrant::client::* functions directly from api-client.sh
#######################################

# This function adds value with logging, so we keep it
qdrant::api::test() {
    if qdrant::client::health_check; then
        log::success "Qdrant API is accessible"
        return 0
    else
        log::error "Qdrant API connection failed"
        return 1
    fi
}

#######################################
# COMPATIBILITY ALIASES
# These exist for backwards compatibility - prefer direct function calls
#######################################

qdrant::start() {
    qdrant::docker::start
}

#######################################
# ESSENTIAL COMMON FUNCTIONS (from common.sh)
#######################################


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
    if ! docker::is_running "$QDRANT_CONTAINER_NAME"; then
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
    if docker::is_running "$QDRANT_CONTAINER_NAME"; then
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

#######################################
# Monitor Qdrant status continuously
# Arguments:
#   $1 - interval in seconds (default: 5)
#######################################
qdrant::status::monitor() {
    local interval="${1:-5}"
    
    log::info "Monitoring Qdrant status (interval: ${interval}s). Press Ctrl+C to stop."
    echo
    
    while true; do
        clear
        echo "=== Qdrant Status Monitor - $(date) ==="
        echo
        
        # Show status
        qdrant::status::check true
        
        echo
        echo "Next update in ${interval} seconds..."
        sleep "$interval"
    done
}

qdrant::display_additional_info() {
    if ! docker::is_running "$QDRANT_CONTAINER_NAME"; then
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
    if cluster_info=$(qdrant::client::get "/cluster" "cluster info" 2>/dev/null); then
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
# INSTALLATION HELPER FUNCTIONS
#######################################

#######################################
# Check installation prerequisites
# Returns: 0 if all prerequisites met, 1 otherwise
#######################################
qdrant::install::check_prerequisites() {
    log::info "Checking prerequisites..."
    
    # Check Docker
    if ! docker::check_daemon; then
        log::error "Docker is not running or not installed"
        return 1
    fi
    
    # Check required commands
    local required_commands=("curl" "jq")
    local missing_commands=()
    
    for cmd in "${required_commands[@]}"; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            missing_commands+=("$cmd")
        fi
    done
    
    if [[ ${#missing_commands[@]} -gt 0 ]]; then
        log::error "Missing required commands: ${missing_commands[*]}"
        log::info "Please install the missing commands and try again"
        return 1
    fi
    
    log::debug "All prerequisites met"
    return 0
}

#######################################
# Create default collections for Vrooli
# Returns: 0 on success, 1 if any failures
#######################################
qdrant::install::create_default_collections() {
    local success_count=0
    local total_count=${#QDRANT_COLLECTION_CONFIGS[@]}
    
    for config in "${QDRANT_COLLECTION_CONFIGS[@]}"; do
        # Parse config: name:vector_size:distance_metric
        local name
        local vector_size
        local distance_metric
        
        IFS=':' read -r name vector_size distance_metric <<< "$config"
        
        log::info "Creating collection: $name"
        
        if qdrant::collections::create "$name" "$vector_size" "$distance_metric" >/dev/null 2>&1; then
            log::debug "Collection '$name' created successfully"
            success_count=$((success_count + 1))
        else
            log::warn "Failed to create collection '$name'"
        fi
    done
    
    log::info "Created $success_count of $total_count default collections"
    
    if [[ $success_count -eq $total_count ]]; then
        log::success "Default collections initialized successfully"
        return 0
    elif [[ $success_count -gt 0 ]]; then
        log::warn "Some default collections could not be created"
        return 0
    else
        log::warn "No default collections were created"
        return 1
    fi
}

#######################################
# Update resource configuration file
# Returns: 0 on success, 1 on failure
#######################################
qdrant::install::update_resource_config() {
    local config_file="${VROOLI_RESOURCES_CONFIG:-}"
    
    if [[ -z "$config_file" ]] || [[ ! -f "$config_file" ]]; then
        log::debug "No resource configuration file to update"
        return 0
    fi
    
    # Create backup
    cp "$config_file" "${config_file}.backup" 2>/dev/null || true
    
    # Update configuration using jq
    local updated_config
    updated_config=$(jq --arg port "$QDRANT_PORT" \
                        --arg grpc_port "$QDRANT_GRPC_PORT" \
                        --arg base_url "$QDRANT_BASE_URL" \
                        --arg grpc_url "$QDRANT_GRPC_URL" \
                        --arg container "$QDRANT_CONTAINER_NAME" \
                        --arg data_dir "$QDRANT_DATA_DIR" \
                        --arg version "$(qdrant::version 2>/dev/null || echo 'unknown')" \
                        '.services.storage.qdrant = {
                            "enabled": true,
                            "port": ($port | tonumber),
                            "grpc_port": ($grpc_port | tonumber),
                            "base_url": $base_url,
                            "grpc_url": $grpc_url,
                            "container_name": $container,
                            "data_directory": $data_dir,
                            "version": $version,
                            "status": "installed",
                            "last_updated": (now | strftime("%Y-%m-%d %H:%M:%S"))
                        }' "$config_file" 2>/dev/null)
    
    if [[ -n "$updated_config" ]]; then
        echo "$updated_config" > "${config_file}.tmp"
        
        if jq . "${config_file}.tmp" >/dev/null 2>&1; then
            mv "${config_file}.tmp" "$config_file"
            log::debug "Resource configuration updated"
            return 0
        fi
    fi
    
    # Cleanup and restore on failure
    trash::safe_remove "${config_file}.tmp" --temp 2>/dev/null || true
    log::warn "Failed to update resource configuration"
    return 1
}

#######################################
# Remove Qdrant from resource configuration
# Returns: 0 on success, 1 on failure
#######################################
qdrant::install::remove_from_resource_config() {
    local config_file="${VROOLI_RESOURCES_CONFIG:-}"
    
    if [[ -z "$config_file" ]] || [[ ! -f "$config_file" ]]; then
        return 0
    fi
    
    if jq 'del(.services.storage.qdrant)' "$config_file" > "${config_file}.tmp" 2>/dev/null; then
        mv "${config_file}.tmp" "$config_file"
        log::debug "Removed from resource configuration"
        return 0
    else
        trash::safe_remove "${config_file}.tmp" --temp 2>/dev/null || true
        log::warn "Failed to remove from resource configuration"
        return 1
    fi
}

#######################################
# Upgrade Qdrant to latest version
# Returns: 0 on success, 1 on failure
#######################################
qdrant::install::upgrade() {
    echo "=== Upgrading Qdrant ==="
    echo
    
    if ! docker::container_exists "$QDRANT_CONTAINER_NAME"; then
        log::error "Qdrant is not installed. Use 'install' action first."
        return 1
    fi
    
    # Get current version if possible
    local current_version="unknown"
    if docker::is_running "$QDRANT_CONTAINER_NAME"; then
        current_version=$(qdrant::version 2>/dev/null || echo "unknown")
        log::info "Current version: $current_version"
    else
        log::info "Qdrant is not running, unable to check current version"
    fi
    
    # Create backup before upgrade
    log::info "Creating backup before upgrade..."
    if qdrant::health::is_healthy; then
        local backup_name="pre-upgrade-$(date +%Y%m%d-%H%M%S)"
        if ! qdrant::backup::create "$backup_name" >/dev/null 2>&1; then
            log::warn "Failed to create backup - continuing with upgrade"
        else
            log::info "Backup created: $backup_name"
        fi
    fi
    
    # Stop current container
    log::info "Stopping current Qdrant container..."
    if ! docker::stop_container "$QDRANT_CONTAINER_NAME"; then
        log::error "Failed to stop Qdrant container"
        return 1
    fi
    
    # Remove old container
    log::info "Removing old container..."
    if ! docker::remove_container "$QDRANT_CONTAINER_NAME"; then
        log::error "Failed to remove old container"
        return 1
    fi
    
    # Pull latest image
    log::info "Pulling latest Qdrant image..."
    if ! docker::pull_image "$QDRANT_IMAGE"; then
        log::error "Failed to pull latest image"
        return 1
    fi
    
    # Create new container with existing data
    log::info "Creating new container with existing data..."
    local init_config
    init_config=$(qdrant::get_init_config)
    
    if ! init::setup_resource "$init_config"; then
        log::error "Failed to create new container"
        return 1
    fi
    
    # Wait for startup
    if ! qdrant::wait_for_ready 60; then
        log::error "Upgraded Qdrant failed to start"
        return 1
    fi
    
    # Get new version
    local new_version
    new_version=$(qdrant::version 2>/dev/null || echo "unknown")
    
    log::success "Upgrade completed successfully"
    log::info "New version: $new_version"
    
    # Show connection info
    qdrant::display_connection_info
    
    return 0
}

#######################################
# Reset Qdrant configuration to defaults
# Returns: 0 on success, 1 on failure
#######################################
qdrant::install::reset_configuration() {
    log::info "Resetting Qdrant configuration to defaults..."
    
    # Stop if running
    if docker::is_running "$QDRANT_CONTAINER_NAME"; then
        log::info "Stopping Qdrant to reset configuration..."
        qdrant::stop || return 1
    fi
    
    # Remove configuration directory
    trash::safe_remove "${QDRANT_CONFIG_DIR}" --production 2>/dev/null || true
    
    # Recreate directories
    mkdir -p "$QDRANT_DATA_DIR" "$QDRANT_CONFIG_DIR" "$QDRANT_SNAPSHOTS_DIR" || return 1
    
    # Restart with clean configuration
    if qdrant::start; then
        log::success "Configuration reset completed"
        return 0
    else
        log::error "Failed to start with reset configuration"
        return 1
    fi
}

