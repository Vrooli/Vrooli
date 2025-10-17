#!/usr/bin/env bash
################################################################################
# Unstructured.io Smoke Test - v2.0 Contract Compliant
# 
# Quick health validation for unstructured-io resource
# Must complete in <30s as per universal.yaml requirements
#
################################################################################

set -euo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"

# Source resource configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Test configuration
TIMEOUT_HEALTH=5
MAX_WAIT=20

log::header "Unstructured.io Smoke Test"

# Test 1: Check if container exists
log::info "Test 1: Checking if container exists..."
if docker ps -a --format "{{.Names}}" | grep -q "^${UNSTRUCTURED_IO_CONTAINER_NAME}$"; then
    log::success "Container exists: ${UNSTRUCTURED_IO_CONTAINER_NAME}"
else
    log::error "Container not found: ${UNSTRUCTURED_IO_CONTAINER_NAME}"
    log::info "Run: vrooli resource unstructured-io manage install"
    exit 1
fi

# Test 2: Check if container is running
log::info "Test 2: Checking if container is running..."
if docker ps --format "{{.Names}}" | grep -q "^${UNSTRUCTURED_IO_CONTAINER_NAME}$"; then
    log::success "Container is running"
else
    log::warning "Container exists but not running, attempting to start..."
    if docker start "${UNSTRUCTURED_IO_CONTAINER_NAME}" &>/dev/null; then
        log::success "Container started successfully"
        # Wait for startup
        sleep 5
    else
        log::error "Failed to start container"
        exit 1
    fi
fi

# Test 3: Health check endpoint
log::info "Test 3: Testing health endpoint..."
wait_time=0
health_ok=false

while [[ $wait_time -lt $MAX_WAIT ]]; do
    if timeout ${TIMEOUT_HEALTH} curl -sf "${UNSTRUCTURED_IO_BASE_URL}/healthcheck" &>/dev/null; then
        health_ok=true
        break
    fi
    sleep 2
    ((wait_time += 2))
    log::info "Waiting for service to be ready... (${wait_time}s/${MAX_WAIT}s)"
done

if [[ "$health_ok" == "true" ]]; then
    # Get actual response for verification
    response=$(timeout ${TIMEOUT_HEALTH} curl -sf "${UNSTRUCTURED_IO_BASE_URL}/healthcheck" 2>/dev/null || echo "{}")
    if echo "$response" | grep -q "OK"; then
        log::success "Health check passed - service is healthy"
    else
        log::warning "Health endpoint responded but status unclear"
        echo "Response: $response"
    fi
else
    log::error "Health check failed - timeout after ${MAX_WAIT}s"
    exit 1
fi

# Test 4: Check processing endpoint availability
log::info "Test 4: Checking processing endpoint..."
if timeout ${TIMEOUT_HEALTH} curl -sf -X OPTIONS "${UNSTRUCTURED_IO_BASE_URL}/general/v0/general" &>/dev/null; then
    log::success "Processing endpoint is accessible"
else
    # Not critical for smoke test, just warn
    log::warning "Processing endpoint not responding to OPTIONS"
fi

# Test 5: Memory and resource check
log::info "Test 5: Checking resource usage..."
container_stats=$(docker stats --no-stream --format "json" "${UNSTRUCTURED_IO_CONTAINER_NAME}" 2>/dev/null || echo "{}")
if [[ -n "$container_stats" ]] && [[ "$container_stats" != "{}" ]]; then
    mem_usage=$(echo "$container_stats" | jq -r '.MemUsage' 2>/dev/null || echo "unknown")
    cpu_usage=$(echo "$container_stats" | jq -r '.CPUPerc' 2>/dev/null || echo "unknown")
    log::success "Resource usage - Memory: ${mem_usage}, CPU: ${cpu_usage}"
else
    log::warning "Could not retrieve container stats"
fi

log::success "Smoke test completed successfully"
exit 0