#!/usr/bin/env bash
# OpenSCAD Configuration Defaults

# Service configuration
export OPENSCAD_NAME="openscad"
export OPENSCAD_DISPLAY_NAME="OpenSCAD 3D CAD"
export OPENSCAD_CATEGORY="execution"
export OPENSCAD_DESCRIPTION="Programmatic 3D CAD modeler using script-based solid modeling"

# Container configuration  
export OPENSCAD_CONTAINER_NAME="${OPENSCAD_CONTAINER_NAME:-vrooli-openscad}"
export OPENSCAD_IMAGE="${OPENSCAD_IMAGE:-openscad/openscad:latest}"
export OPENSCAD_PORT="${OPENSCAD_PORT:-7070}"

# Runtime configuration
export OPENSCAD_DATA_DIR="${OPENSCAD_DATA_DIR:-${HOME}/.openscad}"
export OPENSCAD_SCRIPTS_DIR="${OPENSCAD_DATA_DIR}/scripts"
export OPENSCAD_OUTPUT_DIR="${OPENSCAD_DATA_DIR}/output"
export OPENSCAD_INJECTED_DIR="${APP_ROOT}/resources/openscad/injected"

# Export config function
openscad::export_config() {
    export OPENSCAD_NAME OPENSCAD_DISPLAY_NAME OPENSCAD_CATEGORY OPENSCAD_DESCRIPTION
    export OPENSCAD_CONTAINER_NAME OPENSCAD_IMAGE OPENSCAD_PORT
    export OPENSCAD_DATA_DIR OPENSCAD_SCRIPTS_DIR OPENSCAD_OUTPUT_DIR OPENSCAD_INJECTED_DIR
}