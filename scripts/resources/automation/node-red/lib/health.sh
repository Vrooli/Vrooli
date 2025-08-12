#!/usr/bin/env bash
# Node-RED Health Check Functions
# Provides comprehensive health monitoring using health framework

NODE_RED_HEALTH_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${NODE_RED_HEALTH_LIB_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/health-framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/http-utils.sh"

#######################################
# Get Node-RED health check configuration
# Returns: JSON configuration for health framework
#######################################
node_red::get_health_config() {
    echo '{
        "service_name": "Node-RED",
        "container_name": "'$NODE_RED_CONTAINER_NAME'",
        "checks": {
            "basic": "node_red::check_basic_health",
            "advanced": "node_red::check_advanced_health"
        }
    }'
}

#######################################
# Perform basic health check
# Returns: 0 if healthy, 1 otherwise
#######################################
node_red::check_basic_health() {
    log::info "Checking Node-RED basic health..."
    
    # Check if container is running
    if ! docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        log::error "Node-RED container is not running"
        return 1
    fi
    
    # Check if port is accessible
    if ! http::check_endpoint "$NODE_RED_BASE_URL"; then
        log::error "Node-RED port $NODE_RED_PORT is not accessible"
        return 1
    fi
    
    # Basic HTTP response check
    local url="http://localhost:$NODE_RED_PORT"
    local response_code
    response_code=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
    
    if [[ "$response_code" != "200" ]]; then
        log::error "Node-RED health check failed (HTTP $response_code)"
        return 1
    fi
    
    log::success "Node-RED basic health check passed"
    return 0
}

#######################################
# Perform advanced health check
# Returns: 0 if healthy, 1 otherwise
#######################################
node_red::check_advanced_health() {
    log::info "Checking Node-RED advanced health..."
    
    # Check flows API endpoint
    if ! node_red::check_flows_api; then
        log::error "Node-RED flows API is not responding"
        return 1
    fi
    
    # Check settings API endpoint
    if ! node_red::check_settings_api; then
        log::error "Node-RED settings API is not responding"
        return 1
    fi
    
    # Check websocket endpoint
    if ! node_red::check_websocket_endpoint; then
        log::warn "Node-RED websocket endpoint check failed (non-critical)"
    fi
    
    # Check data directory
    if ! node_red::check_data_directory; then
        log::error "Node-RED data directory issues detected"
        return 1
    fi
    
    log::success "Node-RED advanced health check passed"
    return 0
}

#######################################
# Check Node-RED flows API endpoint
# Returns: 0 if accessible, 1 otherwise
#######################################
node_red::check_flows_api() {
    local flows_url="http://localhost:$NODE_RED_PORT/flows"
    local response_code
    
    response_code=$(curl -s -o /dev/null -w "%{http_code}" "$flows_url" 2>/dev/null || echo "000")
    
    if [[ "$response_code" == "200" ]]; then
        return 0
    fi
    
    log::debug "Flows API returned HTTP $response_code"
    return 1
}

#######################################
# Check Node-RED settings API endpoint
# Returns: 0 if accessible, 1 otherwise
#######################################
node_red::check_settings_api() {
    local settings_url="http://localhost:$NODE_RED_PORT/settings"
    local response_code
    
    response_code=$(curl -s -o /dev/null -w "%{http_code}" "$settings_url" 2>/dev/null || echo "000")
    
    if [[ "$response_code" == "200" ]]; then
        return 0
    fi
    
    log::debug "Settings API returned HTTP $response_code"
    return 1
}

#######################################
# Check Node-RED websocket endpoint
# Returns: 0 if accessible, 1 otherwise
#######################################
node_red::check_websocket_endpoint() {
    local ws_url="http://localhost:$NODE_RED_PORT/comms"
    local response_code
    
    # Websocket endpoints often return different codes
    response_code=$(curl -s -o /dev/null -w "%{http_code}" -H "Upgrade: websocket" -H "Connection: Upgrade" "$ws_url" 2>/dev/null || echo "000")
    
    # Accept various websocket-related response codes
    if [[ "$response_code" == "101" ]] || [[ "$response_code" == "400" ]] || [[ "$response_code" == "426" ]]; then
        return 0
    fi
    
    log::debug "Websocket endpoint returned HTTP $response_code"
    return 1
}

#######################################
# Check Node-RED data directory
# Returns: 0 if healthy, 1 otherwise
#######################################
node_red::check_data_directory() {
    # Check if data directory exists
    if [[ ! -d "$NODE_RED_DATA_DIR" ]]; then
        log::error "Node-RED data directory does not exist: $NODE_RED_DATA_DIR"
        return 1
    fi
    
    # Check if flows file exists (after initial setup)
    if [[ ! -f "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE" ]]; then
        log::warn "Node-RED flows file not found (may be first run)"
    fi
    
    # Check if settings file exists
    if [[ ! -f "$NODE_RED_DATA_DIR/$NODE_RED_SETTINGS_FILE" ]]; then
        log::warn "Node-RED settings file not found"
    fi
    
    # Check directory permissions
    if [[ ! -w "$NODE_RED_DATA_DIR" ]]; then
        log::warn "Node-RED data directory is not writable"
    fi
    
    return 0
}

#######################################
# Perform tiered health check using health framework
# Returns: 0 if healthy, 1 otherwise
#######################################
node_red::health() {
    if ! docker::check_daemon; then
        return 1
    fi
    
    local config
    config=$(node_red::get_health_config)
    health::tiered_check "$config"
}

#######################################
# Get detailed health status information
# Returns: JSON with detailed health information
#######################################
node_red::get_health_status() {
    local status="unknown"
    local details=()
    
    if ! docker::check_daemon; then
        status="error"
        details+=("Docker daemon not accessible")
    elif ! docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        status="not_installed"
        details+=("Node-RED container does not exist")
    elif ! docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        status="stopped"
        details+=("Node-RED container is not running")
    elif node_red::check_basic_health && node_red::check_advanced_health; then
        status="healthy"
        details+=("All health checks passed")
    elif node_red::check_basic_health; then
        status="degraded"
        details+=("Basic health check passed, advanced checks failed")
    else
        status="unhealthy"
        details+=("Basic health check failed")
    fi
    
    local details_json
    details_json=$(printf '%s\n' "${details[@]}" | jq -R . | jq -s .)
    
    echo '{
        "service": "Node-RED",
        "status": "'$status'",
        "timestamp": "'$(date -Iseconds)'",
        "details": '$details_json'
    }'
}

#######################################
# Node-RED health check using health framework
# Returns: 0 on healthy, 1 on failure
#######################################
node_red::health() {
    local config
    config=$(node_red::get_health_config)
    health::tiered_check "$config"
}

#######################################
# Node-RED tiered health check for status engine
# Returns: Health tier via stdout (HEALTHY|DEGRADED|UNHEALTHY)
#######################################
node_red::tiered_health_check() {
    local config
    config=$(node_red::get_health_config)
    health::tiered_check "$config"
}

#######################################
# Monitor Node-RED health continuously
# Arguments:
#   $1 - interval in seconds (default: 30)
#   $2 - max_checks (default: unlimited)
#######################################
node_red::monitor_health() {
    local interval="${1:-30}"
    local max_checks="${2:-0}"
    local check_count=0
    
    log::info "Starting Node-RED health monitoring (interval: ${interval}s)"
    
    while true; do
        ((check_count++))
        
        log::info "Health check #$check_count at $(date)"
        
        if node_red::health; then
            log::success "Node-RED is healthy"
        else
            log::error "Node-RED health check failed"
        fi
        
        if [[ "$max_checks" -gt 0 && "$check_count" -ge "$max_checks" ]]; then
            log::info "Completed $max_checks health checks"
            break
        fi
        
        sleep "$interval"
    done
}