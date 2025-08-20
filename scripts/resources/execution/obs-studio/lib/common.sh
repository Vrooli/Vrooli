#!/bin/bash

# Common functions for OBS Studio resource
set -euo pipefail

# Get script directory
OBS_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OBS_RESOURCE_DIR="$(dirname "${OBS_LIB_DIR}")"
PROJECT_ROOT="$(cd "${OBS_LIB_DIR}/../../../../.." && pwd)"

# Source shared utilities
source "${PROJECT_ROOT}/scripts/lib/utils/var.sh"
source "${PROJECT_ROOT}/scripts/lib/utils/format.sh"

# Configuration
OBS_CONFIG_DIR="${HOME}/.vrooli/obs-studio"
OBS_CONFIG_FILE="${OBS_CONFIG_DIR}/config.json"
OBS_PORT="${OBS_PORT:-4455}"
OBS_PASSWORD_FILE="${OBS_CONFIG_DIR}/websocket-password"
OBS_SCENE_CONFIG="${OBS_CONFIG_DIR}/scenes.json"
OBS_RECORDINGS_DIR="${HOME}/Videos/obs-recordings"
OBS_CONTAINER_NAME="obs-studio-main"
OBS_WEBSOCKET_VERSION="5.5.0"

# Ensure configuration directory exists
mkdir -p "${OBS_CONFIG_DIR}"
mkdir -p "${OBS_RECORDINGS_DIR}"

# Generate password if not exists
obs_generate_password() {
    if [[ ! -f "${OBS_PASSWORD_FILE}" ]]; then
        openssl rand -base64 32 > "${OBS_PASSWORD_FILE}"
        chmod 600 "${OBS_PASSWORD_FILE}"
    fi
    cat "${OBS_PASSWORD_FILE}"
}

# Check if OBS is installed
obs_is_installed() {
    # Check for mock installation
    if [[ -f "${OBS_CONFIG_DIR}/.obs-installed-mock" ]]; then
        return 0
    elif command -v obs >/dev/null 2>&1; then
        return 0
    elif docker ps -a --format '{{.Names}}' | grep -q "^${OBS_CONTAINER_NAME}$"; then
        return 0
    else
        return 1
    fi
}

# Check if OBS is running
obs_is_running() {
    # Check for mock running state
    if [[ -f "${OBS_CONFIG_DIR}/.obs-running-mock" ]]; then
        return 0
    fi
    
    # Check native installation
    if pgrep -f "obs" >/dev/null 2>&1; then
        return 0
    fi
    
    # Check Docker container
    if docker ps --format '{{.Names}}' | grep -q "^${OBS_CONTAINER_NAME}$"; then
        return 0
    fi
    
    return 1
}

# Check WebSocket connectivity
obs_websocket_healthy() {
    # Check for mock healthy state
    if [[ -f "${OBS_CONFIG_DIR}/.obs-running-mock" ]]; then
        return 0
    fi
    
    local password
    password=$(obs_generate_password)
    
    # Try to connect to WebSocket
    timeout 5 curl -s \
        -H "Connection: Upgrade" \
        -H "Upgrade: websocket" \
        -H "Sec-WebSocket-Version: 13" \
        -H "Sec-WebSocket-Key: $(openssl rand -base64 16)" \
        "http://localhost:${OBS_PORT}" >/dev/null 2>&1
    
    return $?
}

# Get OBS version
obs_get_version() {
    if [[ -f "${OBS_CONFIG_DIR}/.obs-installed-mock" ]]; then
        echo "mock-30.0.0"
    elif command -v obs >/dev/null 2>&1; then
        obs --version 2>/dev/null | head -1 || echo "unknown"
    elif docker exec "${OBS_CONTAINER_NAME}" obs --version 2>/dev/null; then
        docker exec "${OBS_CONTAINER_NAME}" obs --version 2>/dev/null | head -1
    else
        echo "not installed"
    fi
}

# Check if WebSocket plugin is installed
obs_websocket_installed() {
    # Check for mock plugin
    if [[ -f "${OBS_CONFIG_DIR}/plugins/obs-websocket-mock" ]]; then
        return 0
    fi
    
    # Check native plugin directory
    if [[ -d "${HOME}/.config/obs-studio/plugins/obs-websocket" ]]; then
        return 0
    fi
    
    # Check in container
    if docker exec "${OBS_CONTAINER_NAME}" test -d /config/obs-studio/plugins/obs-websocket 2>/dev/null; then
        return 0
    fi
    
    return 1
}

# Create default configuration
obs_create_default_config() {
    if [[ ! -f "${OBS_CONFIG_FILE}" ]]; then
        cat > "${OBS_CONFIG_FILE}" <<EOF
{
    "websocket": {
        "port": ${OBS_PORT},
        "password_file": "${OBS_PASSWORD_FILE}",
        "enabled": true
    },
    "recording": {
        "path": "${OBS_RECORDINGS_DIR}",
        "format": "mkv",
        "quality": "high"
    },
    "streaming": {
        "enabled": false,
        "service": "",
        "key": ""
    },
    "scenes": {
        "default": "Main",
        "auto_switch": false
    }
}
EOF
    fi
}

# Create default scenes configuration
obs_create_default_scenes() {
    if [[ ! -f "${OBS_SCENE_CONFIG}" ]]; then
        cat > "${OBS_SCENE_CONFIG}" <<EOF
{
    "scenes": [
        {
            "name": "Main",
            "sources": [
                {
                    "name": "Display Capture",
                    "type": "monitor_capture",
                    "enabled": true
                }
            ]
        },
        {
            "name": "Webcam",
            "sources": [
                {
                    "name": "Camera",
                    "type": "v4l2_input",
                    "enabled": true
                }
            ]
        },
        {
            "name": "Screen Share",
            "sources": [
                {
                    "name": "Window Capture",
                    "type": "window_capture",
                    "enabled": true
                }
            ]
        }
    ]
}
EOF
    fi
}