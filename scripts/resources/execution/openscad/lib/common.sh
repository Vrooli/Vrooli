#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
OPENSCAD_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# OpenSCAD constants
export OPENSCAD_RESOURCE_NAME="openscad"
export OPENSCAD_CONTAINER_NAME="vrooli-openscad"
export OPENSCAD_IMAGE="openscad/openscad:latest"
export OPENSCAD_DATA_DIR="${HOME}/.openscad"
export OPENSCAD_SCRIPTS_DIR="${OPENSCAD_DATA_DIR}/scripts"
export OPENSCAD_OUTPUT_DIR="${OPENSCAD_DATA_DIR}/output"
export OPENSCAD_INJECTED_DIR="${OPENSCAD_LIB_DIR}/../injected"
export OPENSCAD_PORT=7070  # WebGL viewer port if we add web interface later

# Create data directories if they don't exist
openscad::ensure_dirs() {
    mkdir -p "${OPENSCAD_SCRIPTS_DIR}"
    mkdir -p "${OPENSCAD_OUTPUT_DIR}"
    mkdir -p "${OPENSCAD_INJECTED_DIR}"
}

# Check if OpenSCAD is installed (Docker required)
openscad::is_installed() {
    command -v docker >/dev/null 2>&1
}

# Check if OpenSCAD container exists
openscad::container_exists() {
    docker ps -a --format "{{.Names}}" 2>/dev/null | grep -q "^${OPENSCAD_CONTAINER_NAME}$"
}

# Check if OpenSCAD is running
openscad::is_running() {
    docker ps --format "{{.Names}}" 2>/dev/null | grep -q "^${OPENSCAD_CONTAINER_NAME}$"
}

# Get container IP
openscad::get_container_ip() {
    docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${OPENSCAD_CONTAINER_NAME}" 2>/dev/null || echo "unknown"
}