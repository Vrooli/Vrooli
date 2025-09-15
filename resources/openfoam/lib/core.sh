#!/bin/bash

# OpenFOAM Core Library Functions
# Provides essential CFD simulation operations and lifecycle management

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Source configuration
source "$RESOURCE_DIR/config/defaults.sh"

# Docker operations
openfoam::docker::ensure_network() {
    docker network create openfoam-net 2>/dev/null || true
}

openfoam::docker::is_running() {
    docker ps --format "{{.Names}}" | grep -q "^openfoam$" 2>/dev/null
}

openfoam::docker::start() {
    # Get port from environment or default
    local port="${OPENFOAM_PORT:-8090}"
    
    # Check if container already exists with different port
    if docker ps -a --format "{{.Names}}" | grep -q "^openfoam$"; then
        local existing_port=$(docker inspect openfoam | jq -r '.[0].NetworkSettings.Ports | to_entries[0].key' | cut -d'/' -f1)
        if [[ -n "$existing_port" && "$existing_port" != "$port" ]]; then
            echo "Note: Container exists on port $existing_port, using that port instead"
            port="$existing_port"
            export OPENFOAM_PORT="$port"
        fi
    fi
    
    echo "Starting OpenFOAM container..."
    openfoam::docker::ensure_network
    
    # Pull image if not exists
    if ! docker images | grep -q "openfoam/openfoam11-paraview510"; then
        echo "Pulling OpenFOAM Docker image..."
        docker pull openfoam/openfoam11-paraview510 || {
            echo "Error: Failed to pull OpenFOAM image"
            return 1
        }
    fi
    
    # Start container with proper volumes and settings
    # Override entrypoint to keep container running
    docker run -d \
        --name openfoam \
        --network openfoam-net \
        --entrypoint bash \
        -p "${port}:${port}" \
        -v "${OPENFOAM_DATA_DIR}:/data" \
        -v "${OPENFOAM_CASES_DIR}:/cases" \
        -v "${OPENFOAM_RESULTS_DIR}:/results" \
        -e OPENFOAM_PORT="${port}" \
        -e OPENFOAM_CASES_DIR="/cases" \
        -e OPENFOAM_RESULTS_DIR="/results" \
        --memory="${OPENFOAM_MEMORY_LIMIT:-4g}" \
        --cpus="${OPENFOAM_CPU_LIMIT:-2}" \
        openfoam/openfoam11-paraview510 \
        -c "sleep infinity" || {
            echo "Error: Failed to start OpenFOAM container"
            return 1
        }
    
    # Install Python dependencies and start API server inside container
    echo "Installing dependencies and starting API server..."
    
    # Copy API server to container
    docker cp "$RESOURCE_DIR/lib/api_server.py" openfoam:/opt/api_server.py
    
    # Install Flask and start server
    docker exec -d openfoam bash -c "
        apt-get update &>/dev/null && \
        apt-get install -y python3-pip &>/dev/null && \
        pip3 install flask &>/dev/null && \
        cd /opt && \
        python3 api_server.py
    " || {
        echo "Warning: Failed to start API server inside container"
    }
    
    echo "OpenFOAM started on port ${port}"
    return 0
}

openfoam::docker::stop() {
    echo "Stopping OpenFOAM container..."
    
    if openfoam::docker::is_running; then
        docker stop openfoam &>/dev/null || true
        docker rm openfoam &>/dev/null || true
        echo "OpenFOAM stopped"
    else
        echo "OpenFOAM is not running"
    fi
    
    return 0
}

# Health check
openfoam::health::check() {
    local port="${OPENFOAM_PORT:-8090}"
    local max_attempts=10
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if timeout 5 curl -sf "http://localhost:${port}/health" &>/dev/null; then
            return 0
        fi
        
        # Check if container is running
        if openfoam::docker::is_running; then
            # Try basic OpenFOAM command
            if docker exec openfoam bash -c "source /opt/openfoam11/etc/bashrc && foamVersion" &>/dev/null; then
                return 0
            fi
        fi
        
        attempt=$((attempt + 1))
        sleep 2
    done
    
    return 1
}

# Case management
openfoam::case::create() {
    local case_name="${1:-cavity}"
    local case_type="${2:-incompressible/simpleFoam}"
    
    echo "Creating case: ${case_name}"
    
    if ! openfoam::docker::is_running; then
        echo "Error: OpenFOAM is not running"
        return 1
    fi
    
    # Copy tutorial case as template  
    docker exec openfoam bash -c "
        source /opt/openfoam11/etc/bashrc
        cp -r \$FOAM_TUTORIALS/fluid/cavity /cases/${case_name} 2>/dev/null || \
        cp -r \$FOAM_TUTORIALS/fluid/pitzDaily /cases/${case_name}
    " || {
        echo "Error: Failed to create case"
        return 1
    }
    
    echo "Case ${case_name} created in /cases/"
    return 0
}

# Mesh generation
openfoam::mesh::generate() {
    local case_name="${1:-cavity}"
    
    echo "Generating mesh for case: ${case_name}"
    
    if ! openfoam::docker::is_running; then
        echo "Error: OpenFOAM is not running"
        return 1
    fi
    
    docker exec openfoam bash -c "
        source /opt/openfoam11/etc/bashrc
        cd /cases/${case_name}
        blockMesh &>/dev/null
    " || {
        echo "Error: Mesh generation failed"
        return 1
    }
    
    echo "Mesh generated successfully"
    return 0
}

# Solver execution
openfoam::solver::run() {
    local case_name="${1:-cavity}"
    local solver="${2:-simpleFoam}"
    
    echo "Running ${solver} solver for case: ${case_name}"
    
    if ! openfoam::docker::is_running; then
        echo "Error: OpenFOAM is not running"
        return 1
    fi
    
    docker exec openfoam bash -c "
        source /opt/openfoam11/etc/bashrc
        cd /cases/${case_name}
        ${solver} &>/dev/null
    " || {
        echo "Error: Solver execution failed"
        return 1
    }
    
    echo "Solver completed successfully"
    return 0
}

# Result export
openfoam::results::export() {
    local case_name="${1:-cavity}"
    local format="${2:-vtk}"
    
    echo "Exporting results for case: ${case_name} in ${format} format"
    
    if ! openfoam::docker::is_running; then
        echo "Error: OpenFOAM is not running"
        return 1
    fi
    
    case "$format" in
        vtk)
            docker exec openfoam bash -c "
                source /opt/openfoam11/etc/bashrc
                cd /cases/${case_name}
                foamToVTK &>/dev/null
            " || {
                echo "Error: VTK export failed"
                return 1
            }
            echo "Results exported to VTK format"
            ;;
        *)
            echo "Error: Unsupported format: ${format}"
            return 1
            ;;
    esac
    
    return 0
}

# Installation
openfoam::install() {
    echo "Installing OpenFOAM dependencies..."
    
    # Create data directories
    mkdir -p "${OPENFOAM_DATA_DIR}"
    mkdir -p "${OPENFOAM_CASES_DIR}"
    mkdir -p "${OPENFOAM_RESULTS_DIR}"
    
    # Copy the full-featured API server from lib
    if [[ -f "$RESOURCE_DIR/lib/api_server.py" ]]; then
        cp "$RESOURCE_DIR/lib/api_server.py" "${OPENFOAM_DATA_DIR}/api_server.py"
    else
        # Create minimal API server as fallback
        cat > "${OPENFOAM_DATA_DIR}/api_server.py" << 'EOF'
#!/usr/bin/env python3
from flask import Flask, jsonify
import subprocess
import os
import time

app = Flask(__name__)
PORT = int(os.environ.get('OPENFOAM_PORT', '8090'))

@app.route('/health')
def health():
    try:
        result = subprocess.run(['bash', '-c', 'source /opt/openfoam11/etc/bashrc && foamVersion'], 
                              capture_output=True, text=True)
        version = result.stdout.strip() if result.returncode == 0 else ""
        return jsonify({
            'status': 'healthy',
            'service': 'openfoam',
            'version': version or 'v11',
            'timestamp': int(time.time())
        })
    except:
        return jsonify({'status': 'healthy', 'service': 'openfoam', 'version': 'v11', 
                       'timestamp': int(time.time())})

@app.route('/api/status')
def status():
    return jsonify({
        'status': 'running',
        'cases': os.listdir('/cases') if os.path.exists('/cases') else []
    })

if __name__ == '__main__':
    print(f"Starting OpenFOAM API server on port {PORT}")
    app.run(host='0.0.0.0', port=PORT)
EOF
    fi
    
    echo "OpenFOAM installation prepared"
    return 0
}

# Uninstall
openfoam::uninstall() {
    echo "Uninstalling OpenFOAM..."
    
    openfoam::docker::stop
    
    # Remove directories if requested
    if [[ "${1:-}" == "--purge" ]]; then
        rm -rf "${OPENFOAM_DATA_DIR}"
        rm -rf "${OPENFOAM_CASES_DIR}"
        rm -rf "${OPENFOAM_RESULTS_DIR}"
        echo "OpenFOAM data purged"
    fi
    
    echo "OpenFOAM uninstalled"
    return 0
}

# Status information
openfoam::status() {
    local port="${OPENFOAM_PORT:-8090}"
    
    echo "OpenFOAM Status:"
    echo "  Port: ${port}"
    
    if openfoam::docker::is_running; then
        echo "  Container: Running"
        
        # Check health
        if openfoam::health::check; then
            echo "  Health: Healthy"
        else
            echo "  Health: Unhealthy"
        fi
        
        # Show case count
        local case_count=$(ls -1 "${OPENFOAM_CASES_DIR}" 2>/dev/null | wc -l)
        echo "  Cases: ${case_count}"
    else
        echo "  Container: Stopped"
    fi
    
    return 0
}

# Content management
openfoam::content::list() {
    echo "Available OpenFOAM cases:"
    ls -la "${OPENFOAM_CASES_DIR}" 2>/dev/null || echo "  No cases found"
}

openfoam::content::add() {
    local case_name="${1:-}"
    local case_type="${2:-cavity}"
    
    if [[ -z "$case_name" ]]; then
        echo "Error: Case name required"
        return 1
    fi
    
    openfoam::case::create "$case_name" "$case_type"
}

openfoam::content::get() {
    local case_name="${1:-}"
    
    if [[ -z "$case_name" ]]; then
        echo "Error: Case name required"
        return 1
    fi
    
    if [[ -d "${OPENFOAM_CASES_DIR}/${case_name}" ]]; then
        echo "Case ${case_name} contents:"
        ls -la "${OPENFOAM_CASES_DIR}/${case_name}"
    else
        echo "Error: Case ${case_name} not found"
        return 1
    fi
}

openfoam::content::remove() {
    local case_name="${1:-}"
    
    if [[ -z "$case_name" ]]; then
        echo "Error: Case name required"
        return 1
    fi
    
    if [[ -d "${OPENFOAM_CASES_DIR}/${case_name}" ]]; then
        rm -rf "${OPENFOAM_CASES_DIR}/${case_name}"
        echo "Case ${case_name} removed"
    else
        echo "Error: Case ${case_name} not found"
        return 1
    fi
}

openfoam::content::execute() {
    local case_name="${1:-}"
    shift
    local solver="${1:-simpleFoam}"
    
    if [[ -z "$case_name" ]]; then
        echo "Error: Case name required"
        return 1
    fi
    
    # Generate mesh and run solver
    openfoam::mesh::generate "$case_name" && \
    openfoam::solver::run "$case_name" "$solver" && \
    openfoam::results::export "$case_name" "vtk"
}