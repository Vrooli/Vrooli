#!/bin/bash
# Home Assistant Health Check Functions

# Get script directory
HOME_ASSISTANT_HEALTH_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${HOME_ASSISTANT_HEALTH_DIR}/core.sh"

#######################################
# Check if Home Assistant is healthy
# Returns: 0 if healthy, 1 otherwise
#######################################
home_assistant::health::is_healthy() {
    home_assistant::init >/dev/null 2>&1
    
    # First check if container is running
    if ! docker::is_running "$HOME_ASSISTANT_CONTAINER_NAME"; then
        return 1
    fi
    
    # Check API endpoint
    local response
    response=$(timeout 5 curl -s -o /dev/null -w "%{http_code}" "$HOME_ASSISTANT_BASE_URL/api/" 2>/dev/null)
    
    # Home Assistant returns 401 for unauthenticated API requests, which means it's running
    # 200 means it's running and has no auth configured
    if [[ "$response" == "200" || "$response" == "401" ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Get detailed health status
# Returns: JSON object with health details
#######################################
home_assistant::health::get_status() {
    home_assistant::init >/dev/null 2>&1
    
    local is_running="false"
    local is_healthy="false"
    local message="Not installed"
    local api_status="unavailable"
    
    if docker::container_exists "$HOME_ASSISTANT_CONTAINER_NAME"; then
        if docker::is_running "$HOME_ASSISTANT_CONTAINER_NAME"; then
            is_running="true"
            
            if home_assistant::health::is_healthy; then
                is_healthy="true"
                message="Home Assistant is running and healthy"
                api_status="available"
            else
                message="Home Assistant is running but API is not responding"
                api_status="not_responding"
            fi
        else
            message="Home Assistant container exists but is not running"
        fi
    fi
    
    echo "{
        \"running\": $is_running,
        \"healthy\": $is_healthy,
        \"message\": \"$message\",
        \"api_status\": \"$api_status\",
        \"url\": \"$HOME_ASSISTANT_BASE_URL\"
    }"
}

#######################################
# Wait for Home Assistant to be healthy
# Arguments:
#   timeout: Maximum time to wait (default: 60)
# Returns: 0 if healthy, 1 if timeout
#######################################
home_assistant::health::wait_for_healthy() {
    local timeout="${1:-60}"
    local elapsed=0
    local interval=5
    
    log::info "Waiting for Home Assistant to be healthy (timeout: ${timeout}s)..."
    
    while [[ $elapsed -lt $timeout ]]; do
        if home_assistant::health::is_healthy; then
            log::success "Home Assistant is healthy"
            return 0
        fi
        
        sleep "$interval"
        elapsed=$((elapsed + interval))
        log::info "Still waiting... (${elapsed}s elapsed)"
    done
    
    log::error "Timeout waiting for Home Assistant to be healthy"
    return 1
}

# Export functions
export -f home_assistant::health::is_healthy
export -f home_assistant::health::get_status
export -f home_assistant::health::wait_for_healthy