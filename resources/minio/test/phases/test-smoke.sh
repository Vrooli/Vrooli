#!/usr/bin/env bash
################################################################################
# MinIO Smoke Tests - Quick Health Validation
#
# Tests that MinIO service is running and healthy (< 30s)
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
MINIO_DIR="$(builtin cd "${SCRIPT_DIR}/../.." && builtin pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${MINIO_DIR}/../.." && builtin pwd)}"

# Source dependencies
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${MINIO_DIR}/config/defaults.sh"

# Export configuration
minio::export_config 2>/dev/null || true

################################################################################
# Smoke Tests
################################################################################

log::info "Starting MinIO smoke tests..."

FAILED=0

# Test 1: Check if MinIO container exists
log::info "Test 1: Checking MinIO installation..."
CONTAINER_NAME="${MINIO_CONTAINER_NAME:-minio}"

if docker ps -a --filter "name=$CONTAINER_NAME" --format '{{.Names}}' 2>/dev/null | grep -q "$CONTAINER_NAME"; then
    log::success "✓ MinIO container exists"
else
    log::error "✗ MinIO container not found"
    ((FAILED++))
fi

# Test 2: Check if MinIO is running
log::info "Test 2: Checking MinIO runtime status..."

if docker ps --filter "name=$CONTAINER_NAME" --format '{{.Names}}' 2>/dev/null | grep -q "$CONTAINER_NAME"; then
    log::success "✓ MinIO is running"
else
    log::error "✗ MinIO is not running"
    ((FAILED++))
fi

# Test 3: Check health endpoint
log::info "Test 3: Checking MinIO health endpoint..."
API_PORT="${MINIO_PORT:-9000}"

if timeout 5 curl -sf "http://localhost:${API_PORT}/minio/health/live" &>/dev/null; then
    log::success "✓ MinIO health endpoint responds"
else
    log::error "✗ MinIO health endpoint not responding"
    ((FAILED++))
fi

# Test 4: Check API port accessibility
log::info "Test 4: Checking API port accessibility..."

if timeout 5 nc -zv localhost "$API_PORT" &>/dev/null 2>&1; then
    log::success "✓ MinIO API port ${API_PORT} is accessible"
else
    log::error "✗ MinIO API port ${API_PORT} is not accessible"
    ((FAILED++))
fi

# Test 5: Check console port accessibility
log::info "Test 5: Checking console port accessibility..."
CONSOLE_PORT="${MINIO_CONSOLE_PORT:-9001}"

if timeout 5 nc -zv localhost "$CONSOLE_PORT" &>/dev/null 2>&1; then
    log::success "✓ MinIO console port ${CONSOLE_PORT} is accessible"
else
    log::warning "⚠ MinIO console port ${CONSOLE_PORT} is not accessible (optional)"
fi

# Test 6: Quick response time check
log::info "Test 6: Checking health endpoint response time..."
START_TIME=$(date +%s%N)

if timeout 1 curl -sf "http://localhost:${API_PORT}/minio/health/live" &>/dev/null; then
    END_TIME=$(date +%s%N)
    RESPONSE_TIME=$(( (END_TIME - START_TIME) / 1000000 ))
    
    if [[ $RESPONSE_TIME -lt 1000 ]]; then
        log::success "✓ Health endpoint responds in ${RESPONSE_TIME}ms (<1s requirement)"
    else
        log::warning "⚠ Health endpoint slow: ${RESPONSE_TIME}ms"
    fi
else
    log::error "✗ Health endpoint timeout"
    ((FAILED++))
fi

################################################################################
# Results
################################################################################

if [[ $FAILED -gt 0 ]]; then
    log::error "MinIO smoke tests failed: $FAILED tests failed"
    exit 1
else
    log::success "All MinIO smoke tests passed"
    exit 0
fi