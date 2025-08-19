#!/bin/bash
# Gemini status functionality

# Get script directory
GEMINI_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${GEMINI_STATUS_DIR}/core.sh"
# shellcheck disable=SC1091
source "${GEMINI_STATUS_DIR}/../../../lib/status-args.sh"

#######################################
# Collect Gemini status data
# Arguments:
#   --fast: Skip expensive operations
# Returns:
#   Key-value pairs (one per line)
#######################################
gemini::status::collect_data() {
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
    gemini::init >/dev/null 2>&1
    
    local status="unknown"
    local message=""
    local api_base="$GEMINI_API_BASE"
    local model="$GEMINI_DEFAULT_MODEL"
    
    # Check if API key is configured
    if [[ -z "$GEMINI_API_KEY" ]]; then
        status="not_configured"
        message="Gemini API key not configured"
    else
        # Check if it's a placeholder key
        if [[ "$GEMINI_API_KEY" == "placeholder-gemini-key" ]]; then
            # For API services with placeholder keys, use "configured" status
            status="configured"
            message="Gemini configured with placeholder key (requires real API key)"
        else
            # Test connection with real API key (skip in fast mode)
            if [[ "$fast" == "true" ]]; then
                status="running"
                message="Gemini API configured (fast mode)"
            elif gemini::test_connection; then
                status="running"
                message="Gemini API is accessible"
            else
                status="error"
                message="Gemini API is not accessible (check API key validity)"
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
        # For API services with placeholder keys, consider them healthy but note they need real keys
        healthy="false"
    fi
    
    # Get available models (skip in fast mode)
    local models="N/A"
    if [[ "$fast" == "false" && "$status" == "running" ]]; then
        models=$(gemini::list_models 2>/dev/null | tr '\n' ',' | sed 's/,$//')
        if [[ -z "$models" ]]; then
            models="N/A"
        fi
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
    echo "available_models"
    echo "$models"
}

#######################################
# Display Gemini status in text format
# Arguments:
#   data_array: Array of key-value pairs
#######################################
gemini::status::display_text() {
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
    echo "Available Models: ${data[available_models]}"
}

#######################################
# Check Gemini status
# Arguments:
#   --format: Output format (text/json)
#   --fast: Skip expensive operations
# Returns:
#   0 if healthy, 1 otherwise
#######################################
gemini::status() {
    status::run_standard "gemini" "gemini::status::collect_data" "gemini::status::display_text" "$@"
}

# Export function
export -f gemini::status