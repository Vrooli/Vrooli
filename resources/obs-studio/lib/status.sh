#!/bin/bash
# OBS Studio Status Functions

# Get script directory
OBS_STATUS_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${OBS_STATUS_LIB_DIR}/common.sh"
# shellcheck disable=SC1091
source "${OBS_STATUS_LIB_DIR}/../../../lib/status-args.sh"

#######################################
# Collect OBS Studio status data in format-agnostic structure
# Args: [--fast] - Skip expensive operations for faster response
# Returns: Key-value pairs ready for formatting
#######################################
obs::status::collect_data() {
    local fast_mode="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --fast)
                fast_mode="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local status_data=()
    
    # Check installation and running status
    local installed="false"
    local running="false"
    local healthy="false"
    local health_message="OBS Studio not installed"
    
    if obs_is_installed; then
        installed="true"
        
        if obs_is_running; then
            running="true"
            
            if obs_websocket_healthy; then
                healthy="true"
                health_message="OBS Studio running and accessible"
            else
                health_message="OBS Studio running but WebSocket not responding"
            fi
        else
            health_message="OBS Studio installed but not running"
        fi
    fi

    
    # Get version
    local version="unknown"
    if [[ "$installed" == "true" ]]; then
        version=$(obs_get_version 2>/dev/null || echo "unknown")
    fi
    
    # Check WebSocket status
    local websocket_installed="false"
    local websocket_status="not_installed"
    if obs_websocket_installed; then
        websocket_installed="true"
        if [[ "$running" == "true" ]]; then
            if obs_websocket_healthy; then
                websocket_status="connected"
            else
                websocket_status="disconnected"
            fi
        else
            websocket_status="not_running"
        fi
    fi
    
    # Count recordings (skip in fast mode)
    local recording_count=0
    if [[ "$fast_mode" == "false" && -d "${OBS_RECORDINGS_DIR}" ]]; then
        recording_count=$(find "${OBS_RECORDINGS_DIR}" -type f \( -name "*.mkv" -o -name "*.mp4" -o -name "*.flv" \) 2>/dev/null | wc -l)
    fi
    
    # Count scenes (skip in fast mode)
    local scene_count=0
    if [[ "$fast_mode" == "false" && -d "${OBS_SCENES_DIR}" ]]; then
        scene_count=$(find "${OBS_SCENES_DIR}" -type f -name "*.json" 2>/dev/null | wc -l)
    fi
    
    # Check for test results
    local test_status="not_run"
    local test_timestamp=""
    if [[ -f "${OBS_DATA_DIR}/.test-status" ]]; then
        test_status=$(cat "${OBS_DATA_DIR}/.test-status" 2>/dev/null || echo "not_run")
    fi
    if [[ -f "${OBS_DATA_DIR}/.last-test" ]]; then
        test_timestamp=$(cat "${OBS_DATA_DIR}/.last-test" 2>/dev/null || echo "")
    fi
    
    # Build status data array
    status_data+=("name" "obs-studio")
    status_data+=("category" "execution")
    status_data+=("description" "Professional streaming and recording software")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("version" "$version")
    status_data+=("websocket_installed" "$websocket_installed")
    status_data+=("websocket_port" "${OBS_PORT}")
    status_data+=("websocket_status" "$websocket_status")
    status_data+=("recording_path" "${OBS_RECORDINGS_DIR}")
    status_data+=("recording_count" "$recording_count")
    status_data+=("scene_count" "$scene_count")
    status_data+=("test_status" "$test_status")
    status_data+=("test_timestamp" "$test_timestamp")
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Display OBS Studio status in text format
# Args: data_array (key-value pairs)
#######################################
obs::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    echo "[HEADER]  🎬 OBS Studio Status"
    echo ""
    
    # Basic status
    echo "[INFO]    📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        echo "[SUCCESS]    ✅ Installed: Yes"
    else
        echo "[ERROR]      ❌ Installed: No"
    fi
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        echo "[SUCCESS]    ✅ Running: Yes"
    else
        echo "[WARNING]    ⚠️  Running: No"
    fi
    
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        echo "[SUCCESS]    ✅ Health: Healthy"
    else
        echo "[WARNING]    ⚠️  Health: ${data[health_message]:-Unknown}"
    fi
    echo ""
    
    # Configuration
    if [[ "${data[installed]:-false}" == "true" ]]; then
        echo "[INFO]    ⚙️  Configuration:"
        echo "[INFO]       📦 Version: ${data[version]:-unknown}"
        
        if [[ "${data[websocket_installed]:-false}" == "true" ]]; then
            echo "[SUCCESS]    ✅ WebSocket Plugin: Installed"
            echo "[INFO]       🔌 WebSocket Port: ${data[websocket_port]:-unknown}"
            echo "[INFO]       📡 WebSocket Status: ${data[websocket_status]:-unknown}"
        else
            echo "[WARNING]    ⚠️  WebSocket Plugin: Not installed"
        fi
        
        echo "[INFO]       📁 Recording Path: ${data[recording_path]:-unknown}"
        echo "[INFO]       🎥 Recordings: ${data[recording_count]:-0} files"
        echo "[INFO]       🎬 Scenes: ${data[scene_count]:-0} configured"
        echo ""
    fi
    
    # Test results
    if [[ -n "${data[test_timestamp]}" ]]; then
        echo "[INFO]    🧪 Test Results:"
        if [[ "${data[test_status]}" == "passed" ]]; then
            echo "[SUCCESS]    ✅ Status: All tests passed"
        elif [[ "${data[test_status]}" == "failed" ]]; then
            echo "[ERROR]      ❌ Status: Some tests failed"
        else
            echo "[WARNING]    ⚠️  Status: ${data[test_status]:-unknown}"
        fi
        echo "[INFO]       📅 Last Run: ${data[test_timestamp]}"
        echo ""
    fi
    
    # Status message
    echo "[INFO]    📋 Status Message:"
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        echo "[SUCCESS]    ✅ ${data[health_message]:-unknown}"
    else
        echo "[WARNING]    ⚠️  ${data[health_message]:-unknown}"
    fi
}

# Get OBS Studio status
obs_studio_status() {
    status::run_standard "obs-studio" "obs::status::collect_data" "obs::status::display_text" "$@"
}

# Export function
export -f obs_studio_status

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    obs_studio_status "$@"
fi