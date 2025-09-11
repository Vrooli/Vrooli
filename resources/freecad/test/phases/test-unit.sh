#!/usr/bin/env bash
################################################################################
# FreeCAD Resource - Unit Tests
# 
# Tests individual library functions
################################################################################

set -euo pipefail

# Determine paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities and configuration
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${RESOURCE_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${RESOURCE_DIR}/lib/core.sh"

log::info "Starting FreeCAD unit tests..."

# Test 1: Port retrieval
log::info "Test 1: Testing port retrieval..."
port=$(freecad::get_port)
if [[ "$port" =~ ^[0-9]+$ ]] && [[ "$port" -gt 1024 ]] && [[ "$port" -lt 65536 ]]; then
    log::info "✓ Port retrieval works: $port"
else
    log::error "✗ Invalid port retrieved: $port"
    exit 1
fi

# Test 2: Directory initialization
log::info "Test 2: Testing directory initialization..."
freecad::init
if [[ -d "${FREECAD_DATA_DIR}" ]] && \
   [[ -d "${FREECAD_PROJECTS_DIR}" ]] && \
   [[ -d "${FREECAD_SCRIPTS_DIR}" ]] && \
   [[ -d "${FREECAD_EXPORTS_DIR}" ]]; then
    log::info "✓ Directory initialization works"
else
    log::error "✗ Directory initialization failed"
    exit 1
fi

# Test 3: Container name validation
log::info "Test 3: Testing container name..."
if [[ -n "${FREECAD_CONTAINER_NAME}" ]]; then
    log::info "✓ Container name is set: ${FREECAD_CONTAINER_NAME}"
else
    log::error "✗ Container name is not set"
    exit 1
fi

# Test 4: Configuration values
log::info "Test 4: Testing configuration values..."
errors=0

# Check memory limit format
if [[ ! "${FREECAD_MEMORY_LIMIT}" =~ ^[0-9]+[kmg]$ ]]; then
    log::error "✗ Invalid memory limit format: ${FREECAD_MEMORY_LIMIT}"
    ((errors++))
fi

# Check CPU limit
if ! echo "${FREECAD_CPU_LIMIT}" | grep -qE '^[0-9]+(\.[0-9]+)?$'; then
    log::error "✗ Invalid CPU limit format: ${FREECAD_CPU_LIMIT}"
    ((errors++))
fi

# Check thread count
if [[ ! "${FREECAD_THREAD_COUNT}" =~ ^[0-9]+$ ]]; then
    log::error "✗ Invalid thread count: ${FREECAD_THREAD_COUNT}"
    ((errors++))
fi

if [[ $errors -eq 0 ]]; then
    log::info "✓ All configuration values are valid"
else
    log::error "✗ Configuration validation failed"
    exit 1
fi

# Test 5: Export format validation
log::info "Test 5: Testing export format validation..."
valid_formats=("STEP" "IGES" "STL" "OBJ" "DXF" "SVG")
if [[ " ${valid_formats[@]} " =~ " ${FREECAD_DEFAULT_EXPORT_FORMAT} " ]]; then
    log::info "✓ Default export format is valid: ${FREECAD_DEFAULT_EXPORT_FORMAT}"
else
    log::error "✗ Invalid default export format: ${FREECAD_DEFAULT_EXPORT_FORMAT}"
    exit 1
fi

log::info "All unit tests passed successfully"
exit 0