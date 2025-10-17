#!/usr/bin/env bash
# PostgreSQL Resource Environment Exports v2.0
# Self-contained exports without circular dependencies

# Source required utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
export POSTGRES_CONFIG_DIR="${APP_ROOT}/resources/postgres/config"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source secrets management (from scripts/lib, not resource lib)
if ! declare -f secrets::resolve &>/dev/null; then
    source "${APP_ROOT}/scripts/lib/service/secrets.sh"
fi

# Helper for debug output
_postgres_debug() {
    [[ "${DEBUG:-false}" == "true" ]] && echo "[postgres/exports] $*" >&2 || true
}

# Resource metadata
export POSTGRES_RESOURCE_VERSION="2.0.0"
export POSTGRES_RESOURCE_NAME="postgres"
# Use var.sh paths if available, otherwise fallback
export POSTGRES_RESOURCE_DIR="${POSTGRES_RESOURCE_DIR:-${var_SCRIPTS_RESOURCES_DIR}/storage/postgres}"
export POSTGRES_RESOURCE_DIR="${POSTGRES_RESOURCE_DIR:-${APP_ROOT}/resources/postgres}"

# Get port from registry (if function is available)
if declare -f secrets::source_port_registry &>/dev/null; then
    secrets::source_port_registry
fi
export POSTGRES_PORT="${POSTGRES_PORT:-${RESOURCE_PORTS[postgres]:-5433}}"

# Simple instance.conf reader (avoiding circular deps with lib/common.sh)
_postgres_get_instance_config() {
    local instance="${1:-main}"
    local key="$2"
    # Use CONFIG_DIR parent instead of hardcoding path
    local config_file="${POSTGRES_CONFIG_DIR%/config}/instances/${instance}/config/instance.conf"
    
    [[ -f "$config_file" ]] && grep "^${key}=" "$config_file" 2>/dev/null | cut -d'=' -f2- | tr -d '"'
}

# Determine which instance we're working with
export POSTGRES_INSTANCE="${POSTGRES_INSTANCE:-main}"

# Get credentials with proper fallback chain:
# 1. Already set in environment (respect existing)
# 2. Vault (via secrets::resolve) 
# 3. secrets.json (via secrets::resolve)
# 4. instance.conf (for existing instances)
# 5. Defaults

if [[ -z "${POSTGRES_USER:-}" ]]; then
    _postgres_debug "Resolving POSTGRES_USER..."
    POSTGRES_USER=$(secrets::resolve "POSTGRES_USER" 2>/dev/null || true)
    [[ -z "$POSTGRES_USER" ]] && POSTGRES_USER=$(_postgres_get_instance_config "$POSTGRES_INSTANCE" "user")
    [[ -z "$POSTGRES_USER" ]] && POSTGRES_USER="vrooli"
    export POSTGRES_USER
    _postgres_debug "POSTGRES_USER set to: $POSTGRES_USER"
fi

if [[ -z "${POSTGRES_PASSWORD:-}" ]]; then
    _postgres_debug "Resolving POSTGRES_PASSWORD..."
    POSTGRES_PASSWORD=$(secrets::resolve "POSTGRES_PASSWORD" 2>/dev/null || true)
    [[ -z "$POSTGRES_PASSWORD" ]] && POSTGRES_PASSWORD=$(_postgres_get_instance_config "$POSTGRES_INSTANCE" "password")
    export POSTGRES_PASSWORD
    
    # Warn if no password found
    if [[ -z "$POSTGRES_PASSWORD" ]]; then
        if declare -f log::warning &>/dev/null; then
            log::warning "PostgreSQL password not found in vault, secrets.json, or instance.conf"
        else
            echo "[WARNING] PostgreSQL password not found in vault, secrets.json, or instance.conf" >&2
        fi
    else
        _postgres_debug "POSTGRES_PASSWORD resolved successfully"
    fi
fi

if [[ -z "${POSTGRES_DB:-}" ]]; then
    _postgres_debug "Resolving POSTGRES_DB..."
    POSTGRES_DB=$(secrets::resolve "POSTGRES_DB" 2>/dev/null || true)
    [[ -z "$POSTGRES_DB" ]] && POSTGRES_DB=$(_postgres_get_instance_config "$POSTGRES_INSTANCE" "database")
    [[ -z "$POSTGRES_DB" ]] && POSTGRES_DB="vrooli"
    export POSTGRES_DB
    _postgres_debug "POSTGRES_DB set to: $POSTGRES_DB"
fi

# Port might be instance-specific
if [[ -z "${POSTGRES_PORT:-}" ]]; then
    instance_port=$(_postgres_get_instance_config "$POSTGRES_INSTANCE" "port")
    [[ -n "$instance_port" ]] && export POSTGRES_PORT="$instance_port"
fi

# Standard connection variables
export POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
export POSTGRES_SSLMODE="${POSTGRES_SSLMODE:-disable}"

# Computed URLs (only if we have password)
if [[ -n "$POSTGRES_PASSWORD" ]]; then
    export POSTGRES_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=${POSTGRES_SSLMODE}"
    export DATABASE_URL="$POSTGRES_URL"
    export POSTGRES_ADMIN_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/postgres?sslmode=${POSTGRES_SSLMODE}"
    _postgres_debug "PostgreSQL URLs configured successfully"
else
    # Don't export invalid URLs
    unset POSTGRES_URL DATABASE_URL POSTGRES_ADMIN_URL
    _postgres_debug "PostgreSQL URLs not set due to missing password"
fi

# Additional metadata
export POSTGRES_MAX_CONNECTIONS="${POSTGRES_MAX_CONNECTIONS:-100}"
export POSTGRES_POOL_SIZE="${POSTGRES_POOL_SIZE:-10}"

# Health check command
export POSTGRES_HEALTH_CHECK="pg_isready -h ${POSTGRES_HOST} -p ${POSTGRES_PORT} -U ${POSTGRES_USER}"

# Paths
export POSTGRES_SCHEMA_PATH="${POSTGRES_RESOURCE_DIR}/schemas"
export POSTGRES_MIGRATION_PATH="${POSTGRES_RESOURCE_DIR}/migrations"