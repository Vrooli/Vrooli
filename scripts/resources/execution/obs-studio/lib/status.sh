#!/bin/bash

# Status check for OBS Studio
set -euo pipefail

# Get script directory
OBS_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${OBS_STATUS_DIR}/common.sh"

# Check installation status
check_installation() {
    if obs_is_installed; then
        echo "true"
    else
        echo "false"
    fi
}

# Check running status
check_running() {
    if obs_is_running; then
        echo "true"
    else
        echo "false"
    fi
}

# Check health status
check_health() {
    local installed=$(check_installation)
    local running=$(check_running)
    
    if [[ "${installed}" == "false" ]]; then
        echo "not_installed"
    elif [[ "${running}" == "false" ]]; then
        echo "stopped"
    elif obs_websocket_healthy; then
        echo "healthy"
    else
        echo "unhealthy"
    fi
}

# Get detailed status
get_status_details() {
    local installed=$(check_installation)
    local running=$(check_running)
    local health=$(check_health)
    local version=$(obs_get_version)
    local websocket_installed="false"
    local websocket_port="${OBS_PORT}"
    local recording_path="${OBS_RECORDINGS_DIR}"
    
    if obs_websocket_installed; then
        websocket_installed="true"
    fi
    
    # Count recordings if directory exists
    local recording_count=0
    if [[ -d "${OBS_RECORDINGS_DIR}" ]]; then
        recording_count=$(find "${OBS_RECORDINGS_DIR}" -type f \( -name "*.mkv" -o -name "*.mp4" -o -name "*.flv" \) 2>/dev/null | wc -l)
    fi
    
    # Prepare status message
    local status_message=""
    if [[ "${health}" == "healthy" ]]; then
        status_message="OBS Studio is running with WebSocket API available"
    elif [[ "${health}" == "unhealthy" ]]; then
        status_message="OBS Studio is running but WebSocket API is not responding"
    elif [[ "${health}" == "stopped" ]]; then
        status_message="OBS Studio is installed but not running"
    else
        status_message="OBS Studio is not installed"
    fi
    
    # Output in format that can be consumed by format.sh
    cat <<EOF
description="Professional streaming and recording software"
category="execution"
installed="${installed}"
running="${running}"
health="${health}"
version="${version}"
websocket_installed="${websocket_installed}"
websocket_port="${websocket_port}"
recording_path="${recording_path}"
recording_count="${recording_count}"
status_message="${status_message}"
EOF
}

# Format output based on mode
format_output() {
    local mode="${1:-text}"
    local status_data=$(get_status_details)
    
    if [[ "${mode}" == "json" ]]; then
        # Convert to JSON format
        echo "{"
        echo "${status_data}" | while IFS='=' read -r key value; do
            # Remove quotes from value
            value="${value%\"}"
            value="${value#\"}"
            echo "  \"${key}\": \"${value}\","
        done | sed '$ s/,$//'
        echo "}"
    else
        # Text format
        echo "[HEADER]  🎬 OBS Studio Status"
        echo ""
        echo "[INFO]    📊 Basic Status:"
        
        eval "${status_data}"
        
        if [[ "${installed}" == "true" ]]; then
            echo "[SUCCESS]    ✅ Installed: Yes"
        else
            echo "[ERROR]      ❌ Installed: No"
        fi
        
        if [[ "${running}" == "true" ]]; then
            echo "[SUCCESS]    ✅ Running: Yes"
        else
            echo "[WARNING]    ⚠️  Running: No"
        fi
        
        if [[ "${health}" == "healthy" ]]; then
            echo "[SUCCESS]    ✅ Health: Healthy"
        elif [[ "${health}" == "unhealthy" ]]; then
            echo "[WARNING]    ⚠️  Health: Unhealthy (WebSocket not responding)"
        elif [[ "${health}" == "stopped" ]]; then
            echo "[WARNING]    ⚠️  Health: Stopped"
        else
            echo "[ERROR]      ❌ Health: Not installed"
        fi
        
        if [[ "${installed}" == "true" ]]; then
            echo ""
            echo "[INFO]    ⚙️  Configuration:"
            echo "[INFO]       📦 Version: ${version}"
            
            if [[ "${websocket_installed}" == "true" ]]; then
                echo "[SUCCESS]    ✅ WebSocket Plugin: Installed"
                echo "[INFO]       🔌 WebSocket Port: ${websocket_port}"
            else
                echo "[WARNING]    ⚠️  WebSocket Plugin: Not installed"
            fi
            
            echo "[INFO]       📁 Recording Path: ${recording_path}"
            echo "[INFO]       🎥 Recordings: ${recording_count} files"
        fi
        
        echo ""
        echo "[INFO]    📋 Status Message:"
        if [[ "${health}" == "healthy" ]]; then
            echo "[SUCCESS]    ✅ ${status_message}"
        elif [[ "${health}" == "not_installed" ]]; then
            echo "[ERROR]      ❌ ${status_message}"
        else
            echo "[WARNING]    ⚠️  ${status_message}"
        fi
    fi
}

# Main function
main() {
    local format="${1:-text}"
    
    # Use format.sh if available for consistent output
    if [[ -f "${PROJECT_ROOT}/scripts/lib/utils/format.sh" ]]; then
        source "${PROJECT_ROOT}/scripts/lib/utils/format.sh"
        
        # Prepare data for format.sh
        eval "$(get_status_details)"
        
        # Use format_status function from format.sh
        format_status \
            --name "OBS Studio" \
            --installed "${installed}" \
            --running "${running}" \
            --healthy "${health}" \
            --version "${version}" \
            --message "${status_message}" \
            --format "${format}"
    else
        # Fallback to local formatting
        format_output "${format}"
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi