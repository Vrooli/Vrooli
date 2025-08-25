#!/bin/bash

# Start script for OBS Studio
set -euo pipefail

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
OBS_START_DIR="${APP_ROOT}/resources/obs-studio/lib"
source "${OBS_START_DIR}/common.sh"

# Start OBS Studio
start_obs() {
    echo "[INFO] Starting OBS Studio..."
    
    # Check if already running
    if obs_is_running; then
        echo "[INFO] OBS Studio is already running"
        return 0
    fi
    
    # Check if installed
    if ! obs_is_installed; then
        echo "[ERROR] OBS Studio is not installed. Run 'vrooli resource obs-studio install' first"
        return 1
    fi
    
    # Handle mock mode
    if [[ -f "${OBS_CONFIG_DIR}/.obs-installed-mock" ]]; then
        echo "[INFO] Starting OBS Studio in mock mode..."
        touch "${OBS_CONFIG_DIR}/.obs-running-mock"
        
        # Start mock WebSocket server
        echo "[INFO] Starting mock WebSocket server on port ${OBS_PORT}..."
        if [[ -f "${OBS_LIB_DIR}/mock_websocket_server.py" ]]; then
            # Check if already running
            if ! lsof -i:${OBS_PORT} >/dev/null 2>&1; then
                nohup python3 "${OBS_LIB_DIR}/mock_websocket_server.py" \
                    > "${OBS_CONFIG_DIR}/mock_server.log" 2>&1 &
                local mock_pid=$!
                echo ${mock_pid} > "${OBS_CONFIG_DIR}/mock_server.pid"
                echo "[INFO] Mock WebSocket server started with PID: ${mock_pid}"
                
                # Wait for server to be ready
                local max_attempts=10
                local attempt=0
                while [[ ${attempt} -lt ${max_attempts} ]]; do
                    if timeout 2 nc -zv localhost ${OBS_PORT} >/dev/null 2>&1; then
                        echo "[SUCCESS] Mock WebSocket server ready on port ${OBS_PORT}"
                        break
                    fi
                    sleep 1
                    ((attempt++))
                done
                
                if [[ ${attempt} -ge ${max_attempts} ]]; then
                    echo "[WARNING] Mock WebSocket server may not be fully ready"
                fi
            else
                echo "[INFO] Mock WebSocket server already running on port ${OBS_PORT}"
            fi
        fi
        
        echo "[SUCCESS] ✅ OBS Studio mock started successfully"
        return 0
    fi
    
    # Ensure configuration exists
    obs_create_default_config
    obs_create_default_scenes
    
    # Start based on installation method
    if docker ps -a --format '{{.Names}}' | grep -q "^${OBS_CONTAINER_NAME}$"; then
        # Docker installation
        echo "[INFO] Starting OBS Studio Docker container..."
        docker start "${OBS_CONTAINER_NAME}"
        
        # Wait for container to be ready
        local max_attempts=30
        local attempt=0
        while [[ ${attempt} -lt ${max_attempts} ]]; do
            if docker ps --format '{{.Names}}' | grep -q "^${OBS_CONTAINER_NAME}$"; then
                echo "[SUCCESS] OBS Studio container started"
                break
            fi
            sleep 2
            ((attempt++))
        done
        
        if [[ ${attempt} -ge ${max_attempts} ]]; then
            echo "[ERROR] Failed to start OBS Studio container"
            return 1
        fi
        
    else
        # Native installation - start in background
        echo "[INFO] Starting OBS Studio (native)..."
        
        # Start OBS with WebSocket plugin enabled
        nohup obs \
            --minimize-to-tray \
            --startstreaming=false \
            --startrecording=false \
            > "${OBS_CONFIG_DIR}/obs.log" 2>&1 &
        
        local obs_pid=$!
        echo ${obs_pid} > "${OBS_CONFIG_DIR}/obs.pid"
        
        echo "[INFO] OBS Studio started with PID: ${obs_pid}"
    fi
    
    # Wait for WebSocket to be ready
    echo "[INFO] Waiting for WebSocket API to be ready..."
    local max_wait=60
    local waited=0
    
    while [[ ${waited} -lt ${max_wait} ]]; do
        if obs_websocket_healthy; then
            echo "[SUCCESS] WebSocket API is ready on port ${OBS_PORT}"
            break
        fi
        sleep 2
        ((waited+=2))
    done
    
    if [[ ${waited} -ge ${max_wait} ]]; then
        echo "[WARNING] WebSocket API did not become ready in time"
        echo "[INFO] OBS Studio is running but WebSocket may need manual configuration"
    fi
    
    echo "[SUCCESS] ✅ OBS Studio started successfully"
}

# Main function
main() {
    start_obs
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi