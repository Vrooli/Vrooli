#!/usr/bin/env bash
# Mathlib Resource - Core Library Functions

set -euo pipefail

# Source configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../config/defaults.sh"

# Display resource information from runtime.json
mathlib::info() {
    local json_flag="${1:-}"
    local runtime_file="${SCRIPT_DIR}/../config/runtime.json"
    
    if [[ ! -f "${runtime_file}" ]]; then
        echo "Error: runtime.json not found"
        return 1
    fi
    
    if [[ "${json_flag}" == "--json" ]]; then
        cat "${runtime_file}"
    else
        echo "Mathlib Resource Information:"
        echo "=============================="
        jq -r '
            "Resource: " + .resource,
            "Version: " + .version,
            "Description: " + .description,
            "Category: " + .category,
            "Startup Order: " + (.startup_order | tostring),
            "Dependencies: " + (.dependencies | join(", ")),
            "Startup Timeout: " + (.startup_timeout | tostring) + "s",
            "Startup Time Estimate: " + .startup_time_estimate,
            "Recovery Attempts: " + (.recovery_attempts | tostring),
            "Priority: " + .priority
        ' "${runtime_file}"
    fi
}

# Lifecycle management
mathlib::manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "${subcommand}" in
        install)
            mathlib::install "$@"
            ;;
        uninstall)
            mathlib::uninstall "$@"
            ;;
        start)
            mathlib::start "$@"
            ;;
        stop)
            mathlib::stop "$@"
            ;;
        restart)
            mathlib::restart "$@"
            ;;
        *)
            echo "Error: Unknown manage command '${subcommand}'"
            echo "Valid commands: install, uninstall, start, stop, restart"
            return 1
            ;;
    esac
}

# Install Lean 4 and dependencies
mathlib::install() {
    echo "Installing Mathlib (Lean 4) resource..."
    
    # Create directories
    mkdir -p "${MATHLIB_INSTALL_DIR}"
    mkdir -p "${MATHLIB_WORK_DIR}"
    mkdir -p "${MATHLIB_CACHE_DIR}"
    
    # Check if already installed
    if [[ -f "${MATHLIB_INSTALL_DIR}/.installed" ]]; then
        echo "Mathlib is already installed"
        return 2  # Already installed
    fi
    
    echo "Note: Full Lean 4 and Mathlib installation will be implemented by improvers"
    echo "Creating minimal installation marker for scaffolding..."
    
    # Create installation marker
    touch "${MATHLIB_INSTALL_DIR}/.installed"
    
    echo "Mathlib resource installed successfully (scaffolding only)"
    return 0
}

# Uninstall the resource
mathlib::uninstall() {
    echo "Uninstalling Mathlib resource..."
    
    # Stop service if running
    mathlib::stop 2>/dev/null || true
    
    # Remove installation directory
    if [[ -d "${MATHLIB_INSTALL_DIR}" ]]; then
        rm -rf "${MATHLIB_INSTALL_DIR}"
        echo "Removed installation directory"
    fi
    
    # Optionally keep work and cache directories
    local keep_data="${1:-}"
    if [[ "${keep_data}" != "--keep-data" ]]; then
        rm -rf "${MATHLIB_WORK_DIR}"
        rm -rf "${MATHLIB_CACHE_DIR}"
        echo "Removed work and cache directories"
    fi
    
    echo "Mathlib resource uninstalled successfully"
    return 0
}

# Start the service
mathlib::start() {
    echo "Starting Mathlib service..."
    
    # Clean up any stale processes
    pkill -f "mathlib_health.py" 2>/dev/null || true
    rm -f "${MATHLIB_PID_FILE}" 2>/dev/null || true
    
    # Check if already running
    if mathlib::is_running; then
        echo "Mathlib service is already running"
        return 0
    fi
    
    # Create simple health check server for scaffolding
    cat > /tmp/mathlib_health.py << 'EOF'
#!/usr/bin/env python3
from http.server import HTTPServer, BaseHTTPRequestHandler
import json
import os
import sys
import socket

PORT = int(os.environ.get('MATHLIB_PORT', 11458))

class HealthHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            response = {
                'status': 'healthy',
                'service': 'mathlib',
                'version': '0.1.0-scaffold',
                'lean': 'not-installed',
                'mathlib': 'not-installed'
            }
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps(response).encode())
        else:
            self.send_response(404)
            self.end_headers()
    
    def log_message(self, format, *args):
        pass  # Suppress logs

try:
    server = HTTPServer(('localhost', PORT), HealthHandler)
    server.socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    print(f"Mathlib health server started on port {PORT}")
    server.serve_forever()
except OSError as e:
    print(f"Error: Port {PORT} is already in use or not available: {e}")
    sys.exit(1)
except KeyboardInterrupt:
    print("Server stopped")
    sys.exit(0)
EOF
    
    # Start the health server
    nohup python3 /tmp/mathlib_health.py > "${MATHLIB_LOG_FILE}" 2>&1 &
    local pid=$!
    echo "${pid}" > "${MATHLIB_PID_FILE}"
    
    # Wait for startup
    local wait_flag="${1:-}"
    if [[ "${wait_flag}" == "--wait" ]]; then
        echo "Waiting for service to be healthy..."
        if mathlib::wait_for_health; then
            echo "Mathlib service started successfully"
            return 0
        else
            echo "Error: Service failed to become healthy"
            mathlib::stop
            return 1
        fi
    fi
    
    echo "Mathlib service started (PID: ${pid})"
    return 0
}

# Stop the service
mathlib::stop() {
    echo "Stopping Mathlib service..."
    
    if [[ -f "${MATHLIB_PID_FILE}" ]]; then
        local pid=$(cat "${MATHLIB_PID_FILE}")
        if kill -0 "${pid}" 2>/dev/null; then
            kill "${pid}"
            rm -f "${MATHLIB_PID_FILE}"
            echo "Mathlib service stopped"
        else
            echo "Service not running (stale PID file)"
            rm -f "${MATHLIB_PID_FILE}"
        fi
    else
        echo "Mathlib service is not running"
    fi
    
    return 0
}

# Restart the service
mathlib::restart() {
    mathlib::stop
    sleep 2
    mathlib::start "$@"
}

# Check if service is running
mathlib::is_running() {
    if [[ -f "${MATHLIB_PID_FILE}" ]]; then
        local pid=$(cat "${MATHLIB_PID_FILE}")
        if kill -0 "${pid}" 2>/dev/null; then
            return 0
        fi
    fi
    return 1
}

# Wait for service to be healthy
mathlib::wait_for_health() {
    local timeout="${MATHLIB_STARTUP_TIMEOUT}"
    local elapsed=0
    
    while [[ ${elapsed} -lt ${timeout} ]]; do
        if timeout 5 curl -sf "${MATHLIB_HEALTH_ENDPOINT}" > /dev/null 2>&1; then
            return 0
        fi
        sleep 2
        elapsed=$((elapsed + 2))
    done
    
    return 1
}

# Run tests
mathlib::test() {
    local test_type="${1:-all}"
    
    case "${test_type}" in
        smoke|integration|unit|all)
            source "${SCRIPT_DIR}/test.sh"
            mathlib::test::run "${test_type}"
            ;;
        *)
            echo "Error: Unknown test type '${test_type}'"
            echo "Valid types: smoke, integration, unit, all"
            return 1
            ;;
    esac
}

# Content management (future implementation)
mathlib::content() {
    local subcommand="${1:-}"
    shift || true
    
    case "${subcommand}" in
        list)
            echo "Content listing not yet implemented"
            echo "This will list available Lean tactics and libraries"
            ;;
        execute)
            echo "Proof execution not yet implemented"
            echo "This will execute Lean proof files"
            ;;
        *)
            echo "Error: Unknown content command '${subcommand}'"
            echo "Valid commands: list, execute"
            return 1
            ;;
    esac
}

# Show service status
mathlib::status() {
    echo "Mathlib Service Status"
    echo "====================="
    
    if mathlib::is_running; then
        echo "Status: Running"
        if [[ -f "${MATHLIB_PID_FILE}" ]]; then
            echo "PID: $(cat "${MATHLIB_PID_FILE}")"
        fi
        
        # Check health
        if timeout 5 curl -sf "${MATHLIB_HEALTH_ENDPOINT}" > /dev/null 2>&1; then
            echo "Health: Healthy"
            
            # Show health details
            local health_data=$(timeout 5 curl -sf "${MATHLIB_HEALTH_ENDPOINT}" 2>/dev/null || echo "{}")
            echo "Details:"
            echo "${health_data}" | jq -r '
                "  Version: " + .version,
                "  Lean: " + .lean,
                "  Mathlib: " + .mathlib
            ' 2>/dev/null || echo "  Unable to parse health data"
        else
            echo "Health: Unhealthy"
        fi
    else
        echo "Status: Stopped"
    fi
    
    echo ""
    echo "Configuration:"
    echo "  Port: ${MATHLIB_PORT}"
    echo "  Work Dir: ${MATHLIB_WORK_DIR}"
    echo "  Cache Dir: ${MATHLIB_CACHE_DIR}"
}

# View logs
mathlib::logs() {
    local tail_flag="${1:-}"
    local lines="${2:-50}"
    
    if [[ ! -f "${MATHLIB_LOG_FILE}" ]]; then
        echo "No logs available"
        return 0
    fi
    
    if [[ "${tail_flag}" == "--tail" ]]; then
        tail -n "${lines}" "${MATHLIB_LOG_FILE}"
    else
        cat "${MATHLIB_LOG_FILE}"
    fi
}

# Display credentials (not applicable for this resource)
mathlib::credentials() {
    echo "Mathlib does not require credentials for basic usage"
    echo "API Key can be configured via MATHLIB_API_KEY environment variable"
}