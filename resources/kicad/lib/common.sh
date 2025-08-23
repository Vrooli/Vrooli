#!/bin/bash
# KiCad Common Functions

# Get script directory
KICAD_COMMON_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source defaults
source "${KICAD_COMMON_LIB_DIR}/../config/defaults.sh"

# Initialize KiCad directories
kicad::init_dirs() {
    mkdir -p "$KICAD_DATA_DIR"
    mkdir -p "$KICAD_CONFIG_DIR"
    mkdir -p "$KICAD_PROJECTS_DIR"
    mkdir -p "$KICAD_LIBRARIES_DIR"
    mkdir -p "$KICAD_TEMPLATES_DIR"
    mkdir -p "$KICAD_OUTPUTS_DIR"
    mkdir -p "$KICAD_LOGS_DIR"
}

# Check if KiCad is installed
kicad::is_installed() {
    command -v kicad-cli &>/dev/null || command -v kicad &>/dev/null || \
    [[ -x "${KICAD_DATA_DIR}/kicad-cli-mock" ]] || [[ -x "${KICAD_DATA_DIR}/kicad-mock" ]]
}

# Get KiCad version
kicad::get_version() {
    if command -v kicad-cli &>/dev/null; then
        kicad-cli version 2>/dev/null | head -1 | cut -d' ' -f2
    elif command -v kicad &>/dev/null; then
        kicad --version 2>/dev/null | head -1 | cut -d' ' -f2
    elif [[ -x "${KICAD_DATA_DIR}/kicad-cli-mock" ]]; then
        "${KICAD_DATA_DIR}/kicad-cli-mock" version 2>/dev/null
    elif [[ -x "${KICAD_DATA_DIR}/kicad-mock" ]]; then
        "${KICAD_DATA_DIR}/kicad-mock" --version 2>/dev/null | head -1 | cut -d' ' -f2
    else
        echo "not_installed"
    fi
}

# Check if KiCad Python API is available
kicad::check_python_api() {
    $KICAD_PYTHON_VERSION -c "import pcbnew" 2>/dev/null
}

# Export functions
export -f kicad::init_dirs
export -f kicad::is_installed
export -f kicad::get_version
export -f kicad::check_python_api