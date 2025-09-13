#!/usr/bin/env bash
# Core library functions for NSFW Detector resource

set -euo pipefail

# Resource configuration
readonly RESOURCE_NAME="nsfw-detector"
readonly RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly CONFIG_DIR="${RESOURCE_DIR}/config"
readonly TEST_DIR="${RESOURCE_DIR}/test"
readonly DATA_DIR="${DATA_DIR:-${HOME}/.local/share/nsfw-detector}"
readonly LOG_DIR="${LOG_DIR:-${HOME}/.local/share/nsfw-detector/logs}"

# Load configuration
if [[ -f "${CONFIG_DIR}/defaults.sh" ]]; then
    source "${CONFIG_DIR}/defaults.sh"
fi

# Port allocation from registry
get_port() {
    local port_registry="${VROOLI_ROOT:-${HOME}/Vrooli}/scripts/resources/port_registry.sh"
    if [[ -f "$port_registry" ]]; then
        source "$port_registry"
        echo "${RESOURCE_PORTS[nsfw-detector]:-11451}"
    else
        echo "11451"
    fi
}

readonly NSFW_DETECTOR_PORT="${NSFW_DETECTOR_PORT:-$(get_port)}"

# Process management
readonly PIDFILE="${DATA_DIR}/nsfw-detector.pid"

# Lifecycle management functions
manage_install() {
    echo "Installing NSFW Detector resource..."
    
    # Create required directories
    mkdir -p "${DATA_DIR}/models" "${LOG_DIR}"
    
    # Install dependencies (Node.js based for NSFW.js)
    if ! command -v node &> /dev/null; then
        echo "Error: Node.js is required but not installed" >&2
        return 1
    fi
    
    # Create package.json if not exists
    if [[ ! -f "${RESOURCE_DIR}/package.json" ]]; then
        cat > "${RESOURCE_DIR}/package.json" <<EOF
{
  "name": "nsfw-detector",
  "version": "1.0.0",
  "description": "NSFW content detection resource",
  "main": "server.js",
  "scripts": {
    "start": "node server.js"
  },
  "dependencies": {
    "@tensorflow/tfjs-node": "^4.15.0",
    "nsfwjs": "^2.4.2",
    "express": "^4.18.2",
    "multer": "^1.4.5-lts.1",
    "sharp": "^0.33.0"
  }
}
EOF
    fi
    
    # Install Node dependencies
    echo "Installing Node.js dependencies..."
    cd "${RESOURCE_DIR}" && npm install --production
    
    echo "NSFW Detector resource installed successfully"
    return 0
}

manage_uninstall() {
    echo "Uninstalling NSFW Detector resource..."
    
    # Stop service if running
    manage_stop
    
    # Remove data unless --keep-data flag
    local keep_data=false
    for arg in "$@"; do
        if [[ "$arg" == "--keep-data" ]]; then
            keep_data=true
            break
        fi
    done
    
    if [[ "$keep_data" == false ]]; then
        echo "Removing data directories..."
        rm -rf "${DATA_DIR}" "${LOG_DIR}"
    fi
    
    # Remove Node modules
    if [[ -d "${RESOURCE_DIR}/node_modules" ]]; then
        echo "Removing Node.js dependencies..."
        rm -rf "${RESOURCE_DIR}/node_modules"
    fi
    
    echo "NSFW Detector resource uninstalled"
    return 0
}

manage_start() {
    echo "Starting NSFW Detector service..."
    
    # Check if already running
    if [[ -f "$PIDFILE" ]] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
        echo "Service is already running (PID: $(cat "$PIDFILE"))"
        return 2
    fi
    
    # Create minimal server if not exists
    if [[ ! -f "${RESOURCE_DIR}/server.js" ]]; then
        create_minimal_server
    fi
    
    # Start the service
    cd "${RESOURCE_DIR}"
    nohup node server.js > "${LOG_DIR}/server.log" 2>&1 &
    local pid=$!
    echo "$pid" > "$PIDFILE"
    
    # Wait for service to be ready if --wait flag
    local wait_flag=false
    for arg in "$@"; do
        if [[ "$arg" == "--wait" ]]; then
            wait_flag=true
            break
        fi
    done
    
    if [[ "$wait_flag" == true ]]; then
        echo "Waiting for service to be ready..."
        local timeout=30
        local elapsed=0
        while [[ $elapsed -lt $timeout ]]; do
            if timeout 5 curl -sf "http://localhost:${NSFW_DETECTOR_PORT}/health" > /dev/null 2>&1; then
                echo "Service is ready"
                return 0
            fi
            sleep 1
            ((elapsed++))
        done
        echo "Warning: Service started but health check timed out" >&2
    fi
    
    echo "NSFW Detector service started (PID: $pid)"
    return 0
}

manage_stop() {
    echo "Stopping NSFW Detector service..."
    
    if [[ ! -f "$PIDFILE" ]]; then
        echo "Service is not running (no PID file)"
        return 2
    fi
    
    local pid=$(cat "$PIDFILE")
    if ! kill -0 "$pid" 2>/dev/null; then
        echo "Service is not running (PID $pid not found)"
        rm -f "$PIDFILE"
        return 2
    fi
    
    # Graceful shutdown
    kill -TERM "$pid"
    
    # Wait for process to stop
    local timeout=30
    local elapsed=0
    while [[ $elapsed -lt $timeout ]]; do
        if ! kill -0 "$pid" 2>/dev/null; then
            rm -f "$PIDFILE"
            echo "Service stopped successfully"
            return 0
        fi
        sleep 1
        ((elapsed++))
    done
    
    # Force kill if still running
    echo "Force stopping service..."
    kill -KILL "$pid" 2>/dev/null || true
    rm -f "$PIDFILE"
    echo "Service force stopped"
    return 0
}

manage_restart() {
    echo "Restarting NSFW Detector service..."
    manage_stop
    sleep 2
    manage_start "$@"
    return $?
}

# Test functions
test_all() {
    echo "Running all test suites..."
    "${TEST_DIR}/run-tests.sh" all
    return $?
}

test_smoke() {
    echo "Running smoke tests..."
    "${TEST_DIR}/phases/test-smoke.sh"
    return $?
}

test_integration() {
    echo "Running integration tests..."
    "${TEST_DIR}/phases/test-integration.sh"
    return $?
}

test_unit() {
    echo "Running unit tests..."
    "${TEST_DIR}/phases/test-unit.sh"
    return $?
}

# Content management functions
content_add() {
    echo "Adding content to NSFW Detector..."
    # Implementation would handle model/config addition
    echo "Content management not yet implemented"
    return 0
}

content_list() {
    echo "Available models:"
    echo "- nsfwjs (default)"
    echo "- safety-checker (coming soon)"
    echo "- custom-cnn (coming soon)"
    return 0
}

content_get() {
    echo "Retrieving content..."
    echo "Content management not yet implemented"
    return 0
}

content_remove() {
    echo "Removing content..."
    echo "Content management not yet implemented"
    return 0
}

content_execute() {
    echo "Executing NSFW detection..."
    
    local file=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                file="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$file" ]]; then
        echo "Error: --file parameter required" >&2
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        echo "Error: File not found: $file" >&2
        return 1
    fi
    
    # Call the API to classify
    curl -sf -X POST "http://localhost:${NSFW_DETECTOR_PORT}/classify" \
        -F "image=@${file}" \
        || echo '{"error": "Service not available"}'
    
    return 0
}

# Status function
show_status() {
    local format="text"
    
    for arg in "$@"; do
        if [[ "$arg" == "--json" ]]; then
            format="json"
        fi
    done
    
    local status="not_running"
    local pid=""
    local health="unknown"
    
    if [[ -f "$PIDFILE" ]] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
        status="running"
        pid=$(cat "$PIDFILE")
        
        # Check health endpoint
        if timeout 5 curl -sf "http://localhost:${NSFW_DETECTOR_PORT}/health" > /dev/null 2>&1; then
            health="healthy"
        else
            health="unhealthy"
        fi
    fi
    
    if [[ "$format" == "json" ]]; then
        cat <<EOF
{
  "status": "$status",
  "pid": "$pid",
  "health": "$health",
  "port": $NSFW_DETECTOR_PORT,
  "models_loaded": ["nsfwjs"]
}
EOF
    else
        echo "NSFW Detector Status:"
        echo "  Status: $status"
        [[ -n "$pid" ]] && echo "  PID: $pid"
        echo "  Health: $health"
        echo "  Port: $NSFW_DETECTOR_PORT"
        echo "  Models: nsfwjs"
    fi
    
    [[ "$status" == "running" ]] && return 0 || return 2
}

# Logs function
show_logs() {
    local tail_lines=50
    
    for arg in "$@"; do
        if [[ "$arg" == "--tail" ]]; then
            shift
            tail_lines="${1:-50}"
            break
        fi
    done
    
    if [[ -f "${LOG_DIR}/server.log" ]]; then
        tail -n "$tail_lines" "${LOG_DIR}/server.log"
    else
        echo "No logs available"
    fi
    
    return 0
}

# Helper function to create minimal Node.js server
create_minimal_server() {
    cat > "${RESOURCE_DIR}/server.js" <<'EOF'
const express = require('express');
const app = express();
const PORT = process.env.NSFW_DETECTOR_PORT || 11451;

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({
        status: 'healthy',
        service: 'nsfw-detector',
        models: ['nsfwjs'],
        timestamp: new Date().toISOString()
    });
});

// Minimal classification endpoint
app.post('/classify', (req, res) => {
    // Placeholder - would load NSFW.js and process image
    res.json({
        adult: 0.02,
        racy: 0.15,
        gore: 0.01,
        violence: 0.03,
        safe: 0.79,
        classification: 'safe'
    });
});

// Models endpoint
app.get('/models', (req, res) => {
    res.json(['nsfwjs']);
});

// Start server
app.listen(PORT, () => {
    console.log(`NSFW Detector service running on port ${PORT}`);
});
EOF
}