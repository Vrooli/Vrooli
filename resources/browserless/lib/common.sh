#!/usr/bin/env bash
#
# Browserless common functions and variables

set -euo pipefail

# Configuration
export BROWSERLESS_PORT="${BROWSERLESS_PORT:-4110}"
export BROWSERLESS_CONTAINER_NAME="${BROWSERLESS_CONTAINER_NAME:-browserless}"
export BROWSERLESS_IMAGE="${BROWSERLESS_IMAGE:-browserless/chrome:latest}"
export BROWSERLESS_DATA_DIR="${BROWSERLESS_DATA_DIR:-$HOME/.vrooli/browserless}"
export BROWSERLESS_MAX_CONCURRENT_SESSIONS="${BROWSERLESS_MAX_CONCURRENT_SESSIONS:-10}"
export BROWSERLESS_TOKEN="${BROWSERLESS_TOKEN:-}"
export BROWSERLESS_WORKSPACE_DIR="${BROWSERLESS_WORKSPACE_DIR:-$BROWSERLESS_DATA_DIR/workspace}"
export BROWSERLESS_BASE_URL="http://localhost:${BROWSERLESS_PORT}"

# Create data directories
function ensure_directories() {
    mkdir -p "$BROWSERLESS_DATA_DIR"
    mkdir -p "$BROWSERLESS_WORKSPACE_DIR"
}

# Check if browserless is running
function is_running() {
    docker ps --format "{{.Names}}" 2>/dev/null | grep -q "^${BROWSERLESS_CONTAINER_NAME}$"
}

# Get container status
function get_container_status() {
    docker inspect "$BROWSERLESS_CONTAINER_NAME" 2>/dev/null | jq -r '.[0].State.Status' 2>/dev/null || echo "not_found"
}

# Check health endpoint
function check_health() {
    local url="http://localhost:${BROWSERLESS_PORT}/pressure"
    if timeout 5 curl -s "$url" >/dev/null 2>&1; then
        echo "healthy"
    else
        echo "unhealthy"
    fi
}

# Get metrics
function get_metrics() {
    local url="http://localhost:${BROWSERLESS_PORT}/pressure"
    timeout 5 curl -s "$url" 2>/dev/null || echo "{}"
}
