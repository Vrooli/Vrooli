#!/usr/bin/env bash
# Mesa Core Library
# Implements v2.0 universal contract requirements

set -euo pipefail

# Get Mesa port from registry
source "${VROOLI_ROOT:-${HOME}/Vrooli}/scripts/resources/port_registry.sh"
readonly MESA_PORT="${RESOURCE_PORTS["mesa"]:-9512}"
readonly MESA_URL="http://localhost:${MESA_PORT}"
readonly MESA_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
readonly MESA_VENV="${MESA_DIR}/venv"
readonly MESA_DATA="${MESA_DIR}/data"
readonly MESA_LOGS="${MESA_DIR}/logs"
readonly MESA_EXAMPLES="${MESA_DIR}/examples"

# Show resource info (v2.0 requirement)
mesa::show_info() {
    local format="${1:-text}"
    
    if [[ "$format" == "--json" ]]; then
        cat "${MESA_DIR}/config/runtime.json"
    else
        echo "Mesa Agent-Based Simulation Framework"
        echo "======================================"
        jq -r '
            "Startup Order: \(.startup_order)",
            "Dependencies: \(.dependencies | join(", "))",
            "Startup Time: \(.startup_time_estimate)",
            "Timeout: \(.startup_timeout)s",
            "Recovery Attempts: \(.recovery_attempts)",
            "Priority: \(.priority)"
        ' "${MESA_DIR}/config/runtime.json" 2>/dev/null || echo "Error reading runtime.json"
    fi
    return 0
}

# Lifecycle management
mesa::manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            mesa::install "$@"
            ;;
        uninstall)
            mesa::uninstall "$@"
            ;;
        start)
            mesa::start "$@"
            ;;
        stop)
            mesa::stop "$@"
            ;;
        restart)
            mesa::restart "$@"
            ;;
        *)
            echo "Unknown manage subcommand: $subcommand"
            echo "Available: install, uninstall, start, stop, restart"
            return 1
            ;;
    esac
}

# Install Mesa and dependencies
mesa::install() {
    local force="${1:-}"
    
    echo "Installing Mesa Agent-Based Simulation Framework..."
    
    # Check if already installed
    if [[ -d "$MESA_VENV" ]] && [[ "$force" != "--force" ]]; then
        echo "Mesa already installed. Use --force to reinstall."
        return 2
    fi
    
    # Create directories
    mkdir -p "$MESA_DATA" "$MESA_LOGS" "$MESA_EXAMPLES"
    
    # Create Python virtual environment
    echo "Creating Python virtual environment..."
    
    # Try to create venv, fallback to system pip if needed
    if python3 -m venv "$MESA_VENV" 2>/dev/null; then
        echo "Virtual environment created successfully"
    else
        echo "Warning: venv creation failed, using alternative approach..."
        
        # Create structure manually
        mkdir -p "$MESA_VENV/bin" "$MESA_VENV/lib"
        
        # Create wrapper scripts that use system Python with isolated packages
        cat > "$MESA_VENV/bin/python" << PYTHON_EOF
#!/bin/bash
MESA_VENV="$MESA_VENV"
export PYTHONPATH="\${MESA_VENV}/lib:\${PYTHONPATH}"
exec python3 "\$@"
PYTHON_EOF
        chmod +x "$MESA_VENV/bin/python"
        
        # Create pip wrapper that installs to our local directory
        cat > "$MESA_VENV/bin/pip" << 'PIP_EOF'
#!/bin/bash
MESA_VENV=$(cd "$(dirname "$0")/.." && pwd)
export PYTHONPATH="${MESA_VENV}/lib:${PYTHONPATH}"
python3 -m pip install --target="${MESA_VENV}/lib" "$@"
PIP_EOF
        chmod +x "$MESA_VENV/bin/pip"
        
        echo "Created alternative environment structure"
    fi
    
    # Install Mesa and dependencies
    echo "Installing Mesa framework..."
    "${MESA_VENV}/bin/pip" install --upgrade pip
    "${MESA_VENV}/bin/pip" install mesa numpy pandas fastapi uvicorn pydantic
    
    # Validate installation
    if [[ "${2:-}" != "--skip-validation" ]]; then
        echo "Validating installation..."
        "${MESA_VENV}/bin/python" -c "import mesa; print(f'Mesa {mesa.__version__} installed successfully')"
    fi
    
    echo "Mesa installation complete!"
    return 0
}

# Uninstall Mesa
mesa::uninstall() {
    local force="${1:-}"
    local keep_data="${2:-}"
    
    if [[ "$force" != "--force" ]]; then
        read -p "Are you sure you want to uninstall Mesa? [y/N] " -n 1 -r
        echo
        [[ ! $REPLY =~ ^[Yy]$ ]] && return 1
    fi
    
    echo "Stopping Mesa service..."
    mesa::stop 2>/dev/null || true
    
    echo "Removing Mesa installation..."
    rm -rf "$MESA_VENV"
    
    if [[ "$keep_data" != "--keep-data" ]]; then
        echo "Removing data and logs..."
        rm -rf "$MESA_DATA" "$MESA_LOGS"
    fi
    
    echo "Mesa uninstalled successfully"
    return 0
}

# Start Mesa service
mesa::start() {
    local wait="${1:-}"
    
    echo "Starting Mesa service on port ${MESA_PORT}..."
    
    # Check if already running
    if mesa::is_running; then
        echo "Mesa is already running"
        return 2
    fi
    
    # Create log file
    local log_file="${MESA_LOGS}/mesa_$(date +%Y%m%d_%H%M%S).log"
    
    # Start API server (use simple_main for scaffolding)
    cd "$MESA_DIR"
    
    # Check for uvicorn and run API server
    if which uvicorn > /dev/null 2>&1; then
        # Use system uvicorn if available
        nohup python3 -m uvicorn api.robust_main:app \
            --host 0.0.0.0 \
            --port "${MESA_PORT}" \
            > "$log_file" 2>&1 &
    elif [[ -f "${MESA_VENV}/bin/uvicorn" ]]; then
        # Use venv uvicorn
        nohup "${MESA_VENV}/bin/python" -m uvicorn api.robust_main:app \
            --host 0.0.0.0 \
            --port "${MESA_PORT}" \
            > "$log_file" 2>&1 &
    else
        # Run mock server for testing
        echo "Starting Mesa in mock mode (python3-venv required for full functionality)"
        
        # Create a simple mock server using Python directly
        cat > "${MESA_DIR}/mock_server.py" << 'EOF'
import http.server
import json
from datetime import datetime

class MockMesaHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            response = {"status": "healthy", "timestamp": datetime.utcnow().isoformat()}
            self.wfile.write(json.dumps(response).encode())
        elif self.path == '/models':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            response = {"models": ["schelling", "virus", "forest_fire"]}
            self.wfile.write(json.dumps(response).encode())
        else:
            self.send_response(404)
            self.end_headers()
    
    def log_message(self, format, *args):
        return  # Suppress logs

if __name__ == '__main__':
    import sys
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 9512
    server = http.server.HTTPServer(('0.0.0.0', port), MockMesaHandler)
    server.serve_forever()
EOF
        
        nohup python3 "${MESA_DIR}/mock_server.py" "${MESA_PORT}" > "$log_file" 2>&1 &
    fi
    
    local pid=$!
    echo "$pid" > "${MESA_DIR}/.pid"
    
    if [[ "$wait" == "--wait" ]]; then
        echo "Waiting for Mesa to be ready..."
        local max_attempts=30
        local attempt=0
        
        while [[ $attempt -lt $max_attempts ]]; do
            if mesa::health_check; then
                echo "Mesa is ready!"
                return 0
            fi
            sleep 2
            ((attempt++))
        done
        
        echo "Mesa failed to start properly"
        mesa::stop
        return 1
    fi
    
    echo "Mesa started with PID $pid"
    return 0
}

# Stop Mesa service
mesa::stop() {
    echo "Stopping Mesa service..."
    
    if [[ -f "${MESA_DIR}/.pid" ]]; then
        local pid=$(cat "${MESA_DIR}/.pid")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
            rm "${MESA_DIR}/.pid"
            echo "Mesa stopped"
            return 0
        fi
    fi
    
    # Fallback: find and kill by port
    local pid=$(lsof -t -i:${MESA_PORT} 2>/dev/null || true)
    if [[ -n "$pid" ]]; then
        kill "$pid"
        echo "Mesa stopped"
        return 0
    fi
    
    echo "Mesa is not running"
    return 2
}

# Restart Mesa service
mesa::restart() {
    mesa::stop || true
    sleep 2
    mesa::start "$@"
}

# Check if Mesa is running
mesa::is_running() {
    if [[ -f "${MESA_DIR}/.pid" ]]; then
        local pid=$(cat "${MESA_DIR}/.pid")
        kill -0 "$pid" 2>/dev/null && return 0
    fi
    return 1
}

# Health check
mesa::health_check() {
    timeout 5 curl -sf "${MESA_URL}/health" > /dev/null
}

# Show status
mesa::status() {
    local format="${1:-text}"
    
    if mesa::is_running; then
        if mesa::health_check; then
            if [[ "$format" == "--json" ]]; then
                echo '{"status":"running","health":"healthy","port":'${MESA_PORT}'}'
            else
                echo "Mesa Status: Running (Healthy)"
                echo "Port: ${MESA_PORT}"
                echo "URL: ${MESA_URL}"
                echo "PID: $(cat "${MESA_DIR}/.pid" 2>/dev/null || echo 'unknown')"
            fi
        else
            if [[ "$format" == "--json" ]]; then
                echo '{"status":"running","health":"unhealthy","port":'${MESA_PORT}'}'
            else
                echo "Mesa Status: Running (Unhealthy)"
            fi
        fi
    else
        if [[ "$format" == "--json" ]]; then
            echo '{"status":"stopped","health":"n/a","port":'${MESA_PORT}'}'
        else
            echo "Mesa Status: Stopped"
        fi
    fi
    return 0
}

# View logs
mesa::logs() {
    local lines="${1:-50}"
    
    if [[ ! -d "$MESA_LOGS" ]]; then
        echo "No logs directory found"
        return 1
    fi
    
    local latest_log=$(ls -t "${MESA_LOGS}"/*.log 2>/dev/null | head -1)
    if [[ -n "$latest_log" ]]; then
        echo "=== Mesa Logs (last $lines lines) ==="
        tail -n "$lines" "$latest_log"
    else
        echo "No log files found"
        return 1
    fi
}

# Content management
mesa::content() {
    local subcommand="${1:-list}"
    shift || true
    
    case "$subcommand" in
        list)
            mesa::content_list
            ;;
        execute)
            mesa::content_execute "$@"
            ;;
        add)
            mesa::content_add "$@"
            ;;
        remove)
            mesa::content_remove "$@"
            ;;
        *)
            echo "Unknown content subcommand: $subcommand"
            echo "Available: list, execute, add, remove"
            return 1
            ;;
    esac
}

# List available models
mesa::content_list() {
    echo "Available Mesa models:"
    echo "====================="
    
    # Built-in examples
    echo -e "\nBuilt-in Models:"
    echo "  - schelling: Schelling segregation model"
    echo "  - virus: Virus spread epidemiology model"
    echo "  - wealth: Wealth distribution model"
    echo "  - forest_fire: Forest fire spread model"
    
    # Custom models
    if [[ -d "$MESA_EXAMPLES" ]] && ls "$MESA_EXAMPLES"/*.py &>/dev/null; then
        echo -e "\nCustom Models:"
        for model in "$MESA_EXAMPLES"/*.py; do
            local name=$(basename "$model" .py)
            echo "  - $name"
        done
    fi
    
    return 0
}

# Execute a model
mesa::content_execute() {
    local model="${1:-}"
    
    if [[ -z "$model" ]]; then
        echo "Error: Model name required"
        echo "Usage: resource-mesa content execute <model>"
        return 1
    fi
    
    if ! mesa::is_running; then
        echo "Error: Mesa service is not running"
        echo "Start with: resource-mesa manage start"
        return 1
    fi
    
    echo "Executing model: $model"
    
    # Call API to run simulation
    local response=$(curl -sf -X POST "${MESA_URL}/simulate" \
        -H "Content-Type: application/json" \
        -d "{\"model\": \"$model\", \"steps\": 100}" || echo "failed")
    
    if [[ "$response" == "failed" ]]; then
        echo "Failed to execute model"
        return 1
    fi
    
    echo "Simulation completed:"
    echo "$response" | jq '.' 2>/dev/null || echo "$response"
    return 0
}

# Add custom model
mesa::content_add() {
    local file="${1:-}"
    
    if [[ -z "$file" ]] || [[ ! -f "$file" ]]; then
        echo "Error: Valid Python file required"
        echo "Usage: resource-mesa content add <file.py>"
        return 1
    fi
    
    local name=$(basename "$file")
    cp "$file" "$MESA_EXAMPLES/$name"
    echo "Added model: $name"
    return 0
}

# Remove model
mesa::content_remove() {
    local model="${1:-}"
    
    if [[ -z "$model" ]]; then
        echo "Error: Model name required"
        return 1
    fi
    
    if [[ -f "$MESA_EXAMPLES/${model}.py" ]]; then
        rm "$MESA_EXAMPLES/${model}.py"
        echo "Removed model: $model"
        return 0
    else
        echo "Model not found: $model"
        return 1
    fi
}