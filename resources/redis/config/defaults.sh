#!/usr/bin/env bash
# Redis Resource Configuration Defaults
# This file contains all configuration variables for the Redis resource

# Source var.sh to get proper project paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
_REDIS_CONFIG_DIR="${APP_ROOT}/resources/redis/config"
# shellcheck disable=SC1091
source "${_REDIS_CONFIG_DIR}/../../../lib/utils/var.sh"

# Redis Docker Configuration
REDIS_IMAGE="${REDIS_IMAGE:-redis:7-alpine}"
REDIS_CONTAINER_NAME="${REDIS_CONTAINER_NAME:-vrooli-redis-resource}"
REDIS_NETWORK_NAME="${REDIS_NETWORK_NAME:-vrooli-resources}"

# Port Configuration (6380 to avoid conflict with internal Redis on 6379)
REDIS_PORT="${REDIS_PORT:-6380}"
REDIS_INTERNAL_PORT="${REDIS_INTERNAL_PORT:-6379}"

# Volume Configuration (using Docker volumes)
REDIS_VOLUME_NAME="${REDIS_VOLUME_NAME:-vrooli-redis-data}"
REDIS_LOG_VOLUME_NAME="${REDIS_LOG_VOLUME_NAME:-vrooli-redis-logs}"

# Temporary config directory (for generating config file)
REDIS_TEMP_CONFIG_DIR="${REDIS_TEMP_CONFIG_DIR:-/tmp/vrooli-redis-config}"
REDIS_CONFIG_FILE="${REDIS_CONFIG_FILE:-${REDIS_TEMP_CONFIG_DIR}/redis.conf}"

# Redis Configuration
REDIS_MAX_MEMORY="${REDIS_MAX_MEMORY:-2gb}"
REDIS_MAX_MEMORY_POLICY="${REDIS_MAX_MEMORY_POLICY:-allkeys-lru}"
REDIS_PERSISTENCE="${REDIS_PERSISTENCE:-rdb}"  # Options: rdb, aof, both, none
REDIS_SAVE_INTERVALS="${REDIS_SAVE_INTERVALS:-900 1 300 10 60 10000}"  # Default RDB save points
REDIS_AOF_ENABLED="${REDIS_AOF_ENABLED:-no}"  # Append-only file
REDIS_PASSWORD="${REDIS_PASSWORD:-}"  # Empty by default for development
REDIS_DATABASES="${REDIS_DATABASES:-16}"  # Number of databases

# Network Configuration
REDIS_BIND="${REDIS_BIND:-0.0.0.0}"  # Allow connections from any IP in container
REDIS_PROTECTED_MODE="${REDIS_PROTECTED_MODE:-no}"  # Disabled for local development

# Logging Configuration
REDIS_LOGLEVEL="${REDIS_LOGLEVEL:-notice}"  # Options: debug, verbose, notice, warning
REDIS_LOGFILE="${REDIS_LOGFILE:-/dev/stdout}"  # Log to stdout (better for Docker logging)

# Performance Configuration
REDIS_TCP_BACKLOG="${REDIS_TCP_BACKLOG:-511}"
REDIS_TIMEOUT="${REDIS_TIMEOUT:-0}"  # Client idle timeout (0 = disabled)
REDIS_TCP_KEEPALIVE="${REDIS_TCP_KEEPALIVE:-300}"

# Client Configuration for Multi-Tenant Support
REDIS_CLIENT_PREFIX="${REDIS_CLIENT_PREFIX:-vrooli-client}"
REDIS_CLIENT_PORT_START="${REDIS_CLIENT_PORT_START:-6381}"
REDIS_CLIENT_PORT_END="${REDIS_CLIENT_PORT_END:-6399}"

# Health Check Configuration
REDIS_HEALTH_CHECK_INTERVAL="${REDIS_HEALTH_CHECK_INTERVAL:-30s}"
REDIS_HEALTH_CHECK_TIMEOUT="${REDIS_HEALTH_CHECK_TIMEOUT:-5s}"
REDIS_HEALTH_CHECK_RETRIES="${REDIS_HEALTH_CHECK_RETRIES:-3}"

# Initialization Wait Times
REDIS_INITIALIZATION_WAIT="${REDIS_INITIALIZATION_WAIT:-2}"
REDIS_READY_TIMEOUT="${REDIS_READY_TIMEOUT:-30}"

#######################################
# Export all Redis configuration variables
#######################################
redis::export_config() {
    export REDIS_IMAGE
    export REDIS_CONTAINER_NAME
    export REDIS_NETWORK_NAME
    export REDIS_PORT
    export REDIS_INTERNAL_PORT
    export REDIS_VOLUME_NAME
    export REDIS_LOG_VOLUME_NAME
    export REDIS_TEMP_CONFIG_DIR
    export REDIS_CONFIG_FILE
    export REDIS_MAX_MEMORY
    export REDIS_MAX_MEMORY_POLICY
    export REDIS_PERSISTENCE
    export REDIS_SAVE_INTERVALS
    export REDIS_AOF_ENABLED
    export REDIS_PASSWORD
    export REDIS_DATABASES
    export REDIS_BIND
    export REDIS_PROTECTED_MODE
    export REDIS_LOGLEVEL
    export REDIS_LOGFILE
    export REDIS_TCP_BACKLOG
    export REDIS_TIMEOUT
    export REDIS_TCP_KEEPALIVE
    export REDIS_CLIENT_PREFIX
    export REDIS_CLIENT_PORT_START
    export REDIS_CLIENT_PORT_END
    export REDIS_HEALTH_CHECK_INTERVAL
    export REDIS_HEALTH_CHECK_TIMEOUT
    export REDIS_HEALTH_CHECK_RETRIES
    export REDIS_INITIALIZATION_WAIT
    export REDIS_READY_TIMEOUT
}