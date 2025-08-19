#!/bin/bash
# TARS-desktop status functionality

# Get script directory
TARS_DESKTOP_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${TARS_DESKTOP_STATUS_DIR}/core.sh"

# Main status function
tars_desktop::status() {
    local verbose="${1:-false}"
    local format="${2:-text}"
    
    # Initialize
    tars_desktop::init >/dev/null 2>&1
    
    local status="unknown"
    local message=""
    local health="unhealthy"
    
    # Check if installed
    if ! tars_desktop::is_installed; then
        status="not_installed"
        message="TARS-desktop is not installed"
    elif ! tars_desktop::is_running; then
        status="stopped"
        message="TARS-desktop is installed but not running"
    elif tars_desktop::health_check; then
        status="running"
        health="healthy"
        message="TARS-desktop is running and healthy"
    else
        status="running"
        health="unhealthy"
        message="TARS-desktop is running but not responding"
    fi
    
    # Build output data
    local -a output_data=(
        "status" "$status"
        "health" "$health"
        "message" "$message"
        "api_base" "$TARS_DESKTOP_API_BASE"
        "port" "$TARS_DESKTOP_PORT"
    )
    
    # Add verbose details if requested
    if [[ "$verbose" == "true" && "$status" == "running" ]]; then
        # Get capabilities
        local capabilities
        capabilities=$(tars_desktop::get_capabilities 2>/dev/null)
        if [[ -n "$capabilities" ]]; then
            local screen_width screen_height
            screen_width=$(echo "$capabilities" | jq -r '.screen.width' 2>/dev/null || echo "unknown")
            screen_height=$(echo "$capabilities" | jq -r '.screen.height' 2>/dev/null || echo "unknown")
            output_data+=(
                "screen_width" "$screen_width"
                "screen_height" "$screen_height"
            )
        fi
        
        # Check process info
        local pid
        pid=$(pgrep -f "tars_desktop.*server" | head -1)
        if [[ -n "$pid" ]]; then
            output_data+=("pid" "$pid")
        fi
        
        # Installation info
        output_data+=(
            "install_dir" "$TARS_DESKTOP_INSTALL_DIR"
            "venv_dir" "$TARS_DESKTOP_VENV_DIR"
            "model" "$TARS_DESKTOP_MODEL"
            "provider" "$TARS_DESKTOP_MODEL_PROVIDER"
        )
    fi
    
    # Format output using standard formatter
    format::key_value "$format" "${output_data[@]}"
}

# Export function
export -f tars_desktop::status