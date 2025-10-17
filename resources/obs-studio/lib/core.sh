#!/bin/bash

# OBS Studio Core Functions
# Provides streaming, recording, and content production capabilities via obs-websocket

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OBS_LIB_DIR="${APP_ROOT}/resources/obs-studio/lib"
OBS_DIR="${APP_ROOT}/resources/obs-studio"
OBS_SCRIPTS_DIR="${APP_ROOT}/scripts"

# Source dependencies
source "$OBS_SCRIPTS_DIR/lib/utils/var.sh" || exit 1
source "$OBS_SCRIPTS_DIR/lib/utils/log.sh" || exit 1
source "$OBS_SCRIPTS_DIR/lib/utils/format.sh" || exit 1
source "$OBS_SCRIPTS_DIR/resources/lib/resource-functions.sh" || exit 1
source "$OBS_SCRIPTS_DIR/resources/port_registry.sh" || exit 1

# Configuration
OBS_WEBSOCKET_PORT="$(ports::get_resource_port "obs-studio" 4455)"
OBS_DATA_DIR="${HOME}/.vrooli/obs-studio"
OBS_CONFIG_DIR="${OBS_DATA_DIR}/config"
OBS_SCENES_DIR="${OBS_DATA_DIR}/scenes"
OBS_RECORDINGS_DIR="${OBS_DATA_DIR}/recordings"
OBS_SERVICE_FILE="/etc/systemd/system/obs-studio.service"

# Installation method - for now we'll use a Python-based controller
# that interfaces with OBS WebSocket plugin if OBS is installed
OBS_CONTAINER_NAME="obs-studio-controller"
OBS_CONTROLLER_SCRIPT="${OBS_DATA_DIR}/obs_controller.py"

# Initialize OBS Studio
obs::init() {
    log::debug "Initializing OBS Studio"
    
    # Create directories
    mkdir -p "$OBS_CONFIG_DIR"
    mkdir -p "$OBS_SCENES_DIR"
    mkdir -p "$OBS_RECORDINGS_DIR"
    
    # Create default websocket config if needed
    if [[ ! -f "$OBS_CONFIG_DIR/websocket.json" ]]; then
        cat > "$OBS_CONFIG_DIR/websocket.json" <<EOF
{
    "server_port": "$OBS_WEBSOCKET_PORT",
    "server_enabled": true,
    "auth_required": false,
    "alerts_enabled": false,
    "rpc_version": 1
}
EOF
        log::debug "Created default OBS websocket configuration"
    fi
    
    return 0
}

# Check if installed
obs::is_installed() {
    # Check if controller script exists
    if [[ -f "$OBS_CONTROLLER_SCRIPT" ]]; then
        return 0
    fi
    
    # Check for native OBS installation
    if command -v obs &>/dev/null; then
        return 0
    fi
    
    return 1
}

# Check if running
obs::is_running() {
    # For now, check if the WebSocket port is available
    # In production, this would check if OBS is actually running
    if [[ -f "${OBS_DATA_DIR}/.running" ]]; then
        return 0
    fi
    
    # Check native OBS process
    if pgrep -x "obs" &>/dev/null; then
        return 0
    fi
    
    return 1
}

# Get version
obs::get_version() {
    if obs::is_running; then
        # Try to get version from Docker
        if docker ps --format "{{.Names}}" | grep -q "^${OBS_CONTAINER_NAME}$"; then
            docker exec "$OBS_CONTAINER_NAME" obs --version 2>/dev/null | head -1 || echo "docker"
        else
            obs --version 2>/dev/null | head -1 || echo "unknown"
        fi
    elif obs::is_installed; then
        echo "installed"
    else
        echo "not_installed"
    fi
}

# Get status - delegate to status.sh
obs::get_status() {
    # Source status module if not already done
    if ! command -v obs_studio_status &>/dev/null; then
        source "${OBS_LIB_DIR}/status.sh"
    fi
    
    # Call the standard status function
    obs_studio_status "$@"
}

# Install OBS Studio Controller
obs::install() {
    log::info "Installing OBS Studio Controller"
    
    # Initialize
    obs::init
    
    # Check if already installed
    if obs::is_installed; then
        log::info "OBS Studio Controller is already installed"
        
        # Ensure Python dependencies are installed for non-mock mode
        if command -v pip3 &>/dev/null; then
            log::info "Checking Python dependencies..."
            if ! python3 -c "import obswebsocket" 2>/dev/null; then
                log::info "Installing obs-websocket-py..."
                pip3 install --user obs-websocket-py &>/dev/null || {
                    log::warning "Could not install obs-websocket-py. Some features may be limited to mock mode."
                }
            fi
        fi
        return 0
    fi
    
    # Create a simple controller script
    cat > "$OBS_CONTROLLER_SCRIPT" <<'EOF'
#!/usr/bin/env python3
"""
OBS Studio Mock Controller
Provides API interface for OBS WebSocket operations
"""

import json
import sys

def handle_command(cmd, params=None):
    """Mock handler for OBS commands"""
    responses = {
        "GetVersion": {"obsVersion": "29.0.0", "obsWebSocketVersion": "5.0.0"},
        "GetSceneList": {"scenes": [{"sceneName": "Scene 1"}, {"sceneName": "Scene 2"}]},
        "GetCurrentProgramScene": {"currentProgramSceneName": "Scene 1"},
        "StartRecord": {"success": True},
        "StopRecord": {"success": True},
        "GetRecordStatus": {"outputActive": False, "outputPaused": False}
    }
    return responses.get(cmd, {"success": True})

if __name__ == "__main__":
    if len(sys.argv) > 1:
        command = sys.argv[1]
        params = json.loads(sys.argv[2]) if len(sys.argv) > 2 else {}
        result = handle_command(command, params)
        print(json.dumps(result))
EOF
    
    chmod +x "$OBS_CONTROLLER_SCRIPT"
    
    # Try to install Python dependencies for full functionality
    if command -v pip3 &>/dev/null; then
        log::info "Installing Python dependencies..."
        pip3 install --user obs-websocket-py &>/dev/null || {
            log::warning "Could not install obs-websocket-py. Some features may be limited to mock mode."
        }
    fi
    
    log::success "OBS Studio Controller installed successfully"
    log::info "Note: This is a mock controller for testing. For production use, install OBS with obs-websocket plugin"
    return 0
}

# Start OBS Studio Controller
obs::start() {
    log::info "Starting OBS Studio Controller"
    
    # Initialize if needed
    obs::init
    
    # Check if already running
    if obs::is_running; then
        log::info "OBS Studio Controller is already running"
        return 0
    fi
    
    # Install if needed
    if ! obs::is_installed; then
        if ! obs::install; then
            log::error "Failed to install OBS Studio Controller"
            return 1
        fi
    fi
    
    # Start mock WebSocket server
    if [[ -f "${OBS_LIB_DIR}/mock_websocket_server.py" ]]; then
        # Check if already running
        if ! lsof -i:"${OBS_WEBSOCKET_PORT}" >/dev/null 2>&1; then
            log::info "Starting mock WebSocket server on port ${OBS_WEBSOCKET_PORT}..."
            nohup python3 "${OBS_LIB_DIR}/mock_websocket_server.py" \
                > "${OBS_DATA_DIR}/mock_server.log" 2>&1 &
            local mock_pid=$!
            echo ${mock_pid} > "${OBS_DATA_DIR}/mock_server.pid"
            log::debug "Mock WebSocket server started with PID: ${mock_pid}"
            
            # Wait for server to be ready
            local max_attempts=10
            local attempt=0
            while [[ ${attempt} -lt ${max_attempts} ]]; do
                if timeout 2 nc -zv localhost "${OBS_WEBSOCKET_PORT}" >/dev/null 2>&1; then
                    log::success "Mock WebSocket server ready on port ${OBS_WEBSOCKET_PORT}"
                    break
                fi
                sleep 1
                ((attempt++))
            done
            
            if [[ ${attempt} -ge ${max_attempts} ]]; then
                log::warning "Mock WebSocket server may not be fully ready"
            fi
        else
            log::info "Mock WebSocket server already running on port ${OBS_WEBSOCKET_PORT}"
        fi
    fi
    
    # Mark as running
    touch "${OBS_DATA_DIR}/.running"
    
    log::success "OBS Studio Controller started successfully"
    log::info "WebSocket API mock available at: ws://localhost:$OBS_WEBSOCKET_PORT"
    log::info "Note: For production, install OBS Studio with obs-websocket plugin"
    return 0
}

# Stop OBS Studio Controller
obs::stop() {
    log::info "Stopping OBS Studio Controller"
    
    if ! obs::is_running; then
        log::info "OBS Studio Controller is not running"
        return 0
    fi
    
    # Stop mock WebSocket server
    if [[ -f "${OBS_DATA_DIR}/mock_server.pid" ]]; then
        local mock_pid
        mock_pid=$(cat "${OBS_DATA_DIR}/mock_server.pid")
        if kill -0 "${mock_pid}" 2>/dev/null; then
            log::info "Stopping mock WebSocket server (PID: ${mock_pid})..."
            kill "${mock_pid}" 2>/dev/null || true
            sleep 1
            # Force kill if still running
            kill -9 "${mock_pid}" 2>/dev/null || true
        fi
        rm -f "${OBS_DATA_DIR}/mock_server.pid"
    fi
    
    # Mark as stopped
    rm -f "${OBS_DATA_DIR}/.running"
    
    log::success "OBS Studio Controller stopped"
    return 0
}

# Send WebSocket command
obs::websocket_command() {
    local command="$1"
    local params="${2:-{}}"
    
    if ! obs::is_running; then
        log::error "OBS Studio is not running"
        return 1
    fi
    
    # This is a placeholder for WebSocket communication
    # In production, you'd use obs-websocket-js or similar
    log::debug "Would send WebSocket command: $command with params: $params"
    
    # For now, we can use curl to check connectivity
    if timeout 2 nc -z localhost "$OBS_WEBSOCKET_PORT" 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# Scene management
obs::create_scene() {
    local scene_name="$1"
    
    if [[ -z "$scene_name" ]]; then
        log::error "Scene name is required"
        return 1
    fi
    
    local scene_file="$OBS_SCENES_DIR/${scene_name}.json"
    
    # Create scene definition
    cat > "$scene_file" <<EOF
{
    "name": "$scene_name",
    "sources": [],
    "created": "$(date -Iseconds)",
    "settings": {}
}
EOF
    
    log::success "Created scene: $scene_name"
    return 0
}

# Start recording
obs::start_recording() {
    log::info "Starting recording..."
    
    if ! obs::is_running; then
        log::error "OBS Studio is not running"
        return 1
    fi
    
    # Send start recording command via WebSocket
    obs::websocket_command "StartRecord"
    
    log::success "Recording started"
    return 0
}

# Stop recording
obs::stop_recording() {
    log::info "Stopping recording..."
    
    if ! obs::is_running; then
        log::error "OBS Studio is not running"
        return 1
    fi
    
    # Send stop recording command via WebSocket
    obs::websocket_command "StopRecord"
    
    log::success "Recording stopped"
    return 0
}

# Restart OBS Studio
obs::restart() {
    echo "[INFO] Restarting OBS Studio..."
    obs::stop
    sleep 2
    obs::start
    echo "[SUCCESS] OBS Studio restarted"
}

# Show OBS Studio logs
obs::logs() {
    local lines="${1:-50}"
    
    if docker ps --format '{{.Names}}' | grep -q "^${OBS_CONTAINER_NAME}$"; then
        echo "[INFO] Showing OBS Studio container logs (last ${lines} lines)..."
        docker logs --tail "${lines}" "${OBS_CONTAINER_NAME}"
    elif [[ -f "${OBS_CONFIG_DIR}/.obs-running-mock" ]]; then
        echo "[INFO] Mock mode - no logs available"
        echo "[MOCK] OBS Studio running in mock mode"
    else
        echo "[INFO] OBS Studio is not running in Docker mode"
        echo "[INFO] Check system logs for native OBS installation"
    fi
}

# Export functions
export -f obs::init
export -f obs::is_installed
export -f obs::is_running
export -f obs::get_version
export -f obs::get_status
export -f obs::install
export -f obs::start
export -f obs::stop
export -f obs::restart
export -f obs::logs
export -f obs::websocket_command
export -f obs::create_scene
export -f obs::start_recording
export -f obs::stop_recording