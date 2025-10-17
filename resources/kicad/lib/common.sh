#!/bin/bash
# KiCad Common Functions

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
KICAD_COMMON_LIB_DIR="${APP_ROOT}/resources/kicad/lib"

# Source defaults
source "${APP_ROOT}/resources/kicad/config/defaults.sh"

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
    # Check for kicad-cli in standard locations
    command -v kicad-cli &>/dev/null || \
    command -v kicad &>/dev/null || \
    [[ -x "/usr/bin/kicad-cli" ]] || \
    [[ -x "/usr/local/bin/kicad-cli" ]] || \
    [[ -x "/opt/kicad/bin/kicad-cli" ]] || \
    [[ -x "/Applications/KiCad/KiCad.app/Contents/MacOS/kicad-cli" ]] || \
    [[ -x "${KICAD_DATA_DIR}/kicad-cli-mock" ]] || \
    [[ -x "${KICAD_DATA_DIR}/kicad-mock" ]]
}

# Get KiCad version
kicad::get_version() {
    # Try standard command first
    if command -v kicad-cli &>/dev/null; then
        kicad-cli version 2>/dev/null | head -1 | cut -d' ' -f2
    elif command -v kicad &>/dev/null; then
        kicad --version 2>/dev/null | head -1 | cut -d' ' -f2
    # Try known installation locations
    elif [[ -x "/usr/bin/kicad-cli" ]]; then
        /usr/bin/kicad-cli version 2>/dev/null | head -1 | cut -d' ' -f2
    elif [[ -x "/usr/local/bin/kicad-cli" ]]; then
        /usr/local/bin/kicad-cli version 2>/dev/null | head -1 | cut -d' ' -f2
    elif [[ -x "/opt/kicad/bin/kicad-cli" ]]; then
        /opt/kicad/bin/kicad-cli version 2>/dev/null | head -1 | cut -d' ' -f2
    elif [[ -x "/Applications/KiCad/KiCad.app/Contents/MacOS/kicad-cli" ]]; then
        /Applications/KiCad/KiCad.app/Contents/MacOS/kicad-cli version 2>/dev/null | head -1 | cut -d' ' -f2
    # Fall back to mocks
    elif [[ -x "${KICAD_DATA_DIR}/kicad-cli-mock" ]]; then
        "${KICAD_DATA_DIR}/kicad-cli-mock" version 2>/dev/null
    elif [[ -x "${KICAD_DATA_DIR}/kicad-mock" ]]; then
        "${KICAD_DATA_DIR}/kicad-mock" --version 2>/dev/null | head -1 | cut -d' ' -f2
    else
        echo "not_installed"
    fi
}

# Get KiCad CLI path
kicad::get_cli_path() {
    # Return path to working kicad-cli binary
    if command -v kicad-cli &>/dev/null; then
        command -v kicad-cli
    elif [[ -x "/usr/bin/kicad-cli" ]]; then
        echo "/usr/bin/kicad-cli"
    elif [[ -x "/usr/local/bin/kicad-cli" ]]; then
        echo "/usr/local/bin/kicad-cli"
    elif [[ -x "/opt/kicad/bin/kicad-cli" ]]; then
        echo "/opt/kicad/bin/kicad-cli"
    elif [[ -x "/Applications/KiCad/KiCad.app/Contents/MacOS/kicad-cli" ]]; then
        echo "/Applications/KiCad/KiCad.app/Contents/MacOS/kicad-cli"
    elif [[ -x "${KICAD_DATA_DIR}/kicad-cli-mock" ]]; then
        echo "${KICAD_DATA_DIR}/kicad-cli-mock"
    else
        return 1
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
export -f kicad::get_cli_path
export -f kicad::check_python_api