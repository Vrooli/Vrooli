#!/bin/bash
# JupyterHub Resource Configuration Defaults

# Resource identification
export RESOURCE_NAME="jupyterhub"
export RESOURCE_DISPLAY_NAME="JupyterHub"
export RESOURCE_CATEGORY="productivity"

# Port configuration - NEVER hardcode ports
VROOLI_ROOT="${VROOLI_ROOT:-/home/matthalloran8/Vrooli}"
if [[ -f "${VROOLI_ROOT}/scripts/resources/port_registry.sh" ]]; then
    source "${VROOLI_ROOT}/scripts/resources/port_registry.sh"
    export JUPYTERHUB_PORT=$(ports::get_resource_port "jupyterhub")
    export JUPYTERHUB_API_PORT=$(ports::get_resource_port "jupyterhub-api")
    export JUPYTERHUB_PROXY_PORT=$(ports::get_resource_port "jupyterhub-proxy")
fi

# Default to standard ports if registry not available (fallback only for testing)
: ${JUPYTERHUB_PORT:=8000}
: ${JUPYTERHUB_API_PORT:=8001}
: ${JUPYTERHUB_PROXY_PORT:=8081}

# Docker configuration
export JUPYTERHUB_IMAGE="${JUPYTERHUB_IMAGE:-jupyterhub/jupyterhub:4.0}"
export JUPYTERHUB_NETWORK="${JUPYTERHUB_NETWORK:-vrooli-network}"
export JUPYTERHUB_CONTAINER_NAME="${JUPYTERHUB_CONTAINER_NAME:-vrooli-jupyterhub}"

# User notebook images
export JUPYTERHUB_NOTEBOOK_IMAGE="${JUPYTERHUB_NOTEBOOK_IMAGE:-jupyter/scipy-notebook:latest}"
export JUPYTERHUB_NOTEBOOK_IMAGE_MINIMAL="${JUPYTERHUB_NOTEBOOK_IMAGE_MINIMAL:-jupyter/base-notebook:latest}"
export JUPYTERHUB_NOTEBOOK_IMAGE_DATASCIENCE="${JUPYTERHUB_NOTEBOOK_IMAGE_DATASCIENCE:-jupyter/datascience-notebook:latest}"

# Authentication configuration
export JUPYTERHUB_AUTH_TYPE="${JUPYTERHUB_AUTH_TYPE:-native}"  # native, oauth, github, google
export JUPYTERHUB_ADMIN_USER="${JUPYTERHUB_ADMIN_USER:-admin}"
export JUPYTERHUB_ADMIN_PASSWORD="${JUPYTERHUB_ADMIN_PASSWORD:-changeme}"

# OAuth configuration (if using OAuth)
export JUPYTERHUB_OAUTH_CLIENT_ID="${JUPYTERHUB_OAUTH_CLIENT_ID:-}"
export JUPYTERHUB_OAUTH_CLIENT_SECRET="${JUPYTERHUB_OAUTH_CLIENT_SECRET:-}"
export JUPYTERHUB_OAUTH_CALLBACK_URL="${JUPYTERHUB_OAUTH_CALLBACK_URL:-http://localhost:${JUPYTERHUB_PORT}/hub/oauth_callback}"

# Resource limits per user
export JUPYTERHUB_CPU_LIMIT="${JUPYTERHUB_CPU_LIMIT:-2.0}"
export JUPYTERHUB_MEM_LIMIT="${JUPYTERHUB_MEM_LIMIT:-4G}"
export JUPYTERHUB_STORAGE_LIMIT="${JUPYTERHUB_STORAGE_LIMIT:-10G}"

# Spawner configuration
export JUPYTERHUB_SPAWNER_CLASS="${JUPYTERHUB_SPAWNER_CLASS:-dockerspawner.DockerSpawner}"
export JUPYTERHUB_SPAWNER_TIMEOUT="${JUPYTERHUB_SPAWNER_TIMEOUT:-60}"
export JUPYTERHUB_SPAWNER_START_TIMEOUT="${JUPYTERHUB_SPAWNER_START_TIMEOUT:-120}"

# Storage configuration
export JUPYTERHUB_DATA_DIR="${JUPYTERHUB_DATA_DIR:-/data/resources/jupyterhub}"
export JUPYTERHUB_USER_DATA_DIR="${JUPYTERHUB_USER_DATA_DIR:-${JUPYTERHUB_DATA_DIR}/users}"
export JUPYTERHUB_SHARED_DATA_DIR="${JUPYTERHUB_SHARED_DATA_DIR:-${JUPYTERHUB_DATA_DIR}/shared}"
export JUPYTERHUB_CONFIG_DIR="${JUPYTERHUB_CONFIG_DIR:-${JUPYTERHUB_DATA_DIR}/config}"

# Database configuration (PostgreSQL)
export JUPYTERHUB_DB_HOST="${JUPYTERHUB_DB_HOST:-vrooli-postgres}"
export JUPYTERHUB_DB_PORT="${JUPYTERHUB_DB_PORT:-5432}"
export JUPYTERHUB_DB_NAME="${JUPYTERHUB_DB_NAME:-jupyterhub}"
export JUPYTERHUB_DB_USER="${JUPYTERHUB_DB_USER:-jupyterhub}"
export JUPYTERHUB_DB_PASSWORD="${JUPYTERHUB_DB_PASSWORD:-jupyterhub_pass}"

# Redis configuration (optional)
export JUPYTERHUB_REDIS_ENABLED="${JUPYTERHUB_REDIS_ENABLED:-false}"
export JUPYTERHUB_REDIS_HOST="${JUPYTERHUB_REDIS_HOST:-vrooli-redis}"
export JUPYTERHUB_REDIS_PORT="${JUPYTERHUB_REDIS_PORT:-6379}"

# Security configuration
export JUPYTERHUB_COOKIE_SECRET="${JUPYTERHUB_COOKIE_SECRET:-}"  # Generated if empty
export JUPYTERHUB_PROXY_AUTH_TOKEN="${JUPYTERHUB_PROXY_AUTH_TOKEN:-}"  # Generated if empty
export JUPYTERHUB_API_TOKEN="${JUPYTERHUB_API_TOKEN:-}"  # Generated if empty
export JUPYTERHUB_SSL_ENABLED="${JUPYTERHUB_SSL_ENABLED:-false}"

# Performance configuration
export JUPYTERHUB_MAX_USERS="${JUPYTERHUB_MAX_USERS:-100}"
export JUPYTERHUB_CONCURRENT_SPAWN="${JUPYTERHUB_CONCURRENT_SPAWN:-10}"
export JUPYTERHUB_HUB_CPU="${JUPYTERHUB_HUB_CPU:-2}"
export JUPYTERHUB_HUB_MEMORY="${JUPYTERHUB_HUB_MEMORY:-4G}"

# Logging configuration
export JUPYTERHUB_LOG_LEVEL="${JUPYTERHUB_LOG_LEVEL:-INFO}"
export JUPYTERHUB_LOG_FORMAT="${JUPYTERHUB_LOG_FORMAT:-%(asctime)s %(levelname)s %(message)s}"
export JUPYTERHUB_ACCESS_LOG="${JUPYTERHUB_ACCESS_LOG:-true}"

# Cleanup configuration
export JUPYTERHUB_CULL_ENABLED="${JUPYTERHUB_CULL_ENABLED:-true}"
export JUPYTERHUB_CULL_TIMEOUT="${JUPYTERHUB_CULL_TIMEOUT:-3600}"  # 1 hour
export JUPYTERHUB_CULL_INTERVAL="${JUPYTERHUB_CULL_INTERVAL:-600}"  # 10 minutes

# Health check configuration
export JUPYTERHUB_HEALTH_CHECK_INTERVAL="${JUPYTERHUB_HEALTH_CHECK_INTERVAL:-30}"
export JUPYTERHUB_HEALTH_CHECK_TIMEOUT="${JUPYTERHUB_HEALTH_CHECK_TIMEOUT:-5}"
export JUPYTERHUB_HEALTH_CHECK_RETRIES="${JUPYTERHUB_HEALTH_CHECK_RETRIES:-3}"

# Extension configuration
export JUPYTERHUB_ENABLE_EXTENSIONS="${JUPYTERHUB_ENABLE_EXTENSIONS:-true}"
export JUPYTERHUB_EXTENSION_DIR="${JUPYTERHUB_EXTENSION_DIR:-${JUPYTERHUB_DATA_DIR}/extensions}"

# Backup configuration
export JUPYTERHUB_BACKUP_ENABLED="${JUPYTERHUB_BACKUP_ENABLED:-true}"
export JUPYTERHUB_BACKUP_DIR="${JUPYTERHUB_BACKUP_DIR:-${JUPYTERHUB_DATA_DIR}/backups}"
export JUPYTERHUB_BACKUP_RETENTION_DAYS="${JUPYTERHUB_BACKUP_RETENTION_DAYS:-30}"