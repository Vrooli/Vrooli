#!/bin/bash

# Stop script for OBS Studio
set -euo pipefail

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
OBS_STOP_DIR="${APP_ROOT}/resources/obs-studio/lib"
source "${OBS_STOP_DIR}/common.sh"

# Stop OBS Studio
stop_obs() {
    echo "[INFO] Stopping OBS Studio..."
    
    # Check if running
    if ! obs_is_running; then
        echo "[INFO] OBS Studio is not running"
        return 0
    fi
    
    # Handle mock mode
    if [[ -f "${OBS_CONFIG_DIR}/.obs-running-mock" ]]; then
        echo "[INFO] Stopping OBS Studio mock..."
        
        # Stop mock WebSocket server
        if [[ -f "${OBS_CONFIG_DIR}/mock_server.pid" ]]; then
            local mock_pid=$(cat "${OBS_CONFIG_DIR}/mock_server.pid")
            if kill -0 ${mock_pid} 2>/dev/null; then
                echo "[INFO] Stopping mock WebSocket server (PID: ${mock_pid})..."
                kill ${mock_pid} 2>/dev/null || true
                sleep 1
                # Force kill if still running
                kill -9 ${mock_pid} 2>/dev/null || true
            fi
            rm -f "${OBS_CONFIG_DIR}/mock_server.pid"
        fi
        
        rm -f "${OBS_CONFIG_DIR}/.obs-running-mock"
        echo "[SUCCESS] ✅ OBS Studio mock stopped successfully"
        return 0
    fi
    
    # Stop based on installation method
    if docker ps --format '{{.Names}}' | grep -q "^${OBS_CONTAINER_NAME}$"; then
        # Docker installation
        echo "[INFO] Stopping OBS Studio Docker container..."
        docker stop "${OBS_CONTAINER_NAME}"
        echo "[SUCCESS] OBS Studio container stopped"
        
    else
        # Native installation
        if [[ -f "${OBS_CONFIG_DIR}/obs.pid" ]]; then
            local pid=$(cat "${OBS_CONFIG_DIR}/obs.pid")
            if kill -0 ${pid} 2>/dev/null; then
                echo "[INFO] Stopping OBS Studio (PID: ${pid})..."
                kill ${pid}
                
                # Wait for process to stop
                local max_wait=10
                local waited=0
                while [[ ${waited} -lt ${max_wait} ]]; do
                    if ! kill -0 ${pid} 2>/dev/null; then
                        break
                    fi
                    sleep 1
                    ((waited++))
                done
                
                # Force kill if still running
                if kill -0 ${pid} 2>/dev/null; then
                    echo "[WARNING] Process did not stop gracefully, forcing..."
                    kill -9 ${pid}
                fi
            fi
            rm -f "${OBS_CONFIG_DIR}/obs.pid"
        fi
        
        # Also try pkill as fallback
        pkill -f "obs" 2>/dev/null || true
    fi
    
    echo "[SUCCESS] ✅ OBS Studio stopped successfully"
}

# Main function
main() {
    stop_obs
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi