#!/bin/bash
# KiCad Resource Configuration

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# KiCad directories
export KICAD_DATA_DIR="${KICAD_DATA_DIR:-${var_DATA_DIR}/resources/kicad}"
export KICAD_CONFIG_DIR="${KICAD_DATA_DIR}/config"
export KICAD_PROJECTS_DIR="${KICAD_DATA_DIR}/projects"
export KICAD_LIBRARIES_DIR="${KICAD_DATA_DIR}/libraries"
export KICAD_TEMPLATES_DIR="${KICAD_DATA_DIR}/templates"
export KICAD_OUTPUTS_DIR="${KICAD_DATA_DIR}/outputs"
export KICAD_LOGS_DIR="${KICAD_DATA_DIR}/logs"

# KiCad settings
export KICAD_PORT="${KICAD_PORT:-8095}"
export KICAD_HOST="${KICAD_HOST:-localhost}"
export KICAD_PYTHON_VERSION="${KICAD_PYTHON_VERSION:-python3}"
export KICAD_USE_GUI="${KICAD_USE_GUI:-false}"
export KICAD_EXPORT_FORMATS="${KICAD_EXPORT_FORMATS:-gerber,pdf,svg,step}"

# Export config function
kicad::export_config() {
    export KICAD_DATA_DIR
    export KICAD_CONFIG_DIR
    export KICAD_PROJECTS_DIR
    export KICAD_LIBRARIES_DIR
    export KICAD_TEMPLATES_DIR
    export KICAD_OUTPUTS_DIR
    export KICAD_LOGS_DIR
    export KICAD_PORT
    export KICAD_HOST
    export KICAD_PYTHON_VERSION
    export KICAD_USE_GUI
    export KICAD_EXPORT_FORMATS
}