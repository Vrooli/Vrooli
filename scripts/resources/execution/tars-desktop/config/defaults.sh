#!/bin/bash
# TARS-desktop configuration defaults

# Resource settings
export TARS_DESKTOP_NAME="tars-desktop"
export TARS_DESKTOP_CATEGORY="execution"

# Installation settings
export TARS_DESKTOP_REPO="https://github.com/cccc-icu/UI-TARS-desktop"
export TARS_DESKTOP_INSTALL_DIR="${var_DATA_DIR:-/home/matthalloran8/Vrooli/data}/tars-desktop"
export TARS_DESKTOP_CACHE_DIR="${var_CACHE_DIR:-/home/matthalloran8/Vrooli/.cache}/tars-desktop"

# Python environment
export TARS_DESKTOP_PYTHON_VERSION="3.11"
export TARS_DESKTOP_VENV_DIR="${TARS_DESKTOP_INSTALL_DIR}/venv"

# Port configuration - using port registry
export TARS_DESKTOP_PORT="11570"  # Will be registered in port registry

# API configuration
export TARS_DESKTOP_API_BASE="http://localhost:${TARS_DESKTOP_PORT}"
export TARS_DESKTOP_API_KEY="${TARS_DESKTOP_API_KEY:-}"

# Model configuration
export TARS_DESKTOP_MODEL="${TARS_DESKTOP_MODEL:-gpt-4o}"
export TARS_DESKTOP_MODEL_PROVIDER="${TARS_DESKTOP_MODEL_PROVIDER:-openrouter}"

# Runtime settings
export TARS_DESKTOP_MAX_MEMORY="2g"
export TARS_DESKTOP_LOG_LEVEL="${TARS_DESKTOP_LOG_LEVEL:-info}"
export TARS_DESKTOP_TIMEOUT="${TARS_DESKTOP_TIMEOUT:-300}"

# Health check settings
export TARS_DESKTOP_HEALTH_CHECK_INTERVAL="30"
export TARS_DESKTOP_HEALTH_CHECK_TIMEOUT="10"
export TARS_DESKTOP_HEALTH_CHECK_RETRIES="3"

# Export all variables
export -p | grep "^export TARS_DESKTOP_" | cut -d' ' -f3- | while IFS='=' read -r key value; do
    export "$key"
done