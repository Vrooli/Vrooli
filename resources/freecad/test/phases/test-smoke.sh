#!/usr/bin/env bash
################################################################################
# FreeCAD Resource - Smoke Tests
# 
# Quick validation that FreeCAD service is running and healthy
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

# Initialize FreeCAD configuration (sets port)
freecad::init

log::info "Starting FreeCAD smoke tests..."

# Test 1: Check if FreeCAD is installed
log::info "Test 1: Checking FreeCAD installation..."
if docker images | grep -q "${FREECAD_IMAGE%%:*}"; then
    log::info "✓ FreeCAD image found"
else
    log::error "✗ FreeCAD image not found"
    exit 1
fi

# Test 2: Check if FreeCAD is running
log::info "Test 2: Checking if FreeCAD is running..."
if freecad::is_running; then
    log::info "✓ FreeCAD container is running"
else
    log::error "✗ FreeCAD container is not running"
    exit 1
fi

# Test 3: Health check
log::info "Test 3: Performing health check..."
if timeout 5 curl -sf "http://localhost:${FREECAD_PORT}/" &>/dev/null; then
    log::info "✓ FreeCAD service is responding"
else
    log::error "✗ FreeCAD service is not responding"
    exit 1
fi

# Test 4: Check container health
log::info "Test 4: Checking container health..."
container_status=$(docker inspect -f '{{.State.Status}}' "${FREECAD_CONTAINER_NAME}" 2>/dev/null || echo "unknown")
if [[ "$container_status" == "running" ]]; then
    log::info "✓ Container status is healthy: $container_status"
else
    log::error "✗ Container status is unhealthy: $container_status"
    exit 1
fi

# Test 5: Check port availability
log::info "Test 5: Checking port availability..."
if timeout 2 nc -zv localhost "${FREECAD_PORT}" 2>/dev/null; then
    log::info "✓ Port ${FREECAD_PORT} is accessible"
else
    log::error "✗ Port ${FREECAD_PORT} is not accessible"
    exit 1
fi

log::info "All smoke tests passed successfully"
exit 0