#!/usr/bin/env bash
# Apache Superset Default Configuration

# Service configuration - Load from port registry
DEFAULTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PORT_REGISTRY="${DEFAULTS_DIR}/../../../scripts/resources/port_registry.sh"
if [[ -f "$PORT_REGISTRY" ]]; then
    source "$PORT_REGISTRY"
    export SUPERSET_PORT="${SUPERSET_PORT:-${RESOURCE_PORTS["apache-superset"]}}"
    export SUPERSET_POSTGRES_PORT="${SUPERSET_POSTGRES_PORT:-${RESOURCE_PORTS["apache-superset-postgres"]}}"
    export SUPERSET_REDIS_PORT="${SUPERSET_REDIS_PORT:-${RESOURCE_PORTS["apache-superset-redis"]}}"
else
    echo "Error: Port registry not found at $PORT_REGISTRY" >&2
    exit 1
fi
export SUPERSET_WORKER_PORT="${SUPERSET_WORKER_PORT:-5555}"
export SUPERSET_FLOWER_PORT="${SUPERSET_FLOWER_PORT:-5556}"

# Container names
export SUPERSET_CONTAINER_PREFIX="${SUPERSET_CONTAINER_PREFIX:-vrooli-superset}"
export SUPERSET_APP_CONTAINER="${SUPERSET_CONTAINER_PREFIX}-app"
export SUPERSET_POSTGRES_CONTAINER="${SUPERSET_CONTAINER_PREFIX}-postgres"
export SUPERSET_REDIS_CONTAINER="${SUPERSET_CONTAINER_PREFIX}-redis"
export SUPERSET_WORKER_CONTAINER="${SUPERSET_CONTAINER_PREFIX}-worker"
export SUPERSET_WORKER_BEAT_CONTAINER="${SUPERSET_CONTAINER_PREFIX}-worker-beat"

# Docker configuration
export SUPERSET_IMAGE="${SUPERSET_IMAGE:-apache/superset:3.1.0}"
export SUPERSET_POSTGRES_IMAGE="${SUPERSET_POSTGRES_IMAGE:-postgres:14-alpine}"
export SUPERSET_REDIS_IMAGE="${SUPERSET_REDIS_IMAGE:-redis:7-alpine}"
export SUPERSET_NETWORK="${SUPERSET_NETWORK:-vrooli-superset-net}"

# Database configuration (metadata DB)
export SUPERSET_POSTGRES_USER="${SUPERSET_POSTGRES_USER:-superset}"
export SUPERSET_POSTGRES_PASSWORD="${SUPERSET_POSTGRES_PASSWORD:-superset_secure_password}"
export SUPERSET_POSTGRES_DB="${SUPERSET_POSTGRES_DB:-superset}"

# Redis configuration (port already set from registry above)

# Superset configuration
export SUPERSET_ADMIN_USERNAME="${SUPERSET_ADMIN_USERNAME:-admin}"
export SUPERSET_ADMIN_PASSWORD="${SUPERSET_ADMIN_PASSWORD:-admin}"
export SUPERSET_ADMIN_EMAIL="${SUPERSET_ADMIN_EMAIL:-admin@vrooli.local}"
export SUPERSET_SECRET_KEY="${SUPERSET_SECRET_KEY:-$(openssl rand -base64 42 2>/dev/null || echo 'dev-secret-key-change-in-production')}"

# Data directories
export SUPERSET_DATA_DIR="${SUPERSET_DATA_DIR:-${HOME}/.vrooli/apache-superset}"
export SUPERSET_CONFIG_DIR="${SUPERSET_DATA_DIR}/config"
export SUPERSET_POSTGRES_DATA="${SUPERSET_DATA_DIR}/postgres"
export SUPERSET_UPLOADS_DIR="${SUPERSET_DATA_DIR}/uploads"
export SUPERSET_CACHE_DIR="${SUPERSET_DATA_DIR}/cache"

# Vrooli database connections (pre-configured)
export VROOLI_POSTGRES_HOST="${VROOLI_POSTGRES_HOST:-host.docker.internal}"
export VROOLI_POSTGRES_PORT="${VROOLI_POSTGRES_PORT:-5433}"
export VROOLI_QUESTDB_HOST="${VROOLI_QUESTDB_HOST:-host.docker.internal}"
export VROOLI_QUESTDB_PORT="${VROOLI_QUESTDB_PORT:-8812}"  # PostgreSQL wire protocol port
export VROOLI_MINIO_HOST="${VROOLI_MINIO_HOST:-host.docker.internal}"
export VROOLI_MINIO_PORT="${VROOLI_MINIO_PORT:-9000}"

# Timeouts
export SUPERSET_STARTUP_TIMEOUT="${SUPERSET_STARTUP_TIMEOUT:-120}"
export SUPERSET_SHUTDOWN_TIMEOUT="${SUPERSET_SHUTDOWN_TIMEOUT:-30}"
export SUPERSET_HEALTH_CHECK_INTERVAL="${SUPERSET_HEALTH_CHECK_INTERVAL:-5}"

# Logging
export SUPERSET_LOG_LEVEL="${SUPERSET_LOG_LEVEL:-INFO}"
export SUPERSET_LOG_FILE="${SUPERSET_DATA_DIR}/logs/superset.log}"

# Performance tuning
export SUPERSET_WORKERS="${SUPERSET_WORKERS:-4}"
export SUPERSET_WORKER_TIMEOUT="${SUPERSET_WORKER_TIMEOUT:-120}"
export SUPERSET_ROW_LIMIT="${SUPERSET_ROW_LIMIT:-100000}"
export SUPERSET_VIZ_ROW_LIMIT="${SUPERSET_VIZ_ROW_LIMIT:-10000}"

# Common functions
log::info() { echo "[INFO] $*"; }
log::error() { echo "[ERROR] $*" >&2; }
log::success() { echo "[SUCCESS] $*"; }
log::warning() { echo "[WARNING] $*"; }