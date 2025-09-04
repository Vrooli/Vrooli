#!/usr/bin/env bash
# Redis Resource Environment Exports v2.0

# Source required utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
export REDIS_CONFIG_DIR="${APP_ROOT}/resources/redis/config"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source secrets management
if ! declare -f secrets::resolve &>/dev/null; then
    source "${APP_ROOT}/scripts/lib/service/secrets.sh"
fi

# Helper for debug output
_redis_debug() {
    [[ "${DEBUG:-false}" == "true" ]] && echo "[redis/exports] $*" >&2 || true
}

export REDIS_RESOURCE_VERSION="2.0.0"
export REDIS_RESOURCE_NAME="redis"
# Use var.sh paths if available, otherwise fallback
export REDIS_RESOURCE_DIR="${REDIS_RESOURCE_DIR:-${var_SCRIPTS_RESOURCES_DIR}/caching/redis}"
export REDIS_RESOURCE_DIR="${REDIS_RESOURCE_DIR:-${APP_ROOT}/resources/redis}"

# Get port from registry (if function is available)
if declare -f secrets::source_port_registry &>/dev/null; then
    secrets::source_port_registry
fi
export REDIS_PORT="${REDIS_PORT:-${RESOURCE_PORTS[redis]:-6380}}"
export REDIS_HOST="${REDIS_HOST:-localhost}"

# Password (usually empty for local dev)
if [[ -z "${REDIS_PASSWORD:-}" ]]; then
    _redis_debug "Resolving REDIS_PASSWORD..."
    REDIS_PASSWORD=$(secrets::resolve "REDIS_PASSWORD" 2>/dev/null || echo "")
    export REDIS_PASSWORD
    
    if [[ -n "$REDIS_PASSWORD" ]]; then
        _redis_debug "REDIS_PASSWORD resolved successfully"
    else
        _redis_debug "No Redis password configured (normal for local development)"
    fi
fi

# Computed URL
if [[ -n "$REDIS_PASSWORD" ]]; then
    export REDIS_URL="redis://:${REDIS_PASSWORD}@${REDIS_HOST}:${REDIS_PORT}"
    _redis_debug "Redis URL configured with authentication"
else
    export REDIS_URL="redis://${REDIS_HOST}:${REDIS_PORT}"
    _redis_debug "Redis URL configured without authentication"
fi

# Configuration
export REDIS_DB="${REDIS_DB:-0}"
export REDIS_MAX_RETRIES="${REDIS_MAX_RETRIES:-3}"
export REDIS_TIMEOUT="${REDIS_TIMEOUT:-5000}"

# Health check
export REDIS_HEALTH_CHECK="redis-cli -h ${REDIS_HOST} -p ${REDIS_PORT} ping"

# Validate we have minimum required configuration
if [[ -z "$REDIS_PORT" ]] || [[ -z "$REDIS_HOST" ]]; then
    if declare -f log::warning &>/dev/null; then
        log::warning "Redis configuration incomplete: host=$REDIS_HOST port=$REDIS_PORT"
    else
        echo "[WARNING] Redis configuration incomplete: host=$REDIS_HOST port=$REDIS_PORT" >&2
    fi
fi