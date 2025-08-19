#!/bin/bash
# OpenRouter status functionality

# Get script directory
OPENROUTER_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${OPENROUTER_STATUS_DIR}/core.sh"

# Main status function
openrouter::status() {
    local verbose="false"
    local format="text"
    local fast="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --verbose|-v)
                verbose="true"
                shift
                ;;
            --format)
                format="${2:-text}"
                shift 2
                ;;
            --fast)
                fast="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Source format utilities
    source "${OPENROUTER_STATUS_DIR}/../../../../lib/utils/format.sh"
    
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
    
    # Build output data for format utility
    local -a output_data=(
        "status" "$status"
        "running" "$running"
        "healthy" "$healthy"
        "message" "$message"
        "api_base" "$api_base"
        "default_model" "$model"
    )
    
    # Add verbose details if requested and available (skip in fast mode)
    if [[ "$verbose" == "true" && "$status" == "running" && "$fast" == "false" ]]; then
        # Get usage info
        local usage
        usage=$(openrouter::get_usage 2>/dev/null)
        if [[ -n "$usage" ]]; then
            local limit usage_amount
            limit=$(echo "$usage" | jq -r '.data.limit' 2>/dev/null || echo "unknown")
            usage_amount=$(echo "$usage" | jq -r '.data.usage' 2>/dev/null || echo "unknown")
            output_data+=("limit" "$limit" "usage" "$usage_amount")
        fi
    elif [[ "$verbose" == "true" && "$fast" == "true" ]]; then
        output_data+=("limit" "N/A" "usage" "N/A")
    fi
    
    # Format output using standard formatter
    format::key_value "$format" "${output_data[@]}"
}

# Export function
export -f openrouter::status