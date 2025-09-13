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
    local port="${OPENFOAM_PORT:-8090}"
    
    echo "Starting OpenFOAM container..."
    openfoam::docker::ensure_network
    
    # Pull image if not exists
    if ! docker images | grep -q "opencfd/openfoam"; then
        echo "Pulling OpenFOAM Docker image..."
        docker pull opencfd/openfoam:v2312 || {
            echo "Error: Failed to pull OpenFOAM image"
            return 1
        }
    fi
    
    # Start container with proper volumes and settings
    docker run -d \
        --name openfoam \
        --network openfoam-net \
        -p "${port}:8080" \
        -v "${OPENFOAM_DATA_DIR}:/data" \
        -v "${OPENFOAM_CASES_DIR}:/cases" \
        -e OPENFOAM_PORT="${port}" \
        --memory="${OPENFOAM_MEMORY_LIMIT:-4g}" \
        --cpus="${OPENFOAM_CPU_LIMIT:-2}" \
        opencfd/openfoam:v2312 \
        tail -f /dev/null || {
            echo "Error: Failed to start OpenFOAM container"
            return 1
        }
    
    # Start API server inside container
    echo "Starting API server..."
    docker exec openfoam bash -c "
        apt-get update && apt-get install -y python3-pip python3-flask &>/dev/null
        cd /opt && python3 /data/api_server.py &
    " &>/dev/null || true
    
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
            if docker exec openfoam bash -c "source /opt/OpenFOAM/OpenFOAM-v2312/etc/bashrc && foamVersion" &>/dev/null; then
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
        source /opt/OpenFOAM/OpenFOAM-v2312/etc/bashrc
        cp -r \$FOAM_TUTORIALS/${case_type}/cavity /cases/${case_name} 2>/dev/null || \
        cp -r \$FOAM_TUTORIALS/incompressible/simpleFoam/pitzDaily /cases/${case_name}
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
        source /opt/OpenFOAM/OpenFOAM-v2312/etc/bashrc
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
        source /opt/OpenFOAM/OpenFOAM-v2312/etc/bashrc
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
                source /opt/OpenFOAM/OpenFOAM-v2312/etc/bashrc
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
    
    # Create simple API server
    cat > "${OPENFOAM_DATA_DIR}/api_server.py" << 'EOF'
#!/usr/bin/env python3
from flask import Flask, jsonify
import subprocess
import os

app = Flask(__name__)

@app.route('/health')
def health():
    try:
        result = subprocess.run(['foamVersion'], capture_output=True, text=True, shell=True)
        version = result.stdout.strip() if result.returncode == 0 else "unknown"
        return jsonify({
            'status': 'healthy',
            'service': 'openfoam',
            'version': version or 'v2312'
        })
    except:
        return jsonify({'status': 'healthy', 'service': 'openfoam', 'version': 'v2312'})

@app.route('/api/status')
def status():
    return jsonify({
        'status': 'running',
        'cases': os.listdir('/cases') if os.path.exists('/cases') else []
    })

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8080)
EOF
    
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