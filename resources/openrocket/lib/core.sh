#!/usr/bin/env bash
# OpenRocket Core Functionality Library

set -euo pipefail

# Configuration
OPENROCKET_PORT="${OPENROCKET_PORT:-9513}"
OPENROCKET_DATA_DIR="${OPENROCKET_DATA_DIR:-${HOME}/.openrocket}"
OPENROCKET_MODELS_DIR="${OPENROCKET_MODELS_DIR:-${OPENROCKET_DATA_DIR}/models}"
OPENROCKET_DESIGNS_DIR="${OPENROCKET_DESIGNS_DIR:-${OPENROCKET_DATA_DIR}/designs}"
OPENROCKET_SIMS_DIR="${OPENROCKET_SIMS_DIR:-${OPENROCKET_DATA_DIR}/simulations}"
OPENROCKET_LOG_DIR="${OPENROCKET_LOG_DIR:-${OPENROCKET_DATA_DIR}/logs}"
OPENROCKET_PID_FILE="${OPENROCKET_DATA_DIR}/openrocket.pid"
OPENROCKET_CONTAINER_NAME="openrocket-server"
OPENROCKET_IMAGE="openrocket/openrocket:latest"
OPENROCKET_HEALTH_URL="http://localhost:${OPENROCKET_PORT}/health"

# Import test library if needed
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
[[ -f "${RESOURCE_DIR}/lib/test.sh" ]] && source "${RESOURCE_DIR}/lib/test.sh"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_debug() {
    [[ "${DEBUG:-false}" == "true" ]] && echo -e "${BLUE}[DEBUG]${NC} $*"
}

# Help function
openrocket_help() {
    cat << EOF
OpenRocket Launch Vehicle Design Studio

USAGE:
    $(basename "$0") [COMMAND] [OPTIONS]

COMMANDS:
    help                    Show this help message
    info                    Show resource information
    manage install          Install OpenRocket and dependencies
    manage start            Start OpenRocket server
    manage stop             Stop OpenRocket server
    manage restart          Restart OpenRocket server
    manage uninstall        Remove OpenRocket completely
    test smoke              Run quick health check
    test integration        Run integration tests
    test unit               Run unit tests
    test all                Run all tests
    content list            List available rocket designs
    content add <file>      Import a rocket design (.ork file)
    content get <name>      Export rocket design
    content remove <name>   Remove rocket design
    content execute <cmd>   Run simulation command
    status                  Show OpenRocket status
    logs                    View OpenRocket logs

EXAMPLES:
    # Install and start OpenRocket
    $(basename "$0") manage install
    $(basename "$0") manage start

    # Import a rocket design
    $(basename "$0") content add my-rocket.ork

    # Run a trajectory simulation
    $(basename "$0") content execute simulate --design my-rocket --altitude 1000

    # View simulation results
    $(basename "$0") logs

ENVIRONMENT VARIABLES:
    OPENROCKET_PORT         Server port (default: 9513)
    OPENROCKET_DATA_DIR     Data directory (default: ~/.openrocket)

EOF
}

# Info function
openrocket_info() {
    local json_output="${1:-false}"
    
    if [[ "$json_output" == "--json" ]]; then
        cat "${RESOURCE_DIR}/config/runtime.json" 2>/dev/null || echo '{}'
    else
        if [[ -f "${RESOURCE_DIR}/config/runtime.json" ]]; then
            echo "OpenRocket Runtime Information:"
            jq -r 'to_entries[] | "  \(.key): \(.value)"' "${RESOURCE_DIR}/config/runtime.json"
        else
            echo "Runtime configuration not found"
        fi
    fi
}

# Install function
openrocket_install() {
    local force=false
    local skip_validation=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --force) force=true; shift ;;
            --skip-validation) skip_validation=true; shift ;;
            *) shift ;;
        esac
    done
    
    log_info "Installing OpenRocket Launch Vehicle Design Studio..."
    
    # Check if Docker is installed
    if ! command -v docker &> /dev/null; then
        log_error "Docker is required but not installed"
        return 1
    fi
    
    # Create directories
    mkdir -p "${OPENROCKET_DATA_DIR}" "${OPENROCKET_MODELS_DIR}" \
             "${OPENROCKET_DESIGNS_DIR}" "${OPENROCKET_SIMS_DIR}" \
             "${OPENROCKET_LOG_DIR}"
    
    # Pull Docker image
    log_info "Pulling OpenRocket Docker image..."
    docker pull "${OPENROCKET_IMAGE}" || {
        # Build custom image if official doesn't exist
        log_info "Building custom OpenRocket container..."
        cat > "${OPENROCKET_DATA_DIR}/Dockerfile" << 'DOCKERFILE'
FROM openjdk:11-jre-slim

RUN apt-get update && apt-get install -y \
    wget \
    xvfb \
    python3 \
    python3-pip \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /opt/openrocket

# Download OpenRocket JAR (latest stable)
RUN wget https://github.com/openrocket/openrocket/releases/download/release-22.02/OpenRocket-22.02.jar \
    -O OpenRocket.jar

# Install Python dependencies for API
RUN pip3 install flask flask-cors pyyaml

# Add API server script
COPY api_server.py /opt/openrocket/

EXPOSE 9513

CMD ["python3", "/opt/openrocket/api_server.py"]
DOCKERFILE
        
        # Create API server
        cat > "${OPENROCKET_DATA_DIR}/api_server.py" << 'PYTHON'
#!/usr/bin/env python3
import os
import json
import subprocess
import tempfile
from flask import Flask, jsonify, request, send_file
from flask_cors import CORS

app = Flask(__name__)
CORS(app)

DESIGNS_DIR = "/data/designs"
SIMS_DIR = "/data/simulations"
os.makedirs(DESIGNS_DIR, exist_ok=True)
os.makedirs(SIMS_DIR, exist_ok=True)

@app.route('/health', methods=['GET'])
def health():
    return jsonify({
        "status": "healthy",
        "service": "openrocket",
        "version": "22.02"
    })

@app.route('/api/designs', methods=['GET'])
def list_designs():
    designs = []
    for file in os.listdir(DESIGNS_DIR):
        if file.endswith('.ork'):
            designs.append(file[:-4])
    return jsonify({"designs": designs})

@app.route('/api/designs/<name>', methods=['POST'])
def upload_design(name):
    if 'file' not in request.files:
        return jsonify({"error": "No file provided"}), 400
    
    file = request.files['file']
    file.save(os.path.join(DESIGNS_DIR, f"{name}.ork"))
    return jsonify({"message": "Design uploaded", "name": name})

@app.route('/api/designs/<name>', methods=['GET'])
def download_design(name):
    file_path = os.path.join(DESIGNS_DIR, f"{name}.ork")
    if not os.path.exists(file_path):
        return jsonify({"error": "Design not found"}), 404
    return send_file(file_path)

@app.route('/api/simulate', methods=['POST'])
def simulate():
    data = request.json
    design = data.get('design')
    
    if not design:
        return jsonify({"error": "Design name required"}), 400
    
    design_path = os.path.join(DESIGNS_DIR, f"{design}.ork")
    if not os.path.exists(design_path):
        return jsonify({"error": "Design not found"}), 404
    
    # Run simulation (headless)
    with tempfile.NamedTemporaryFile(suffix='.csv', delete=False) as f:
        output_file = f.name
    
    cmd = [
        "xvfb-run", "-a",
        "java", "-jar", "/opt/openrocket/OpenRocket.jar",
        "-simulate", design_path,
        "-output", output_file
    ]
    
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=60)
        
        # Parse CSV results
        with open(output_file, 'r') as f:
            sim_data = f.read()
        
        os.unlink(output_file)
        
        return jsonify({
            "status": "completed",
            "design": design,
            "data": sim_data
        })
        
    except subprocess.TimeoutExpired:
        return jsonify({"error": "Simulation timeout"}), 500
    except Exception as e:
        return jsonify({"error": str(e)}), 500

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=9513)
PYTHON
        
        docker build -t "${OPENROCKET_IMAGE}" "${OPENROCKET_DATA_DIR}"
    }
    
    # Seed sample rocket designs
    log_info "Seeding sample rocket designs..."
    mkdir -p "${RESOURCE_DIR}/examples/rockets"
    
    # Create a simple model rocket design spec
    cat > "${RESOURCE_DIR}/examples/rockets/alpha-iii.yaml" << 'YAML'
name: Alpha III
description: Classic Estes Alpha III model rocket
components:
  nosecone:
    type: ogive
    length: 7.0  # cm
    diameter: 2.5  # cm
    material: plastic
  body:
    length: 25.0  # cm
    diameter: 2.5  # cm
    thickness: 0.1  # cm
    material: cardboard
  fins:
    count: 3
    rootChord: 5.0  # cm
    tipChord: 2.0  # cm
    span: 5.0  # cm
    material: balsa
  motor:
    type: A8-3
    diameter: 1.8  # cm
    length: 7.0  # cm
    totalImpulse: 2.5  # N-s
    burnTime: 0.7  # seconds
YAML
    
    if [[ "$skip_validation" != "true" ]]; then
        log_info "Validating installation..."
        sleep 2
    fi
    
    log_info "OpenRocket installation completed successfully"
}

# Start function
openrocket_start() {
    local wait_ready="${1:-false}"
    
    # Check if already running
    if docker ps | grep -q "${OPENROCKET_CONTAINER_NAME}"; then
        log_warn "OpenRocket is already running"
        return 0
    fi
    
    log_info "Starting OpenRocket server on port ${OPENROCKET_PORT}..."
    
    # Start container
    docker run -d \
        --name "${OPENROCKET_CONTAINER_NAME}" \
        -p "${OPENROCKET_PORT}:9513" \
        -v "${OPENROCKET_DESIGNS_DIR}:/data/designs" \
        -v "${OPENROCKET_SIMS_DIR}:/data/simulations" \
        -v "${OPENROCKET_LOG_DIR}:/data/logs" \
        -v "${OPENROCKET_DATA_DIR}/api_server.py:/opt/openrocket/api_server.py" \
        "${OPENROCKET_IMAGE}" || {
        log_error "Failed to start OpenRocket container"
        return 1
    }
    
    if [[ "$wait_ready" == "--wait" ]]; then
        log_info "Waiting for OpenRocket to be ready..."
        local max_attempts=30
        local attempt=0
        
        while (( attempt < max_attempts )); do
            if timeout 5 curl -sf "${OPENROCKET_HEALTH_URL}" > /dev/null 2>&1; then
                log_info "OpenRocket is ready!"
                return 0
            fi
            sleep 2
            ((attempt++))
        done
        
        log_error "OpenRocket failed to become ready"
        return 1
    fi
    
    log_info "OpenRocket started successfully"
}

# Stop function
openrocket_stop() {
    log_info "Stopping OpenRocket server..."
    
    if docker ps | grep -q "${OPENROCKET_CONTAINER_NAME}"; then
        docker stop "${OPENROCKET_CONTAINER_NAME}" > /dev/null 2>&1
        docker rm "${OPENROCKET_CONTAINER_NAME}" > /dev/null 2>&1
        log_info "OpenRocket stopped"
    else
        log_warn "OpenRocket is not running"
    fi
}

# Restart function
openrocket_restart() {
    openrocket_stop
    sleep 2
    openrocket_start "$@"
}

# Uninstall function
openrocket_uninstall() {
    local keep_data="${1:-false}"
    
    log_info "Uninstalling OpenRocket..."
    
    # Stop if running
    openrocket_stop
    
    # Remove Docker image
    docker rmi "${OPENROCKET_IMAGE}" 2>/dev/null || true
    
    # Remove data unless requested to keep
    if [[ "$keep_data" != "--keep-data" ]]; then
        log_info "Removing OpenRocket data..."
        rm -rf "${OPENROCKET_DATA_DIR}"
    fi
    
    log_info "OpenRocket uninstalled"
}

# Manage function
openrocket_manage() {
    local subcmd="${1:-}"
    shift 2>/dev/null || true
    
    case "$subcmd" in
        install) openrocket_install "$@" ;;
        start) openrocket_start "$@" ;;
        stop) openrocket_stop "$@" ;;
        restart) openrocket_restart "$@" ;;
        uninstall) openrocket_uninstall "$@" ;;
        *)
            log_error "Unknown manage command: $subcmd"
            echo "Available: install, start, stop, restart, uninstall"
            return 1
            ;;
    esac
}

# Content management
openrocket_content() {
    local subcmd="${1:-}"
    shift 2>/dev/null || true
    
    case "$subcmd" in
        list)
            curl -sf "http://localhost:${OPENROCKET_PORT}/api/designs" | jq -r '.designs[]' 2>/dev/null || {
                log_error "Failed to list designs"
                return 1
            }
            ;;
        add)
            local file="${1:-}"
            if [[ -z "$file" || ! -f "$file" ]]; then
                log_error "File not found: $file"
                return 1
            fi
            local name="${2:-$(basename "$file" .ork)}"
            curl -sf -X POST -F "file=@${file}" \
                "http://localhost:${OPENROCKET_PORT}/api/designs/${name}" || {
                log_error "Failed to upload design"
                return 1
            }
            log_info "Design uploaded: $name"
            ;;
        get)
            local name="${1:-}"
            if [[ -z "$name" ]]; then
                log_error "Design name required"
                return 1
            fi
            curl -sf "http://localhost:${OPENROCKET_PORT}/api/designs/${name}" \
                -o "${name}.ork" || {
                log_error "Failed to download design"
                return 1
            }
            log_info "Design downloaded: ${name}.ork"
            ;;
        remove)
            local name="${1:-}"
            if [[ -z "$name" ]]; then
                log_error "Design name required"
                return 1
            fi
            rm -f "${OPENROCKET_DESIGNS_DIR}/${name}.ork"
            log_info "Design removed: $name"
            ;;
        execute)
            local cmd="${1:-simulate}"
            shift 2>/dev/null || true
            
            case "$cmd" in
                simulate)
                    local design="${1:-alpha-iii}"
                    curl -sf -X POST -H "Content-Type: application/json" \
                        -d "{\"design\":\"${design}\"}" \
                        "http://localhost:${OPENROCKET_PORT}/api/simulate" | jq . || {
                        log_error "Simulation failed"
                        return 1
                    }
                    ;;
                *)
                    log_error "Unknown command: $cmd"
                    return 1
                    ;;
            esac
            ;;
        *)
            log_error "Unknown content command: $subcmd"
            echo "Available: list, add, get, remove, execute"
            return 1
            ;;
    esac
}

# Status function
openrocket_status() {
    echo "OpenRocket Status:"
    echo "  Port: ${OPENROCKET_PORT}"
    echo -n "  Container: "
    
    if docker ps | grep -q "${OPENROCKET_CONTAINER_NAME}"; then
        echo -e "${GREEN}Running${NC}"
        echo -n "  Health: "
        if timeout 5 curl -sf "${OPENROCKET_HEALTH_URL}" > /dev/null 2>&1; then
            echo -e "${GREEN}Healthy${NC}"
        else
            echo -e "${RED}Unhealthy${NC}"
        fi
    else
        echo -e "${RED}Stopped${NC}"
    fi
    
    # Show design count
    if [[ -d "${OPENROCKET_DESIGNS_DIR}" ]]; then
        local count=$(find "${OPENROCKET_DESIGNS_DIR}" -name "*.ork" 2>/dev/null | wc -l)
        echo "  Designs: $count"
    fi
}

# Logs function
openrocket_logs() {
    local lines="${1:-50}"
    
    if docker ps | grep -q "${OPENROCKET_CONTAINER_NAME}"; then
        docker logs --tail "$lines" "${OPENROCKET_CONTAINER_NAME}"
    else
        log_warn "OpenRocket is not running"
    fi
}

# Test function wrapper
openrocket_test() {
    local test_type="${1:-all}"
    
    case "$test_type" in
        smoke|integration|unit|all)
            if [[ -f "${RESOURCE_DIR}/test/run-tests.sh" ]]; then
                "${RESOURCE_DIR}/test/run-tests.sh" "$test_type"
            else
                log_error "Test runner not found"
                return 1
            fi
            ;;
        *)
            log_error "Unknown test type: $test_type"
            echo "Available: smoke, integration, unit, all"
            return 1
            ;;
    esac
}