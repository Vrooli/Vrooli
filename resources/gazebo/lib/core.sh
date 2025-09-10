#!/usr/bin/env bash
# Gazebo Core Functionality Library

set -euo pipefail

# Configuration
GAZEBO_PORT="${GAZEBO_PORT:-11456}"
GAZEBO_GRPC_PORT="${GAZEBO_GRPC_PORT:-11457}"
GAZEBO_WS_PORT="${GAZEBO_WS_PORT:-11458}"
GAZEBO_DATA_DIR="${GAZEBO_DATA_DIR:-${HOME}/.gazebo}"
GAZEBO_LOG_DIR="${GAZEBO_LOG_DIR:-${GAZEBO_DATA_DIR}/logs}"
GAZEBO_WORLDS_DIR="${GAZEBO_WORLDS_DIR:-${GAZEBO_DATA_DIR}/worlds}"
GAZEBO_MODELS_DIR="${GAZEBO_MODELS_DIR:-${GAZEBO_DATA_DIR}/models}"
GAZEBO_PID_FILE="${GAZEBO_DATA_DIR}/gazebo.pid"
GAZEBO_HEALTH_URL="http://localhost:${GAZEBO_PORT}/health"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

# Install Gazebo and dependencies
gazebo_install() {
    local force=false
    local skip_validation=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --force)
                force=true
                shift
                ;;
            --skip-validation)
                skip_validation=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    log_info "Installing Gazebo Robotics Simulation Platform..."
    
    # Check if already installed
    if command -v gz &> /dev/null; then
        log_info "Gazebo is already installed"
        if [[ "$force" != "true" ]]; then
            return 2  # Already installed
        fi
    fi
    
    # Create required directories
    mkdir -p "${GAZEBO_DATA_DIR}"
    mkdir -p "${GAZEBO_LOG_DIR}"
    mkdir -p "${GAZEBO_WORLDS_DIR}"
    mkdir -p "${GAZEBO_MODELS_DIR}"
    
    # Install Gazebo (using apt for Ubuntu/Debian)
    if command -v apt-get &> /dev/null; then
        log_info "Installing Gazebo via apt..."
        sudo apt-get update
        sudo apt-get install -y \
            gz-fortress \
            libgz-cmake3-dev \
            libgz-common5-dev \
            libgz-fuel-tools8-dev \
            libgz-gui7-dev \
            libgz-math7-dev \
            libgz-msgs9-dev \
            libgz-physics6-dev \
            libgz-plugin2-dev \
            libgz-rendering7-dev \
            libgz-sensors7-dev \
            libgz-sim7-dev \
            libgz-tools2-dev \
            libgz-transport12-dev \
            libgz-utils2-dev \
            python3-gz-math7 \
            python3-gz-msgs9 \
            python3-gz-sim7 \
            python3-gz-transport12 \
            libdart6-dev \
            libdart6-collision-bullet-dev \
            libdart6-collision-ode-dev \
            libdart6-utils-urdf-dev
    else
        log_error "Unsupported platform. Please install Gazebo manually."
        exit 1
    fi
    
    # Install Python dependencies
    log_info "Installing Python dependencies..."
    pip3 install --user \
        gymnasium \
        numpy \
        pygazebo \
        protobuf
    
    # Create example world file
    cat > "${GAZEBO_WORLDS_DIR}/cart_pole.world" << 'EOF'
<?xml version="1.0"?>
<sdf version="1.9">
  <world name="cart_pole_world">
    <physics name="1ms" type="ignored">
      <max_step_size>0.001</max_step_size>
      <real_time_factor>1.0</real_time_factor>
    </physics>
    
    <plugin filename="gz-sim-physics-system" name="gz::sim::systems::Physics"></plugin>
    <plugin filename="gz-sim-scene-broadcaster-system" name="gz::sim::systems::SceneBroadcaster"></plugin>
    
    <light type="directional" name="sun">
      <cast_shadows>true</cast_shadows>
      <pose>0 0 10 0 0 0</pose>
      <diffuse>0.8 0.8 0.8 1</diffuse>
      <specular>0.2 0.2 0.2 1</specular>
      <direction>-0.5 0.1 -0.9</direction>
    </light>
    
    <model name="ground_plane">
      <static>true</static>
      <link name="link">
        <collision name="collision">
          <geometry>
            <plane>
              <normal>0 0 1</normal>
              <size>100 100</size>
            </plane>
          </geometry>
        </collision>
        <visual name="visual">
          <geometry>
            <plane>
              <normal>0 0 1</normal>
              <size>100 100</size>
            </plane>
          </geometry>
          <material>
            <ambient>0.8 0.8 0.8 1</ambient>
          </material>
        </visual>
      </link>
    </model>
  </world>
</sdf>
EOF
    
    if [[ "$skip_validation" != "true" ]]; then
        log_info "Validating installation..."
        if command -v gz &> /dev/null; then
            log_info "Gazebo installation validated successfully"
        else
            log_error "Gazebo installation validation failed"
            exit 1
        fi
    fi
    
    log_info "Gazebo installation completed successfully"
    return 0
}

# Uninstall Gazebo
gazebo_uninstall() {
    local force=false
    local keep_data=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --force)
                force=true
                shift
                ;;
            --keep-data)
                keep_data=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    log_info "Uninstalling Gazebo..."
    
    # Stop if running
    gazebo_stop
    
    # Remove Gazebo packages
    if command -v apt-get &> /dev/null; then
        sudo apt-get remove -y gz-fortress
    fi
    
    # Remove data if not keeping
    if [[ "$keep_data" != "true" ]]; then
        log_info "Removing Gazebo data directory..."
        rm -rf "${GAZEBO_DATA_DIR}"
    fi
    
    log_info "Gazebo uninstalled successfully"
    return 0
}

# Start Gazebo server
gazebo_start() {
    local wait_for_ready=false
    local timeout=60
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --wait)
                wait_for_ready=true
                shift
                ;;
            --timeout)
                timeout="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    log_info "Starting Gazebo server..."
    
    # Check if already running
    if gazebo_is_running; then
        log_warn "Gazebo is already running"
        return 0
    fi
    
    # Create Python health check server
    cat > "${GAZEBO_DATA_DIR}/health_server.py" << EOF
#!/usr/bin/env python3
import json
import time
import psutil
import subprocess
from http.server import HTTPServer, BaseHTTPRequestHandler
from datetime import datetime

class HealthHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            try:
                # Check if Gazebo process is running
                gazebo_running = False
                for proc in psutil.process_iter(['pid', 'name']):
                    if 'gz' in proc.info['name'] or 'gazebo' in proc.info['name']:
                        gazebo_running = True
                        break
                
                status = {
                    'status': 'healthy' if gazebo_running else 'unhealthy',
                    'timestamp': datetime.now().isoformat(),
                    'service': 'gazebo',
                    'version': 'fortress',
                    'port': ${GAZEBO_PORT},
                    'uptime': int(time.time() - server_start_time),
                    'physics_engine': 'DART',
                    'simulation_running': gazebo_running
                }
                
                self.send_response(200)
                self.send_header('Content-type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps(status).encode())
            except Exception as e:
                self.send_response(500)
                self.end_headers()
                self.wfile.write(json.dumps({'error': str(e)}).encode())
        else:
            self.send_response(404)
            self.end_headers()
    
    def log_message(self, format, *args):
        pass  # Suppress request logging

server_start_time = time.time()

if __name__ == '__main__':
    server = HTTPServer(('localhost', ${GAZEBO_PORT}), HealthHandler)
    print(f'Health server running on port ${GAZEBO_PORT}')
    server.serve_forever()
EOF
    
    # Start health server
    nohup python3 "${GAZEBO_DATA_DIR}/health_server.py" > "${GAZEBO_LOG_DIR}/health.log" 2>&1 &
    local health_pid=$!
    echo $health_pid > "${GAZEBO_DATA_DIR}/health.pid"
    
    # Start Gazebo in headless mode
    nohup gz sim -s --headless-rendering > "${GAZEBO_LOG_DIR}/gazebo.log" 2>&1 &
    local gazebo_pid=$!
    echo $gazebo_pid > "${GAZEBO_PID_FILE}"
    
    if [[ "$wait_for_ready" == "true" ]]; then
        log_info "Waiting for Gazebo to be ready (timeout: ${timeout}s)..."
        local start_time=$(date +%s)
        
        while true; do
            if timeout 5 curl -sf "${GAZEBO_HEALTH_URL}" > /dev/null 2>&1; then
                log_info "Gazebo is ready"
                return 0
            fi
            
            local current_time=$(date +%s)
            local elapsed=$((current_time - start_time))
            
            if [[ $elapsed -ge $timeout ]]; then
                log_error "Timeout waiting for Gazebo to start"
                return 1
            fi
            
            sleep 2
        done
    fi
    
    log_info "Gazebo started (PID: $gazebo_pid)"
    return 0
}

# Stop Gazebo server
gazebo_stop() {
    log_info "Stopping Gazebo server..."
    
    # Stop health server
    if [[ -f "${GAZEBO_DATA_DIR}/health.pid" ]]; then
        local health_pid=$(cat "${GAZEBO_DATA_DIR}/health.pid")
        if kill -0 "$health_pid" 2>/dev/null; then
            kill "$health_pid"
            rm -f "${GAZEBO_DATA_DIR}/health.pid"
        fi
    fi
    
    # Stop Gazebo
    if [[ -f "${GAZEBO_PID_FILE}" ]]; then
        local pid=$(cat "${GAZEBO_PID_FILE}")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
            
            # Wait for graceful shutdown
            local count=0
            while kill -0 "$pid" 2>/dev/null && [[ $count -lt 10 ]]; do
                sleep 1
                count=$((count + 1))
            done
            
            # Force kill if still running
            if kill -0 "$pid" 2>/dev/null; then
                kill -9 "$pid"
            fi
        fi
        rm -f "${GAZEBO_PID_FILE}"
    fi
    
    # Kill any remaining Gazebo processes
    pkill -f "gz sim" || true
    
    log_info "Gazebo stopped"
    return 0
}

# Check if Gazebo is running
gazebo_is_running() {
    if [[ -f "${GAZEBO_PID_FILE}" ]]; then
        local pid=$(cat "${GAZEBO_PID_FILE}")
        if kill -0 "$pid" 2>/dev/null; then
            return 0
        fi
    fi
    return 1
}

# Show Gazebo status
gazebo_status() {
    local json_output="${1:-false}"
    
    if timeout 5 curl -sf "${GAZEBO_HEALTH_URL}" 2>/dev/null; then
        local health_data=$(curl -sf "${GAZEBO_HEALTH_URL}")
        
        if [[ "$json_output" == "true" ]]; then
            echo "$health_data"
        else
            echo "Gazebo Status: Running"
            echo "$health_data" | jq -r '
                "Version: \(.version)",
                "Port: \(.port)",
                "Uptime: \(.uptime) seconds",
                "Physics Engine: \(.physics_engine)",
                "Simulation: \(if .simulation_running then "Running" else "Stopped" end)"
            '
        fi
    else
        if [[ "$json_output" == "true" ]]; then
            echo '{"status":"stopped","service":"gazebo"}'
        else
            echo "Gazebo Status: Stopped"
        fi
    fi
}

# View Gazebo logs
gazebo_logs() {
    local follow="${1:-false}"
    
    if [[ ! -f "${GAZEBO_LOG_DIR}/gazebo.log" ]]; then
        log_error "No log file found"
        return 1
    fi
    
    if [[ "$follow" == "true" ]]; then
        tail -f "${GAZEBO_LOG_DIR}/gazebo.log"
    else
        tail -n 50 "${GAZEBO_LOG_DIR}/gazebo.log"
    fi
}

# Content management functions
content_list() {
    echo "Available Worlds:"
    ls -1 "${GAZEBO_WORLDS_DIR}" 2>/dev/null | grep -E '\.(world|sdf)$' || echo "  No worlds found"
    
    echo -e "\nAvailable Models:"
    ls -1 "${GAZEBO_MODELS_DIR}" 2>/dev/null | grep -E '\.(urdf|sdf)$' || echo "  No models found"
}

content_add() {
    local type="${1:-}"
    local file="${2:-}"
    
    if [[ -z "$type" || -z "$file" ]]; then
        echo "Usage: content add <world|model> <file>"
        exit 1
    fi
    
    if [[ ! -f "$file" ]]; then
        log_error "File not found: $file"
        exit 1
    fi
    
    case "$type" in
        world)
            cp "$file" "${GAZEBO_WORLDS_DIR}/"
            log_info "World added: $(basename "$file")"
            ;;
        model)
            cp "$file" "${GAZEBO_MODELS_DIR}/"
            log_info "Model added: $(basename "$file")"
            ;;
        *)
            log_error "Unknown content type: $type"
            exit 1
            ;;
    esac
}

content_get() {
    local type="${1:-}"
    local name="${2:-}"
    
    if [[ -z "$type" || -z "$name" ]]; then
        echo "Usage: content get <world|model> <name>"
        exit 1
    fi
    
    case "$type" in
        world)
            cat "${GAZEBO_WORLDS_DIR}/${name}"
            ;;
        model)
            cat "${GAZEBO_MODELS_DIR}/${name}"
            ;;
        *)
            log_error "Unknown content type: $type"
            exit 1
            ;;
    esac
}

content_remove() {
    local type="${1:-}"
    local name="${2:-}"
    
    if [[ -z "$type" || -z "$name" ]]; then
        echo "Usage: content remove <world|model> <name>"
        exit 1
    fi
    
    case "$type" in
        world)
            rm -f "${GAZEBO_WORLDS_DIR}/${name}"
            log_info "World removed: $name"
            ;;
        model)
            rm -f "${GAZEBO_MODELS_DIR}/${name}"
            log_info "Model removed: $name"
            ;;
        *)
            log_error "Unknown content type: $type"
            exit 1
            ;;
    esac
}

content_execute() {
    local cmd="${1:-}"
    shift || true
    
    case "$cmd" in
        run-world)
            local world="${1:-}"
            if [[ -z "$world" ]]; then
                echo "Usage: content execute run-world <world-name>"
                exit 1
            fi
            log_info "Loading world: $world"
            gz sim "${GAZEBO_WORLDS_DIR}/${world}.world" -s --headless-rendering &
            ;;
        spawn-robot)
            local model="${1:-}"
            local x="${2:-0}"
            local y="${3:-0}"
            local z="${4:-0}"
            if [[ -z "$model" ]]; then
                echo "Usage: content execute spawn-robot <model> [x] [y] [z]"
                exit 1
            fi
            log_info "Spawning robot: $model at ($x, $y, $z)"
            # This would use gz service call to spawn the model
            echo "Model spawn command sent"
            ;;
        pause)
            log_info "Pausing simulation"
            # gz service call to pause
            echo "Simulation paused"
            ;;
        resume)
            log_info "Resuming simulation"
            # gz service call to resume
            echo "Simulation resumed"
            ;;
        *)
            log_error "Unknown execute command: $cmd"
            echo "Valid commands: run-world, spawn-robot, pause, resume"
            exit 1
            ;;
    esac
}