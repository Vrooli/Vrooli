#!/usr/bin/env bash
#
# Browserless common functions and variables

set -euo pipefail

# Configuration - only set if not already defined by defaults.sh
[[ -z "${BROWSERLESS_PORT:-}" ]] && export BROWSERLESS_PORT="4110"
[[ -z "${BROWSERLESS_CONTAINER_NAME:-}" ]] && export BROWSERLESS_CONTAINER_NAME="vrooli-browserless"
[[ -z "${BROWSERLESS_IMAGE:-}" ]] && export BROWSERLESS_IMAGE="ghcr.io/browserless/chrome@sha256:96cc9039f44c8a7b277846783f18c1ec501a7f8b1b12bdfc2bc1f9c3f84a9a17"
[[ -z "${BROWSERLESS_DATA_DIR:-}" ]] && export BROWSERLESS_DATA_DIR="$HOME/.vrooli/browserless"
[[ -z "${BROWSERLESS_MAX_BROWSERS:-}" ]] && export BROWSERLESS_MAX_BROWSERS="1"
# Align concurrent session cap with MAX_BROWSERS default unless explicitly overridden
[[ -z "${BROWSERLESS_MAX_CONCURRENT_SESSIONS:-}" ]] && export BROWSERLESS_MAX_CONCURRENT_SESSIONS="${BROWSERLESS_MAX_BROWSERS:-1}"
[[ -z "${BROWSERLESS_TOKEN:-}" ]] && export BROWSERLESS_TOKEN=""
[[ -z "${BROWSERLESS_WORKSPACE_DIR:-}" ]] && export BROWSERLESS_WORKSPACE_DIR="${BROWSERLESS_DATA_DIR}/workspace"
[[ -z "${BROWSERLESS_BASE_URL:-}" ]] && export BROWSERLESS_BASE_URL="http://localhost:${BROWSERLESS_PORT}"
[[ -z "${BROWSERLESS_DEFAULT_LAUNCH_ARGS:-}" ]] && export BROWSERLESS_DEFAULT_LAUNCH_ARGS="--no-sandbox --disable-dev-shm-usage --disable-gpu --disable-software-rasterizer --disable-dev-tools --disable-features=TranslateUI --disable-extensions --disable-background-networking --no-first-run --mute-audio"

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
