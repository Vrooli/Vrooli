#!/usr/bin/env bash
# Qdrant Resource Environment Exports v2.0
# Self-contained exports without circular dependencies
# Mirrors the Ollama exports.sh pattern for lifecycle compatibility

# Source required utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Helper for debug output
_qdrant_debug() {
    [[ "${DEBUG:-false}" == "true" ]] && echo "[qdrant/exports] $*" >&2 || true
}

# Resource metadata
export QDRANT_RESOURCE_VERSION="2.0.0"
export QDRANT_RESOURCE_NAME="qdrant"

# Get port from registry (if available)
if [[ -f "${APP_ROOT}/scripts/resources/port_registry.sh" ]]; then
    source "${APP_ROOT}/scripts/resources/port_registry.sh" 2>/dev/null || true
fi
export QDRANT_PORT="${QDRANT_PORT:-${RESOURCE_PORTS[qdrant]:-6333}}"
export QDRANT_GRPC_PORT="${QDRANT_GRPC_PORT:-6334}"

# Service URLs
export QDRANT_HOST="${QDRANT_HOST:-localhost}"
export QDRANT_URL="http://${QDRANT_HOST}:${QDRANT_PORT}"
export QDRANT_BASE_URL="$QDRANT_URL"
export QDRANT_GRPC_URL="grpc://${QDRANT_HOST}:${QDRANT_GRPC_PORT}"

# Container/storage configuration
export QDRANT_CONTAINER_NAME="${QDRANT_CONTAINER_NAME:-qdrant}"
export QDRANT_DATA_DIR="${QDRANT_DATA_DIR:-${HOME}/.qdrant/data}"
export QDRANT_CONFIG_DIR="${QDRANT_CONFIG_DIR:-${HOME}/.qdrant/config}"
export QDRANT_SNAPSHOTS_DIR="${QDRANT_SNAPSHOTS_DIR:-${HOME}/.qdrant/snapshots}"
export QDRANT_IMAGE="${QDRANT_IMAGE:-qdrant/qdrant:latest}"
export QDRANT_VERSION="${QDRANT_VERSION:-latest}"

# API Key (empty = no authentication)
export QDRANT_API_KEY="${QDRANT_API_KEY:-}"

# Health check command
export QDRANT_HEALTH_CHECK="curl -s ${QDRANT_URL}/healthz"

_qdrant_debug "Qdrant exports configured: URL=${QDRANT_URL}, PORT=${QDRANT_PORT}"
