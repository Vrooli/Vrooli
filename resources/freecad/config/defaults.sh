#!/usr/bin/env bash
################################################################################
# FreeCAD Resource - Default Configuration
# 
# Parametric 3D CAD modeler with Python API
################################################################################

# Service Configuration
export FREECAD_ENABLED="${FREECAD_ENABLED:-true}"
# Using the minimal Python image approach for now
export FREECAD_IMAGE="${FREECAD_IMAGE:-python:3.11-slim}"
export FREECAD_CONTAINER_NAME="${FREECAD_CONTAINER_NAME:-vrooli-freecad}"

# Port Configuration - MUST use port registry
# Port is retrieved dynamically from port_registry.sh
export FREECAD_PORT="${FREECAD_PORT:-}"  # Set by lifecycle scripts

# Display Configuration (Xvfb for headless operation)
export FREECAD_DISPLAY="${FREECAD_DISPLAY:-:99}"
export FREECAD_DISPLAY_RESOLUTION="${FREECAD_DISPLAY_RESOLUTION:-1920x1080x24}"

# Resource Limits
export FREECAD_MEMORY_LIMIT="${FREECAD_MEMORY_LIMIT:-4096m}"
export FREECAD_CPU_LIMIT="${FREECAD_CPU_LIMIT:-2.0}"
export FREECAD_THREAD_COUNT="${FREECAD_THREAD_COUNT:-4}"

# Storage Configuration
export FREECAD_DATA_DIR="${FREECAD_DATA_DIR:-${HOME}/.vrooli/freecad/data}"
export FREECAD_PROJECTS_DIR="${FREECAD_PROJECTS_DIR:-${HOME}/.vrooli/freecad/projects}"
export FREECAD_SCRIPTS_DIR="${FREECAD_SCRIPTS_DIR:-${HOME}/.vrooli/freecad/scripts}"
export FREECAD_EXPORTS_DIR="${FREECAD_EXPORTS_DIR:-${HOME}/.vrooli/freecad/exports}"

# Python Configuration
export FREECAD_PYTHON_VERSION="${FREECAD_PYTHON_VERSION:-3.9}"
export FREECAD_MACRO_PATH="${FREECAD_MACRO_PATH:-/usr/share/freecad/Macro}"

# Workbench Configuration
export FREECAD_ENABLE_FEM="${FREECAD_ENABLE_FEM:-true}"
export FREECAD_ENABLE_PATH="${FREECAD_ENABLE_PATH:-true}"
export FREECAD_ENABLE_ASSEMBLY="${FREECAD_ENABLE_ASSEMBLY:-true}"
export FREECAD_ENABLE_ARCH="${FREECAD_ENABLE_ARCH:-false}"

# Export Configuration
export FREECAD_DEFAULT_EXPORT_FORMAT="${FREECAD_DEFAULT_EXPORT_FORMAT:-STEP}"
export FREECAD_EXPORT_QUALITY="${FREECAD_EXPORT_QUALITY:-high}"
export FREECAD_MESH_RESOLUTION="${FREECAD_MESH_RESOLUTION:-0.1}"

# Performance Configuration
export FREECAD_CACHE_SIZE="${FREECAD_CACHE_SIZE:-512}"
export FREECAD_UNDO_LEVELS="${FREECAD_UNDO_LEVELS:-20}"
export FREECAD_AUTO_SAVE="${FREECAD_AUTO_SAVE:-true}"
export FREECAD_AUTO_SAVE_INTERVAL="${FREECAD_AUTO_SAVE_INTERVAL:-10}"

# Network Configuration
export FREECAD_NETWORK="${FREECAD_NETWORK:-vrooli-network}"

# Logging Configuration
export FREECAD_LOG_LEVEL="${FREECAD_LOG_LEVEL:-INFO}"
export FREECAD_LOG_FILE="${FREECAD_LOG_FILE:-${HOME}/.vrooli/freecad/logs/freecad.log}"
export FREECAD_DEBUG="${FREECAD_DEBUG:-false}"

# Health Check Configuration
export FREECAD_HEALTH_CHECK_INTERVAL="${FREECAD_HEALTH_CHECK_INTERVAL:-30}"
export FREECAD_HEALTH_CHECK_TIMEOUT="${FREECAD_HEALTH_CHECK_TIMEOUT:-5}"
export FREECAD_HEALTH_CHECK_RETRIES="${FREECAD_HEALTH_CHECK_RETRIES:-3}"

# Timeout Configuration
export FREECAD_STARTUP_TIMEOUT="${FREECAD_STARTUP_TIMEOUT:-60}"
export FREECAD_STOP_TIMEOUT="${FREECAD_STOP_TIMEOUT:-30}"
export FREECAD_SCRIPT_TIMEOUT="${FREECAD_SCRIPT_TIMEOUT:-300}"