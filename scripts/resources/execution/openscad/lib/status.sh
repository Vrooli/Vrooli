#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
OPENSCAD_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source format helper if available
if [[ -f "${OPENSCAD_STATUS_DIR}/../../../../lib/utils/format.sh" ]]; then
    source "${OPENSCAD_STATUS_DIR}/../../../../lib/utils/format.sh"
fi

# Dependencies are expected to be sourced by caller

# Get OpenSCAD status
openscad::status() {
    local format="${1:-text}"
    
    # Shift off format argument if present
    if [[ "$#" -gt 0 ]]; then
        shift
    fi
    
    local name="openscad"
    local status="stopped"
    local installed="false"
    local running="false"
    local healthy="false"
    local message="OpenSCAD is not installed"
    local version="unknown"
    local scripts_count=0
    local outputs_count=0
    
    if openscad::is_installed; then
        installed="true"
        
        if openscad::is_running; then
            status="running"
            running="true"
            
            # Check if we can execute commands in the container
            if docker exec "${OPENSCAD_CONTAINER_NAME}" sh -c "openscad --version 2>&1" 2>/dev/null | grep -q "OpenSCAD"; then
                healthy="true"
                version=$(docker exec "${OPENSCAD_CONTAINER_NAME}" sh -c "openscad --version 2>&1" 2>/dev/null | head -1 | sed 's/OpenSCAD version //' || echo "unknown")
                message="OpenSCAD is running and healthy"
                
                # Count scripts and outputs
                scripts_count=$(find "${OPENSCAD_SCRIPTS_DIR}" -name "*.scad" 2>/dev/null | wc -l || echo 0)
                outputs_count=$(find "${OPENSCAD_OUTPUT_DIR}" -name "*.stl" -o -name "*.off" -o -name "*.png" 2>/dev/null | wc -l || echo 0)
            else
                message="OpenSCAD container is running but not responding"
            fi
        elif openscad::container_exists; then
            message="OpenSCAD container exists but is not running"
        else
            message="OpenSCAD is installed but not running"
        fi
    elif command -v docker >/dev/null 2>&1; then
        message="OpenSCAD is not installed (run 'resource-openscad install')"
    else
        message="Docker is required but not installed"
    fi
    
    # Output directly without format helper
    if [[ "$format" == "json" ]]; then
        cat <<EOF
{"name":"${name}","status":"${status}","installed":${installed},"running":${running},"healthy":${healthy},"version":"${version}","scripts":${scripts_count},"outputs":${outputs_count},"message":"${message}","description":"Programmatic 3D CAD modeler","category":"execution"}
EOF
    else
        cat <<EOF
Name: ${name}
Status: ${status}
Installed: ${installed}
Running: ${running}
Healthy: ${healthy}
Version: ${version}
Scripts: ${scripts_count}
Outputs: ${outputs_count}
Message: ${message}
Description: Programmatic 3D CAD modeler
Category: execution
EOF
    fi
}