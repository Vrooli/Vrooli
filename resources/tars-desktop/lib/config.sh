#!/bin/bash
# TARS-desktop configuration

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
TARS_LIB_DIR="${APP_ROOT}/resources/tars-desktop/lib"
TARS_RESOURCE_DIR="${APP_ROOT}/resources/tars-desktop"

# Source shared utilities
source "$TARS_RESOURCE_DIR/../../../lib/utils/var.sh"

# Configuration
export TARS_DESKTOP_NAME="${TARS_DESKTOP_NAME:-tars-desktop}"
export TARS_DESKTOP_PORT="${TARS_DESKTOP_PORT:-11570}"
export TARS_DESKTOP_VERSION="${TARS_DESKTOP_VERSION:-latest}"
export TARS_DESKTOP_DATA_DIR="${var_DATA_DIR:-/home/matthalloran8/Vrooli/data}/tars-desktop"
export TARS_DESKTOP_SCRIPTS_DIR="${TARS_DESKTOP_DATA_DIR}/scripts"
export TARS_DESKTOP_VENV_DIR="${TARS_DESKTOP_DATA_DIR}/venv"
export TARS_DESKTOP_LOG_FILE="${TARS_DESKTOP_DATA_DIR}/tars-desktop.log"
export TARS_DESKTOP_PID_FILE="${TARS_DESKTOP_DATA_DIR}/tars-desktop.pid"
export TARS_DESKTOP_REPO_URL="https://github.com/OpenAdaptAI/OpenAdapt.git"
export TARS_DESKTOP_API_BASE="http://localhost:${TARS_DESKTOP_PORT}"

# Ensure data directory exists
mkdir -p "$TARS_DESKTOP_DATA_DIR"
mkdir -p "$TARS_DESKTOP_SCRIPTS_DIR"