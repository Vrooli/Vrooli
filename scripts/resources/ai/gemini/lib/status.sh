#!/bin/bash
# Gemini status functionality

# Get script directory
GEMINI_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${GEMINI_STATUS_DIR}/core.sh"

# Main status function
gemini::status() {
    local verbose="${1:-false}"
    local format="${2:-text}"
    
    # Source format utilities
    source "${GEMINI_STATUS_DIR}/../../../../lib/utils/format.sh"
    
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
            # Test connection with real API key
            if gemini::test_connection; then
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
    
    # Build output data
    local -a output_data=(
        "status" "$status"
        "running" "$running"
        "healthy" "$healthy"
        "message" "$message"
        "api_base" "$api_base"
        "default_model" "$model"
    )
    
    # Add verbose details if requested and available
    if [[ "$verbose" == "true" && "$status" == "running" ]]; then
        # Get available models
        local models
        models=$(gemini::list_models 2>/dev/null | tr '\n' ',' | sed 's/,$//')
        if [[ -n "$models" ]]; then
            output_data+=("available_models" "$models")
        fi
    fi
    
    # Format output using standard formatter
    format::key_value "$format" "${output_data[@]}"
}

# Export function
export -f gemini::status