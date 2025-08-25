#!/bin/bash
# OpenRouter status functionality

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENROUTER_STATUS_DIR="${APP_ROOT}/resources/openrouter/lib"

# Source dependencies
source "${OPENROUTER_STATUS_DIR}/core.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/status-args.sh"

#######################################
# Collect OpenRouter status data
# Arguments:
#   --fast: Skip expensive operations
# Returns:
#   Key-value pairs (one per line)
#######################################
openrouter::status::collect_data() {
    local fast="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --fast)
                fast="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Initialize
    openrouter::init >/dev/null 2>&1
    
    local status="unknown"
    local message=""
    local api_base="$OPENROUTER_API_BASE"
    local model="$OPENROUTER_DEFAULT_MODEL"
    
    # Check if API key is configured
    if [[ -z "$OPENROUTER_API_KEY" ]]; then
        status="not_configured"
        message="OpenRouter API key not configured"
    else
        # Check if it's a placeholder key
        if [[ "$OPENROUTER_API_KEY" == "sk-placeholder-key" ]]; then
            # For API services with placeholder keys, use "configured" status
            # This indicates it's installed but needs real credentials
            status="configured"
            message="OpenRouter configured with placeholder key (requires real API key)"
        else
            # Test connection with real API key (skip in fast mode)
            if [[ "$fast" == "true" ]]; then
                status="running"
                message="OpenRouter API configured (fast mode)"
            elif openrouter::test_connection; then
                status="running"
                message="OpenRouter API is accessible"
            else
                status="error"
                message="OpenRouter API is not accessible (check API key validity)"
            fi
        fi
    fi
    
    # Determine running status
    # API services are considered "running" if they're configured or operational
    local running="false"
    if [[ "$status" == "running" || "$status" == "configured" ]]; then
        running="true"
    fi
    
    # Determine health status
    local healthy="false"
    if [[ "$status" == "running" ]]; then
        healthy="true"
    elif [[ "$status" == "configured" ]]; then
        # For API services with placeholder keys, consider them unhealthy but operational
        healthy="false"
    fi
    
    # Get usage info (skip in fast mode)
    local limit="N/A"
    local usage_amount="N/A"
    if [[ "$fast" == "false" && "$status" == "running" ]]; then
        local usage
        usage=$(openrouter::get_usage 2>/dev/null)
        if [[ -n "$usage" ]]; then
            limit=$(echo "$usage" | jq -r '.data.limit' 2>/dev/null || echo "unknown")
            usage_amount=$(echo "$usage" | jq -r '.data.usage' 2>/dev/null || echo "unknown")
        fi
    fi
    
    # Check test results
    local test_status="not_run"
    local test_timestamp="N/A"
    local test_file="${var_ROOT_DIR}/data/test-results/openrouter-test.json"
    
    if [[ -f "$test_file" ]]; then
        test_status=$(jq -r '.status // "unknown"' "$test_file" 2>/dev/null || echo "unknown")
        test_timestamp=$(jq -r '.timestamp // "N/A"' "$test_file" 2>/dev/null || echo "N/A")
    fi
    
    # Output data as key-value pairs
    echo "status"
    echo "$status"
    echo "running"
    echo "$running"
    echo "healthy"
    echo "$healthy"
    echo "message"
    echo "$message"
    echo "api_base"
    echo "$api_base"
    echo "default_model"
    echo "$model"
    echo "limit"
    echo "$limit"
    echo "usage"
    echo "$usage_amount"
    echo "test_status"
    echo "$test_status"
    echo "test_timestamp"
    echo "$test_timestamp"
}

#######################################
# Display OpenRouter status in text format
# Arguments:
#   data_array: Array of key-value pairs
#######################################
openrouter::status::display_text() {
    local -a data_array=("$@")
    
    # Convert array to associative array for easier access
    local -A data
    for ((i=0; i<${#data_array[@]}; i+=2)); do
        data["${data_array[i]}"]="${data_array[i+1]}"
    done
    
    echo "Status: ${data[status]}"
    echo "Running: ${data[running]}"
    echo "Healthy: ${data[healthy]}"
    echo "Message: ${data[message]}"
    echo "API Base: ${data[api_base]}"
    echo "Default Model: ${data[default_model]}"
    echo "Limit: ${data[limit]}"
    echo "Usage: ${data[usage]}"
    echo "Test Status: ${data[test_status]}"
    echo "Test Timestamp: ${data[test_timestamp]}"
}

#######################################
# Check OpenRouter status
# Arguments:
#   --format: Output format (text/json)
#   --fast: Skip expensive operations
# Returns:
#   0 if healthy, 1 otherwise
#######################################
openrouter::status() {
    status::run_standard "openrouter" "openrouter::status::collect_data" "openrouter::status::display_text" "$@"
}

# Export function
export -f openrouter::status