#!/usr/bin/env bash
# ====================================================================
# Standardized Status Helper for Vrooli Resources
# ====================================================================
#
# Provides consistent status reporting across all resource management
# scripts, addressing the status inconsistency issues identified during
# integration testing.
#
# This helper provides:
# 1. Standardized JSON status format
# 2. Consistent health check patterns
# 3. Integration readiness detection
# 4. Actionable next step recommendations
#
# Usage:
#   source status-helper.sh
#   status_helper::check_service "vault" "8200" "/v1/sys/health"
#   status_helper::format_response "healthy" "Service is operational"
#
# ====================================================================

# Prevent multiple inclusions
if [[ -n "${STATUS_HELPER_LOADED:-}" ]]; then
    return 0
fi
STATUS_HELPER_LOADED=1

# Status constants
readonly STATUS_HEALTHY="healthy"
readonly STATUS_DEGRADED="degraded"
readonly STATUS_ERROR="error"
readonly STATUS_UNKNOWN="unknown"

#######################################
# Format a standardized status response
# Arguments:
#   $1 - service name
#   $2 - status (healthy|degraded|error|unknown)
#   $3 - message
#   $4 - details (optional JSON object)
#   $5 - next_actions (optional array of strings)
#   $6 - integration_ready (optional true/false)
#######################################
status_helper::format_response() {
    local service="$1"
    local status="$2"
    local message="$3"
    local details="${4:-{}}"
    local next_actions="${5:-[]}"
    local integration_ready="${6:-false}"
    
    # Ensure details is valid JSON
    if ! echo "$details" | jq . >/dev/null 2>&1; then
        details="{\"raw\": $(echo "$details" | jq -R .)}"
    fi
    
    # Ensure next_actions is valid JSON array
    if ! echo "$next_actions" | jq . >/dev/null 2>&1; then
        next_actions="[]"
    fi
    
    # Generate timestamp
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%S.%3NZ")
    
    # Build JSON response
    jq -n \
        --arg service "$service" \
        --arg status "$status" \
        --arg message "$message" \
        --argjson details "$details" \
        --argjson next_actions "$next_actions" \
        --arg integration_ready "$integration_ready" \
        --arg timestamp "$timestamp" \
        '{
            service: $service,
            status: $status,
            message: $message,
            details: $details,
            next_actions: $next_actions,
            integration_ready: ($integration_ready == "true"),
            timestamp: $timestamp
        }'
}

#######################################
# Check if a service is running on a port
# Arguments:
#   $1 - port number
# Returns:
#   0 - service is running
#   1 - service is not running
#######################################
status_helper::is_port_listening() {
    local port="$1"
    
    # Use netstat to check if port is listening
    if command -v netstat >/dev/null 2>&1; then
        netstat -ln 2>/dev/null | grep -q ":$port "
        return $?
    fi
    
    # Fallback to lsof
    if command -v lsof >/dev/null 2>&1; then
        lsof -i ":$port" >/dev/null 2>&1
        return $?
    fi
    
    # Final fallback to curl
    curl -s --max-time 1 "http://localhost:$port" >/dev/null 2>&1
    return $?
}

#######################################
# Perform HTTP health check
# Arguments:
#   $1 - base URL
#   $2 - health endpoint (optional, defaults to /)
#   $3 - expected status code (optional, defaults to 200)
# Returns:
#   0 - health check passed
#   1 - health check failed
#######################################
status_helper::check_http_health() {
    local base_url="$1"
    local health_endpoint="${2:-/}"
    local expected_status="${3:-200}"
    
    local full_url="${base_url}${health_endpoint}"
    local response
    
    # Perform HTTP request with timeout
    response=$(curl -s -w "%{http_code}" --max-time 5 "$full_url" 2>/dev/null || echo "000")
    local http_code="${response: -3}"
    
    [[ "$http_code" == "$expected_status" ]]
}

#######################################
# Check Docker container status
# Arguments:
#   $1 - container name
# Returns:
#   0 - container is running
#   1 - container is not running or doesn't exist
#######################################
status_helper::check_docker_container() {
    local container_name="$1"
    
    if ! command -v docker >/dev/null 2>&1; then
        return 1
    fi
    
    docker ps --format "table {{.Names}}" 2>/dev/null | grep -q "^$container_name$"
}

#######################################
# Get Docker container details
# Arguments:
#   $1 - container name
# Outputs:
#   JSON object with container details
#######################################
status_helper::get_docker_details() {
    local container_name="$1"
    
    if ! command -v docker >/dev/null 2>&1; then
        echo "{}"
        return
    fi
    
    local container_info
    container_info=$(docker inspect "$container_name" 2>/dev/null || echo "[]")
    
    if [[ "$container_info" == "[]" ]]; then
        echo "{\"exists\": false}"
        return
    fi
    
    # Extract key information
    echo "$container_info" | jq '.[0] | {
        exists: true,
        id: .Id[0:12],
        state: .State.Status,
        started_at: .State.StartedAt,
        ports: [.NetworkSettings.Ports // {} | to_entries[] | select(.value != null) | {
            container_port: .key,
            host_port: (.value[0].HostPort // null)
        }],
        health: (.State.Health.Status // null)
    }'
}

#######################################
# Comprehensive service check
# Arguments:
#   $1 - service name
#   $2 - port number
#   $3 - health endpoint (optional)
#   $4 - container name (optional)
# Outputs:
#   Standardized JSON status response
#######################################
status_helper::check_service() {
    local service="$1"
    local port="$2"
    local health_endpoint="${3:-/}"
    local container_name="${4:-$service}"
    
    local status="$STATUS_UNKNOWN"
    local message="Service status unknown"
    local details="{}"
    local next_actions="[]"
    local integration_ready="false"
    
    # Check if port is listening
    local port_listening=false
    if status_helper::is_port_listening "$port"; then
        port_listening=true
    fi
    
    # Check Docker container if specified
    local container_running=false
    local container_details="{}"
    if [[ -n "$container_name" ]]; then
        if status_helper::check_docker_container "$container_name"; then
            container_running=true
        fi
        container_details=$(status_helper::get_docker_details "$container_name")
    fi
    
    # Check HTTP health
    local http_healthy=false
    local http_response=""
    if [[ "$port_listening" == true ]]; then
        if status_helper::check_http_health "http://localhost:$port" "$health_endpoint"; then
            http_healthy=true
            http_response=$(curl -s --max-time 5 "http://localhost:$port$health_endpoint" 2>/dev/null || echo "{}")
        fi
    fi
    
    # Determine overall status
    if [[ "$http_healthy" == true ]]; then
        status="$STATUS_HEALTHY"
        message="Service is healthy and responding"
        integration_ready="true"
        next_actions='["Service is ready for integration"]'
    elif [[ "$port_listening" == true ]]; then
        status="$STATUS_DEGRADED"
        message="Service is running but not responding to health checks"
        next_actions='["Check service logs", "Restart service if needed"]'
    elif [[ "$container_running" == true ]]; then
        status="$STATUS_DEGRADED"
        message="Container is running but service is not accessible"
        next_actions='["Check port configuration", "Check service startup logs"]'
    else
        status="$STATUS_ERROR"
        message="Service is not running"
        next_actions='["Start service", "Check configuration"]'
    fi
    
    # Build details object
    details=$(jq -n \
        --arg port "$port" \
        --arg health_endpoint "$health_endpoint" \
        --arg port_listening "$port_listening" \
        --arg container_running "$container_running" \
        --argjson container_details "$container_details" \
        --arg http_healthy "$http_healthy" \
        --argjson http_response "$http_response" \
        '{
            port: $port,
            health_endpoint: $health_endpoint,
            port_listening: ($port_listening == "true"),
            container_running: ($container_running == "true"),
            container_details: $container_details,
            http_healthy: ($http_healthy == "true"),
            http_response: $http_response
        }')
    
    # Format and return response
    status_helper::format_response \
        "$service" \
        "$status" \
        "$message" \
        "$details" \
        "$next_actions" \
        "$integration_ready"
}

#######################################
# Check integration readiness between services
# Arguments:
#   $1 - primary service name
#   $2 - secondary service name
#   $3 - integration test function (optional)
# Outputs:
#   JSON object with integration status
#######################################
status_helper::check_integration() {
    local primary="$1"
    local secondary="$2"
    local test_function="${3:-}"
    
    local integration_status="unknown"
    local integration_message="Integration status not tested"
    local integration_details="{}"
    
    # If test function provided, run it
    if [[ -n "$test_function" ]] && command -v "$test_function" >/dev/null 2>&1; then
        if "$test_function" "$primary" "$secondary"; then
            integration_status="ready"
            integration_message="Services can communicate successfully"
        else
            integration_status="blocked" 
            integration_message="Service communication failed"
        fi
    fi
    
    # Build integration response
    jq -n \
        --arg primary "$primary" \
        --arg secondary "$secondary" \
        --arg status "$integration_status" \
        --arg message "$integration_message" \
        --argjson details "$integration_details" \
        '{
            primary_service: $primary,
            secondary_service: $secondary,
            integration_status: $status,
            message: $message,
            details: $details
        }'
}

#######################################
# Display status in human-readable format
# Arguments:
#   $1 - JSON status response
#######################################
status_helper::display_human() {
    local json_status="$1"
    
    # Extract fields
    local service
    local status
    local message
    local integration_ready
    
    service=$(echo "$json_status" | jq -r '.service')
    status=$(echo "$json_status" | jq -r '.status')
    message=$(echo "$json_status" | jq -r '.message')
    integration_ready=$(echo "$json_status" | jq -r '.integration_ready')
    
    # Choose emoji and color based on status
    local emoji
    local color
    case "$status" in
        "healthy")
            emoji="✅"
            color='\033[0;32m'  # Green
            ;;
        "degraded")
            emoji="⚠️"
            color='\033[1;33m'  # Yellow
            ;;
        "error")
            emoji="❌"
            color='\033[0;31m'  # Red
            ;;
        *)
            emoji="❓"
            color='\033[0;37m'  # White
            ;;
    esac
    
    # Print status line
    echo -e "${color}${emoji} ${service}: ${message}\033[0m"
    
    # Show integration readiness
    if [[ "$integration_ready" == "true" ]]; then
        echo "   🔗 Integration ready"
    fi
    
    # Show next actions if any
    local next_actions
    next_actions=$(echo "$json_status" | jq -r '.next_actions[]?' 2>/dev/null || true)
    if [[ -n "$next_actions" ]]; then
        echo "   💡 Next steps:"
        echo "$next_actions" | while read -r action; do
            echo "      • $action"
        done
    fi
}

#######################################
# Validate that required dependencies are available
#######################################
status_helper::validate_dependencies() {
    local missing_deps=()
    
    if ! command -v jq >/dev/null 2>&1; then
        missing_deps+=("jq")
    fi
    
    if ! command -v curl >/dev/null 2>&1; then
        missing_deps+=("curl")
    fi
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        echo "Missing required dependencies: ${missing_deps[*]}" >&2
        echo "Install with: sudo apt-get install ${missing_deps[*]}" >&2
        return 1
    fi
    
    return 0
}

# Validate dependencies when loaded
if ! status_helper::validate_dependencies; then
    echo "Warning: status-helper.sh loaded with missing dependencies" >&2
fi